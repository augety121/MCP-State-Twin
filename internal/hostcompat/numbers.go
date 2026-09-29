package hostcompat

import (
	"errors"

	"gopkg.in/yaml.v3"
)

var limitFields = []string{
	"providerRequests", "toolCalls", "wallTimeMs", "maxTraceBytes",
	"retriesPerProviderRequest", "retriesPerToolCall", "repeatedIdenticalCalls",
}

// Called only AFTER bounded strict decoding has rejected unknown fields,
// aliases, duplicates and excessive depth. yaml.v3 can truncate a YAML float
// into a Go int; inspect the original scalar tags rather than the coerced value.
func explicitScalarFields(raw []byte, report bool) error {
	prefix := "HOST_TARGET"
	if report {
		prefix = "HOST_REPORT"
	}
	var doc yaml.Node
	if yaml.Unmarshal(raw, &doc) != nil || len(doc.Content) != 1 {
		return errors.New(prefix + "_DECODE_INVALID")
	}
	root := doc.Content[0]
	for _, field := range limitFields {
		if !scalarAt(root, "!!int", "trial", "limits", field) {
			return errors.New(prefix + "_INTEGER_FIELDS_REQUIRED")
		}
	}
	if report {
		if !scalarAt(root, "!!int", "trial", "index") ||
			!scalarAt(root, "!!int", "evidence", "assertionSummary", "passed") ||
			!scalarAt(root, "!!int", "evidence", "assertionSummary", "failed") {
			return errors.New(prefix + "_INTEGER_FIELDS_REQUIRED")
		}
		// Missing/null values become false in Go; YAML 1.1 no/off spellings
		// also coerce to false. Neither is an explicit YAML boolean declaration.
		if !scalarAt(root, "!!bool", "redaction", "secretsDetected") {
			return errors.New("HOST_REPORT_REDACTION_BOOLEAN_REQUIRED")
		}
	}
	return nil
}

func scalarAt(n *yaml.Node, tag string, path ...string) bool {
	for _, key := range path {
		if n == nil || n.Kind != yaml.MappingNode {
			return false
		}
		var next *yaml.Node
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == key {
				next = n.Content[i+1]
				break
			}
		}
		n = next
	}
	return n != nil && n.Kind == yaml.ScalarNode && n.Tag == tag
}
