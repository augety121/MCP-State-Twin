package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/augety121/mcp-state-twin/internal/engine"
	"github.com/augety121/mcp-state-twin/internal/server"
	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/store"
	"github.com/augety121/mcp-state-twin/internal/world"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	ctx := context.Background()
	var err error
	switch os.Args[1] {
	case "validate":
		err = runValidate(ctx, os.Args[2:])
	case "init":
		err = runInit(ctx, os.Args[2:])
	case "call":
		err = runCall(ctx, os.Args[2:])
	case "state":
		err = runState(ctx, os.Args[2:])
	case "snapshot":
		err = runSnapshot(ctx, os.Args[2:])
	case "fork":
		err = runFork(ctx, os.Args[2:])
	case "diff":
		err = runDiff(ctx, os.Args[2:])
	case "serve":
		err = runServe(os.Args[2:])
	case "version":
		fmt.Println(server.Version)
	case "help", "-h", "--help":
		usage()
		return
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `statetwin - deterministic stateful MCP test worlds

Usage:
  statetwin validate --spec twin.yaml
  statetwin init --spec twin.yaml --fixture state.json --db twin.db --snapshot base
  statetwin call --spec twin.yaml --db twin.db --branch main --tool get_issue --input '{...}'
  statetwin state --db twin.db --branch main
  statetwin snapshot --db twin.db --branch main --name base
  statetwin fork --db twin.db --snapshot base --branch run-a
  statetwin diff --db twin.db --before run-a --after run-b
  statetwin serve --spec twin.yaml --fixture state.json --db twin.db

Control-plane authentication is read from STATETWIN_CONTROL_TOKEN.`)
}

func runValidate(_ context.Context, args []string) error {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	specPath := flags.String("spec", "", "TwinSpec YAML path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *specPath == "" {
		return errors.New("--spec is required")
	}
	twin, err := spec.Load(*specPath)
	if err != nil {
		return err
	}
	stateStore, err := store.Open(":memory:")
	if err != nil {
		return err
	}
	defer stateStore.Close()
	runtime, err := engine.New(twin, stateStore)
	if err != nil {
		return err
	}
	return printJSON(map[string]any{"valid": true, "specDigest": runtime.Digest(), "tools": runtime.ToolNames()})
}

func runInit(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	specPath := flags.String("spec", "", "TwinSpec YAML path")
	fixturePath := flags.String("fixture", "", "initial state JSON path")
	dbPath := flags.String("db", "statetwin.db", "SQLite database path")
	branch := flags.String("branch", "main", "initial branch ID")
	snapshot := flags.String("snapshot", "base", "initial snapshot name; empty disables")
	if err := flags.Parse(args); err != nil {
		return err
	}
	runtime, stateStore, err := loadRuntime(*specPath, *dbPath)
	if err != nil {
		return err
	}
	defer stateStore.Close()
	initial, err := loadFixture(*fixturePath)
	if err != nil {
		return err
	}
	if err := runtime.Initialize(ctx, *branch, initial); err != nil {
		return err
	}
	result := map[string]any{"branch": *branch, "specDigest": runtime.Digest()}
	if *snapshot != "" {
		created, err := stateStore.CreateSnapshot(ctx, *snapshot, *branch)
		if err != nil {
			return err
		}
		result["snapshot"] = created
	}
	return printJSON(result)
}

func runCall(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("call", flag.ContinueOnError)
	specPath := flags.String("spec", "", "TwinSpec YAML path")
	dbPath := flags.String("db", "statetwin.db", "SQLite database path")
	branch := flags.String("branch", "main", "branch ID")
	tool := flags.String("tool", "", "tool name")
	inputRaw := flags.String("input", "{}", "JSON object")
	if err := flags.Parse(args); err != nil {
		return err
	}
	runtime, stateStore, err := loadRuntime(*specPath, *dbPath)
	if err != nil {
		return err
	}
	defer stateStore.Close()
	var input map[string]any
	if err := json.Unmarshal([]byte(*inputRaw), &input); err != nil {
		return fmt.Errorf("decode --input: %w", err)
	}
	result, err := runtime.Call(ctx, *branch, *tool, input)
	if err != nil {
		return err
	}
	return printJSON(result)
}

func runState(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("state", flag.ContinueOnError)
	dbPath := flags.String("db", "statetwin.db", "SQLite database path")
	branch := flags.String("branch", "main", "branch ID")
	if err := flags.Parse(args); err != nil {
		return err
	}
	stateStore, err := store.Open(*dbPath)
	if err != nil {
		return err
	}
	defer stateStore.Close()
	value, err := stateStore.Branch(ctx, *branch)
	if err != nil {
		return err
	}
	return printJSON(value)
}

