package baseline

import (
	"errors"
	"sort"
)

type Usage struct {
	Status       string   `json:"status"`
	InputTokens  *int     `json:"inputTokens,omitempty"`
	OutputTokens *int     `json:"outputTokens,omitempty"`
	Cost         *float64 `json:"cost,omitempty"`
	Currency     string   `json:"currency,omitempty"`
}
type Observation struct {
	TrialID         string   `json:"trialId"`
	State           string   `json:"state"`
	Verified        bool     `json:"verified"`
	Cleanup         string   `json:"cleanup"`
	Success         bool     `json:"success"`
	PolicyFailures  int      `json:"policyFailures"`
	IllegalAttempts int      `json:"illegalAttempts"`
	IllegalEffects  int      `json:"illegalEffects"`
	ObservedModel   string   `json:"observedModel"`
	LatencySeconds  *float64 `json:"latencySeconds,omitempty"`
	Usage           Usage    `json:"usage"`
}
type Metrics struct {
	Planned             int       `json:"planned"`
	Scorable            int       `json:"scorable"`
	Complete            int       `json:"complete"`
	Successes           int       `json:"successes"`
	Missing             int       `json:"missing"`
	Unverified          int       `json:"unverified"`
	Interrupted         int       `json:"interrupted"`
	PolicyFailures      int       `json:"policyFailures"`
	IllegalAttempts     int       `json:"illegalAttempts"`
	IllegalEffects      int       `json:"illegalEffects"`
	SuccessRate         float64   `json:"successRate"`
	ScorableRate        float64   `json:"scorableRate"`
	CostStatus          string    `json:"costStatus"`
	Cost                *float64  `json:"cost,omitempty"`
	Currency            string    `json:"currency,omitempty"`
	SuccessfulLatencies []float64 `json:"successfulLatencies"`
}
type Reliability struct {
	ConfigID     string `json:"configId"`
	EntryID      string `json:"entryId"`
	Successes    int    `json:"successes"`
	Planned      int    `json:"planned"`
	AllSucceeded bool   `json:"allSucceeded"`
}
type Report struct {
	Format          string              `json:"format"`
	PlanID          string              `json:"planId"`
	PlanRevision    string              `json:"planRevision"`
	Mode            string              `json:"mode"`
	Purpose         string              `json:"purpose"`
	Decision        string              `json:"decision"`
	Reason          string              `json:"reason"`
	Metrics         map[string]*Metrics `json:"metrics"`
	Reliability     []Reliability       `json:"repeatReliability"`
	Trials          []Observation       `json:"trials"`
	Interval        Interval            `json:"interval"`
	SourceTrust     string              `json:"sourceTrust"`
	HostObservation string              `json:"hostObservation"`
	Shards          map[string]string   `json:"shards"`
}

