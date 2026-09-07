package agenthost

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

// decodeJSON rejects duplicate keys before decoding. JSON number spelling is
// retained until conversion to an exact int64 or a finite JSON float.
func decodeJSON(data []byte, max int, target any) error {
	if len(data) == 0 || len(data) > max || !utf8.Valid(data) {
		return errors.New("HOST_PROTOCOL_ERROR")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := checkValue(d, 0); err != nil {
		return errors.New("HOST_PROTOCOL_ERROR")
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("HOST_PROTOCOL_ERROR")
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := d.Decode(target); err != nil {
		return errors.New("HOST_PROTOCOL_ERROR")
	}
	return nil
}

// DecodeDocument adds closed-field admission for local evidence/configuration.
// The caller normalizes dynamic numbers only where its data model requires it.
func DecodeDocument(data []byte, max int, target any) error {
	if err := decodeJSON(data, max, target); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	d.DisallowUnknownFields()
	if d.Decode(target) != nil {
		return errors.New("HOST_PROTOCOL_ERROR")
	}
	return nil
}

func NormalizeNumbers(v any) (any, error) { return numbers(v) }

func checkValue(d *json.Decoder, depth int) error {
	if depth > 32 {
		return errors.New("depth")
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	if delim != '{' && delim != '[' {
		return errors.New("unexpected delimiter")
	}
	seen := map[string]bool{}
	for d.More() {
		if delim == '{' {
			key, err := d.Token()
			if err != nil {
				return err
			}
			s, ok := key.(string)
			if !ok || seen[s] {
				return errors.New("duplicate or invalid key")
			}
			seen[s] = true
		}
		if err := checkValue(d, depth+1); err != nil {
			return err
		}
	}
	end, err := d.Token()
	if err != nil || (delim == '{' && end != json.Delim('}')) || (delim == '[' && end != json.Delim(']')) {
		return errors.New("invalid close")
	}
	return nil
}

func numbers(v any) (any, error) {
	switch x := v.(type) {
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i, nil
		}
		f, err := x.Float64()
		if err != nil {
			return nil, errors.New("HOST_PROTOCOL_ERROR")
		}
		return f, nil
	case map[string]any:
		for k, value := range x {
			n, err := numbers(value)
			if err != nil {
				return nil, err
			}
			x[k] = n
		}
	case []any:
		for i, value := range x {
			n, err := numbers(value)
			if err != nil {
				return nil, err
			}
			x[i] = n
		}
	}
	return v, nil
}
