// Package plugin owns the trusted lifecycle around one Task-scoped MCP session.
package plugin

import (
	"context"
	"errors"
	"time"

	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/task"
)

const PlanFormat = "statetwin.dev/plugin-session-plan/v1alpha1"
const ReportFormat = "statetwin.dev/plugin-session-report/v1alpha1"

type Host struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	Model           string `json:"model"`
	FreshSession    bool   `json:"freshSession"`
	ToolsOnly       bool   `json:"toolsOnly"`
	AdditionalTools int    `json:"additionalTools"`
}
type Plan struct {
	Format          string `json:"format"`
	ID              string `json:"id"`
	Pack            string `json:"pack"`
	Profile         string `json:"profile"`
	EntryID         string `json:"entryId"`
	Host            Host   `json:"host"`
	Output          string `json:"output"`
	Mode            string `json:"mode"`
	DeadlineSeconds int    `json:"deadlineSeconds"`
}
type Prepared struct {
	Plan  Plan
	Pack  *baselinepack.Prepared
	Entry *baselinepack.PreparedEntry
}

func Prepare(parent context.Context, root, name string) (*Prepared, error) {
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	r := baselinepack.NewReader(ctx, root)
	raw, err := r.Read(name, 64<<10)
	if err != nil {
		return nil, err
	}
	var p Plan
	if baselinepack.Decode(raw, 64<<10, &p) != nil || p.Validate() != nil {
		return nil, errors.New("PLUGIN_PLAN_INVALID")
	}
	pack, err := baselinepack.PrepareWith(r, p.Pack, p.Profile)
	if err != nil {
		return nil, err
	}
	e := pack.Find(p.EntryID)
	if e == nil || p.DeadlineSeconds > e.Task.Budgets.EpisodeSeconds || r.RejectOutput(p.Output) != nil {
		return nil, errors.New("PLUGIN_PLAN_INVALID")
	}
	return &Prepared{p, pack, e}, nil
}
func (p Plan) Validate() error {
	if p.Format != PlanFormat || !baselinepack.Label(p.ID) || !baselinepack.Label(p.EntryID) || task.PortablePath(p.Pack) != nil || task.PortablePath(p.Profile) != nil || task.PortablePath(p.Output) != nil || p.DeadlineSeconds < 1 || p.DeadlineSeconds > 180 || p.Mode != "offline-contract" || !p.Host.FreshSession || !p.Host.ToolsOnly || p.Host.AdditionalTools != 0 || p.Host.Model != "mock/statetwin" {
		return errors.New("PLUGIN_PLAN_INVALID")
	}
	if !((p.Host.Name == "mcp-client" && p.Host.Version == "go-sdk-1.8.0") || (p.Host.Name == "inspect" && p.Host.Version == "0.3.275")) {
		return errors.New("PLUGIN_HOST_UNSUPPORTED")
	}
	return nil
}
