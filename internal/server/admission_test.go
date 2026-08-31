package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAdmissionRejectsWithoutQueueingAndRecovers(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int64
	admission, err := NewAdmissionHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			close(entered)
		}
		<-release
		w.WriteHeader(http.StatusNoContent)
	}), 1)
	if err != nil {
		t.Fatal(err)
	}

	first := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		admission.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
		close(done)
	}()
	<-entered

	second := httptest.NewRecorder()
	admission.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/", nil))
	if second.Code != http.StatusServiceUnavailable || second.Header().Get("Retry-After") != "1" {
		t.Fatalf("overload response status=%d headers=%v", second.Code, second.Header())
	}
	if !strings.Contains(second.Body.String(), `"code":"SERVER_BUSY"`) || calls.Load() != 1 {
		t.Fatalf("overload reached application or lost typed error: calls=%d body=%s", calls.Load(), second.Body.String())
	}

	close(release)
	<-done
	third := httptest.NewRecorder()
	thirdRequest := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(context.Background())
	admission.ServeHTTP(third, thirdRequest)
	if third.Code != http.StatusNoContent || calls.Load() != 2 {
		t.Fatalf("capacity did not recover: status=%d calls=%d", third.Code, calls.Load())
	}
	stats := admission.Stats()
	if stats.Limit != 1 || stats.Active != 0 || stats.Rejected != 1 {
		t.Fatalf("unexpected admission stats: %#v", stats)
	}
}

func TestAdmissionConfigurationFailsClosed(t *testing.T) {
	for _, limit := range []int{-1, 0, 1025} {
		if _, err := NewAdmissionHandler(http.NotFoundHandler(), limit); err == nil {
			t.Fatalf("limit %d accepted", limit)
		}
	}
	if _, err := NewAdmissionHandler(nil, 1); err == nil {
		t.Fatal("nil handler accepted")
	}
}
