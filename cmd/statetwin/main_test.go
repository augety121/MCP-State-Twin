package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHardenedHTTPServerDefaults(t *testing.T) {
	server := hardenedHTTPServer("127.0.0.1:0", http.NotFoundHandler())
	if server.ReadHeaderTimeout != 5*time.Second ||
		server.ReadTimeout != 30*time.Second ||
		server.WriteTimeout != 30*time.Second ||
		server.IdleTimeout != 60*time.Second ||
		server.MaxHeaderBytes != 1<<20 {
		t.Fatalf("unexpected HTTP server limits: %#v", server)
	}
}

func TestRunCompatibilityRequiresValidatedReport(t *testing.T) {
	if err := runCompatibility(nil); err == nil || !strings.Contains(err.Error(), "validate subcommand") {
		t.Fatalf("missing compatibility subcommand error = %v", err)
	}
	if err := runCompatibility([]string{"validate"}); err == nil || !strings.Contains(err.Error(), "--report") {
		t.Fatalf("missing compatibility report error = %v", err)
	}
	if err := runCompatibility([]string{"publish"}); err == nil || !strings.Contains(err.Error(), "validate subcommand") {
		t.Fatalf("unsupported compatibility subcommand error = %v", err)
	}
	if err := runCompatibility([]string{"validate", "report.yaml"}); err == nil || !strings.Contains(err.Error(), "positional") {
		t.Fatalf("positional compatibility report error = %v", err)
	}
}

func TestHardenedHTTPServerRejectsSlowHeaders(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	handlerCalled := make(chan struct{}, 1)
	httpServer := hardenedHTTPServer(listener.Addr().String(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled <- struct{}{}
		w.WriteHeader(http.StatusNoContent)
	}))
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- httpServer.Serve(listener)
	}()
	t.Cleanup(func() {
		_ = httpServer.Close()
		if serveErr := <-serveErrors; serveErr != nil && serveErr != http.ErrServerClosed {
			t.Errorf("HTTP server stopped with %v", serveErr)
		}
	})

	connection, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err := fmt.Fprint(connection, "GET / HTTP/1.1\r\nHost: local\r\nX-Slow:"); err != nil {
		t.Fatal(err)
	}

	time.Sleep(httpServer.ReadHeaderTimeout + 250*time.Millisecond)
	if err := connection.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	_, _ = fmt.Fprint(connection, " complete\r\n\r\n")
	response, readErr := http.ReadResponse(bufio.NewReader(connection), &http.Request{Method: http.MethodGet})
	if readErr == nil {
		defer response.Body.Close()
		if response.StatusCode < http.StatusBadRequest {
			t.Fatalf("slow header response status = %d, want connection close or 4xx", response.StatusCode)
		}
	} else {
		var networkError net.Error
		if errors.As(readErr, &networkError) && networkError.Timeout() {
			t.Fatalf("server left the slow-header connection open past the read deadline: %v", readErr)
		}
	}
	select {
	case <-handlerCalled:
		t.Fatal("slow header request reached the application handler")
	default:
	}
}

func TestLoadFixtureRejectsTrailingAndOversizedDocuments(t *testing.T) {
	directory := t.TempDir()
	trailing := filepath.Join(directory, "trailing.json")
	if err := os.WriteFile(trailing, []byte(`{"entities":{},"sequences":{}} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadFixture(trailing); err == nil || !strings.Contains(err.Error(), "one JSON value") {
		t.Fatalf("expected trailing JSON rejection, got %v", err)
	}

	oversized := filepath.Join(directory, "oversized.json")
	if err := os.WriteFile(oversized, []byte(strings.Repeat(" ", maxFixtureBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadFixture(oversized); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected fixture size rejection, got %v", err)
	}
}

func TestServeRejectsWhitespaceControlToken(t *testing.T) {
	t.Setenv("STATETWIN_CONTROL_TOKEN", "not a token")
	err := runServe([]string{"--spec", "does-not-matter.yaml"})
	if err == nil || !strings.Contains(err.Error(), "must not contain whitespace") {
		t.Fatalf("runServe error = %v", err)
	}
}
