package main

import (
	"net/http"
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

func TestServeRejectsWhitespaceControlToken(t *testing.T) {
	t.Setenv("STATETWIN_CONTROL_TOKEN", "not a token")
	err := runServe([]string{"--spec", "does-not-matter.yaml"})
	if err == nil || !strings.Contains(err.Error(), "must not contain whitespace") {
		t.Fatalf("runServe error = %v", err)
	}
}
