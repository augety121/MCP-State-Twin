package logging

import (
	"errors"
	"strings"
	"testing"
)

func TestRedactsOperationalSecretsAndIdentifiers(t *testing.T) {
	message := "Authorization: Bearer abc123 token=secret-value api_key=sk-12345678901234567890 provider sk-abcdefghijklmnop1234 contact=alice@example.com\n-----BEGIN PRIVATE KEY-----secret-----END PRIVATE KEY-----"
	redacted := Redact(message)
	for _, value := range []string{"abc123", "secret-value", "sk-12345678901234567890", "sk-abcdefghijklmnop1234", "alice@example.com", "BEGIN PRIVATE KEY"} {
		if strings.Contains(redacted, value) {
			t.Fatalf("redacted message still contains %q: %s", value, redacted)
		}
	}
	for _, marker := range []string{"[REDACTED]", "[REDACTED_TOKEN]", "[REDACTED_EMAIL]", "[REDACTED_PRIVATE_KEY]"} {
		if !strings.Contains(redacted, marker) {
			t.Fatalf("redacted message is missing marker %q: %s", marker, redacted)
		}
	}
}

func TestSafeErrorPreservesNonSensitiveContext(t *testing.T) {
	if got := SafeError(errors.New("database unavailable")); got != "database unavailable" {
		t.Fatalf("unexpected safe error: %q", got)
	}
}
