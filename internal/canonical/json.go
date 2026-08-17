package canonical

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// JSON returns the stable JSON representation used by State Twin artifacts.
// The supported value domain is ordinary JSON plus Go structs with JSON tags.
// encoding/json sorts string map keys, and the decode/re-encode pass removes
// insignificant source formatting while preserving integer precision.
func JSON(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical input: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var normalized any
	if err := decoder.Decode(&normalized); err != nil {
		return nil, fmt.Errorf("normalize canonical input: %w", err)
	}
	result, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal normalized input: %w", err)
	}
	return result, nil
}

func Digest(v any) (string, error) {
	data, err := JSON(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
