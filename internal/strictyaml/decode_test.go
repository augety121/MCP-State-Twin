package strictyaml

import (
	"strings"
	"testing"
)

func TestDecodeOneRejectsAliasesTagsAndMultipleDocuments(t *testing.T) {
	for name, input := range map[string]string{
		"alias":    "value: &shared x\ncopy: *shared\n",
		"tag":      "value: !!str x\n",
		"multiple": "value: x\n---\nvalue: y\n",
	} {
		t.Run(name, func(t *testing.T) {
			var target struct {
				Value string `yaml:"value"`
			}
			if err := DecodeOne([]byte(input), 1024, "test", &target); err == nil {
				t.Fatal("expected strict YAML rejection")
			}
		})
	}
}

func TestDecodeOneRejectsUnknownFieldsAndSize(t *testing.T) {
	var target struct {
		Value string `yaml:"value"`
	}
	if err := DecodeOne([]byte("value: x\nunknown: y\n"), 1024, "test", &target); err == nil || !strings.Contains(err.Error(), "field unknown") {
		t.Fatalf("expected unknown-field rejection, got %v", err)
	}
	if err := DecodeOne([]byte(strings.Repeat("x", 9)), 8, "test", &target); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected size rejection, got %v", err)
	}
}

func TestDecodeOneRejectsDeepDocuments(t *testing.T) {
	input := strings.Repeat("[", MaxDocumentDepth+2) + "x" + strings.Repeat("]", MaxDocumentDepth+2)
	var target any
	if err := DecodeOne([]byte(input), 4096, "test", &target); err == nil || !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("expected depth rejection, got %v", err)
	}
}
