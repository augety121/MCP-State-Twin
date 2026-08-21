package limits

import (
	"bytes"
	"encoding/json"
)

func unmarshalCanonical(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode(target)
}
