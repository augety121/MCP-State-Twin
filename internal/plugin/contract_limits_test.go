package plugin

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/testfixture"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Only spawned copies enter this helper. The pipe holder exits when its test
// closes the private connection, including when the regression assertion fails.
func TestPluginContractLimitsHelper(t *testing.T) {
	var args []string
	for i, arg := range os.Args {
		if arg == "--contract-limits" {
			args = os.Args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		return
	}
	mode, address := args[0], args[1]
	if mode == "holder" {
		c, err := net.DialTimeout("tcp", address, 5*time.Second)
		if err != nil {
			os.Exit(2)
		}
		_ = c.SetDeadline(time.Now().Add(45 * time.Second))
		_, _ = c.Write([]byte{1})
		_, _ = io.Copy(io.Discard, c)
		os.Exit(0)
	}
	root, name := args[5], args[7]
	if strings.HasPrefix(mode, "inherited-") {
		binary, _ := os.Executable()
		holder := exec.Command(binary, "-test.run=^TestPluginContractLimitsHelper$", "--", "--contract-limits", "holder", address)
		holder.Stderr = os.Stderr
		if holder.Start() != nil {
			os.Exit(2)
		}
		if mode == "inherited-startup" {
			os.Exit(0)
		}
	}
	p, err := Prepare(context.Background(), root, name)
	if err != nil {
		os.Exit(2)
	}
	if mode == "inherited-sealed" || mode == "inherited-nonzero" || mode == "inherited-linger" || strings.HasPrefix(mode, "sealed-") {
		_, err = Serve(context.Background(), root, p, os.Stdin, os.Stdout, os.Getenv("STATETWIN_PLUGIN_CONTROL"), os.Getenv("STATETWIN_PLUGIN_CONTROL_TOKEN"))
		if err != nil {
			os.Exit(2)
		}
		if strings.HasSuffix(mode, "linger") {
			time.Sleep(30 * time.Second)
		}
		if strings.HasSuffix(mode, "nonzero") {
			os.Exit(7)
		}
		os.Exit(0)
	}
	if mode == "initialize" {
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "contract-startup-test", Version: "v1"}, nil)
	for _, name := range p.Entry.Task.Tools {
		server.AddTool(&mcp.Tool{Name: name, InputSchema: map[string]any{"type": "object"}}, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, error) { return nil, nil })
	}
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if mode == "tools" && method == "tools/list" {
				<-ctx.Done()
				return nil, ctx.Err()
			}
			return next(ctx, method, req)
		}
	})
	l, err := listenControl(os.Getenv("STATETWIN_PLUGIN_CONTROL"))
	if err != nil {
		os.Exit(2)
	}
	_, err = server.Connect(context.Background(), &mcp.StdioTransport{}, nil)
	if err != nil {
		os.Exit(2)
	}
	c, err := l.Accept()
	if err != nil {
		os.Exit(2)
	}
	_, _ = io.Copy(io.Discard, c) // Deliberately never acknowledge private ready.
	os.Exit(0)
}

