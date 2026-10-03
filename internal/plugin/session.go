package plugin

import (
	"bufio"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Report struct {
	Format          string `json:"format"`
	Plan            Plan   `json:"plan"`
	PackID          string `json:"packId"`
	PackRevision    string `json:"packRevision"`
	TaskID          string `json:"taskId"`
	Lifecycle       string `json:"lifecycle"`
	Execution       string `json:"execution"`
	Evidence        string `json:"evidence"`
	Cleanup         string `json:"cleanup"`
	Decision        string `json:"decision"`
	ToolAttempts    int    `json:"toolAttempts"`
	FailureCode     string `json:"failureCode"`
	TerminalRef     string `json:"terminalRef"`
	SourceTrust     string `json:"sourceTrust"`
	HostObservation string `json:"hostObservation"`
}
type ControlRequest struct {
	Version     string `json:"version"`
	SessionID   string `json:"sessionId"`
	Sequence    int    `json:"sequence"`
	Operation   string `json:"operation"`
	HostOutcome string `json:"hostOutcome,omitempty"`
	Answer      any    `json:"answer,omitempty"`
}
type ControlResponse struct {
	Version     string  `json:"version"`
	SessionID   string  `json:"sessionId"`
	Sequence    int     `json:"sequence"`
	Status      string  `json:"status"`
	FailureCode string  `json:"failureCode,omitempty"`
	Report      *Report `json:"report,omitempty"`
}
type hello struct {
	Version   string `json:"version"`
	SessionID string `json:"sessionId"`
	Token     string `json:"token"`
}

type limitedWriter struct{ io.WriteCloser }

func (w limitedWriter) Write(p []byte) (int, error) {
	if len(p) > 1<<20 || logging.ContainsSensitive(string(p)) {
		return 0, errors.New("PLUGIN_RESOURCE_LIMIT")
	}
	n, e := w.WriteCloser.Write(p)
	if e == nil && n != len(p) {
		e = io.ErrShortWrite
	}
	return n, e
}

// Serve uses only the supplied bounded streams and an authenticated private IPC
// endpoint. The token and endpoint are never written to an artifact or stdout.
func Serve(parent context.Context, root string, p *Prepared, in io.ReadCloser, out io.WriteCloser, address, token string) (*Report, error) {
	if p == nil || p.Plan.Validate() != nil || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(token) {
		return nil, errors.New("PLUGIN_CONTROL_DENIED")
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(p.Plan.DeadlineSeconds)*time.Second)
	defer cancel()
	l, err := listenControl(address)
	if err != nil {
		return nil, err
	}
	defer l.Close()
	s, err := claim(root, p.Plan.Output, p.Plan)
	if err != nil {
		return nil, err
	}
	defer s.root.Close()
	w, err := agenteval.NewPluginWorld(ctx, p.Entry.Task, p.Entry.Bundle)
	if err != nil {
		return nil, err
	}
	defer w.Finish("cancelled", nil)
	initialized := make(chan struct{})
	var initOnce sync.Once
	server := mcp.NewServer(&mcp.Implementation{Name: "SIMULATED/StateTwinTask", Version: "baseline-plugin-v1"}, &mcp.ServerOptions{
		SupportedProtocolVersions: []string{p.Pack.Profile.ProtocolProfile},
		Logger:                    slog.New(slog.NewTextHandler(io.Discard, nil)),
		InitializedHandler:        func(context.Context, *mcp.InitializedRequest) { initOnce.Do(func() { close(initialized) }) },
	})
	for _, tool := range w.Tools() {
		server.AddTool(tool, func(callCtx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var input map[string]any
			if req.Params == nil || agenthost.DecodeDocument(req.Params.Arguments, 1<<20, &input) != nil || input == nil {
				return nil, errors.New("PLUGIN_INPUT_INVALID")
			}
			if _, err := agenthost.NormalizeNumbers(input); err != nil {
				return nil, errors.New("PLUGIN_INPUT_INVALID")
			}
			e, err := w.Call(callCtx, agenteval.Call{Tool: tool.Name, Input: input})
			if err != nil {
				return nil, err
			}
			raw, _ := json.Marshal(e.Result)
			return &mcp.CallToolResult{StructuredContent: e.Result, Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}}, IsError: e.ErrorClass != ""}, nil
		})
	}
	var frames atomic.Int32
	var controlReady atomic.Bool
	inflight := make(chan struct{}, 4)
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(c context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method == "tools/call" && !controlReady.Load() {
				return nil, errors.New("PLUGIN_STATE_CONFLICT")
			}
			if frames.Add(1) > 128 {
				cancel()
				return nil, errors.New("PLUGIN_RESOURCE_LIMIT")
			}
			select {
			case inflight <- struct{}{}:
				defer func() { <-inflight }()
			default:
				return nil, errors.New("PLUGIN_RESOURCE_LIMIT")
			}
			r, err := next(c, method, req)
			if err == nil && method == "tools/list" {
				initOnce.Do(func() { close(initialized) })
			}
			if err != nil {
				return nil, errors.New("PLUGIN_MCP_REQUEST_FAILED")
			}
			return r, nil
		}
	})
	stopIO := context.AfterFunc(ctx, func() { in.Close(); out.Close(); l.Close() })
	defer stopIO()
	session, err := server.Connect(ctx, &mcp.IOTransport{Reader: in, Writer: limitedWriter{out}, MaxLineLength: 1 << 20}, nil)
	if err != nil {
		return nil, errors.New("PLUGIN_STARTUP_FAILED")
	}
	defer session.Close()
	wireDone := make(chan struct{})
	go func() { _ = session.Wait(); close(wireDone) }()
	var sealed atomic.Bool
	controlDone := make(chan error, 1)
	var report *Report
	var sealMu sync.Mutex
	seal := func(outcome string, answer any) (*Report, error) {
		sealMu.Lock()
		defer sealMu.Unlock()
		if report != nil {
			return report, nil
		}
		terminal, e := w.Finish(outcome, answer)
		if e != nil {
			return nil, e
		}
		verified := false
		if terminal.Execution == "completed" && terminal.Cleanup == "complete" {
			checkCtx, end := context.WithTimeout(context.Background(), time.Duration(p.Entry.Task.Budgets.CleanupSeconds)*time.Second)
			verified = agenteval.VerifyPluginTerminal(checkCtx, terminal, p.Entry.Task, p.Entry.Bundle) == nil
			end()
		}
		r := reportFor(p, terminal, verified)
		if e = s.publish("terminal.json", terminal, limits.MaxReportBytes); e != nil {
			return nil, e
		}
		if e = s.publish("session-report.json", r, 1<<20); e != nil {
			return nil, e
		}
		report = r
		sealed.Store(true)
		return report, nil
	}
	go func() {
		controlDone <- controlLoop(ctx, l, p.Plan.ID, token, initialized, seal, func() { controlReady.Store(true) })
	}()
	var failure error
	controlReceived := false
	select {
	case failure = <-controlDone:
		controlReceived = true
	case <-wireDone:
		if sealed.Load() {
			failure = <-controlDone
			controlReceived = true
		} else {
			failure = errors.New("PLUGIN_INTERRUPTED")
		}
	case <-ctx.Done():
		failure = errors.New("PLUGIN_INTERRUPTED")
	}
	if !sealed.Load() {
		_, e := seal("cancelled", nil)
		if e != nil {
			failure = e
		}
	}
	cancel()
	l.Close()
	if !controlReceived {
		<-controlDone
	}
	_ = session.Close()
	<-wireDone
	// Closing the listener or the context also releases authentication/reads.
	if report == nil {
		return nil, failure
	}
	if failure != nil {
		return report, failure
	}
	return report, nil
}

