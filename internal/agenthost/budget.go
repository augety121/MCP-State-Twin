package agenthost

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/augety121/mcp-state-twin/internal/task"
)

// Governor provides atomic admission, not rollback of already admitted work.
// Callers execute one request/tool at a time and must join it before cleanup.
type Governor struct {
	mu       sync.Mutex
	limits   task.Budgets
	usage    Usage
	stopped  bool
	deadline time.Time
}

type Usage struct {
	ModelRequests int `json:"modelRequests"`
	ToolAttempts  int `json:"toolAttempts"`
	ContentBytes  int `json:"contentBytes"`
}

func NewGovernor(b task.Budgets) (*Governor, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return &Governor{limits: b, deadline: time.Now().Add(time.Duration(b.EpisodeSeconds) * time.Second)}, nil
}

func (g *Governor) Admit(ctx context.Context, kind string, size int) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.stopped {
		return errors.New("EPISODE_STOPPED")
	}
	if err := ctx.Err(); err != nil {
		g.stopped = true
		return err
	}
	if !time.Now().Before(g.deadline) {
		g.stopped = true
		return errors.New("EPISODE_TIMEOUT")
	}
	if size < 0 || size > g.limits.TraceBytes-(64<<10)-g.usage.ContentBytes {
		g.stopped = true
		return errors.New("BUDGET_EXHAUSTED")
	}
	switch kind {
	case "model":
		if g.usage.ModelRequests >= g.limits.ModelRequests {
			g.stopped = true
			return errors.New("BUDGET_EXHAUSTED")
		}
		g.usage.ModelRequests++
	case "tool":
		if g.usage.ToolAttempts >= g.limits.ToolAttempts {
			g.stopped = true
			return errors.New("BUDGET_EXHAUSTED")
		}
		g.usage.ToolAttempts++
	case "content":
	default:
		g.stopped = true
		return errors.New("HOST_PROTOCOL_ERROR")
	}
	g.usage.ContentBytes += size
	return nil
}

func (g *Governor) Stop()        { g.mu.Lock(); g.stopped = true; g.mu.Unlock() }
func (g *Governor) Usage() Usage { g.mu.Lock(); defer g.mu.Unlock(); return g.usage }
