package hostcompat

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestReportRequiresExplicitRedactionBoolean(t *testing.T) {
	for _, level := range []string{"verified", "experimental", "regressed"} {
		for _, token := range []string{"omit", "null", "~", `"false"`, "no", "off", "0", "true", "false"} {
			t.Run(level+"/"+token, func(t *testing.T) {
				r := validGenericReport()
				r.Claim.Level = level
				raw, err := yaml.Marshal(r)
				if err != nil {
					t.Fatal(err)
				}
				replacement := "secretsDetected: " + token
				if token == "omit" {
					replacement = ""
				}
				raw = bytes.Replace(raw, []byte("secretsDetected: false"), []byte(replacement), 1)
				got, err := Decode(raw)
				if token == "false" {
					if err != nil || got == nil {
						t.Fatal(err)
					}
				} else if err == nil || got != nil {
					t.Fatal("missing/coerced/unsafe redaction declaration admitted")
				}
			})
		}
	}
	for _, token := range []string{"null", `"false"`, "0", "true", "false"} {
		raw, err := json.Marshal(validGenericReport())
		if err != nil {
			t.Fatal(err)
		}
		raw = bytes.Replace(raw, []byte(`"secretsDetected":false`), []byte(`"secretsDetected":`+token), 1)
		got, err := Decode(raw)
		if token == "false" {
			if err != nil || got == nil {
				t.Fatal(err)
			}
		} else if err == nil || got != nil {
			t.Fatal("unsafe JSON redaction declaration admitted", token)
		}
	}
}

func TestReportFileAdmission(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "report.yaml")
	raw, err := yaml.Marshal(validGenericReport())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(good, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(good); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{dir, filepath.Join(dir, "private-input-missing")} {
		if got, err := Load(path); err == nil || got != nil || strings.Contains(err.Error(), dir) {
			t.Fatal("invalid file admitted or path echoed", err)
		}
	}
	after, err := os.ReadFile(good)
	if err != nil || !bytes.Equal(raw, after) {
		t.Fatal("input modified", err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(good, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("Windows symlink privileges unavailable")
		}
		t.Fatal(err)
	}
	if got, err := Load(link); err == nil || got != nil {
		t.Fatal("symlink report admitted")
	}
}
