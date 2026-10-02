package plugin

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

type shortCloser struct{}

func (shortCloser) Write(p []byte) (int, error) { return len(p) / 2, nil }
func (shortCloser) Close() error                { return nil }
func TestPluginStdioLimitsAndOutput(t *testing.T) {
	w := limitedWriter{shortCloser{}}
	if _, e := w.Write([]byte("{}\n")); !errors.Is(e, io.ErrShortWrite) {
		t.Fatal(e)
	}
	if _, e := w.Write(bytes.Repeat([]byte("x"), (1<<20)+1)); e == nil {
		t.Fatal("oversize output")
	}
	for depth := 16; depth <= 17; depth++ {
		var tree any = "leaf"
		for i := 0; i < depth; i++ {
			tree = []any{tree}
		}
		if controlDepth(tree, 0) != (depth == 16) {
			t.Fatal("depth", depth)
		}
	}
	scan := bufio.NewScanner(strings.NewReader(strings.Repeat("x", (64<<10)+1) + "\n"))
	scan.Buffer(make([]byte, 4096), 64<<10)
	var request ControlRequest
	if readControl(scan, &request) == nil {
		t.Fatal("oversize private frame")
	}
}
