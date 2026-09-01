package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/engine"
	"github.com/augety121/mcp-state-twin/internal/episode"
	"github.com/augety121/mcp-state-twin/internal/governor"
	"github.com/augety121/mcp-state-twin/internal/hostcompat"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/augety121/mcp-state-twin/internal/provider"
	statetwinscenario "github.com/augety121/mcp-state-twin/internal/scenario"
	"github.com/augety121/mcp-state-twin/internal/server"
	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/store"
	"github.com/augety121/mcp-state-twin/internal/world"
)

const maxFixtureBytes = 16 << 20

var activeExecutionProfile = func() governor.Profile {
	profile, err := governor.Resolve(governor.Options{})
	if err != nil {
		panic(err)
	}
	return profile
}()

func main() {
	global, args, err := parseGlobalExecutionArgs(os.Args[1:])
	if err != nil {
		log.Printf("error: %s", logging.SafeError(err))
		os.Exit(2)
	}
	profile, err := governor.Resolve(governor.Options{
		CLIMode: global.mode, CLIMaxProcs: global.maxProcs,
		CLIMemoryMiB: global.memoryMiB, CLIMaxInFlight: global.maxInFlight,
		EnvMode: os.Getenv("STATETWIN_EXECUTION_MODE"), EnvMaxProcs: os.Getenv("STATETWIN_MAX_PROCS"),
		EnvMemoryMiB: os.Getenv("STATETWIN_MEMORY_LIMIT_MIB"), EnvMaxInFlight: os.Getenv("STATETWIN_MAX_INFLIGHT"),
	})
	if err != nil {
		log.Printf("error: %s", logging.SafeError(err))
		os.Exit(2)
	}
	if _, err := governor.Apply(profile); err != nil {
		log.Printf("error: %s", logging.SafeError(err))
		os.Exit(2)
	}
	activeExecutionProfile = profile
	if len(args) < 1 {
		usage()
		os.Exit(2)
	}
	ctx := context.Background()
	switch args[0] {
	case "validate":
		err = runValidate(ctx, args[1:])
	case "init":
		err = runInit(ctx, args[1:])
	case "call":
		err = runCall(ctx, args[1:])
	case "state":
		err = runState(ctx, args[1:])
	case "snapshot":
		err = runSnapshot(ctx, args[1:])
	case "fork":
		err = runFork(ctx, args[1:])
	case "diff":
		err = runDiff(ctx, args[1:])
	case "scenario":
		err = runScenario(ctx, args[1:])
	case "serve":
		err = runServe(args[1:])
	case "version":
		fmt.Println(server.Version)
	case "protocols":
		err = printJSON(server.CurrentProtocolEvidence())
	case "limits":
		err = runLimits()
	case "execution-profile":
		err = printJSON(profile)
	case "compatibility":
		err = runCompatibility(args[1:])
	case "bundle":
		err = runBundle(args[1:])
	case "episode":
		err = runEpisode(ctx, args[1:])
	case "provider":
		err = runProviderSmoke(ctx, args[1:])
	case "help", "-h", "--help":
		usage()
		return
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		log.Printf("error: %s", logging.SafeError(err))
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `statetwin - deterministic stateful MCP test worlds

Usage:
	statetwin [--execution-mode MODE] [--max-procs N] [--memory-limit-mib N] [--max-inflight N] COMMAND

Execution policy:
  The default mode is quiet and limits the Go runtime to one logical CPU slot.
  STATETWIN_EXECUTION_MODE, STATETWIN_MAX_PROCS, STATETWIN_MEMORY_LIMIT_MIB,
  and STATETWIN_MAX_INFLIGHT provide environment defaults;
  root command-line flags take precedence. These are soft process-local controls,
  not OS/RSS hard quotas or distributed rate limiting.

Commands:
  statetwin validate --spec twin.yaml
  statetwin init --spec twin.yaml --fixture state.json --db twin.db --snapshot base
  statetwin call --spec twin.yaml --db twin.db --branch main --tool get_issue --input '{...}'
  statetwin state --db twin.db --branch main
  statetwin snapshot --db twin.db --branch main --name base
  statetwin fork --db twin.db --snapshot base --branch run-a
  statetwin diff --db twin.db --before run-a --after run-b
  statetwin scenario --spec twin.yaml --fixture state.json --scenario scenario.yaml
  statetwin serve --spec twin.yaml --fixture state.json --db twin.db
  statetwin protocols
  statetwin limits
  statetwin execution-profile
  statetwin compatibility validate --report report.yaml
  statetwin bundle build --manifest bundle.yaml --out twin.stb
  statetwin bundle verify --bundle twin.stb
  statetwin episode run --bundle twin.stb --id episode-001 [--scenario path] [--journal episodes.db]
  statetwin episode inspect --journal episodes.db --id episode-001
  statetwin episode submit --bundle twin.stb --id episode-001 --journal episodes.db [--effect-profile hermetic]
  statetwin episode task --journal episodes.db --id episode-001
  statetwin episode cancel --journal episodes.db --id episode-001
  statetwin episode coordinator --journal episodes.db [--addr 127.0.0.1:8092] [--tls-cert cert.pem --tls-key key.pem]
  statetwin episode worker --coordinator http://127.0.0.1:8092 --id worker-001 [--once]
  statetwin provider smoke --provider openai|anthropic --model MODEL --runtime-revision GIT_SHA --mcp-url https://... --prompt "..." --out report.json

Control-plane authentication is read from STATETWIN_CONTROL_TOKEN.
Episode coordinator authentication is read from STATETWIN_COORDINATOR_TOKEN.`)
}

