package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
)

// Every new command has a closed flag set and shares one operation deadline.
func runOfflineDelivery(parent context.Context, command string, args []string) error {
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	argumentCodes := map[string]string{"suite-review": "SUITE_REVIEW_ARGUMENTS_INVALID", "suite-assess-reviewed": "REVIEWED_ASSESSMENT_ARGUMENTS_INVALID", "cases": "CASE_ARGUMENTS_INVALID", "qualify": "CASE_ARGUMENTS_INVALID", "inventory": "INV_ARGUMENTS_INVALID", "retention-preview": "RETENTION_ARGUMENTS_INVALID", "suite-export": "EXPORT_ARGUMENTS_INVALID", "suite-import": "IMPORT_ARGUMENTS_INVALID"}
	root := f.String("root", ".", "trusted quiescent root")
	var out, expect, catalog, suite, policy, registry, mode, archive, cases string
	format := "json"
	switch command {
	case "suite-review", "suite-assess-reviewed":
		f.StringVar(&out, "out", "", "relative suite output")
		f.StringVar(&expect, "expect", "", "independent expectation")
		f.StringVar(&catalog, "tasks", "", "independent task catalog")
		if command == "suite-review" {
			f.StringVar(&suite, "suite", "", "relative suite plan")
		} else {
			f.StringVar(&policy, "policy", "", "candidate-pass-v1 or both-pass-v1")
		}
	case "cases", "qualify":
		f.StringVar(&cases, "cases", "", "task case manifest")
	case "inventory", "retention-preview":
		f.StringVar(&registry, "registry", "", "explicit suite registry")
		if command == "inventory" {
			f.StringVar(&mode, "mode", "", "metadata or replay")
		} else {
			f.StringVar(&policy, "policy", "", "retention policy")
		}
	case "suite-export", "suite-import":
		f.StringVar(&out, "out", "", "suite directory")
		f.StringVar(&archive, "archive", "", "archive path")
	default:
		return errors.New("OFFLINE_ARGUMENTS_INVALID")
	}
	if command != "suite-export" && command != "suite-import" {
		f.StringVar(&format, "format", "json", "json or markdown")
	}
	if err := f.Parse(args); err != nil {
		return errors.New(argumentCodes[command])
	}
	bad := f.NArg() != 0 || (format != "json" && format != "markdown")
	switch command {
	case "suite-review":
		bad = bad || out == "" || expect == "" || catalog == "" || suite == ""
	case "suite-assess-reviewed":
		bad = bad || out == "" || expect == "" || catalog == "" || (policy != "candidate-pass-v1" && policy != "both-pass-v1")
	case "cases", "qualify":
		bad = bad || cases == ""
	case "inventory":
		bad = bad || registry == "" || (mode != "metadata" && mode != "replay")
	case "retention-preview":
		bad = bad || registry == "" || policy == ""
	default:
		bad = bad || out == "" || archive == ""
	}
	if bad {
		return errors.New(argumentCodes[command])
	}
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	var result any
	var err error
	gate := ""
	switch command {
	case "suite-review":
		r, e := agenteval.ReviewSuite(ctx, *root, suite, out, expect, catalog)
		err = e
		if r != nil {
			result = r
			if r.Decision != "matched" {
				gate = "SUITE_REVIEW_NOT_MATCHED"
			}
		}
	case "suite-assess-reviewed":
		r, e := agenteval.AssessReviewedSuite(ctx, *root, out, expect, catalog, policy)
		err = e
		if r != nil {
			result = r
			if r.Decision != "passed" {
				gate = "REVIEWED_ASSESSMENT_GATE_NOT_SATISFIED"
			}
		}
	case "cases":
		r, e := agenteval.RunCases(ctx, *root, cases)
		err = e
		if r != nil {
			result = r
			if r.Decision != "matched" {
				gate = "CASE_GATE_NOT_SATISFIED"
			}
		}
	case "qualify":
		r, e := agenteval.QualifyTasks(ctx, *root, cases)
		err = e
		if r != nil {
			result = r
			if r.Decision != "qualified" {
				gate = "CASE_GATE_NOT_SATISFIED"
			}
		}
	case "inventory":
		r, e := agenteval.InventorySuites(ctx, *root, registry, mode)
		err = e
		if r != nil {
			result = r
			if r.Completion != "complete" {
				gate = "INV_INCOMPLETE"
			}
		}
	case "retention-preview":
		r, e := agenteval.PreviewRetention(ctx, *root, registry, policy)
		err = e
		if r != nil {
			result = r
			if r.Completion != "complete" {
				gate = "RETENTION_INCOMPLETE"
			}
		}
	case "suite-export":
		r, e := agenteval.ExportSuite(ctx, *root, out, archive)
		err = e
		if r != nil {
			result = r
		}
	case "suite-import":
		r, e := agenteval.ImportSuite(ctx, *root, archive, out)
		err = e
		if r != nil {
			result = r
		}
	}
	// Partial diagnostics remain useful on interruption; a completed success
	// must never be printed after the shared operation deadline.
	if ctx.Err() != nil && err == nil {
		return ctx.Err()
	}
	if result != nil {
		raw, marshalErr := json.MarshalIndent(result, "", "  ")
		if marshalErr != nil {
			return errors.New("OFFLINE_REPORT_INVALID")
		}
		if len(raw) > 1<<20 {
			return errors.New("OFFLINE_REPORT_RESOURCE_LIMIT")
		}
		if format == "markdown" {
			_, marshalErr = fmt.Printf("# Offline %s\n\nSynthetic inputs; provenance not proven.\n\n```json\n%s\n```\n", command, raw)
		} else {
			_, marshalErr = fmt.Printf("%s\n", raw)
		}
		if marshalErr != nil {
			return marshalErr
		}
	}
	if err != nil {
		return err
	}
	if gate != "" {
		return errors.New(gate)
	}
	return nil
}
