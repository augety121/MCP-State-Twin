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

func TestPrivateSchedulerPaginationPreviewAndAdvanceNext(t *testing.T) {
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

	for _, body := range []string{
		`{"id":"first","branch":"main","dueAt":"2026-08-01T01:00:00Z","kind":"signal","payload":{}}`,
		`{"id":"second","branch":"main","dueAt":"2026-08-01T02:00:00Z","kind":"signal","payload":{}}`,
	} {
		response := do(http.MethodPost, "/v1/scheduler/events", body)
		response.Body.Close()
		if response.StatusCode != http.StatusCreated {
			t.Fatalf("schedule status = %d", response.StatusCode)
		}
	}

	firstPageResponse := do(http.MethodGet, "/v1/scheduler/events?branch=main&status=pending&limit=1", "")
	var firstPage store.ScheduledEventPage
	if err := json.NewDecoder(firstPageResponse.Body).Decode(&firstPage); err != nil {
		firstPageResponse.Body.Close()
		t.Fatal(err)
	}
	firstPageResponse.Body.Close()
	if firstPageResponse.StatusCode != http.StatusOK || len(firstPage.Events) != 1 || firstPage.NextCursor == "" {
		t.Fatalf("first page status=%d page=%#v", firstPageResponse.StatusCode, firstPage)
	}
	secondPageResponse := do(http.MethodGet, "/v1/scheduler/events?branch=main&status=pending&limit=1&cursor="+firstPage.NextCursor, "")
	var secondPage store.ScheduledEventPage
	if err := json.NewDecoder(secondPageResponse.Body).Decode(&secondPage); err != nil {
		secondPageResponse.Body.Close()
		t.Fatal(err)
	}
	secondPageResponse.Body.Close()
	if secondPageResponse.StatusCode != http.StatusOK || len(secondPage.Events) != 1 || secondPage.Events[0].ID != "second" {
		t.Fatalf("second page status=%d page=%#v", secondPageResponse.StatusCode, secondPage)
	}

	previewResponse := do(http.MethodGet, "/v1/scheduler/next?branch=main", "")
	var preview store.NextDueBatch
	if err := json.NewDecoder(previewResponse.Body).Decode(&preview); err != nil {
		previewResponse.Body.Close()
		t.Fatal(err)
	}
	previewResponse.Body.Close()
	if previewResponse.StatusCode != http.StatusOK || preview.DueAt != "2026-08-01T01:00:00Z" || preview.BatchSize != 1 {
		t.Fatalf("preview status=%d batch=%#v", previewResponse.StatusCode, preview)
	}

	stepResponse := do(http.MethodPost, "/v1/clock/advance-next", `{"branch":"main","expectedHeadVersion":2}`)
	var step store.SchedulerStepResult
	if err := json.NewDecoder(stepResponse.Body).Decode(&step); err != nil {
		stepResponse.Body.Close()
		t.Fatal(err)
	}
	stepResponse.Body.Close()
	if stepResponse.StatusCode != http.StatusOK || len(step.Delivered) != 1 || step.Delivered[0].ID != "first" || !step.ClockAdvanced || step.NextDueAt != "2026-08-01T02:00:00Z" {
		t.Fatalf("step status=%d result=%#v", stepResponse.StatusCode, step)
	}

	stalePage := do(http.MethodGet, "/v1/scheduler/events?branch=main&status=pending&limit=1&cursor="+firstPage.NextCursor, "")
	defer stalePage.Body.Close()
	if stalePage.StatusCode != http.StatusConflict {
		t.Fatalf("stale page status = %d, want 409", stalePage.StatusCode)
	}
}