type globalExecutionOptions struct {
	mode        string
	maxProcs    string
	memoryMiB   string
	maxInFlight string
}

func parseGlobalExecutionArgs(args []string) (globalExecutionOptions, []string, error) {
	var options globalExecutionOptions
	for len(args) > 0 {
		name, value, hasValue := strings.Cut(args[0], "=")
		switch name {
		case "--execution-mode", "--max-procs", "--memory-limit-mib", "--max-inflight":
			if !hasValue {
				if len(args) < 2 {
					return options, nil, fmt.Errorf("%s requires a value", name)
				}
				value, args = args[1], args[1:]
			}
			if strings.TrimSpace(value) == "" {
				return options, nil, fmt.Errorf("%s requires a non-empty value", name)
			}
			if name == "--execution-mode" {
				if options.mode != "" {
					return options, nil, errors.New("--execution-mode may be specified only once")
				}
				options.mode = value
			} else if name == "--max-procs" {
				if options.maxProcs != "" {
					return options, nil, errors.New("--max-procs may be specified only once")
				}
				options.maxProcs = value
			} else if name == "--memory-limit-mib" {
				if options.memoryMiB != "" {
					return options, nil, errors.New("--memory-limit-mib may be specified only once")
				}
				options.memoryMiB = value
			} else {
				if options.maxInFlight != "" {
					return options, nil, errors.New("--max-inflight may be specified only once")
				}
				options.maxInFlight = value
			}
			args = args[1:]
		default:
			return options, args, nil
		}
	}
	return options, args, nil
}

func runProviderSmoke(parent context.Context, args []string) error {
	if len(args) == 0 || args[0] != "smoke" {
		return errors.New("provider requires the smoke subcommand")
	}
	flags := flag.NewFlagSet("provider smoke", flag.ContinueOnError)
	providerName := flags.String("provider", "", "openai or anthropic")
	model := flags.String("model", "", "exact provider model identifier")
	runtimeRevision := flags.String("runtime-revision", server.Revision, "exact 40- or 64-character source revision")
	mcpURL := flags.String("mcp-url", "", "public HTTPS MCP endpoint")
	prompt := flags.String("prompt", "", "synthetic prompt that requires MCP tool use")
	outputPath := flags.String("out", "", "new ProviderSmokeReport JSON path")
	timeout := flags.Duration("timeout", 5*time.Minute, "bounded provider run timeout")
	poll := flags.Duration("poll", 2*time.Second, "OpenAI background polling interval")
	syntheticOnly := flags.Bool("synthetic-only", false, "attest that endpoint, prompt, and data are synthetic")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("provider smoke does not accept positional arguments")
	}
	if *providerName == "" || *model == "" || *mcpURL == "" || *prompt == "" || *outputPath == "" || !*syntheticOnly {
		return errors.New("--provider, --model, --mcp-url, --prompt, --out, and --synthetic-only are required")
	}
	keyName := "OPENAI_API_KEY"
	if *providerName == "anthropic" {
		keyName = "ANTHROPIC_API_KEY"
	}
	apiKey := os.Getenv(keyName)
	if apiKey == "" {
		return fmt.Errorf("%s must be set", keyName)
	}
	ctx, cancel := context.WithTimeout(parent, *timeout)
	defer cancel()
	report, err := provider.Run(ctx, provider.Request{
		Provider: *providerName, Model: *model, RuntimeVersion: server.Version, RuntimeRevision: *runtimeRevision,
		Prompt: *prompt, MCPServerURL: *mcpURL,
		MCPAuthorization: os.Getenv("STATETWIN_MCP_AUTHORIZATION"), APIKey: apiKey,
		Timeout: *timeout, PollInterval: *poll, SyntheticOnly: true,
	})
	if report != nil {
		if writeErr := writeJSONFile(*outputPath, report); writeErr != nil {
			return errors.Join(err, writeErr)
		}
		if printErr := printJSON(report); printErr != nil {
			return errors.Join(err, printErr)
		}
	}
	return err
}

