//go:build linux || darwin

package hostcompat

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestDeclarationFIFORejectedBeforeOpen(t *testing.T) {
	if path := os.Getenv("STATETWIN_TEST_DECLARATION_FIFO"); path != "" {
		if r, err := Load(path); r != nil || err == nil || err.Error() != "HOST_REPORT_FILE_INVALID" {
			t.Fatal("FIFO report admitted", err)
		}
		if target, err := LoadTarget(path); target != nil || err == nil || err.Error() != "HOST_TARGET_FILE_INVALID" {
			t.Fatal("FIFO target admitted", err)
		}
		return
	}
	path := filepath.Join(t.TempDir(), "input.fifo")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	// A regression to os.Open would block without a writer. Isolate the read
	// in a bounded child so the test reports a failure instead of hanging CI.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDeclarationFIFORejectedBeforeOpen$")
	cmd.Env = append(os.Environ(), "STATETWIN_TEST_DECLARATION_FIFO="+path)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("FIFO admission failed: %v (deadline: %v)\n%s", err, ctx.Err(), output)
	}
}
