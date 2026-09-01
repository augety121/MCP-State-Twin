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

func TestRuntimeBackedControlPlaneExecutesScheduledTwinSpecAction(t *testing.T) {
	runtime, stateStore := referenceRuntime(t)
	server := httptest.NewServer(NewControlPlaneWithRuntime(stateStore, "test-secret", runtime, "close_issue"))
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

	invalid := do(http.MethodPost, "/v1/scheduler/events", `{"id":"bad","branch":"main","dueAt":"2026-08-01T01:00:00Z","kind":"tool-call","action":{"tool":"close_issue","input":{"owner":"octo"}}}`)
	invalid.Body.Close()
	if invalid.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid action admission status = %d", invalid.StatusCode)
	}
	unknown := do(http.MethodPost, "/v1/scheduler/events", `{"id":"unknown","branch":"main","dueAt":"2026-08-01T01:00:00Z","kind":"tool-call","action":{"tool":"missing_tool","input":{}}}`)
	unknown.Body.Close()
	if unknown.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown action admission status = %d", unknown.StatusCode)
	}

	created := do(http.MethodPost, "/v1/scheduler/events", `{"id":"close-later","branch":"main","dueAt":"2026-08-01T01:00:00Z","kind":"tool-call","action":{"tool":"close_issue","input":{"owner":"octo","repository":"demo","number":1}},"expectedHeadVersion":0}`)
	created.Body.Close()
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("action create status = %d", created.StatusCode)
	}

	ordinary := do(http.MethodPost, "/v1/clock/advance", `{"branch":"main","to":"2026-08-01T02:00:00Z","expectedHeadVersion":1}`)
	ordinary.Body.Close()
	if ordinary.StatusCode != http.StatusConflict {
		t.Fatalf("ordinary action advance status = %d", ordinary.StatusCode)
	}

	advanced := do(http.MethodPost, "/v1/clock/advance-next", `{"branch":"main","expectedHeadVersion":1}`)
	var result store.SchedulerStepResult
	if err := json.NewDecoder(advanced.Body).Decode(&result); err != nil {
		advanced.Body.Close()
		t.Fatal(err)
	}
	advanced.Body.Close()
	if advanced.StatusCode != http.StatusOK || result.ExecutedActions != 1 || len(result.Processed) != 1 || result.Processed[0].Status != store.SchedulerCompleted {
		t.Fatalf("action advance status=%d result=%#v", advanced.StatusCode, result)
	}
	branch, err := stateStore.Branch(t.Context(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.State.Entities["issue"]["octo/demo#1"]["state"] != "closed" || branch.CallCount != 1 {
		t.Fatalf("scheduled close branch = %#v", branch)
	}
	listed := do(http.MethodGet, "/v1/scheduler/events?branch=main&status=completed&limit=10", "")
	var page store.ScheduledEventPage
	if err := json.NewDecoder(listed.Body).Decode(&page); err != nil {
		listed.Body.Close()
		t.Fatal(err)
	}
	listed.Body.Close()
	if listed.StatusCode != http.StatusOK || len(page.Events) != 1 || page.Events[0].ID != "close-later" {
		t.Fatalf("completed action page status=%d page=%#v", listed.StatusCode, page)
	}

	plainServer := httptest.NewServer(NewControlPlane(stateStore, "plain-secret", "close_issue"))
	t.Cleanup(plainServer.Close)
	request, _ := http.NewRequest(http.MethodPost, plainServer.URL+"/v1/scheduler/events", bytes.NewBufferString(`{"id":"no-runtime","branch":"main","dueAt":"2026-08-01T02:00:00Z","kind":"tool-call","action":{"tool":"close_issue","input":{"owner":"octo","repository":"demo","number":1}}}`))
	request.Header.Set("Authorization", "Bearer plain-secret")
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("non-runtime action admission status = %d", response.StatusCode)
	}
}
