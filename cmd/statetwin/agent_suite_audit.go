package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
)

func runSuiteAudit(ctx context.Context, args []string) error {
	f := flag.NewFlagSet("eval "+args[0], flag.ContinueOnError)
	root := f.String("root", ".", "trusted quiescent root")
	out := f.String("out", "", "relative suite evidence directory")
	format := f.String("format", "json", "json or markdown")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	if f.NArg() != 0 || *out == "" || (*format != "json" && *format != "markdown") {
		return errors.New("SUITE_ARGUMENTS_INVALID")
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	r, err := agenteval.InspectSuite(ctx, *root, *out)
	if err != nil {
		return err
	}
	if *format == "markdown" {
		_, err = fmt.Print(r.Markdown())
	} else {
		err = printJSON(r)
	}
	if err != nil {
		return err
	}
	if r.State == "invalid" {
		return errors.New("SUITE_DIRECTORY_INVALID")
	}
	if args[0] == "suite-verify" && !r.RegressionGatePassed {
		return errors.New("SUITE_GATE_NOT_SATISFIED")
	}
	return nil
}
