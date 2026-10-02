package plugin

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type boundedStderr struct {
	left     atomic.Int64
	exceeded atomic.Bool
}

// Package-private observation seam for owned-child crash qualification.
type contractStageKey struct{}

func contractStage(ctx context.Context, stage string, cmd *exec.Cmd) {
	if hook, ok := ctx.Value(contractStageKey{}).(func(string, *exec.Cmd)); ok {
		hook(stage, cmd)
	}
}

func (b *boundedStderr) Write(p []byte) (int, error) {
	if b.left.Add(-int64(len(p))) < 0 {
		b.exceeded.Store(true)
		return 0, errors.New("PLUGIN_RESOURCE_LIMIT")
	}
	return len(p), nil
}

// RunContract executes a synthetic witness over real stdio in an independently
// owned child. It is explicitly contract evidence, never a model evaluation.
func RunContract(parent context.Context, command []string, root, planName string, w *agenteval.Witness) (*Report, error) {
	return runContract(parent, command, root, planName, w, &ContractMetrics{})
}

type ContractMetrics struct {
	PreflightSeconds      float64 `json:"preflightSeconds"`
	StartupSeconds        float64 `json:"startupSeconds"`
	ToolsSeconds          float64 `json:"toolsSeconds"`
	GradingAndSealSeconds float64 `json:"gradingAndSealSeconds"`
	CleanupSeconds        float64 `json:"cleanupSeconds"`
	InspectionSeconds     float64 `json:"inspectionSeconds"`
	TotalSeconds          float64 `json:"totalSeconds"`
	PeakRSSBytes          int64   `json:"peakRssBytes"`
	RSSStatus             string  `json:"rssStatus"`
}

func RunContractMeasured(parent context.Context, command []string, root, planName string, w *agenteval.Witness) (*Report, ContractMetrics, error) {
	var metrics ContractMetrics
	r, e := runContract(parent, command, root, planName, w, &metrics)
	return r, metrics, e
}
func runContract(parent context.Context, command []string, root, planName string, w *agenteval.Witness, metrics *ContractMetrics) (*Report, error) {
	started := time.Now()
	phase := started
	metrics.RSSStatus = "unavailable"
	defer func() { metrics.TotalSeconds = time.Since(started).Seconds() }()
	p, err := Prepare(parent, root, planName)
	if err != nil {
		return nil, err
	}
	metrics.PreflightSeconds = time.Since(phase).Seconds()
	if len(command) == 0 || w == nil || !w.SyntheticOnly || w.TaskID != p.Entry.Task.ID || p.Plan.Host.Name != "mcp-client" {
		return nil, errors.New("PLUGIN_PLAN_INVALID")
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(p.Plan.DeadlineSeconds+20)*time.Second)
	defer cancel()
	random := make([]byte, 32)
	if _, err = rand.Read(random); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(random)
	name := "statetwin-" + token[:32]
	var address string
	if runtime.GOOS == "windows" {
		address = `\\.\pipe\` + name
	} else {
		dir, e := os.MkdirTemp("/tmp", "stp-")
		if e != nil {
			return nil, e
		}
		defer os.RemoveAll(dir)
		address = filepath.Join(dir, name+".sock")
	}
	args := append(append([]string{}, command[1:]...), "plugin", "serve", "--root", root, "--session-plan", planName)
	cmd := exec.CommandContext(ctx, command[0], args...)
	cmd.Env = append(os.Environ(), "STATETWIN_PLUGIN_CONTROL="+address, "STATETWIN_PLUGIN_CONTROL_TOKEN="+token)
	stderr := &boundedStderr{}
	stderr.left.Store(256 << 10)
	cmd.Stderr = stderr
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		in.Close()
		return nil, err
	}
	phase = time.Now()
	if err = cmd.Start(); err != nil {
		in.Close()
		out.Close()
		return nil, errors.New("PLUGIN_STARTUP_FAILED")
	}
	measure := residentMeasure(cmd)
	defer func() {
		if rss, ok := measure(); ok {
			metrics.PeakRSSBytes = rss
			metrics.RSSStatus = "measured"
		}
	}()
	waited := false
	defer func() {
		in.Close()
		out.Close()
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	contractStage(ctx, "started", cmd)
	client := mcp.NewClient(&mcp.Implementation{Name: "StateTwinContract", Version: "v1"}, nil)
	session, err := client.Connect(ctx, &mcp.IOTransport{Reader: out, Writer: in, MaxLineLength: 1 << 20}, &mcp.ClientSessionOptions{ProtocolVersion: p.Pack.Profile.ProtocolProfile})
	if err != nil {
		return nil, errors.New("PLUGIN_STARTUP_FAILED")
	}
	defer session.Close()
	contractStage(ctx, "initialized", cmd)
	// Modern tools-first clients establish readiness through tools/list.
	listed, err := session.ListTools(ctx, nil)
	if err != nil || len(listed.Tools) != len(p.Entry.Task.Tools) {
		return nil, errors.New("PLUGIN_PROJECTION_INVALID")
	}
	c, err := connectControl(ctx, address)
	if err != nil {
		return nil, errors.New("PLUGIN_CONTROL_DENIED")
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(time.Duration(p.Plan.DeadlineSeconds+10) * time.Second))
	if json.NewEncoder(c).Encode(hello{"v1", p.Plan.ID, token}) != nil {
		return nil, errors.New("PLUGIN_CONTROL_DENIED")
	}
	scan := bufio.NewScanner(c)
	scan.Buffer(make([]byte, 4096), 64<<10)
	var ready ControlResponse
	if readControl(scan, &ready) != nil || ready.Status != "ready" {
		return nil, errors.New("PLUGIN_STARTUP_FAILED")
	}
	contractStage(ctx, "ready", cmd)
	metrics.StartupSeconds = time.Since(phase).Seconds()
	phase = time.Now()
	for _, call := range w.Calls {
		_, err = session.CallTool(ctx, &mcp.CallToolParams{Name: call.Tool, Arguments: call.Input})
		if err != nil {
			return nil, errors.New("PLUGIN_MCP_REQUEST_FAILED")
		}
	}
	contractStage(ctx, "called", cmd)
	metrics.ToolsSeconds = time.Since(phase).Seconds()
	phase = time.Now()
	request := ControlRequest{Version: "v1", SessionID: p.Plan.ID, Sequence: 1, Operation: "finish", HostOutcome: "completed", Answer: w.Answer}
	if json.NewEncoder(c).Encode(request) != nil {
		return nil, errors.New("PLUGIN_CONTROL_DENIED")
	}
	var response ControlResponse
	if readControl(scan, &response) != nil || response.Status != "sealed" || response.Report == nil {
		return nil, errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	contractStage(ctx, "sealed", cmd)
	metrics.GradingAndSealSeconds = time.Since(phase).Seconds()
	phase = time.Now()
	c.Close()
	in.Close()
	_ = session.Close()
	err = cmd.Wait()
	waited = true
	metrics.CleanupSeconds = time.Since(phase).Seconds()
	phase = time.Now()
	if err != nil || stderr.exceeded.Load() {
		return response.Report, errors.New("PLUGIN_EXECUTION_FAILED")
	}
	checked, err := Inspect(ctx, root, planName)
	metrics.InspectionSeconds = time.Since(phase).Seconds()
	if err != nil {
		return checked, err
	}
	return checked, nil
}

var _ io.Writer = (*boundedStderr)(nil)