func runSnapshot(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	dbPath := flags.String("db", "statetwin.db", "SQLite database path")
	branch := flags.String("branch", "main", "branch ID")
	name := flags.String("name", "", "snapshot name")
	if err := flags.Parse(args); err != nil {
		return err
	}
	stateStore, err := store.Open(*dbPath)
	if err != nil {
		return err
	}
	defer stateStore.Close()
	value, err := stateStore.CreateSnapshot(ctx, *name, *branch)
	if err != nil {
		return err
	}
	return printJSON(value)
}

func runFork(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("fork", flag.ContinueOnError)
	dbPath := flags.String("db", "statetwin.db", "SQLite database path")
	snapshot := flags.String("snapshot", "", "snapshot name")
	branch := flags.String("branch", "", "new branch ID")
	if err := flags.Parse(args); err != nil {
		return err
	}
	stateStore, err := store.Open(*dbPath)
	if err != nil {
		return err
	}
	defer stateStore.Close()
	if err := stateStore.Fork(ctx, *snapshot, *branch); err != nil {
		return err
	}
	return printJSON(map[string]any{"branch": *branch, "snapshot": *snapshot})
}

func runDiff(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("diff", flag.ContinueOnError)
	dbPath := flags.String("db", "statetwin.db", "SQLite database path")
	before := flags.String("before", "", "before branch")
	after := flags.String("after", "", "after branch")
	if err := flags.Parse(args); err != nil {
		return err
	}
	stateStore, err := store.Open(*dbPath)
	if err != nil {
		return err
	}
	defer stateStore.Close()
	changes, err := stateStore.DiffBranches(ctx, *before, *after)
	if err != nil {
		return err
	}
	return printJSON(map[string]any{"changes": changes})
}

func runServe(args []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	specPath := flags.String("spec", "", "TwinSpec YAML path")
	fixturePath := flags.String("fixture", "", "initial state JSON path")
	dbPath := flags.String("db", "statetwin.db", "SQLite database path")
	branch := flags.String("branch", "main", "initial branch ID")
	dataAddr := flags.String("data-addr", "127.0.0.1:8090", "agent data-plane address")
	controlAddr := flags.String("control-addr", "127.0.0.1:8091", "private control-plane address")
	if err := flags.Parse(args); err != nil {
		return err
	}
	token := os.Getenv("STATETWIN_CONTROL_TOKEN")
	if token == "" {
		return errors.New("STATETWIN_CONTROL_TOKEN must be set")
	}
	if strings.ContainsAny(token, " \t\r\n") {
		return errors.New("STATETWIN_CONTROL_TOKEN must not contain whitespace")
	}
	runtime, stateStore, err := loadRuntime(*specPath, *dbPath)
	if err != nil {
		return err
	}
	defer stateStore.Close()
	initial, err := loadFixture(*fixturePath)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runtime.Initialize(ctx, *branch, initial); err != nil {
		return err
	}

	dataServer := hardenedHTTPServer(*dataAddr, server.NewDataPlane(runtime))
	controlServer := hardenedHTTPServer(*controlAddr, server.NewControlPlane(stateStore, token))
	for name, addr := range map[string]string{"data": *dataAddr, "control": *controlAddr} {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return fmt.Errorf("invalid %s address: %w", name, err)
		}
		if ip := net.ParseIP(host); ip != nil && !ip.IsLoopback() {
			log.Printf("warning: %s plane is bound outside loopback (%s); configure network policy and TLS", name, addr)
		}
	}

	errorsCh := make(chan error, 2)
	go func() {
		log.Printf("agent data plane: http://%s/mcp/%s", *dataAddr, *branch)
		errorsCh <- dataServer.ListenAndServe()
	}()
	go func() {
		log.Printf("private control plane: http://%s/v1", *controlAddr)
		errorsCh <- controlServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
	case err := <-errorsCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = dataServer.Shutdown(shutdownCtx)
	_ = controlServer.Shutdown(shutdownCtx)
	return nil
}

func hardenedHTTPServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

func loadRuntime(specPath, dbPath string) (*engine.Runtime, *store.Store, error) {
	if specPath == "" {
		return nil, nil, errors.New("--spec is required")
	}
	twin, err := spec.Load(specPath)
	if err != nil {
		return nil, nil, err
	}
	stateStore, err := store.Open(dbPath)
	if err != nil {
		return nil, nil, err
	}
	runtime, err := engine.New(twin, stateStore)
	if err != nil {
		stateStore.Close()
		return nil, nil, err
	}
	return runtime, stateStore, nil
}

func loadFixture(path string) (*world.State, error) {
	if path == "" {
		return world.New(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fixture: %w", err)
	}
	var state world.State
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&state); err != nil {
		return nil, fmt.Errorf("decode fixture: %w", err)
	}
	state.Normalize()
	return &state, nil
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
