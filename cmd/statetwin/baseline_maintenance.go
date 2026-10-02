package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agentapi"
	"github.com/augety121/mcp-state-twin/internal/baseline"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/task"
)

func baselineMaintenance(ctx context.Context, args []string) (any, error) {
	f := flag.NewFlagSet("baseline maintenance", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	root := f.String("root", ".", "root")
	pack := f.String("pack", "", "pack")
	profile := f.String("profile", "", "profile")
	registry := f.String("registry", "", "local registry")
	id := f.String("pack-id", "", "pack identity")
	revision := f.String("revision", "", "pack revision")
	event := f.String("event", "", "event document")
	plan := f.String("live-plan", "", "accepted API bridge plan")
	entry := f.String("entry", "", "pack entry")
	at := f.String("at", "", "explicit assessment UTC time")
	allow := f.Bool("allow-live", false, "assess explicit live opt-in, does not send a request")
	used := f.Int("used-requests", 0, "already admitted requests")
	claim := f.String("claim", "", "claim document")
	binding := f.String("binding", "", "current binding document")
	if f.Parse(args[1:]) != nil || f.NArg() != 0 {
		return nil, errors.New("BASELINE_FLAGS_INVALID")
	}
	switch args[0] {
	case "register":
		if *pack == "" || *profile == "" || *registry == "" {
			return nil, errors.New("BASELINE_FLAGS_INVALID")
		}
		r, e := baselinepack.Register(ctx, *root, *pack, *profile, *registry)
		if e != nil {
			return nil, e
		}
		return struct {
			PackID   string `json:"packId"`
			Revision string `json:"revision"`
			State    string `json:"state"`
			Review   string `json:"reviewIndependence"`
		}{r.PackID, r.Revision, "frozen", r.ReviewIndependence}, nil
	case "revision-state", "transition":
		if *registry == "" || *id == "" || *revision == "" {
			return nil, errors.New("BASELINE_FLAGS_INVALID")
		}
		if args[0] == "transition" {
			raw, e := task.ReadFile(*root, *event, 64<<10)
			if e != nil {
				return nil, e
			}
			var ev baselinepack.RevisionEvent
			if baselinepack.Decode(raw, 64<<10, &ev) != nil {
				return nil, errors.New("BASELINE_REVISION_EVENT_INVALID")
			}
			if e = baselinepack.Transition(*root, *registry, *id, *revision, ev); e != nil {
				return nil, e
			}
		}
		state, e := baselinepack.RevisionState(*root, *registry, *id, *revision)
		return map[string]string{"packId": *id, "revision": *revision, "state": state}, e
	case "live-check":
		now, e := time.Parse(time.RFC3339Nano, *at)
		if e != nil {
			return nil, errors.New("BASELINE_TIME_INVALID")
		}
		p, e := baselinepack.Prepare(ctx, *root, *pack, *profile)
		if e != nil {
			return nil, e
		}
		raw, e := task.ReadFile(*root, *plan, task.MaxBytes+(16<<10))
		if e != nil {
			return nil, e
		}
		live, e := agentapi.DecodePlan(raw)
		if e != nil {
			return nil, e
		}
		return baseline.CheckLive(live, p.Find(*entry), now, *allow, *used)
	case "claim-check":
		now, e := time.Parse(time.RFC3339Nano, *at)
		if e != nil {
			return nil, errors.New("BASELINE_TIME_INVALID")
		}
		raw, e := task.ReadFile(*root, *claim, 64<<10)
		if e != nil {
			return nil, e
		}
		var c baseline.Claim
		if baselinepack.Decode(raw, 64<<10, &c) != nil {
			return nil, errors.New("BASELINE_CLAIM_INVALID")
		}
		raw, e = task.ReadFile(*root, *binding, 64<<10)
		if e != nil {
			return nil, e
		}
		var b baseline.Binding
		if baselinepack.Decode(raw, 64<<10, &b) != nil {
			return nil, errors.New("BASELINE_CLAIM_INVALID")
		}
		return baseline.AssessClaim(c, b, now)
	}
	return nil, errors.New("BASELINE_COMMAND_UNSUPPORTED")
}
