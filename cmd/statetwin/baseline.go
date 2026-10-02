package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/baseline"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/plugin"
)

func runBaseline(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("BASELINE_COMMAND_REQUIRED")
	}
	switch args[0] {
	case "register", "revision-state", "transition", "live-check", "claim-check":
		result, err := baselineMaintenance(ctx, args)
		if result != nil {
			raw, e := json.MarshalIndent(result, "", "  ")
			if e != nil {
				return errors.New("BASELINE_OUTPUT_FAILED")
			}
			n, e := fmt.Fprintln(out, string(raw))
			if e != nil || n != len(raw)+1 {
				return errors.New("BASELINE_OUTPUT_FAILED")
			}
		}
		return err
	}
	f := flag.NewFlagSet("baseline", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	root := f.String("root", ".", "input root")
	plan := f.String("plan", "", "baseline plan")
	output := f.String("out", "", "new output directory")
	pack := f.String("pack", "", "pack manifest")
	profile := f.String("profile", "", "plugin profile")
	shard := f.Int("shard", 0, "explicit shard index")
	allowLive := f.Bool("allow-live", false, "explicitly permit approved API-bridge shard")
	if f.Parse(args[1:]) != nil || f.NArg() != 0 {
		return errors.New("BASELINE_FLAGS_INVALID")
	}
	var result any
	var err error
	if *allowLive && args[0] != "shard" {
		return errors.New("BASELINE_FLAGS_INVALID")
	}
	switch args[0] {
	case "variants", "qualify":
		if args[0] == "qualify" {
			if *pack == "" || *profile == "" || *output != "" || *plan != "" || *shard != 0 {
				return errors.New("BASELINE_FLAGS_INVALID")
			}
			result, err = baselinepack.Qualify(ctx, *root, *pack, *profile)
			break
		}
		if *pack == "" || *profile == "" || *output == "" || *plan != "" || *shard != 0 {
			return errors.New("BASELINE_FLAGS_INVALID")
		}
		result, err = baselinepack.GenerateVariants(ctx, *root, *pack, *profile, *output)
	case "check", "freeze", "shard", "assess":
		if *plan == "" || *pack != "" || *profile != "" {
			return errors.New("BASELINE_FLAGS_INVALID")
		}
		switch args[0] {
		case "check":
			if *output != "" || *shard != 0 {
				return errors.New("BASELINE_FLAGS_INVALID")
			}
			var p *baseline.Prepared
			p, err = baseline.Prepare(ctx, *root, *plan)
			if p != nil {
				result = p.Frozen
			}
		case "freeze":
			if *output == "" || *shard != 0 {
				return errors.New("BASELINE_FLAGS_INVALID")
			}
			result, err = baseline.Freeze(ctx, *root, *plan, *output)
		case "assess":
			if *output != "" || *shard != 0 {
				return errors.New("BASELINE_FLAGS_INVALID")
			}
			result, err = baseline.Assess(ctx, *root, *plan)
		case "shard":
			if *output != "" || *shard < 1 {
				return errors.New("BASELINE_FLAGS_INVALID")
			}
			if *allowLive {
				result, err = baseline.RunLiveShard(ctx, *root, *plan, *shard, true, func() string { return os.Getenv("OPENAI_API_KEY") })
				break
			}
			binary, e := os.Executable()
			if e != nil {
				return e
			}
			result, err = baseline.RunShard(ctx, *root, *plan, *shard, func(c context.Context, r, n string, w *agenteval.Witness) (*plugin.Report, error) {
				return plugin.RunContract(c, []string{binary}, r, n, w)
			})
		}
	default:
		return errors.New("BASELINE_COMMAND_UNSUPPORTED")
	}
	if result != nil {
		raw, e := json.MarshalIndent(result, "", "  ")
		if e != nil {
			return errors.New("BASELINE_OUTPUT_FAILED")
		}
		n, e := fmt.Fprintln(out, string(raw))
		if e != nil || n != len(raw)+1 {
			return errors.New("BASELINE_OUTPUT_FAILED")
		}
	}
	return err
}
