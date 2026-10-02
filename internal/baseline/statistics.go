package baseline

import (
	"math"
	"sort"
)

type Cluster struct {
	Family     string
	Domain     string
	Difference float64
}
type Interval struct {
	Algorithm  string   `json:"algorithm"`
	Seed       uint64   `json:"seed"`
	Samples    int      `json:"samples"`
	Families   int      `json:"families"`
	Observed   float64  `json:"observedDifference"`
	Lower      *float64 `json:"lower,omitempty"`
	Upper      *float64 `json:"upper,omitempty"`
	Status     string   `json:"status"`
	Population string   `json:"population"`
}

// splitmix64 is fixed here rather than relying on Go's default RNG/version.
type random struct{ state uint64 }

func (r *random) next() uint64 {
	r.state += 0x9e3779b97f4a7c15
	z := r.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}
func (r *random) index(n int) int {
	size := uint64(n)
	limit := uint64(-size) % size
	for {
		x := r.next()
		if x >= limit {
			return int(x % size)
		}
	}
}

func Bootstrap(input []Cluster, repeats int, seed uint64) Interval {
	result := Interval{Algorithm: Algorithm, Seed: seed, Samples: 10000, Families: len(input), Status: "insufficient-sample", Population: "selected families in the two declared synthetic domains; no AGI inference"}
	clusters := append([]Cluster(nil), input...)
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].Family < clusters[j].Family })
	groups := map[string][]float64{}
	seen := map[string]bool{}
	for _, c := range clusters {
		if seen[c.Family] || c.Domain == "" || math.IsNaN(c.Difference) || math.IsInf(c.Difference, 0) || c.Difference < -1 || c.Difference > 1 {
			result.Status = "invalid"
			return result
		}
		seen[c.Family] = true
		groups[c.Domain] = append(groups[c.Domain], c.Difference)
	}
	domains := []string{}
	for domain := range groups {
		domains = append(domains, domain)
	}
	sort.Strings(domains)
	if len(domains) != 2 {
		return result
	}
	for _, domain := range domains {
		for _, v := range groups[domain] {
			result.Observed += v / (2 * float64(len(groups[domain])))
		}
	}
	if repeats < 3 || len(clusters) < 20 || len(groups[domains[0]]) < 8 || len(groups[domains[1]]) < 8 {
		return result
	}
	rng := random{seed}
	samples := make([]float64, 10000)
	for i := range samples {
		for _, domain := range domains {
			values := groups[domain]
			sum := 0.0
			for range values {
				sum += values[rng.index(len(values))]
			}
			samples[i] += sum / (2 * float64(len(values)))
		}
	}
	sort.Float64s(samples)
	if samples[0] == samples[len(samples)-1] {
		result.Status = "degenerate"
		return result
	}
	// Fixed nearest-rank percentile convention, documented with numeric goldens.
	low, high := samples[249], samples[9749]
	result.Lower = &low
	result.Upper = &high
	result.Status = "estimated"
	return result
}
