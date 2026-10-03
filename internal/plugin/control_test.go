package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPluginControlIsolationAndSequence(t *testing.T) {
	for _, condition := range []string{"token", "session", "unknown", "sequence", "forged-grade", "valid"} {
		t.Run(condition, func(t *testing.T) {
			address := testControlAddress(t)
			l, e := listenControl(address)
			if e != nil {
				t.Fatal(e)
			}
			defer l.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			ready := make(chan struct{})
			close(ready)
			var seals atomic.Int32
			done := make(chan error, 1)
			go func() {
				done <- controlLoop(ctx, l, "session", strings.Repeat("a", 64), ready, func(string, any) (*Report, error) { seals.Add(1); return &Report{Decision: "failed"}, nil })
			}()
			c, e := connectControl(ctx, address)
			if e != nil {
				t.Fatal(e)
			}
			defer c.Close()
			_ = c.SetDeadline(time.Now().Add(4 * time.Second))
			hello := hello{"v1", "session", strings.Repeat("a", 64)}
			if condition == "token" {
				hello.Token = strings.Repeat("b", 64)
			}
			if condition == "session" {
				hello.SessionID = "other"
			}
			json.NewEncoder(c).Encode(hello)
			scan := bufio.NewScanner(c)
			scan.Buffer(make([]byte, 4096), 64<<10)
			var response ControlResponse
			if condition == "token" || condition == "session" {
				if readControl(scan, &response) == nil {
					t.Fatal("unauthenticated ready")
				}
			} else {
				if readControl(scan, &response) != nil || response.Status != "ready" {
					t.Fatal("missing ready")
				}
				req := ControlRequest{Version: "v1", SessionID: "session", Sequence: 1, Operation: "finish", HostOutcome: "completed"}
				switch condition {
				case "unknown":
					req.Operation = "reset"
				case "sequence":
					req.Sequence = 2
				}
				if condition == "forged-grade" {
					c.Write([]byte(`{"version":"v1","sessionId":"session","sequence":1,"operation":"finish","hostOutcome":"completed","grade":1}` + "\n"))
				} else {
					json.NewEncoder(c).Encode(req)
				}
				e = readControl(scan, &response)
				if condition == "valid" {
					if e != nil || response.Report == nil || response.Report.Decision != "failed" {
						t.Fatal(response, e)
					}
				} else if e == nil {
					t.Fatal("invalid control accepted")
				}
			}
			c.Close()
			e = <-done
			if condition == "valid" {
				if e != nil || seals.Load() != 1 {
					t.Fatal(e, seals.Load())
				}
			} else if e == nil || seals.Load() != 0 {
				t.Fatal(e, seals.Load())
			}
		})
	}
}

func TestPluginPreflightLastReferenceNoEffects(t *testing.T) {
	// Full-pack admission precedes listener creation and output claim, including
	// entries that are not selected for this particular session.
	s := startSession(t)
	s.client.Close()
	<-s.done
	raw := s.p.Pack.Reader.Inputs["baseline-pack.json"].Raw
	var pack map[string]any
	json.Unmarshal(raw, &pack)
	entries := pack["entries"].([]any)
	entries[len(entries)-1].(map[string]any)["task"] = "missing.json"
	raw, _ = json.Marshal(pack)
	if e := os.WriteFile(filepath.Join(s.root, "baseline-pack.json"), raw, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := Prepare(context.Background(), s.root, "plugin-session.json"); e == nil {
		t.Fatal("last reference ignored")
	}
}
