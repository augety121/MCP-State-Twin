package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	statetwinserver "github.com/augety121/mcp-state-twin/internal/server"
)

func TestHardenedHTTPServerDefaults(t *testing.T) {
	httpServer, err := hardenedHTTPServer("127.0.0.1:0", http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	if httpServer.ReadHeaderTimeout != 5*time.Second ||
		httpServer.ReadTimeout != 30*time.Second ||
		httpServer.WriteTimeout != 30*time.Second ||
		httpServer.IdleTimeout != 60*time.Second ||
		httpServer.MaxHeaderBytes != 1<<20 {
		t.Fatalf("unexpected HTTP server limits: %#v", httpServer)
	}
	admission, ok := httpServer.Handler.(*statetwinserver.AdmissionHandler)
	if !ok || admission.Stats().Limit != activeExecutionProfile.MaxInFlightRequests {
		t.Fatalf("HTTP admission is not bound to the execution profile: %#v", httpServer.Handler)
	}
	second, err := hardenedHTTPServer("127.0.0.1:0", http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	if second.Handler == httpServer.Handler {
		t.Fatal("separate listeners unexpectedly share one admission pool")
	}
}

func TestParseGlobalExecutionArgs(t *testing.T) {
	options, remaining, err := parseGlobalExecutionArgs([]string{
		"--execution-mode", "balanced", "--max-procs=2", "--memory-limit-mib", "768", "--max-inflight=6", "scenario", "--spec", "twin.yaml",
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.mode != "balanced" || options.maxProcs != "2" || options.memoryMiB != "768" || options.maxInFlight != "6" {
		t.Fatalf("unexpected options: %#v", options)
	}
	if strings.Join(remaining, " ") != "scenario --spec twin.yaml" {
		t.Fatalf("unexpected remaining arguments: %q", remaining)
	}
	for _, args := range [][]string{
		{"--execution-mode"},
		{"--max-procs="},
		{"--memory-limit-mib"},
		{"--execution-mode", "quiet", "--execution-mode", "balanced", "version"},
	} {
		if _, _, err := parseGlobalExecutionArgs(args); err == nil {
			t.Fatalf("expected invalid root options to fail: %q", args)
		}
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

func TestProviderSmokeRequiresExplicitEvidenceInputsAndEnvironmentKey(t *testing.T) {
	if err := runProviderSmoke(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "smoke subcommand") {
		t.Fatalf("missing provider subcommand error = %v", err)
	}
	if err := runProviderSmoke(context.Background(), []string{"smoke"}); err == nil || !strings.Contains(err.Error(), "--provider") {
		t.Fatalf("missing provider flags error = %v", err)
	}
	t.Setenv("OPENAI_API_KEY", "")
	err := runProviderSmoke(context.Background(), []string{"smoke", "--provider", "openai", "--model", "gpt-test", "--mcp-url", "https://example.invalid/mcp", "--prompt", "use tool", "--out", filepath.Join(t.TempDir(), "report.json"), "--synthetic-only"})
	if err == nil || !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("missing provider credential error = %v", err)
	}
}

func TestBundleAndEpisodeCommandsFailClosed(t *testing.T) {
	if err := runBundle(nil); err == nil || !strings.Contains(err.Error(), "build or verify") {
		t.Fatalf("missing bundle subcommand error = %v", err)
	}
	if err := runBundle([]string{"publish"}); err == nil || !strings.Contains(err.Error(), "build or verify") {
		t.Fatalf("unsupported bundle subcommand error = %v", err)
	}
	if err := runBundle([]string{"build"}); err == nil || !strings.Contains(err.Error(), "--manifest and --out") {
		t.Fatalf("missing bundle build arguments error = %v", err)
	}
	if err := runBundle([]string{"verify", "bundle.stb"}); err == nil || !strings.Contains(err.Error(), "positional") {
		t.Fatalf("positional bundle argument error = %v", err)
	}
	if err := runEpisode(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "coordinator") {
		t.Fatalf("missing episode subcommand error = %v", err)
	}
	if err := runEpisode(context.Background(), []string{"run"}); err == nil || !strings.Contains(err.Error(), "--bundle and --id") {
		t.Fatalf("missing episode arguments error = %v", err)
	}
	if err := runEpisode(context.Background(), []string{"start"}); err == nil || !strings.Contains(err.Error(), "coordinator") {
		t.Fatalf("unsupported episode subcommand error = %v", err)
	}
	if err := runEpisode(context.Background(), []string{"inspect"}); err == nil || !strings.Contains(err.Error(), "--journal and --id") {
		t.Fatalf("missing Episode inspect arguments error = %v", err)
	}
	if err := runEpisode(context.Background(), []string{"inspect", "episode-1"}); err == nil || !strings.Contains(err.Error(), "positional") {
		t.Fatalf("positional Episode inspect error = %v", err)
	}
	if err := runEpisode(context.Background(), []string{"submit"}); err == nil || !strings.Contains(err.Error(), "--bundle") {
		t.Fatalf("missing Episode submit error = %v", err)
	}
	if err := runEpisode(context.Background(), []string{"worker"}); err == nil || !strings.Contains(err.Error(), "--coordinator") {
		t.Fatalf("missing Episode worker error = %v", err)
	}
}

func TestEpisodeCoordinatorRefusesInsecureNonLoopbackAndMissingToken(t *testing.T) {
	t.Setenv("STATETWIN_COORDINATOR_TOKEN", "")
	if err := runEpisodeCoordinator([]string{"--journal", "ignored.db", "--addr", "0.0.0.0:8092"}); err == nil || !strings.Contains(err.Error(), "require TLS") {
		t.Fatalf("non-loopback plaintext coordinator error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "episodes.db")
	if err := runEpisodeCoordinator([]string{"--journal", path, "--tls-cert", "cert-only.pem"}); err == nil || !strings.Contains(err.Error(), "provided together") {
		t.Fatalf("partial TLS configuration error = %v", err)
	}
}

func TestEvidenceWriterRefusesExistingOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evidence.json")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONFile(path, map[string]any{"replace": true}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected existing evidence refusal, got %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing evidence changed: data=%q err=%v", data, err)
	}
}

func TestHardenedHTTPServerRejectsSlowHeaders(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	handlerCalled := make(chan struct{}, 1)
	httpServer, err := hardenedHTTPServer(listener.Addr().String(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled <- struct{}{}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err != nil {
		t.Fatal(err)
	}
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