func runBundle(args []string) error {
	if len(args) == 0 {
		return errors.New("bundle requires build or verify subcommand")
	}
	switch args[0] {
	case "build":
		flags := flag.NewFlagSet("bundle build", flag.ContinueOnError)
		manifestPath := flags.String("manifest", "", "TwinBundle source manifest path")
		outputPath := flags.String("out", "", "output .stb path")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("bundle build does not accept positional arguments")
		}
		if *manifestPath == "" || *outputPath == "" {
			return errors.New("--manifest and --out are required")
		}
		result, err := bundle.Build(*manifestPath, *outputPath)
		if err != nil {
			return err
		}
		return printJSON(result)
	case "verify":
		flags := flag.NewFlagSet("bundle verify", flag.ContinueOnError)
		bundlePath := flags.String("bundle", "", "TwinBundle .stb path")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		if flags.NArg() != 0 {
			return errors.New("bundle verify does not accept positional arguments")
		}
		if *bundlePath == "" {
			return errors.New("--bundle is required")
		}
		result, err := bundle.Verify(*bundlePath)
		if err != nil {
			return err
		}
		return printJSON(result)
	default:
		return errors.New("bundle requires build or verify subcommand")
	}
}

func runEpisode(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("episode requires run, inspect, submit, task, cancel, coordinator, or worker subcommand")
	}
	switch args[0] {
	case "run":
		return runEpisodeRun(ctx, args[1:])
	case "inspect":
		return runEpisodeInspect(ctx, args[1:])
	case "submit":
		return runEpisodeSubmit(ctx, args[1:])
	case "task":
		return runEpisodeTask(ctx, args[1:])
	case "cancel":
		return runEpisodeCancel(ctx, args[1:])
	case "coordinator":
		return runEpisodeCoordinator(args[1:])
	case "worker":
		return runEpisodeWorker(args[1:])
	default:
		return errors.New("episode requires run, inspect, submit, task, cancel, coordinator, or worker subcommand")
	}
}

func runEpisodeRun(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("episode run", flag.ContinueOnError)
	bundlePath := flags.String("bundle", "", "TwinBundle .stb path")
	episodeID := flags.String("id", "", "stable local episode identifier")
	scenarioPath := flags.String("scenario", "", "declared Scenario path; optional for one-scenario bundles")
	outputPath := flags.String("out", "", "optional evidence JSON output path")
	journalPath := flags.String("journal", "", "optional durable local Episode Journal SQLite path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("episode run does not accept positional arguments")
	}
	if *bundlePath == "" || *episodeID == "" {
		return errors.New("--bundle and --id are required")
	}
	artifact, err := bundle.Open(*bundlePath)
	if err != nil {
		return err
	}
	var evidence *episode.Envelope
	if *journalPath == "" {
		evidence, err = episode.Run(ctx, artifact, *episodeID, *scenarioPath, server.Version, server.Revision)
	} else {
		journal, openErr := episode.OpenJournal(*journalPath)
		if openErr != nil {
			return openErr
		}
		defer journal.Close()
		evidence, _, err = journal.RunPersistent(ctx, artifact, *episodeID, *scenarioPath, server.Version, server.Revision)
	}
	if err != nil {
		return err
	}
	if *outputPath != "" {
		if err := writeJSONFile(*outputPath, evidence); err != nil {
			return err
		}
	}
	if err := printJSON(evidence); err != nil {
		return err
	}
	if evidence.Evidence == nil || evidence.Evidence.Report == nil || !evidence.Evidence.Report.Passed {
		return errors.New("episode assertions failed")
	}
	return nil
}

