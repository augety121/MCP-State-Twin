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

func runSuiteAssessment(ctx context.Context, args []string) error {
	f := flag.NewFlagSet("eval suite-assess", flag.ContinueOnError)
	root := f.String("root", ".", "trusted quiescent root")
	out := f.String("out", "", "relative suite evidence directory")
	expect := f.String("expect", "", "independent expectation outside result directory")
	policy := f.String("policy", "", "candidate-pass-v1 or both-pass-v1")
	format := f.String("format", "json", "json or markdown")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 || *out == "" || *expect == "" || (*policy != "candidate-pass-v1" && *policy != "both-pass-v1") || (*format != "json" && *format != "markdown") {
		return errors.New("ASSESSMENT_ARGUMENTS_INVALID")
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	r, err := agenteval.AssessSuite(ctx, *root, *out, *expect, *policy)
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
	if r.Decision != "passed" {
		return errors.New("ASSESSMENT_GATE_NOT_SATISFIED")
	}
	return nil
}
