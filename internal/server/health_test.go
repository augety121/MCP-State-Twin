package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestControlHealthIsAuthenticatedAndRedacted(t *testing.T) {
	_, stateStore := referenceRuntime(t)
	control := NewControlPlane(stateStore, "health-secret", "close_issue")

	unauthorized := httptest.NewRecorder()
	control.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/v1/health/live", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated health status=%d", unauthorized.Code)
	}

	for _, path := range []string{"/v1/health/live", "/v1/health/ready"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer health-secret")
		response := httptest.NewRecorder()
		control.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s status=%d body=%s", path, response.Code, response.Body.String())
		}
		body := response.Body.String()
		var decoded map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("%s invalid JSON: %v", path, err)
		}
		check := strings.TrimPrefix(path, "/v1/health/")
		want := map[string]any{"format": "statetwin.dev/health/v1alpha1", "check": check, "status": "ok", "version": Version}
		if !reflect.DeepEqual(decoded, want) {
			t.Fatalf("%s response=%#v want=%#v", path, decoded, want)
		}
		for _, forbidden := range []string{"health-secret", "main", "close_issue", "snapshot", "branch"} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("%s exposed %q in %s", path, forbidden, body)
			}
		}
	}
}

func TestControlReadinessFailsClosedWithoutStorageDetails(t *testing.T) {
	_, stateStore := referenceRuntime(t)
	control := NewControlPlane(stateStore, "health-secret")
	if err := stateStore.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/health/ready", nil)
	request.Header.Set("Authorization", "Bearer health-secret")
	response := httptest.NewRecorder()
	control.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), `"code":"NOT_READY"`) {
		t.Fatalf("readiness status=%d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(strings.ToLower(response.Body.String()), "database") || strings.Contains(response.Body.String(), "sql:") {
		t.Fatalf("readiness exposed storage internals: %s", response.Body.String())
	}
}
