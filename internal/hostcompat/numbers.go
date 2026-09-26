package hostcompat

import "gopkg.in/yaml.v3"

var limitFields = []string{
	"providerRequests", "toolCalls", "wallTimeMs", "maxTraceBytes",
	"retriesPerProviderRequest", "retriesPerToolCall", "repeatedIdenticalCalls",
}

// Called only AFTER bounded strict decoding has rejected unknown fields,
// aliases, duplicates and excessive depth. yaml.v3 can truncate a YAML float
// into a Go int; inspect the original scalar tags rather than the coerced value.
func explicitIntegerFields(raw []byte, report bool) bool {
	var doc yaml.Node
	if yaml.Unmarshal(raw, &doc) != nil || len(doc.Content) != 1 {
		return false
	}
	root := doc.Content[0]
	for _, field := range limitFields {
		if !integerAt(root, "trial", "limits", field) {
			return false
		}
	}
	return !report || (integerAt(root, "trial", "index") &&
		integerAt(root, "evidence", "assertionSummary", "passed") &&
		integerAt(root, "evidence", "assertionSummary", "failed"))
}

func integerAt(n *yaml.Node, path ...string) bool {
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
	return n != nil && n.Kind == yaml.ScalarNode && n.Tag == "!!int"
}