// Decide is an internal pure policy function. Public assessment obtains these
// observations only from verified terminals, never from imported score JSON.
func Decide(f Frozen, observations []Observation) (*Report, error) {
	if len(f.Plan.Configs) != 2 || f.Plan.Policy.Delta == nil {
		return nil, errors.New("BASELINE_PLAN_INVALID")
	}
	r := &Report{Format: ReportFormat, PlanID: f.Plan.ID, PlanRevision: f.Plan.Revision, Mode: f.Plan.Mode, Purpose: f.Plan.Purpose, Decision: "invalid", Reason: "missing-or-invalid-evidence", Metrics: map[string]*Metrics{}, Trials: []Observation{}, Reliability: []Reliability{}, SourceTrust: "local-operator-asserted", HostObservation: "contract-test"}
	r.Shards = map[string]string{}
	if f.Plan.Mode == "api-bridge-live" {
		r.HostObservation = "api-bridge-receipts-not-product-evidence"
	}
	byID := map[string]Observation{}
	planned := map[string]bool{}
	for _, t := range f.Trials {
		planned[t.ID] = true
	}
	for _, o := range observations {
		if _, ok := byID[o.TrialID]; ok || !planned[o.TrialID] || o.PolicyFailures < 0 || o.IllegalAttempts < 0 || o.IllegalEffects < 0 {
			return nil, errors.New("BASELINE_EVIDENCE_INVALID")
		}
		byID[o.TrialID] = o
	}
	for _, c := range f.Plan.Configs {
		r.Metrics[c.ID] = &Metrics{CostStatus: "unknown", SuccessfulLatencies: []float64{}}
	}
	valid := true
	hardFail := false
	family := map[string]map[string][]float64{}
	domains := map[string]string{}
	floorSuccess := map[string]int{}
	floorTotal := map[string]int{}
	reliability := map[string]*Reliability{}
	costs := map[string]float64{}
	currency := map[string]string{}
	knownCost := map[string]int{}
	for _, t := range f.Trials {
		m := r.Metrics[t.ConfigID]
		if m == nil {
			return nil, errors.New("BASELINE_PLAN_INVALID")
		}
		m.Planned++
		o, exists := byID[t.ID]
		if !exists {
			o = Observation{TrialID: t.ID, State: "missing", ObservedModel: "unknown", Usage: Usage{Status: "unavailable"}}
		}
		r.Trials = append(r.Trials, o)
		key := t.ConfigID + "/" + t.EntryID
		if reliability[key] == nil {
			reliability[key] = &Reliability{ConfigID: t.ConfigID, EntryID: t.EntryID}
		}
		rr := reliability[key]
		rr.Planned++
		scorable := exists && o.State == "completed" && o.Verified && o.Cleanup == "complete"
		if !exists {
			m.Missing++
		}
		if exists && !o.Verified {
			m.Unverified++
		}
		if exists && o.State != "completed" {
			m.Interrupted++
		}
		if !scorable {
			valid = false
		} else {
			m.Scorable++
			m.Complete++
		}
		success := scorable && o.Success
		if success {
			m.Successes++
			rr.Successes++
			if o.LatencySeconds != nil && *o.LatencySeconds >= 0 {
				m.SuccessfulLatencies = append(m.SuccessfulLatencies, *o.LatencySeconds)
			}
		}
		m.PolicyFailures += o.PolicyFailures
		m.IllegalAttempts += o.IllegalAttempts
		m.IllegalEffects += o.IllegalEffects
		if o.Usage.Status == "reported" && o.Usage.Cost != nil && *o.Usage.Cost >= 0 && o.Usage.Currency != "" {
			if currency[t.ConfigID] == "" {
				currency[t.ConfigID] = o.Usage.Currency
			}
			if currency[t.ConfigID] == o.Usage.Currency {
				knownCost[t.ConfigID]++
				costs[t.ConfigID] += *o.Usage.Cost
			}
		}
		if family[t.FamilyID] == nil {
			family[t.FamilyID] = map[string][]float64{}
		}
		v := 0.0
		if success {
			v = 1
		}
		family[t.FamilyID][t.ConfigID] = append(family[t.FamilyID][t.ConfigID], v)
		domains[t.FamilyID] = t.Domain
		if t.ConfigID == f.Plan.Configs[1].ID {
			floorTotal[t.Domain]++
			if success {
				floorSuccess[t.Domain]++
			}
			if o.IllegalEffects > 0 || (t.Required && !success) {
				hardFail = true
			}
		}
	}
	for id, m := range r.Metrics {
		if m.Planned > 0 {
			m.SuccessRate = float64(m.Successes) / float64(m.Planned)
			m.ScorableRate = float64(m.Scorable) / float64(m.Planned)
		}
		if knownCost[id] == m.Planned && m.Planned > 0 {
			v := costs[id]
			m.Cost = &v
			m.Currency = currency[id]
			m.CostStatus = "reported"
		}
	}
	keys := []string{}
	for key := range reliability {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		rr := reliability[key]
		rr.AllSucceeded = rr.Successes == rr.Planned
		r.Reliability = append(r.Reliability, *rr)
	}
	// New policy failures are evaluated by matched entry/repeat, never netted
	// against gains on other tasks or masked by a baseline's higher total.
	pairs := map[string]map[string]Observation{}
	for _, t := range f.Trials {
		key := t.EntryID + "/" + string(rune(t.Repeat))
		if pairs[key] == nil {
			pairs[key] = map[string]Observation{}
		}
		pairs[key][t.ConfigID] = byID[t.ID]
	}
	for _, pair := range pairs {
		a, b := pair[f.Plan.Configs[0].ID], pair[f.Plan.Configs[1].ID]
		if b.PolicyFailures > a.PolicyFailures {
			hardFail = true
		}
	}
	clusters := []Cluster{}
	for id, configs := range family {
		a, b := configs[f.Plan.Configs[0].ID], configs[f.Plan.Configs[1].ID]
		if len(a) == 0 || len(a) != len(b) {
			valid = false
			continue
		}
		sum := 0.0
		for i := range a {
			sum += b[i] - a[i]
		}
		clusters = append(clusters, Cluster{id, domains[id], sum / float64(len(a))})
	}
	r.Interval = Bootstrap(clusters, f.Plan.Repeats, f.Plan.Policy.Seed)
	if !valid {
		return r, nil
	}
	if hardFail {
		r.Decision = "fail"
		r.Reason = "policy-or-required-task"
		return r, nil
	}
	for domain, total := range floorTotal {
		floor, ok := f.Plan.Policy.Floors[domain]
		if !ok {
			r.Decision = "inconclusive"
			r.Reason = "absolute-floor-unset"
			return r, nil
		}
		if float64(floorSuccess[domain])/float64(total) < floor {
			r.Decision = "fail"
			r.Reason = "absolute-floor"
			return r, nil
		}
	}
	r.Decision = "inconclusive"
	r.Reason = r.Interval.Status
	if r.Interval.Status != "estimated" {
		return r, nil
	}
	if *r.Interval.Lower >= -*f.Plan.Policy.Delta {
		r.Decision = "pass"
		r.Reason = "absolute-and-noninferiority-gates"
	} else if *r.Interval.Upper < -*f.Plan.Policy.Delta {
		r.Decision = "fail"
		r.Reason = "inferiority"
	} else {
		r.Reason = "interval-crosses-margin"
	}
	return r, nil
}