func reportFor(p *Prepared, t *agenteval.PluginTerminal, verified bool) *Report {
	r := &Report{Format: ReportFormat, Plan: p.Plan, PackID: p.Pack.Pack.ID, PackRevision: p.Pack.Pack.Revision, TaskID: p.Entry.Task.ID, Lifecycle: "sealed", Execution: t.Execution, Evidence: "partial", Cleanup: t.Cleanup, Decision: "invalid", ToolAttempts: t.ToolAttempts, FailureCode: t.FailureCode, TerminalRef: "terminal.json", SourceTrust: "local-operator-asserted", HostObservation: "contract-test"}
	if verified {
		r.Evidence = "world-replay-verified"
		r.Decision = "failed"
		if t.Evaluation.Outcome == p.Entry.Task.ExpectedOutcome {
			r.Decision = "passed"
		}
	}
	if !verified && r.FailureCode == "" {
		r.FailureCode = "PLUGIN_EVIDENCE_INVALID"
	}
	return r
}

func readControl(scan *bufio.Scanner, dst any) error {
	if !scan.Scan() {
		if scan.Err() != nil {
			return errors.New("PLUGIN_CONTROL_READ_FAILED")
		}
		return io.EOF
	}
	// This private authenticated channel necessarily carries its bootstrap
	// capability. Strictly decode it, but never run artifact redaction or log it.
	if err := agenthost.DecodeDocument(scan.Bytes(), 64<<10, dst); err != nil {
		return errors.New("PLUGIN_CONTROL_DENIED")
	}
	var tree any
	if json.Unmarshal(scan.Bytes(), &tree) != nil || !controlDepth(tree, 0) {
		return errors.New("PLUGIN_RESOURCE_LIMIT")
	}
	return nil
}
func controlDepth(v any, n int) bool {
	if n > 16 {
		return false
	}
	switch x := v.(type) {
	case map[string]any:
		for _, child := range x {
			if !controlDepth(child, n+1) {
				return false
			}
		}
	case []any:
		for _, child := range x {
			if !controlDepth(child, n+1) {
				return false
			}
		}
	}
	return true
}
func sendControl(c net.Conn, r ControlResponse) error {
	raw, err := json.Marshal(r)
	if err != nil || len(raw) > 64<<10 {
		return errors.New("PLUGIN_RESOURCE_LIMIT")
	}
	_ = c.SetWriteDeadline(time.Now().Add(5 * time.Second))
	n, err := c.Write(append(raw, '\n'))
	if err == nil && n != len(raw)+1 {
		err = io.ErrShortWrite
	}
	return err
}
func controlLoop(ctx context.Context, l net.Listener, id, token string, initialized <-chan struct{}, seal func(string, any) (*Report, error), onReady ...func()) error {
	startup := time.AfterFunc(10*time.Second, func() { l.Close() })
	defer startup.Stop()
	c, err := l.Accept()
	if err != nil {
		return errors.New("PLUGIN_STARTUP_TIMEOUT")
	}
	defer c.Close()
	stop := context.AfterFunc(ctx, func() { c.Close() })
	defer stop()
	_ = c.SetReadDeadline(time.Now().Add(10 * time.Second))
	scan := bufio.NewScanner(c)
	scan.Buffer(make([]byte, 4096), 64<<10)
	var h hello
	if readControl(scan, &h) != nil || h.Version != "v1" || h.SessionID != id || subtle.ConstantTimeCompare([]byte(token), []byte(h.Token)) != 1 {
		return errors.New("PLUGIN_CONTROL_DENIED")
	}
	select {
	case <-initialized:
	case <-ctx.Done():
		return errors.New("PLUGIN_INTERRUPTED")
	case <-time.After(10 * time.Second):
		return errors.New("PLUGIN_STARTUP_TIMEOUT")
	}
	startup.Stop()
	for _, ready := range onReady {
		ready()
	}
	if sendControl(c, ControlResponse{Version: "v1", SessionID: id, Status: "ready"}) != nil {
		return errors.New("PLUGIN_CONTROL_DENIED")
	}
	_ = c.SetReadDeadline(time.Time{})
	var previous ControlRequest
	var previousResponse ControlResponse
	done := false
	for count := 0; count < 128; count++ {
		var req ControlRequest
		if err := readControl(scan, &req); err != nil {
			if done && errors.Is(err, io.EOF) {
				return nil
			}
			return errors.New("PLUGIN_INTERRUPTED")
		}
		if req.Version != "v1" || req.SessionID != id || req.Sequence < 1 {
			return errors.New("PLUGIN_CONTROL_DENIED")
		}
		if req.Sequence == previous.Sequence && baselinepack.Equal(req, previous) {
			if sendControl(c, previousResponse) != nil {
				return errors.New("PLUGIN_OUTPUT_FAILED")
			}
			continue
		}
		if req.Sequence != previous.Sequence+1 {
			return errors.New("PLUGIN_STATE_CONFLICT")
		}
		res := ControlResponse{Version: "v1", SessionID: id, Sequence: req.Sequence, Status: "running"}
		switch req.Operation {
		case "status":
			if req.HostOutcome != "" || req.Answer != nil {
				return errors.New("PLUGIN_PLAN_INVALID")
			}
			if done {
				res.Status = "sealed"
			}
		case "finish", "abort":
			if done {
				return errors.New("PLUGIN_STATE_CONFLICT")
			}
			if req.HostOutcome != "completed" && req.HostOutcome != "cancelled" && req.HostOutcome != "failed" {
				return errors.New("PLUGIN_PLAN_INVALID")
			}
			if req.Operation == "abort" && (req.HostOutcome == "completed" || req.Answer != nil) {
				return errors.New("PLUGIN_PLAN_INVALID")
			}
			res.Report, err = seal(req.HostOutcome, req.Answer)
			if err != nil {
				return err
			}
			res.Status = "sealed"
			done = true
			_ = c.SetReadDeadline(time.Now().Add(time.Second))
		default:
			return errors.New("PLUGIN_CONTROL_DENIED")
		}
		if sendControl(c, res) != nil {
			return errors.New("PLUGIN_OUTPUT_FAILED")
		}
		previous, previousResponse = req, res
	}
	return errors.New("PLUGIN_RESOURCE_LIMIT")
}
