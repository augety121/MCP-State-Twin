package spec

import "testing"

func FuzzDecodeTwinSpec(f *testing.F) {
	f.Add([]byte(`apiVersion: statetwin.dev/v1alpha1
kind: Twin
metadata:
  name: fuzz
  upstream: {protocol: mcp, status: unbound}
  fidelity: {level: L1, status: unverified}
clock: {mode: virtual, initial: "2026-08-01T00:00:00Z"}
state:
  entities:
    item: {key: [id]}
tools:
  - name: get_item
    description: Get an item.
    inputSchema: {type: object}
`))
	f.Add([]byte("---\n---\n---\n"))
	f.Add([]byte("apiVersion: [unterminated"))

	f.Fuzz(func(t *testing.T, data []byte) {
		twin, err := Decode(data)
		if err != nil {
			return
		}
		if _, err := twin.Digest(); err != nil {
			t.Fatalf("validated TwinSpec cannot be digested: %v", err)
		}
		if _, err := twin.SurfaceDigest(); err != nil {
			t.Fatalf("validated TwinSpec surface cannot be digested: %v", err)
		}
	})
}
