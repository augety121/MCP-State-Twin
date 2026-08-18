package strictyaml

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

const MaxDocumentDepth = 128

// DecodeOne decodes one bounded YAML document with a closed field set.
// Anchors, aliases, and explicit tags are rejected before typed decoding.
func DecodeOne(data []byte, maxBytes int, label string, target any) error {
	if len(data) > maxBytes {
		return fmt.Errorf("decode %s: document exceeds %d bytes", label, maxBytes)
	}

	nodeDecoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := nodeDecoder.Decode(&document); err != nil {
		return fmt.Errorf("decode %s: %w", label, err)
	}
	if err := rejectExtensions(&document, 0); err != nil {
		return fmt.Errorf("decode %s: %w", label, err)
	}
	var trailing yaml.Node
	if err := nodeDecoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode %s: multiple YAML documents are not allowed", label)
		}
		return fmt.Errorf("decode %s trailing document: %w", label, err)
	}

	typedDecoder := yaml.NewDecoder(bytes.NewReader(data))
	typedDecoder.KnownFields(true)
	if err := typedDecoder.Decode(target); err != nil {
		return fmt.Errorf("decode %s: %w", label, err)
	}
	return nil
}

func rejectExtensions(node *yaml.Node, depth int) error {
	if node == nil {
		return errors.New("empty YAML document")
	}
	if depth > MaxDocumentDepth {
		return fmt.Errorf("YAML document exceeds depth limit %d", MaxDocumentDepth)
	}
	if node.Kind == yaml.AliasNode || node.Anchor != "" {
		return errors.New("YAML anchors and aliases are not allowed")
	}
	if node.Style&yaml.TaggedStyle != 0 {
		return errors.New("explicit YAML tags are not allowed")
	}
	for _, child := range node.Content {
		if err := rejectExtensions(child, depth+1); err != nil {
			return err
		}
	}
	return nil
}
