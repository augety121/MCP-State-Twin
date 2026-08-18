package main

import (
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