func runEpisodeInspect(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("episode inspect", flag.ContinueOnError)
	journalPath := flags.String("journal", "", "durable local Episode Journal SQLite path")
	episodeID := flags.String("id", "", "Episode identifier")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("episode inspect does not accept positional arguments")
	}
	if *journalPath == "" || *episodeID == "" {
		return errors.New("--journal and --id are required")
	}
	journal, err := episode.OpenJournal(*journalPath)
	if err != nil {
		return err
	}
	defer journal.Close()
	record, err := journal.Get(ctx, *episodeID)
	if err != nil {
		return err
	}
	return printJSON(record)
}

func runEpisodeSubmit(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("episode submit", flag.ContinueOnError)
	bundlePath := flags.String("bundle", "", "TwinBundle .stb path")
	episodeID := flags.String("id", "", "immutable Episode identifier")
	scenarioPath := flags.String("scenario", "", "declared Scenario path")
	journalPath := flags.String("journal", "", "Episode Journal SQLite path")
	effectProfile := flags.String("effect-profile", string(episode.EffectHermetic), "hermetic or external")
	maxAttempts := flags.Int("max-attempts", 3, "bounded attempt budget")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("episode submit does not accept positional arguments")
	}
	if *bundlePath == "" || *episodeID == "" || *journalPath == "" {
		return errors.New("--bundle, --id, and --journal are required")
	}
	info, err := os.Lstat(*bundlePath)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > int64(limits.MaxBundleCompressed) {
		return errors.New("TwinBundle must be a bounded regular non-symlink file")
	}
	data, err := os.ReadFile(*bundlePath)
	if err != nil {
		return err
	}
	journal, err := episode.OpenJournal(*journalPath)
	if err != nil {
		return err
	}
	defer journal.Close()
	task, _, err := journal.Submit(ctx, data, *episodeID, *scenarioPath, server.Version, server.Revision, episode.EffectProfile(*effectProfile), *maxAttempts)
	if err != nil {
		return err
	}
	return printJSON(task)
}

func runEpisodeTask(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("episode task", flag.ContinueOnError)
	journalPath := flags.String("journal", "", "Episode Journal SQLite path")
	episodeID := flags.String("id", "", "Episode identifier")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *journalPath == "" || *episodeID == "" {
		return errors.New("--journal and --id are required and positional arguments are not accepted")
	}
	journal, err := episode.OpenJournal(*journalPath)
	if err != nil {
		return err
	}
	defer journal.Close()
	task, err := journal.GetTask(ctx, *episodeID)
	if err != nil {
		return err
	}
	return printJSON(task)
}

func runEpisodeCancel(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("episode cancel", flag.ContinueOnError)
	journalPath := flags.String("journal", "", "Episode Journal SQLite path")
	episodeID := flags.String("id", "", "Episode identifier")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *journalPath == "" || *episodeID == "" {
		return errors.New("--journal and --id are required and positional arguments are not accepted")
	}
	journal, err := episode.OpenJournal(*journalPath)
	if err != nil {
		return err
	}
	defer journal.Close()
	task, err := journal.CancelTask(ctx, *episodeID)
	if err != nil {
		return err
	}
	return printJSON(task)
}

