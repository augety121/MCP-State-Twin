package episode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestCoordinatorRequiresAuthRejectsUnknownFieldsAndServesClaim(t *testing.T) {
	journal, err := OpenJournal(filepath.Join(t.TempDir(), "episodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	coordinator, err := NewCoordinator(journal, "synthetic-control-token")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(coordinator)
	defer server.Close()

	unauthorized, err := http.Post(server.URL+"/v1/claims", "application/json", bytes.NewBufferString(`{"workerId":"w","leaseSeconds":30}`))
	if err != nil {
		t.Fatal(err)
	}
	unauthorized.Body.Close()
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorized.StatusCode)
	}

	bad := authenticatedRequest(t, http.MethodPost, server.URL+"/v1/claims", "synthetic-control-token", `{"workerId":"w","leaseSeconds":30,"unknown":true}`)
	if bad.StatusCode != http.StatusBadRequest {
		bad.Body.Close()
		t.Fatalf("unknown-field status = %d", bad.StatusCode)
	}
	bad.Body.Close()

	data := buildIssueTrackerBundleBytes(t)
	submitBody, err := json.Marshal(map[string]any{
		"bundle": data, "episodeId": "http-remote-001", "runtimeVersion": "0.2.0-dev",
		"runtimeRevision": "revision-1", "effectProfile": "hermetic", "maxAttempts": 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	submitted := authenticatedRequest(t, http.MethodPost, server.URL+"/v1/episodes", "synthetic-control-token", string(submitBody))
	if submitted.StatusCode != http.StatusCreated {
		submitted.Body.Close()
		t.Fatalf("submit status = %d", submitted.StatusCode)
	}
	submitted.Body.Close()

	client, err := NewCoordinatorClient(server.URL, "synthetic-control-token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	claim, err := client.Claim(context.Background(), "worker-http", 30, EffectHermetic)
	if err != nil {
		t.Fatal(err)
	}
	if claim.EpisodeID != "http-remote-001" || claim.AttemptID == "" || claim.FencingToken != 1 {
		t.Fatalf("claim = %+v", claim)
	}
	heartbeat, err := client.Heartbeat(context.Background(), claim, 30)
	if err != nil || heartbeat.CancelRequested || heartbeat.LeaseUntil == "" {
		t.Fatalf("heartbeat = %+v, %v", heartbeat, err)
	}
}

func TestCoordinatorNeverEchoesBearerTokenInErrors(t *testing.T) {
	journal, err := OpenJournal(filepath.Join(t.TempDir(), "episodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	token := "synthetic-super-secret-token"
	coordinator, err := NewCoordinator(journal, token)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/claims", bytes.NewBufferString(`{"workerId":"worker","leaseSeconds":30}`))
	request.Header.Set("Authorization", "Bearer wrong-"+token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	coordinator.ServeHTTP(recorder, request)
	if bytes.Contains(recorder.Body.Bytes(), []byte(token)) {
		t.Fatal("coordinator error echoed bearer token")
	}
}

func TestCoordinatorClientRejectsUnsafeURLAndReturnsTypedErrors(t *testing.T) {
	for _, endpoint := range []string{"http://example.com", "https://user:secret@example.com"} {
		if _, err := NewCoordinatorClient(endpoint, "synthetic-token", nil); err == nil {
			t.Fatalf("unsafe endpoint %q was accepted", endpoint)
		}
	}
	journal, err := OpenJournal(filepath.Join(t.TempDir(), "episodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	coordinator, err := NewCoordinator(journal, "synthetic-control-token")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(coordinator)
	defer server.Close()
	client, err := NewCoordinatorClient(server.URL, "synthetic-control-token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Claim(context.Background(), "empty-worker", 30, EffectHermetic)
	var requestError *CoordinatorRequestError
	if !errors.As(err, &requestError) || requestError.StatusCode != http.StatusNotFound || requestError.Code != "NO_TASK" {
		t.Fatalf("empty claim error = %#v", err)
	}
}

func authenticatedRequest(t *testing.T, method, url, token, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
