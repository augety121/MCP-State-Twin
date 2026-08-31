package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/store"
)

func TestPrivateSchedulerLifecycleAndClockDelivery(t *testing.T) {
	_, stateStore := referenceRuntime(t)
	server := httptest.NewServer(NewControlPlane(stateStore, "test-secret"))
	t.Cleanup(server.Close)

	do := func(method, path, body string) *http.Response {
		t.Helper()
		request, err := http.NewRequest(method, server.URL+path, bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", "Bearer test-secret")
		request.Header.Set("Content-Type", "application/json")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		return response
	}

	created := do(http.MethodPost, "/v1/scheduler/events", `{"id":"webhook-arrives","branch":"main","dueAt":"2026-08-01T01:00:00Z","priority":5,"kind":"signal","payload":{"type":"issue.updated"},"expectedHeadVersion":0}`)
	if created.StatusCode != http.StatusCreated {
		created.Body.Close()
		t.Fatalf("create status = %d", created.StatusCode)
	}
	created.Body.Close()

	listed := do(http.MethodGet, "/v1/scheduler/events?branch=main", "")
	var listing struct {
		Format string           `json:"format"`
		Policy string           `json:"policy"`
		Digest string           `json:"digest"`
		Events []map[string]any `json:"events"`
	}
	if err := json.NewDecoder(listed.Body).Decode(&listing); err != nil {
		listed.Body.Close()
		t.Fatal(err)
	}
	listed.Body.Close()
	if listing.Format != store.SchedulerFormat || listing.Policy != store.SchedulerPolicy || len(listing.Events) != 1 || listing.Digest == "" {
		t.Fatalf("listing = %#v", listing)
	}

	advanced := do(http.MethodPost, "/v1/clock/advance", `{"branch":"main","by":"2h","expectedHeadVersion":1}`)
	var result store.ClockAdvanceResult
	if err := json.NewDecoder(advanced.Body).Decode(&result); err != nil {
		advanced.Body.Close()
		t.Fatal(err)
	}
	advanced.Body.Close()
	if advanced.StatusCode != http.StatusOK || len(result.Delivered) != 1 || result.Delivered[0].Status != store.SchedulerDelivered {
		t.Fatalf("advance status=%d result=%#v", advanced.StatusCode, result)
	}

	canceled := do(http.MethodPost, "/v1/scheduler/events/cancel", `{"id":"webhook-arrives","branch":"main"}`)
	defer canceled.Body.Close()
	if canceled.StatusCode != http.StatusConflict {
		t.Fatalf("cancel delivered event status = %d, want 409", canceled.StatusCode)
	}
}
