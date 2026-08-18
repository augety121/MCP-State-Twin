package logging

import "regexp"

var (
	authorizationPattern = regexp.MustCompile(`(?i)(authorization\s*[:=]\s*(?:bearer|basic)\s+)[^\s,;]+`)
	keyValuePattern      = regexp.MustCompile(`(?i)((?:api[_-]?key|token|secret|password)\s*[:=]\s*)[^\s,;]+`)
	knownTokenPattern    = regexp.MustCompile(`(?i)\b(?:sk|rk)-[A-Za-z0-9_-]{16,}\b|\bgh[pousr]_[A-Za-z0-9]{20,}\b|\bAIza[0-9A-Za-z_-]{30,}\b`)
	privateKeyPattern    = regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z0-9 ]*PRIVATE KEY-----`)
	emailPattern         = regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}\b`)
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
	message = privateKeyPattern.ReplaceAllString(message, "[REDACTED_PRIVATE_KEY]")
	message = authorizationPattern.ReplaceAllString(message, "$1[REDACTED]")
	message = keyValuePattern.ReplaceAllString(message, "$1[REDACTED]")
	message = knownTokenPattern.ReplaceAllString(message, "[REDACTED_TOKEN]")
	message = emailPattern.ReplaceAllString(message, "[REDACTED_EMAIL]")
	return message
}
