package logging

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestRedactsOperationalSecretsAndIdentifiers(t *testing.T) {
	fakeAPIKey := "sk-" + strings.Repeat("a", 20)
	fakeProviderKey := "sk-" + strings.Repeat("b", 20)
	message := "Authorization: Bearer abc123 token=secret-value api_key=" + fakeAPIKey + " provider " + fakeProviderKey + " contact=alice@example.com\n-----BEGIN PRIVATE KEY-----secret-----END PRIVATE KEY-----"
	redacted := Redact(message)
	for _, value := range []string{"abc123", "secret-value", fakeAPIKey, fakeProviderKey, "alice@example.com", "BEGIN PRIVATE KEY"} {
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

func TestContainsSensitive(t *testing.T) {
	if !ContainsSensitive("Authorization: Bearer abc123") {
		t.Fatal("authorization credential was not detected")
	}
	if ContainsSensitive("sha256:0123456789abcdef") {
		t.Fatal("digest-like value was incorrectly classified as sensitive")
	}
}

func TestStructuredJSONCredentialAdmission(t *testing.T) {
	for _, raw := range []string{
		`{"api_key":"synthetic-private-sentinel"}`,
		`{"nested":[{"api\u005fkey":"synthetic-private-sentinel"}]}`,
		`{"Authorization":"Bearer synthetic-private-sentinel"}`,
		`{"authorization":["Bearer synthetic-private-sentinel"]}`,
		`{"refresh-token":"synthetic-private-sentinel"}`,
		`{"password":123456}`,
		`{"secret":{"nested":"synthetic-private-sentinel"}}`,
		`{"arguments":"{\"api_key\":\"synthetic-private-sentinel\"}"}`,
		`{"text":"api_key\u003dsynthetic-private-sentinel"}`,
	} {
		if !ContainsSensitive(raw) {
			t.Fatal("structured credential missed", raw)
		}
		if strings.Contains(Redact(raw), "synthetic-private-sentinel") {
			t.Fatal("structured secret leaked")
		}
	}
	for _, raw := range []string{`{"authorization":[{"tool":"get_issue","equals":{"owner":"octo"}}]}`, `{"tokens":{"input":1,"output":2,"total":3}}`, `{"policy":{"required":true},"events":[]}`, `{"task":"token refresh review","value":null}`, `[null,false,12,"ordinary text"]`} {
		if ContainsSensitive(raw) {
			t.Fatal("noncredential metadata rejected", raw)
		}
	}
}

func TestJSONCredentialScanResourceBounds(t *testing.T) {
	deep := strings.Repeat("[", 129) + "null" + strings.Repeat("]", 129)
	if !ContainsSensitive(deep) {
		t.Fatal("excessive JSON depth silently admitted")
	}
	nested := `{"ordinary":"text"}`
	for i := 0; i < 6; i++ {
		b, _ := json.Marshal(nested)
		nested = string(b)
	}
	if !ContainsSensitive(nested) {
		t.Fatal("excessive embedded JSON silently skipped")
	}
}