func runEpisodeCoordinator(args []string) error {
	flags := flag.NewFlagSet("episode coordinator", flag.ContinueOnError)
	journalPath := flags.String("journal", "", "Episode Journal SQLite path")
	address := flags.String("addr", "127.0.0.1:8092", "coordinator listen address")
	tlsCert := flags.String("tls-cert", "", "TLS certificate PEM")
	tlsKey := flags.String("tls-key", "", "TLS private key PEM")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *journalPath == "" {
		return errors.New("--journal is required and positional arguments are not accepted")
	}
	if (*tlsCert == "") != (*tlsKey == "") {
		return errors.New("--tls-cert and --tls-key must be provided together")
	}
	host, _, err := net.SplitHostPort(*address)
	if err != nil {
		return fmt.Errorf("invalid coordinator address: %w", err)
	}
	ip := net.ParseIP(host)
	if *tlsCert == "" && (ip == nil || !ip.IsLoopback()) {
		return errors.New("non-loopback coordinator listeners require TLS")
	}
	token := os.Getenv("STATETWIN_COORDINATOR_TOKEN")
	journal, err := episode.OpenJournal(*journalPath)
	if err != nil {
		return err
	}
	defer journal.Close()
	coordinator, err := episode.NewCoordinator(journal, token)
	if err != nil {
		return err
	}
	httpServer, err := hardenedHTTPServer(*address, coordinator)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errorsCh := make(chan error, 1)
	go func() {
		if *tlsCert != "" {
			errorsCh <- httpServer.ListenAndServeTLS(*tlsCert, *tlsKey)
			return
		}
		errorsCh <- httpServer.ListenAndServe()
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
	return httpServer.Shutdown(shutdownCtx)
}

func runEpisodeWorker(args []string) error {
	flags := flag.NewFlagSet("episode worker", flag.ContinueOnError)
	coordinatorURL := flags.String("coordinator", "", "Episode coordinator base URL")
	workerID := flags.String("id", "", "stable worker identifier")
	leaseSeconds := flags.Int("lease-seconds", 60, "lease and heartbeat extension seconds")
	once := flags.Bool("once", false, "exit after one claim or an empty queue")
	poll := flags.Duration("poll", 2*time.Second, "empty-queue polling interval")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || *coordinatorURL == "" || *workerID == "" {
		return errors.New("--coordinator and --id are required and positional arguments are not accepted")
	}
	if *poll <= 0 || *leaseSeconds < 3 || *leaseSeconds > limits.MaxLeaseSeconds {
		return errors.New("--poll must be positive and --lease-seconds must be within 3..resource limit")
	}
	client, err := episode.NewCoordinatorClient(*coordinatorURL, os.Getenv("STATETWIN_COORDINATOR_TOKEN"), nil)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	for {
		claim, err := client.Claim(ctx, *workerID, *leaseSeconds, episode.EffectHermetic)
		if err != nil {
			var requestError *episode.CoordinatorRequestError
			if errors.As(err, &requestError) && requestError.Code == "NO_TASK" {
				if *once {
					return nil
				}
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(*poll):
					continue
				}
			}
			return err
		}
		if err := executeHermeticClaim(ctx, client, claim, *leaseSeconds); err != nil {
			return err
		}
		if *once {
			return nil
		}
	}
}

func executeHermeticClaim(parent context.Context, client *episode.CoordinatorClient, claim *episode.Claim, leaseSeconds int) error {
	if claim.EffectProfile != episode.EffectHermetic {
		return errors.New("scripted worker refuses non-hermetic claim")
	}
	if claim.Runtime.Version != server.Version || claim.Runtime.Revision != server.Revision {
		return client.Fail(parent, claim, episode.CommitNoEffect, "WORKER_ERROR")
	}
	artifact, err := bundle.OpenBytes(claim.Bundle)
	if err != nil {
		_ = client.Fail(parent, claim, episode.CommitNoEffect, "WORKER_ERROR")
		return err
	}
	runCtx, cancel := context.WithCancel(parent)
	defer cancel()
	heartbeatErrors := make(chan error, 1)
	stopHeartbeat := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Duration(leaseSeconds) * time.Second / 3)
		defer ticker.Stop()
		for {
			select {
			case <-stopHeartbeat:
				heartbeatErrors <- nil
				return
			case <-runCtx.Done():
				heartbeatErrors <- nil
				return
			case <-ticker.C:
				result, err := client.Heartbeat(runCtx, claim, leaseSeconds)
				if err != nil {
					heartbeatErrors <- err
					cancel()
					return
				}
				if result.CancelRequested {
					cancel()
				}
			}
		}
	}()
	evidence, runErr := episode.Run(runCtx, artifact, claim.EpisodeID, claim.ScenarioPath, claim.Runtime.Version, claim.Runtime.Revision)
	close(stopHeartbeat)
	heartbeatErr := <-heartbeatErrors
	if runErr != nil {
		class := "WORKER_ERROR"
		if errors.Is(runCtx.Err(), context.Canceled) {
			class = "CANCELLED"
		}
		if failErr := client.Fail(parent, claim, episode.CommitNoEffect, class); failErr != nil {
			return errors.Join(runErr, heartbeatErr, failErr)
		}
		return errors.Join(runErr, heartbeatErr)
	}
	if heartbeatErr != nil {
		return heartbeatErr
	}
	return client.Complete(parent, claim, evidence)
}

