package plugin

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"

	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
)

const PerformanceFormat = "statetwin.dev/plugin-performance/v1alpha2"
const PerformanceLimit = 4 << 20

// PerformanceRun preserves observations, not caller-computed percentiles or a
// caller-selected threshold. It is local operator evidence, not attestation.
type PerformanceRun struct {
	Format           string                 `json:"format"`
	RunID            string                 `json:"runId"`
	Source           string                 `json:"source"`
	ProcessScope     string                 `json:"processScope"`
	BaselineScope    string                 `json:"baselineScope"`
	Environment      PerformanceEnvironment `json:"environment"`
	Workload         PerformanceWorkload    `json:"workload"`
	Sessions         int                    `json:"sessions"`
	GoroutinesBefore int                    `json:"goroutinesBefore"`
	GoroutinesAfter  int                    `json:"goroutinesAfter"`
	Pairs            []PerformancePair      `json:"pairs"`
}

type PerformanceEnvironment struct {
	OS              string `json:"os"`
	Arch            string `json:"arch"`
	CPU             string `json:"cpu"`
	Go              string `json:"go"`
	BuildLabel      string `json:"buildLabel"`
	RuntimeVersion  string `json:"runtimeVersion"`
	RuntimeRevision string `json:"runtimeRevision"`
	GOMAXPROCS      int    `json:"gomaxprocs"`
	Race            bool   `json:"race"`
}

type PerformanceWorkload struct {
	PackID          string `json:"packId"`
	PackRevision    string `json:"packRevision"`
	EntryID         string `json:"entryId"`
	TaskID          string `json:"taskId"`
	Profile         string `json:"profile"`
	ProtocolProfile string `json:"protocolProfile"`
}

type PerformancePair struct {
	ID              int             `json:"id"`
	ProcessSeconds  float64         `json:"processSeconds"`
	BaselineSeconds float64         `json:"baselineSeconds"`
	Metrics         ContractMetrics `json:"metrics"`
}

type PerformanceDistribution struct {
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
	Max float64 `json:"max"`
}

type PerformanceSummary struct {
	ProcessSeconds         PerformanceDistribution            `json:"processSeconds"`
	BaselineSeconds        PerformanceDistribution            `json:"baselineSeconds"`
	PhaseSeconds           map[string]PerformanceDistribution `json:"phaseSeconds"`
	RatioOfP95             float64                            `json:"ratioOfP95"`
	DifferenceOfP95Seconds float64                            `json:"differenceOfP95Seconds"`
	PairedOverheadSeconds  PerformanceDistribution            `json:"pairedOverheadSeconds"`
	PeakChildRSSBytes      int64                              `json:"peakChildRssBytes"`
	RSSStatus              string                             `json:"rssStatus"`
}

type PerformanceGate struct {
	Metric string  `json:"metric"`
	Actual float64 `json:"actual"`
	Limit  float64 `json:"limit"`
	Passed bool    `json:"passed"`
}

type PerformanceAssessment struct {
	Format      string                  `json:"format"`
	Status      string                  `json:"status"`
	Decision    string                  `json:"decision"`
	SourceTrust string                  `json:"sourceTrust"`
	Scope       string                  `json:"scope"`
	Reason      string                  `json:"reason"`
	RunID       string                  `json:"runId,omitempty"`
	Environment *PerformanceEnvironment `json:"environment,omitempty"`
	Workload    *PerformanceWorkload    `json:"workload,omitempty"`
	Samples     int                     `json:"samples"`
	Summary     *PerformanceSummary     `json:"summary,omitempty"`
	Gates       []PerformanceGate       `json:"gates"`
}

