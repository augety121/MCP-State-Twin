package server

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/store"
)

type ControlPlane struct {
	store *store.Store
	token string
	mux   *http.ServeMux
}

func NewControlPlane(stateStore *store.Store, token string) *ControlPlane {
	c := &ControlPlane{store: stateStore, token: token, mux: http.NewServeMux()}
	c.mux.HandleFunc("GET /v1/branches/{branch}", c.getBranch)
	c.mux.HandleFunc("POST /v1/snapshots", c.createSnapshot)
	c.mux.HandleFunc("POST /v1/forks", c.fork)
	c.mux.HandleFunc("POST /v1/resets", c.reset)
	c.mux.HandleFunc("POST /v1/clock/advance", c.advanceClock)
	c.mux.HandleFunc("GET /v1/diff", c.diff)
	return c
}

func (c *ControlPlane) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if c.token == "" {
		writeError(w, http.StatusServiceUnavailable, "CONTROL_AUTH_REQUIRED", "control plane requires a non-empty token")
		return
	}
	scheme, provided, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(provided) != provided || strings.ContainsAny(provided, " \t\r\n") {
		writeError(w, http.StatusUnauthorized, "AUTH_DENIED", "valid control-plane bearer token required")
		return
	}
	if len(provided) != len(c.token) || subtle.ConstantTimeCompare([]byte(provided), []byte(c.token)) != 1 {
		writeError(w, http.StatusUnauthorized, "AUTH_DENIED", "valid control-plane bearer token required")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	c.mux.ServeHTTP(w, r)
}

func (c *ControlPlane) getBranch(w http.ResponseWriter, r *http.Request) {
	branch, err := c.store.Branch(r.Context(), r.PathValue("branch"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, branch)
}

func (c *ControlPlane) createSnapshot(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Name   string `json:"name"`
		Branch string `json:"branch"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	snapshot, err := c.store.CreateSnapshot(r.Context(), request.Name, request.Branch)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, snapshot)
}

func (c *ControlPlane) fork(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Snapshot string `json:"snapshot"`
		Branch   string `json:"branch"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := c.store.Fork(r.Context(), request.Snapshot, request.Branch); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"branch": request.Branch, "snapshot": request.Snapshot})
}

func (c *ControlPlane) reset(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Snapshot string `json:"snapshot"`
		Branch   string `json:"branch"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if err := c.store.Reset(r.Context(), request.Snapshot, request.Branch); err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"branch": request.Branch, "resetTo": request.Snapshot})
}

func (c *ControlPlane) advanceClock(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Branch              string `json:"branch"`
		By                  string `json:"by"`
		To                  string `json:"to"`
		ExpectedHeadVersion *int64 `json:"expectedHeadVersion"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	if (request.By == "") == (request.To == "") {
		writeError(w, http.StatusBadRequest, "CLOCK_INVALID", "exactly one of by or to is required")
		return
	}
	branch, err := c.store.Branch(r.Context(), request.Branch)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	target := branch.Clock
	if request.By != "" {
		duration, parseErr := time.ParseDuration(request.By)
		if parseErr != nil || duration <= 0 {
			writeError(w, http.StatusBadRequest, "CLOCK_INVALID", "by must be a positive Go duration")
			return
		}
		target = target.Add(duration)
	} else {
		target, err = time.Parse(time.RFC3339Nano, request.To)
		if err != nil {
			writeError(w, http.StatusBadRequest, "CLOCK_INVALID", "to must be RFC3339")
			return
		}
	}
	if err := c.store.AdvanceClock(r.Context(), request.Branch, target, request.ExpectedHeadVersion); err != nil {
		writeStoreError(w, err)
		return
	}
	updated, err := c.store.Branch(r.Context(), request.Branch)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"branch": request.Branch, "clock": updated.Clock, "headVersion": updated.HeadVersion})
}

func (c *ControlPlane) diff(w http.ResponseWriter, r *http.Request) {
	changes, err := c.store.DiffBranches(r.Context(), r.URL.Query().Get("before"), r.URL.Query().Get("after"))
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"changes": changes})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	body := http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "INVALID_INPUT", "request must contain one JSON value")
		return false
	}
	return true
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrBranchNotFound), errors.Is(err, store.ErrSnapshotNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, store.ErrBranchConflict):
		writeError(w, http.StatusConflict, "BRANCH_CONFLICT", err.Error())
	case errors.Is(err, store.ErrClockRegression), errors.Is(err, store.ErrClockLimit):
		writeError(w, http.StatusBadRequest, "CLOCK_INVALID", err.Error())
	default:
		writeError(w, http.StatusBadRequest, "CONTROL_ERROR", err.Error())
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