func runCompatibility(args []string) error {
	if len(args) == 0 || args[0] != "validate" {
		return errors.New("compatibility requires the validate subcommand")
	}
	flags := flag.NewFlagSet("compatibility validate", flag.ContinueOnError)
	reportPath := flags.String("report", "", "HostCompatibilityReport YAML path")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("compatibility validate does not accept positional arguments")
	}
	if *reportPath == "" {
		return errors.New("--report is required")
	}
	report, err := hostcompat.Load(*reportPath)
	if err != nil {
		return err
	}
	digest, err := report.Digest()
	if err != nil {
		return err
	}
	return printJSON(map[string]any{
		"valid": true, "format": report.Format, "profile": report.Host.Profile,
		"claimLevel": report.Claim.Level, "reportDigest": digest,
	})
}

func runLimits() error {
	digest, err := limits.Digest()
	if err != nil {
		return err
	}
	return printJSON(map[string]any{"profile": limits.Default(), "digest": digest})
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
	return printJSON(map[string]any{
		"valid": true, "specDigest": runtime.Digest(), "surfaceDigest": runtime.SurfaceDigest(), "tools": runtime.ToolNames(),
	})
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

func runScenario(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("scenario", flag.ContinueOnError)
	specPath := flags.String("spec", "", "TwinSpec YAML path")
	fixturePath := flags.String("fixture", "", "initial state JSON path")
	scenarioPath := flags.String("scenario", "", "Scenario YAML path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *specPath == "" {
		return errors.New("--spec is required")
	}
	if *scenarioPath == "" {
		return errors.New("--scenario is required")
	}
	twin, err := spec.Load(*specPath)
	if err != nil {
		return err
	}
	initial, err := loadFixture(*fixturePath)
	if err != nil {
		return err
	}
	scenario, err := statetwinscenario.Load(*scenarioPath)
	if err != nil {
		return err
	}
	report, err := statetwinscenario.Run(ctx, twin, initial, scenario, server.Version)
	if err != nil {
		return err
	}
	if err := printJSON(report); err != nil {
		return err
	}
	if !report.Passed {
		return errors.New("scenario assertions failed")
	}
	return nil
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

	dataServer, err := hardenedHTTPServer(*dataAddr, server.NewDataPlane(runtime))
	if err != nil {
		return err
	}
	toolNames := make([]string, 0, len(runtime.Spec().Tools))
	for _, tool := range runtime.Spec().Tools {
		toolNames = append(toolNames, tool.Name)
	}
	controlServer, err := hardenedHTTPServer(*controlAddr, server.NewControlPlaneWithRuntime(stateStore, token, runtime, toolNames...))
	if err != nil {
		return err
	}
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

func hardenedHTTPServer(address string, handler http.Handler) (*http.Server, error) {
	admission, err := server.NewAdmissionHandler(handler, activeExecutionProfile.MaxInFlightRequests)
	if err != nil {
		return nil, err
	}
	return &http.Server{
		Addr:              address,
		Handler:           admission,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}, nil
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
	if len(data) > maxFixtureBytes {
		return nil, fmt.Errorf("decode fixture: document exceeds %d bytes", maxFixtureBytes)
	}
	var state world.State
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&state); err != nil {
		return nil, fmt.Errorf("decode fixture: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("decode fixture: document must contain one JSON value")
	}
	state.Normalize()
	return &state, nil
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeJSONFile(path string, value any) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("evidence output %q already exists", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect evidence output: %w", err)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode JSON output: %w", err)
	}
	data = append(data, '\n')
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".statetwin-evidence-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary evidence: %w", err)
	}
	name := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(name)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("secure temporary evidence: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("write evidence: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync evidence: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close evidence: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("publish evidence: %w", err)
	}
	committed = true
	return nil
}
