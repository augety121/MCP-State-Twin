package logging

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	authorizationPattern    = regexp.MustCompile(`(?i)(authorization\s*[:=]\s*(?:bearer|basic)\s+)[^\s,;]+`)
	keyValuePattern         = regexp.MustCompile(`(?i)((?:api[_-]?key|token|secret|password)\s*[:=]\s*)[^\s,;]+`)
	knownTokenPattern       = regexp.MustCompile(`(?i)\b(?:sk|rk)-[A-Za-z0-9_-]{16,}\b|\bgh[pousr]_[A-Za-z0-9]{20,}\b|\bAIza[0-9A-Za-z_-]{30,}\b`)
	privateKeyPattern       = regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)
	emailPattern            = regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`)
	jsonCredentialPattern   = regexp.MustCompile(`(?i)("(?:authorization|api[_-]?key|access[_-]?token|refresh[_-]?token|client[_-]?secret|private[_-]?key|token|secret|password)"\s*:\s*)"(?:[^"\\]|\\.)*"`)
	credentialKeyNormalizer = strings.NewReplacer("_", "", "-", "")
)

// SafeError is for operational logs only. It does not change errors returned
// to an MCP client or persisted in the audit record.
func SafeError(err error) string {
	if err == nil {
		return ""
	}
	return Redact(err.Error())
}

func Redact(message string) string {
	if json.Valid([]byte(message)) && sensitiveJSON(message, 0) {
		return "[REDACTED_JSON]"
	}
	message = jsonCredentialPattern.ReplaceAllString(message, `${1}"[REDACTED]"`)
	message = privateKeyPattern.ReplaceAllString(message, "[REDACTED_PRIVATE_KEY]")
	message = authorizationPattern.ReplaceAllString(message, "$1[REDACTED]")
	message = keyValuePattern.ReplaceAllString(message, "$1[REDACTED]")
	message = knownTokenPattern.ReplaceAllString(message, "[REDACTED_TOKEN]")
	message = emailPattern.ReplaceAllString(message, "[REDACTED_EMAIL]")
	return message
}

// Stream valid JSON rather than allocating a second arbitrary object tree.
// Decode field names/values before matching, including escaped key names and
// bounded JSON carried inside string fields (for example tool arguments).
// This is finite credential-pattern admission, not universal secret discovery.
func sensitiveJSON(message string, embedded int) bool {
	d := json.NewDecoder(strings.NewReader(message))
	d.UseNumber()
	type frame struct {
		object, key bool
		name        string
		authArray   bool
	}
	stack := []frame{}
	consume := func() {
		if len(stack) > 0 && stack[len(stack)-1].object {
			f := &stack[len(stack)-1]
			f.key = true
			f.name = ""
		}
	}
	for d.More() || len(stack) > 0 {
		token, err := d.Token()
		if err != nil {
			return false
		}
		if len(stack) > 0 {
			f := &stack[len(stack)-1]
			if f.object && f.key {
				if key, ok := token.(string); ok {
					f.name = key
					f.key = false
					continue
				}
			}
			_, scalarString := token.(string)
			if f.authArray && scalarString && token != "" {
				return true
			}
			// Task authorization is an array of resource rules, not an HTTP
			// credential. Only its string-valued header form is credential-like.
			credential := credentialField(f.name) && (strings.ToLower(f.name) != "authorization" || scalarString)
			if f.object && !f.key && credential && token != nil && token != "" {
				return true
			}
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{', '[':
				authArray := false
				if delim == '[' && len(stack) > 0 {
					parent := stack[len(stack)-1]
					authArray = parent.authArray || strings.EqualFold(parent.name, "authorization")
				}
				consume()
				stack = append(stack, frame{object: delim == '{', key: delim == '{', authArray: authArray})
				if len(stack) > 128 {
					return true
				}
			case '}', ']':
				if len(stack) == 0 {
					return false
				}
				stack = stack[:len(stack)-1]
			}
			continue
		}
		if text, ok := token.(string); ok {
			if plainSensitive(text) {
				return true
			}
			if json.Valid([]byte(text)) {
				if embedded >= 4 || sensitiveJSON(text, embedded+1) {
					return true
				}
			}
		}
		consume()
	}
	return false
}
func credentialField(key string) bool {
	key = strings.ToLower(credentialKeyNormalizer.Replace(key))
	switch key {
	case "authorization", "apikey", "accesstoken", "refreshtoken", "clientsecret", "privatekey", "token", "secret", "password":
		return true
	}
	return false
}
func plainSensitive(message string) bool {
	return authorizationPattern.MatchString(message) || keyValuePattern.MatchString(message) || knownTokenPattern.MatchString(message) || privateKeyPattern.MatchString(message) || emailPattern.MatchString(message) || jsonCredentialPattern.MatchString(message)
}

// ContainsSensitive detects known text and structured JSON credential-like or
// email/private-key patterns. It is intended for fail-closed artifact admission;
// callers should not log the rejected value.
func ContainsSensitive(message string) bool {
	return plainSensitive(message) || (json.Valid([]byte(message)) && sensitiveJSON(message, 0))
}
