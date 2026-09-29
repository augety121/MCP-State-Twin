package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/task"
)

func runAgentSuite(ctx context.Context, args []string) error {
	command := args[0]
	f := flag.NewFlagSet("eval "+command, flag.ContinueOnError)
	root := f.String("root", ".", "trusted input and output root")
	plan := f.String("suite", "", "relative synthetic offline suite plan")
	var out string
	if command == "suite" {
		f.StringVar(&out, "out", "", "new relative suite evidence directory; no overwrite/resume")
	}
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	if f.NArg() != 0 || *plan == "" || (command == "suite" && out == "") {
		return errors.New("SUITE_ARGUMENTS_INVALID")
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	raw, err := task.ReadFile(*root, *plan, agenteval.MaxSuitePlanBytes)
	if err != nil {
		return errors.New("SUITE_INPUT_INVALID")
	}
	prepared, err := agenteval.PrepareSuite(ctx, *root, raw)
	if err != nil {
		return err
	}
	if command == "suite-preflight" {
		return printJSON(prepared.Summary())
	}
	r, runErr := agenteval.RunSuite(ctx, *root, out, prepared)
	if r != nil {
		if err := printJSON(r); err != nil {
			return err
		}
	}
	if runErr != nil {
		return runErr
	}
	if r == nil || r.ExecutionStatus != "completed" || r.ComparisonStatus != "complete" || r.Comparison == nil || r.Comparison.Decision != "no_regression_observed" {
		return errors.New("SUITE_GATE_NOT_SATISFIED")
	}
	return nil
}
