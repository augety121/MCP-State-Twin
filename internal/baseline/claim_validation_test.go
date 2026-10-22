package baseline

import (
	"reflect"
	"testing"
	"time"
)

func TestClaimValidatesBeforeStatusClassification(t *testing.T) {
	now := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	b := Binding{"runtime", "head", "host", "version", "adapter", "protocol", "pack", "revision", "oracle", "statistics", "evidence"}
	claim := Claim{Format: "statetwin.dev/baseline-host-claim/v1alpha1", Binding: b, Kind: "api", ObservedAt: now.Format(time.RFC3339), ValidUntil: now.Add(time.Hour).Format(time.RFC3339), Observation: "contract-test", Outcome: "passed", EvidenceRef: "synthetic.json"}
	for _, status := range []string{"current", "expired", "revoked", "failed"} {
		t.Run(status, func(t *testing.T) {
			c, at := claim, now
			want := "experimental"
			switch status {
			case "expired":
				at, want = now.Add(2*time.Hour), "stale"
			case "revoked":
				c.Revoked, c.RevocationReason, want = true, "regression", "regressed"
			case "failed":
				c.Outcome, want = "failed", "regressed"
			}
			if r, err := AssessClaim(c, b, at); err != nil || r.Status != want || r.PublicationAllowed {
				t.Fatalf("valid %s: %+v %v", status, r, err)
			}
			check := func(in Claim, current Binding) {
				t.Helper()
				if r, err := AssessClaim(in, current, at); err == nil || err.Error() != "BASELINE_CLAIM_INVALID" || r.PublicationAllowed {
					t.Fatalf("invalid declaration was classified: %+v %v", r, err)
				}
			}
			for i := 0; i < reflect.TypeOf(b).NumField(); i++ {
				for _, blank := range []string{"", " \t\n"} {
					incomplete := b
					reflect.ValueOf(&incomplete).Elem().Field(i).SetString(blank)
					bad := c
					bad.Binding = incomplete
					check(bad, incomplete)
					check(c, incomplete)
				}
			}
			for _, field := range []string{"Outcome", "Observation", "EvidenceRef"} {
				bad := c
				reflect.ValueOf(&bad).Elem().FieldByName(field).SetString(" ")
				check(bad, b)
			}
			bad := c
			bad.Revoked, bad.RevocationReason = true, " "
			check(bad, b)
			bad.Revoked, bad.RevocationReason = false, "contradictory-revocation"
			check(bad, b)
		})
	}
}
