// Package agentapi owns the opt-in, fixed-endpoint Responses transport.
// It is not an upstream tool passthrough or a remote MCP server.
package agentapi

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/task"
)

const PlanFormat = "statetwin.dev/agent-live-plan/v1alpha1"
const Profile = "openai-responses-local-bridge-v1alpha1"
const Endpoint = "https://api.openai.com/v1/responses"
const CostPolicy = "request-capped-cost-unknown"

var id = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
var modelID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
var digest = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

type Plan struct {
	Format                string     `json:"format"`
	ID                    string     `json:"id"`
	Provider              string     `json:"provider"`
	Profile               string     `json:"profile"`
	Model                 string     `json:"model"`
	Task                  *task.Task `json:"task"`
	BundleDigest          string     `json:"bundleDigest"`
	IssuedAt              string     `json:"issuedAt"`
	ExpiresAt             string     `json:"expiresAt"`
	MaxRequests           int        `json:"maxRequests"`
	MaxOutputTokens       int        `json:"maxOutputTokens"`
	CostPolicy            string     `json:"costPolicy"`
	Approved              bool       `json:"approved"`
	SyntheticDataApproved bool       `json:"syntheticDataApproved"`
	UnknownCostApproved   bool       `json:"unknownCostApproved"`
}

func ValidModel(model string) bool {
	return modelID.MatchString(model) && !strings.HasPrefix(model, "mock-")
}

func DecodePlan(data []byte) (*Plan, error) {
	var p Plan
	if agenthost.DecodeDocument(data, task.MaxBytes+(16<<10), &p) != nil || p.Task == nil {
		return nil, errors.New("LIVE_PLAN_INVALID")
	}
	for _, rule := range p.Task.Authority {
		if _, err := agenthost.NormalizeNumbers(rule.Equals); err != nil {
			return nil, errors.New("LIVE_PLAN_INVALID")
		}
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

// Validate is timeless structural admission. Historical plans are not made
// invalid merely because their approval window has since expired.
func (p *Plan) Validate() error {
	if p == nil || p.Format != PlanFormat || !id.MatchString(p.ID) || p.Provider != "openai" || p.Profile != Profile || !ValidModel(p.Model) || !digest.MatchString(p.BundleDigest) || p.CostPolicy != CostPolicy {
		return errors.New("LIVE_PLAN_INVALID")
	}
	if p.Task.Validate() != nil || p.MaxRequests < 1 || p.MaxRequests > p.Task.Budgets.ModelRequests || p.MaxOutputTokens < 1 || p.MaxOutputTokens > 8192 {
		return errors.New("LIVE_PLAN_INVALID")
	}
	issued, e1 := time.Parse(time.RFC3339Nano, p.IssuedAt)
	expires, e2 := time.Parse(time.RFC3339Nano, p.ExpiresAt)
	if e1 != nil || e2 != nil || !expires.After(issued) || expires.Sub(issued) > 24*time.Hour {
		return errors.New("LIVE_PLAN_INVALID")
	}
	return nil
}

func (p *Plan) Authorize(now time.Time, allowLive bool) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if !allowLive || !p.Approved || !p.SyntheticDataApproved || !p.UnknownCostApproved {
		return errors.New("LIVE_NOT_AUTHORIZED")
	}
	issued, _ := time.Parse(time.RFC3339Nano, p.IssuedAt)
	expires, _ := time.Parse(time.RFC3339Nano, p.ExpiresAt)
	if now.Before(issued) || !now.Before(expires) {
		return errors.New("LIVE_APPROVAL_EXPIRED_OR_NOT_YET_VALID")
	}
	return nil
}

func (p *Plan) OutputDirectory() string { return ".statetwin/live/" + p.ID }
