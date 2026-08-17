package canonical

import "testing"

func TestDigestIgnoresMapInsertionOrder(t *testing.T) {
	a := map[string]any{"b": 2, "a": map[string]any{"y": true, "x": "v"}}
	b := map[string]any{"a": map[string]any{"x": "v", "y": true}, "b": 2}

	da, err := Digest(a)
	if err != nil {
		t.Fatal(err)
	}
	db, err := Digest(b)
	if err != nil {
		t.Fatal(err)
	}
	if da != db {
		t.Fatalf("digests differ: %s != %s", da, db)
	}
}

func TestJSONRejectsUnsupportedValues(t *testing.T) {
	if _, err := JSON(map[string]any{"bad": func() {}}); err == nil {
		t.Fatal("expected unsupported value error")
	}
}
