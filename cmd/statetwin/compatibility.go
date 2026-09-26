package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"

	"github.com/augety121/mcp-state-twin/internal/hostcompat"
)

func runCompatibility(args []string) error {
	return runCompatibilityTo(args, os.Stdout)
}

func runCompatibilityTo(args []string, output io.Writer) error {
	if len(args) == 0 || (args[0] != "validate" && args[0] != "assess") {
		return errors.New("compatibility requires the validate subcommand or assess subcommand")
	}
	flags := flag.NewFlagSet("compatibility "+args[0], flag.ContinueOnError)
	reportPath := flags.String("report", "", "HostCompatibilityReport YAML path")
	var at string
	var requireFresh bool
	if args[0] == "assess" {
		flags.StringVar(&at, "at", "", "explicit RFC3339 UTC assessment time ending in Z")
		flags.BoolVar(&requireFresh, "require-fresh", false, "fail unless declared verified and within time window; NOT proof of compatibility")
	}
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("compatibility does not accept positional arguments")
	}
	if *reportPath == "" {
		return errors.New("--report is required")
	}
	if args[0] == "assess" && at == "" {
		return errors.New("--at is required for assess")
	}
	r, err := hostcompat.Load(*reportPath)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if args[0] == "validate" {
		digest, err := r.Digest()
		if err != nil {
			return err
		}
		return encoder.Encode(map[string]any{
			"valid": true, "format": r.Format, "profile": r.Host.Profile,
			"claimLevel": r.Claim.Level, "reportDigest": digest,
			"validationScope": "structure-only", "provenance": "not_verified", "publicationAllowed": false,
		})
	}
	assessment, err := hostcompat.Assess(r, at)
	if err != nil {
		return err
	}
	if err := encoder.Encode(assessment); err != nil {
		return err
	}
	if requireFresh && !assessment.ClaimTimeEligible {
		return errors.New("HOST_REPORT_NOT_TIME_ELIGIBLE")
	}
	return nil
}
