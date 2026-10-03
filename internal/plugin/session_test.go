package plugin

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/testfixture"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func testControlAddress(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 16)
	_, _ = rand.Read(raw)
	name := "statetwin-" + hex.EncodeToString(raw)
	if runtime.GOOS == "windows" {
		return `\\.\pipe\` + name
	}
	dir, err := os.MkdirTemp("/tmp", "stp-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, name+".sock")
}

type runningSession struct {
	client  *mcp.ClientSession
	control net.Conn
	scan    *bufio.Scanner
	done    chan error
	cancel  context.CancelFunc
	root    string
	p       *Prepared
}

func startSession(t *testing.T) *runningSession {
	t.Helper()
	root := testfixture.Baseline(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	p, err := Prepare(ctx, root, "plugin-session.json")
	if err != nil {
		t.Fatal(err)
	}
	address := testControlAddress(t)
	token := strings.Repeat("a", 64)
	serverIn, clientOut := io.Pipe()
	clientIn, serverOut := io.Pipe()
	done := make(chan error, 1)
	go func() { _, e := Serve(ctx, root, p, serverIn, serverOut, address, token); done <- e }()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-plugin", Version: "v1"}, nil)
	session, err := client.Connect(ctx, &mcp.IOTransport{Reader: clientIn, Writer: clientOut, MaxLineLength: 1 << 20}, &mcp.ClientSessionOptions{ProtocolVersion: "2025-11-25"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	conn, err := dialControl(ctx, address)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err = json.NewEncoder(conn).Encode(hello{"v1", p.Plan.ID, token}); err != nil {
		t.Fatal(err)
	}
	scan := bufio.NewScanner(conn)
	scan.Buffer(make([]byte, 4096), 64<<10)
	var ready ControlResponse
	if readControl(scan, &ready) != nil || ready.Status != "ready" {
		t.Fatal(ready)
	}
	return &runningSession{session, conn, scan, done, cancel, root, p}
}
func (s *runningSession) controlRequest(t *testing.T, req ControlRequest) ControlResponse {
	t.Helper()
	if err := json.NewEncoder(s.control).Encode(req); err != nil {
		t.Fatal(err)
	}
	var response ControlResponse
	if err := readControl(s.scan, &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func TestPluginStdioLifecycleAndInspection(t *testing.T) {
	s := startSession(t)
	ctx := context.Background()
	listed, err := s.client.ListTools(ctx, nil)
	if err != nil || len(listed.Tools) != len(s.p.Entry.Task.Tools) {
		t.Fatal(err)
	}
	for _, tool := range listed.Tools {
		if strings.Contains(tool.Name, "reset") || strings.Contains(tool.Name, "grade") {
			t.Fatal(tool.Name)
		}
	}
	_, err = s.client.CallTool(ctx, &mcp.CallToolParams{Name: "reset", Arguments: map[string]any{}})
	if err == nil {
		t.Fatal("control callable")
	}
	r, err := s.client.CallTool(ctx, &mcp.CallToolParams{Name: "get_issue", Arguments: map[string]any{"owner": "octo", "repository": "demo", "number": 1}})
	if err != nil || r.IsError {
		t.Fatal(r, err)
	}
	answer := map[string]any{"repository": "octo/demo", "number": 1, "state": "open", "title": "Target issue"}
	request := ControlRequest{Version: "v1", SessionID: s.p.Plan.ID, Sequence: 1, Operation: "finish", HostOutcome: "completed", Answer: answer}
	response := s.controlRequest(t, request)
	if response.Report == nil || response.Report.Decision != "passed" {
		t.Fatal(response)
	}
	duplicate := s.controlRequest(t, request)
	if duplicate.Report == nil || duplicate.Report.ToolAttempts != 1 {
		t.Fatal(duplicate)
	}
	s.control.Close()
	if err = <-s.done; err != nil {
		t.Fatal(err)
	}
	inspected, err := Inspect(ctx, s.root, "plugin-session.json")
	if err != nil || inspected.Decision != "passed" {
		t.Fatal(inspected, err)
	}
	recovery, err := Recover(ctx, s.root, "plugin-session.json", "recovered")
	if err != nil || recovery.Resumable || recovery.OriginalDecision != "passed" {
		t.Fatal(recovery, err)
	}
	copyReport, err := Inspect(ctx, filepath.Join(s.root, "recovered"), "plugin-session.json")
	if err != nil || copyReport.Decision != "passed" {
		t.Fatal(copyReport, err)
	}
	if _, err = Recover(ctx, s.root, "plugin-session.json", "recovered"); err == nil {
		t.Fatal("recovery overwrote output")
	}
	if err = os.WriteFile(filepath.Join(s.root, s.p.Plan.Output, "unknown.json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Inspect(ctx, s.root, "plugin-session.json"); err == nil {
		t.Fatal("unknown member")
	}
}

func TestPluginEOFAndAuthority(t *testing.T) {
	s := startSession(t)
	r, err := s.client.CallTool(context.Background(), &mcp.CallToolParams{Name: "close_issue", Arguments: map[string]any{"owner": "octo", "repository": "demo", "number": 1}})
	if err != nil || !r.IsError {
		t.Fatal("readonly Task authority bypass", r, err)
	}
	s.client.Close()
	if err = <-s.done; err == nil {
		t.Fatal("EOF reported complete")
	}
	report, err := Inspect(context.Background(), s.root, "plugin-session.json")
	if err == nil || report == nil || report.Decision != "invalid" || report.Execution == "completed" {
		t.Fatal(report, err)
	}
}
