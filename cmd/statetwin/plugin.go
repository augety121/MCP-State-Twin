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
	"strings"
	"syscall"

	"github.com/augety121/mcp-state-twin/internal/baseline"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/plugin"
)

func runPlugin(parent context.Context, args []string) error {
	return runPluginTo(parent, args, os.Stdout)
}
func runPluginTo(parent context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("PLUGIN_COMMAND_REQUIRED")
	}
	command := args[0]
	flags := flag.NewFlagSet("plugin", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("root", ".", "input root")
	pack := flags.String("pack", "", "pack manifest")
	profile := flags.String("profile", "", "plugin profile")
	plan := flags.String("session-plan", "", "session plan")
	destination := flags.String("out", "", "new recovery directory")
	kind := flags.String("type", "", "schema type")
	format := flags.String("format", "json", "json or markdown")
	if flags.Parse(args[1:]) != nil || flags.NArg() != 0 || (*format != "json" && *format != "markdown") {
		return errors.New("PLUGIN_FLAGS_INVALID")
	}
	ctx, cancel := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	var result any
	var err error
	if command != "recover" && *destination != "" {
		return errors.New("PLUGIN_FLAGS_INVALID")
	}
	if command != "schema" && *kind != "" {
		return errors.New("PLUGIN_FLAGS_INVALID")
	}
	switch command {
	case "schema":
		if *plan != "" || *pack != "" || *profile != "" {
			return errors.New("PLUGIN_FLAGS_INVALID")
		}
		switch *kind {
		case "pack":
			result = baselinepack.Schema(baselinepack.Pack{})
		case "profile":
			result = baselinepack.Schema(baselinepack.Profile{})
		case "session":
			result = baselinepack.Schema(plugin.Plan{})
		case "report":
			result = baselinepack.Schema(plugin.Report{})
		case "baseline-plan":
			result = baselinepack.Schema(baseline.Plan{})
		case "baseline-report":
			result = baselinepack.Schema(baseline.Report{})
		default:
			return errors.New("PLUGIN_FLAGS_INVALID")
		}
	case "recover":
		if *plan == "" || *pack != "" || *profile != "" || *destination == "" {
			return errors.New("PLUGIN_FLAGS_INVALID")
		}
		result, err = plugin.Recover(ctx, *root, *plan, *destination)
	case "projection":
		if *plan == "" || *pack != "" || *profile != "" {
			return errors.New("PLUGIN_FLAGS_INVALID")
		}
		p, e := plugin.Prepare(ctx, *root, *plan)
		if e != nil {
			return e
		}
		result = struct {
			Task    baselinepack.PublicProjection `json:"task"`
			Plan    plugin.Plan                   `json:"plan"`
			Profile baselinepack.Profile          `json:"profile"`
		}{p.Entry.Projection(), p.Plan, p.Pack.Profile}
	case "check", "describe":
		if *pack == "" || *profile == "" || *plan != "" {
			return errors.New("PLUGIN_FLAGS_INVALID")
		}
		var p *baselinepack.Prepared
		p, err = baselinepack.Prepare(ctx, *root, *pack, *profile)
		if err != nil {
			return err
		}
		result = p.Summary()
		if command == "describe" {
			entries := []baselinepack.PublicProjection{}
			for _, e := range p.Entries {
				entries = append(entries, e.Projection())
			}
			result = struct {
				Check   baselinepack.Check              `json:"check"`
				Profile baselinepack.Profile            `json:"profile"`
				Tasks   []baselinepack.PublicProjection `json:"tasks"`
				Adapter string                          `json:"adapter"`
			}{p.Summary(), p.Profile, entries, "inspect-0.3.275/python-3.12/mcp-1.26.0"}
		}
	case "serve":
		if *plan == "" || *pack != "" || *profile != "" || *format != "json" {
			return errors.New("PLUGIN_FLAGS_INVALID")
		}
		p, e := plugin.Prepare(ctx, *root, *plan)
		if e != nil {
			return e
		}
		// serve owns protocol stdout; no CLI report may be appended to it.
		_, err = plugin.Serve(ctx, *root, p, os.Stdin, os.Stdout, os.Getenv("STATETWIN_PLUGIN_CONTROL"), os.Getenv("STATETWIN_PLUGIN_CONTROL_TOKEN"))
		return err
	case "inspect":
		if *plan == "" || *pack != "" || *profile != "" {
			return errors.New("PLUGIN_FLAGS_INVALID")
		}
		result, err = plugin.Inspect(ctx, *root, *plan)
	default:
		return errors.New("PLUGIN_COMMAND_UNSUPPORTED")
	}
	if result != nil {
		raw, e := json.MarshalIndent(result, "", "  ")
		if e != nil {
			return errors.New("PLUGIN_OUTPUT_FAILED")
		}
		if *format == "markdown" {
			raw = []byte("# State Twin plugin\n\n```json\n" + strings.ReplaceAll(string(raw), "`", "\\u0060") + "\n```\n")
		}
		n, e := fmt.Fprintln(out, string(raw))
		if e != nil || n != len(raw)+1 {
			return errors.New("PLUGIN_OUTPUT_FAILED")
		}
	}
	return err
}