// AssessPerformance applies the existing SPEC-0078 §12.2 interpretation:
// ratio of p95s and difference of p95s, never p95 of per-pair ratios/deltas.
// Paired overhead is explicitly diagnostic. No input can change the gates.
func AssessPerformance(raw []byte) *PerformanceAssessment {
	r := &PerformanceAssessment{Format: "statetwin.dev/plugin-performance-assessment/v1alpha1", Status: "invalid", Decision: "unavailable", SourceTrust: "local-operator-asserted", Scope: "go-stdio-contract-performance-only", Reason: "PERFORMANCE_INPUT_INVALID", Gates: []PerformanceGate{}}
	var header map[string]any
	if agenthost.DecodeDocument(raw, PerformanceLimit, &header) != nil {
		return r
	}
	if header["format"] == "statetwin.dev/plugin-performance/v1alpha1" || header["format"] == "statetwin.dev/inspect-performance/v1alpha1" {
		r.Status, r.Reason = "incomplete", "PERFORMANCE_LEGACY_NOT_RECOMPUTABLE"
		return r
	}
	var run PerformanceRun
	if baselinepack.Decode(raw, PerformanceLimit, &run) != nil || run.Format != PerformanceFormat || !baselinepack.Label(run.RunID) || run.Source != "mock-contract" || run.ProcessScope != "full-go-stdio-contract-v1" || run.BaselineScope != "preloaded-run-witness-v1" || run.Sessions < len(run.Pairs) || run.Sessions > 10000 || run.GoroutinesBefore < 1 || run.GoroutinesAfter < 1 || len(run.Pairs) < 30 || len(run.Pairs) > 1000 {
		return r
	}
	e, w := run.Environment, run.Workload
	for _, s := range []string{e.OS, e.Arch, e.CPU, e.Go, e.BuildLabel, e.RuntimeVersion, e.RuntimeRevision, w.PackID, w.PackRevision, w.EntryID, w.TaskID, w.Profile, w.ProtocolProfile} {
		if s == "" || s != strings.TrimSpace(s) || len(s) > 256 {
			return r
		}
	}
	if e.Race || e.GOMAXPROCS < 1 || e.GOMAXPROCS > 65536 {
		return r
	}
	process, baseline, overhead := []float64{}, []float64{}, []float64{}
	phases := map[string][]float64{}
	peak := int64(0)
	rssMeasured := true
	for i, p := range run.Pairs {
		m := p.Metrics
		if p.ID != i+1 || !duration(p.ProcessSeconds, true) || !duration(p.BaselineSeconds, true) || !duration(m.TotalSeconds, true) || m.TotalSeconds > p.ProcessSeconds || !duration(m.StartupSeconds, true) {
			return r
		}
		values := map[string]float64{"preflight": m.PreflightSeconds, "startup": m.StartupSeconds, "tools": m.ToolsSeconds, "gradingAndSeal": m.GradingAndSealSeconds, "cleanup": m.CleanupSeconds, "inspection": m.InspectionSeconds}
		sum := 0.0
		for name, v := range values {
			if !duration(v, false) {
				return r
			}
			sum += v
			phases[name] = append(phases[name], v)
		}
		if sum > m.TotalSeconds+1e-9 || m.PeakRSSBytes < 0 || (m.RSSStatus != "measured" && m.RSSStatus != "unavailable") || (m.RSSStatus == "measured" && m.PeakRSSBytes == 0) || (m.RSSStatus == "unavailable" && m.PeakRSSBytes != 0) {
			return r
		}
		rssMeasured = rssMeasured && m.RSSStatus == "measured"
		peak = max(peak, m.PeakRSSBytes)
		process = append(process, p.ProcessSeconds)
		baseline = append(baseline, p.BaselineSeconds)
		overhead = append(overhead, p.ProcessSeconds-p.BaselineSeconds)
	}
	s := &PerformanceSummary{ProcessSeconds: distribution(process), BaselineSeconds: distribution(baseline), PairedOverheadSeconds: distribution(overhead), PhaseSeconds: map[string]PerformanceDistribution{}, PeakChildRSSBytes: peak, RSSStatus: "measured"}
	if !rssMeasured {
		s.RSSStatus = "unavailable-or-partial"
	}
	for name, values := range phases {
		s.PhaseSeconds[name] = distribution(values)
	}
	s.RatioOfP95 = s.ProcessSeconds.P95 / s.BaselineSeconds.P95
	if math.IsInf(s.RatioOfP95, 0) || math.IsNaN(s.RatioOfP95) {
		return r
	}
	s.DifferenceOfP95Seconds = s.ProcessSeconds.P95 - s.BaselineSeconds.P95
	r.Status, r.Decision, r.Reason = "valid", "passed", "PERFORMANCE_THRESHOLDS_PASSED"
	r.RunID, r.Environment, r.Workload, r.Samples, r.Summary = run.RunID, &e, &w, len(run.Pairs), s
	r.Gates = []PerformanceGate{{"startupP95Seconds", s.PhaseSeconds["startup"].P95, 2, false}, {"ratioOfP95", s.RatioOfP95, 1.5, false}, {"differenceOfP95Seconds", s.DifferenceOfP95Seconds, 2, false}}
	for i := range r.Gates {
		r.Gates[i].Passed = r.Gates[i].Actual <= r.Gates[i].Limit
		if !r.Gates[i].Passed {
			r.Decision, r.Reason = "failed", "PERFORMANCE_THRESHOLDS_FAILED"
		}
	}
	return r
}

func duration(v float64, positive bool) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 3600 && (!positive || v > 0)
}

// Nearest rank: with 30 samples, p50 is index 14 and p95 is index 28.
func distribution(values []float64) PerformanceDistribution {
	x := append([]float64(nil), values...)
	sort.Float64s(x)
	return PerformanceDistribution{x[int(math.Ceil(float64(len(x))*.5))-1], x[int(math.Ceil(float64(len(x))*.95))-1], x[len(x)-1]}
}

func ReadPerformanceAssessment(ctx context.Context, root, name string) (*PerformanceAssessment, error) {
	raw, err := baselinepack.NewReader(ctx, root).Read(name, PerformanceLimit)
	if err != nil {
		return nil, err
	}
	r := AssessPerformance(raw)
	if r.Decision != "passed" {
		return r, errors.New(r.Reason)
	}
	return r, nil
}