func contractLimitFixture(t *testing.T) (string, *agenteval.Witness) {
	t.Helper()
	root := testfixture.Baseline(t)
	raw, err := os.ReadFile(filepath.Join(root, "baseline-pack.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pack baselinepack.Pack
	if json.Unmarshal(raw, &pack) != nil {
		t.Fatal("pack decode")
	}
	pack.Entries = pack.Entries[:1]
	raw, _ = json.Marshal(pack)
	if err = os.WriteFile(filepath.Join(root, "baseline-pack.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"issue-tracker/agent-tasks/read-issue.json", "issue-tracker/reviewed/tasks/read-issue.json"} {
		raw, err = os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		raw = []byte(strings.Replace(string(raw), `"cleanupSeconds": 10`, `"cleanupSeconds": 1`, 1))
		if err = os.WriteFile(filepath.Join(root, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	raw, err = os.ReadFile(filepath.Join(root, "issue-tracker/agent-witnesses/read-issue.json"))
	if err != nil {
		t.Fatal(err)
	}
	w, err := agenteval.DecodeWitness(raw)
	if err != nil {
		t.Fatal(err)
	}
	return root, w
}

func TestPluginContractInheritedStderrCleanup(t *testing.T) {
	for _, mode := range []string{"inherited-startup", "inherited-sealed", "inherited-nonzero", "inherited-linger"} {
		t.Run(mode, func(t *testing.T) {
			root, witness := contractLimitFixture(t)
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			_ = listener.(*net.TCPListener).SetDeadline(time.Now().Add(20 * time.Second))
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			binary, _ := os.Executable()
			type result struct {
				report *Report
				err    error
			}
			done := make(chan result, 1)
			var sealed time.Time
			ctx = context.WithValue(ctx, contractStageKey{}, func(stage string, _ *exec.Cmd) {
				if stage == "sealed" {
					sealed = time.Now()
				}
			})
			go func() {
				r, e := RunContract(ctx, []string{binary, "-test.run=^TestPluginContractLimitsHelper$", "--", "--contract-limits", mode, listener.Addr().String()}, root, "plugin-session.json", witness)
				done <- result{r, e}
			}()
			conn, err := listener.Accept()
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close() // Releases only this test's stderr-holding descendant.
			_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
			var ready [1]byte
			if _, err = io.ReadFull(conn, ready[:]); err != nil {
				t.Fatal(err)
			}
			select {
			case got := <-done:
				if got.err == nil {
					t.Fatal("incomplete cleanup accepted")
				}
				if mode != "inherited-startup" && (got.err.Error() != "PLUGIN_CLEANUP_FAILED" || got.report == nil || got.report.Cleanup != "failed" || got.report.Decision != "invalid") {
					t.Fatal(got.report, got.err)
				}
				if mode == "inherited-linger" && (sealed.IsZero() || time.Since(sealed) >= 2*time.Second) {
					t.Fatal("child exit and stderr drain used consecutive cleanup budgets", time.Since(sealed))
				}
			case <-time.After(8 * time.Second):
				conn.Close()
				cancel()
				<-done
				t.Fatal("child Wait exceeded cleanup budget while descendant held stderr")
			}
		})
	}
}

func TestPluginContractSealedChildFailure(t *testing.T) {
	for _, mode := range []string{"sealed-linger", "sealed-nonzero"} {
		t.Run(mode, func(t *testing.T) {
			root, witness := contractLimitFixture(t)
			binary, _ := os.Executable()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var child *exec.Cmd
			var sealed time.Time
			ctx = context.WithValue(ctx, contractStageKey{}, func(stage string, cmd *exec.Cmd) {
				if stage == "sealed" {
					child, sealed = cmd, time.Now()
				}
			})
			r, err := RunContract(ctx, []string{binary, "-test.run=^TestPluginContractLimitsHelper$", "--", "--contract-limits", mode, "unused"}, root, "plugin-session.json", witness)
			if err == nil || r == nil || r.Decision != "invalid" || child == nil || child.ProcessState == nil || time.Since(sealed) > 3*time.Second {
				t.Fatal("sealed child failure accepted or cleanup deadline exceeded", r, err, time.Since(sealed))
			}
			if mode == "sealed-linger" && (r.Cleanup != "failed" || err.Error() != "PLUGIN_CLEANUP_FAILED") {
				t.Fatal(r, err)
			}
			if mode == "sealed-nonzero" && (r.Execution != "failed" || r.Cleanup != "complete" || err.Error() != "PLUGIN_EXECUTION_FAILED") {
				t.Fatal(r, err)
			}
		})
	}
}

func TestPluginContractStartupDeadline(t *testing.T) {
	for _, mode := range []string{"initialize", "tools", "control"} {
		t.Run(mode, func(t *testing.T) {
			root, witness := contractLimitFixture(t)
			binary, _ := os.Executable()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var child *exec.Cmd
			var started time.Time
			ctx = context.WithValue(ctx, contractStageKey{}, func(stage string, cmd *exec.Cmd) {
				if stage == "started" {
					child, started = cmd, time.Now()
				}
			})
			_, err := RunContract(ctx, []string{binary, "-test.run=^TestPluginContractLimitsHelper$", "--", "--contract-limits", mode, "unused"}, root, "plugin-session.json", witness)
			if err == nil || err.Error() != "PLUGIN_STARTUP_TIMEOUT" || child == nil || child.ProcessState == nil || time.Since(started) > 15*time.Second {
				t.Fatal("startup deadline or owned-child reap failed", err, time.Since(started))
			}
		})
	}
}
