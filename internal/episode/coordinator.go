package episode

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/limits"
)

const maxCoordinatorBody = limits.MaxBundleCompressed*2 + limits.MaxReportBytes

type Coordinator struct {
	journal *Journal
	token   string
	mux     *http.ServeMux
}

func NewCoordinator(journal *Journal, token string) (*Coordinator, error) {
	if journal == nil || journal.db == nil {
		return nil, errors.New("Episode Journal is required")
	}
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return nil, errors.New("coordinator bearer token is required and must not contain whitespace")
	}
	c := &Coordinator{journal: journal, token: token, mux: http.NewServeMux()}
	c.mux.HandleFunc("POST /v1/episodes", c.submit)
	c.mux.HandleFunc("GET /v1/episodes/{id}", c.inspect)
	c.mux.HandleFunc("POST /v1/claims", c.claim)
	c.mux.HandleFunc("POST /v1/episodes/{id}/heartbeat", c.heartbeat)
	c.mux.HandleFunc("POST /v1/episodes/{id}/complete", c.complete)
	c.mux.HandleFunc("POST /v1/episodes/{id}/fail", c.fail)
	c.mux.HandleFunc("POST /v1/episodes/{id}/cancel", c.cancel)
	c.mux.HandleFunc("POST /v1/recover", c.recover)
	return c, nil
}

func (c *Coordinator) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	scheme, provided, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(provided) != provided || len(provided) != len(c.token) || subtle.ConstantTimeCompare([]byte(provided), []byte(c.token)) != 1 {
		coordinatorError(w, http.StatusUnauthorized, "AUTH_DENIED", "valid coordinator bearer token required")
		return
	}
	c.mux.ServeHTTP(w, r)
}

func (c *Coordinator) submit(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Bundle          []byte        `json:"bundle"`
		EpisodeID       string        `json:"episodeId"`
		ScenarioPath    string        `json:"scenarioPath"`
		RuntimeVersion  string        `json:"runtimeVersion"`
		RuntimeRevision string        `json:"runtimeRevision"`
		EffectProfile   EffectProfile `json:"effectProfile"`
		MaxAttempts     int           `json:"maxAttempts"`
	}
	if !decodeCoordinatorJSON(w, r, &request) {
		return
	}
	task, created, err := c.journal.Submit(r.Context(), request.Bundle, request.EpisodeID, request.ScenarioPath, request.RuntimeVersion, request.RuntimeRevision, request.EffectProfile, request.MaxAttempts)
	if err != nil {
		writeCoordinatorError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	coordinatorJSON(w, status, task)
}

func (c *Coordinator) inspect(w http.ResponseWriter, r *http.Request) {
	task, err := c.journal.GetTask(r.Context(), r.PathValue("id"))
	if err != nil {
		writeCoordinatorError(w, err)
		return
	}
	coordinatorJSON(w, http.StatusOK, task)
}

func (c *Coordinator) claim(w http.ResponseWriter, r *http.Request) {
	var request struct {
		WorkerID      string        `json:"workerId"`
		LeaseSeconds  int           `json:"leaseSeconds"`
		EffectProfile EffectProfile `json:"effectProfile,omitempty"`
	}
	if !decodeCoordinatorJSON(w, r, &request) {
		return
	}
	claim, err := c.journal.ClaimProfile(r.Context(), request.WorkerID, time.Duration(request.LeaseSeconds)*time.Second, request.EffectProfile)
	if err != nil {
		writeCoordinatorError(w, err)
		return
	}
	coordinatorJSON(w, http.StatusOK, claim)
}

func (c *Coordinator) heartbeat(w http.ResponseWriter, r *http.Request) {
	var request struct {
		AttemptID    string `json:"attemptId"`
		FencingToken int64  `json:"fencingToken"`
		LeaseSeconds int    `json:"leaseSeconds"`
	}
	if !decodeCoordinatorJSON(w, r, &request) {
		return
	}
	result, err := c.journal.Heartbeat(r.Context(), r.PathValue("id"), request.AttemptID, request.FencingToken, time.Duration(request.LeaseSeconds)*time.Second)
	if err != nil {
		writeCoordinatorError(w, err)
		return
	}
	coordinatorJSON(w, http.StatusOK, result)
}

func (c *Coordinator) complete(w http.ResponseWriter, r *http.Request) {
	var request struct {
		AttemptID    string    `json:"attemptId"`
		FencingToken int64     `json:"fencingToken"`
		Evidence     *Envelope `json:"evidence"`
	}
	if !decodeCoordinatorJSON(w, r, &request) {
		return
	}
	evidence, reused, err := c.journal.CompleteClaim(r.Context(), r.PathValue("id"), request.AttemptID, request.FencingToken, request.Evidence)
	if err != nil {
		writeCoordinatorError(w, err)
		return
	}
	coordinatorJSON(w, http.StatusOK, map[string]any{"evidence": evidence, "reused": reused})
}

func (c *Coordinator) fail(w http.ResponseWriter, r *http.Request) {
	var request struct {
		AttemptID    string      `json:"attemptId"`
		FencingToken int64       `json:"fencingToken"`
		CommitState  CommitState `json:"commitState"`
		ErrorClass   string      `json:"errorClass"`
	}
	if !decodeCoordinatorJSON(w, r, &request) {
		return
	}
	task, err := c.journal.FailClaim(r.Context(), r.PathValue("id"), request.AttemptID, request.FencingToken, request.CommitState, request.ErrorClass)
	if err != nil {
		writeCoordinatorError(w, err)
		return
	}
	coordinatorJSON(w, http.StatusOK, task)
}

func (c *Coordinator) cancel(w http.ResponseWriter, r *http.Request) {
	if !decodeCoordinatorJSON(w, r, &struct{}{}) {
		return
	}
	task, err := c.journal.CancelTask(r.Context(), r.PathValue("id"))
	if err != nil {
		writeCoordinatorError(w, err)
		return
	}
	coordinatorJSON(w, http.StatusOK, task)
}

func (c *Coordinator) recover(w http.ResponseWriter, r *http.Request) {
	if !decodeCoordinatorJSON(w, r, &struct{}{}) {
		return
	}
	count, err := c.journal.RecoverExpired(r.Context())
	if err != nil {
		writeCoordinatorError(w, err)
		return
	}
	coordinatorJSON(w, http.StatusOK, map[string]any{"recovered": count})
}

func decodeCoordinatorJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if r.Header.Get("Content-Type") != "application/json" {
		coordinatorError(w, http.StatusUnsupportedMediaType, "INVALID_INPUT", "content-type must be application/json")
		return false
	}
	body := http.MaxBytesReader(w, r.Body, maxCoordinatorBody)
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		coordinatorError(w, http.StatusBadRequest, "INVALID_INPUT", "request body is invalid")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		coordinatorError(w, http.StatusBadRequest, "INVALID_INPUT", "request must contain one JSON value")
		return false
	}
	return true
}

func writeCoordinatorError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrEpisodeNotFound):
		coordinatorError(w, http.StatusNotFound, "NOT_FOUND", "Episode task not found")
	case errors.Is(err, ErrNoClaimableEpisode):
		coordinatorError(w, http.StatusNotFound, "NO_TASK", "no claimable Episode")
	case errors.Is(err, ErrStaleAttempt):
		coordinatorError(w, http.StatusConflict, "STALE_ATTEMPT", "attempt lease or fencing token is stale")
	case errors.Is(err, ErrCancelRequested):
		coordinatorError(w, http.StatusConflict, "CANCEL_REQUESTED", "Episode cancellation is already requested")
	case errors.Is(err, ErrEpisodeConflict):
		coordinatorError(w, http.StatusConflict, "EPISODE_CONFLICT", "Episode identity or terminal state conflicts")
	case strings.Contains(err.Error(), "RESOURCE_LIMIT"):
		coordinatorError(w, http.StatusRequestEntityTooLarge, "RESOURCE_LIMIT", "coordinator resource limit exceeded")
	default:
		coordinatorError(w, http.StatusBadRequest, "COORDINATOR_ERROR", "coordinator request rejected")
	}
}

func coordinatorError(w http.ResponseWriter, status int, code, message string) {
	coordinatorJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func coordinatorJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

type CoordinatorClient struct {
	baseURL string
	token   string
	client  *http.Client
}

type CoordinatorRequestError struct {
	StatusCode int
	Code       string
}

func (e *CoordinatorRequestError) Error() string {
	return fmt.Sprintf("coordinator request failed (%d %s)", e.StatusCode, e.Code)
}

func NewCoordinatorClient(baseURL, token string, client *http.Client) (*CoordinatorClient, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("coordinator URL must be an absolute http(s) URL without query or fragment")
	}
	if parsed.User != nil {
		return nil, errors.New("coordinator URL must not contain credentials")
	}
	if parsed.Scheme == "http" {
		ip := net.ParseIP(parsed.Hostname())
		if ip == nil || !ip.IsLoopback() {
			return nil, errors.New("plaintext coordinator URL must use a loopback IP")
		}
	}
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return nil, errors.New("coordinator bearer token is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &CoordinatorClient{baseURL: strings.TrimRight(baseURL, "/"), token: token, client: client}, nil
}

func (c *CoordinatorClient) Claim(ctx context.Context, workerID string, leaseSeconds int, profile EffectProfile) (*Claim, error) {
	var result Claim
	err := c.call(ctx, http.MethodPost, "/v1/claims", map[string]any{"workerId": workerID, "leaseSeconds": leaseSeconds, "effectProfile": profile}, &result)
	return &result, err
}

func (c *CoordinatorClient) Heartbeat(ctx context.Context, claim *Claim, leaseSeconds int) (*HeartbeatResult, error) {
	var result HeartbeatResult
	err := c.call(ctx, http.MethodPost, "/v1/episodes/"+url.PathEscape(claim.EpisodeID)+"/heartbeat", map[string]any{"attemptId": claim.AttemptID, "fencingToken": claim.FencingToken, "leaseSeconds": leaseSeconds}, &result)
	return &result, err
}

func (c *CoordinatorClient) Complete(ctx context.Context, claim *Claim, evidence *Envelope) error {
	return c.call(ctx, http.MethodPost, "/v1/episodes/"+url.PathEscape(claim.EpisodeID)+"/complete", map[string]any{"attemptId": claim.AttemptID, "fencingToken": claim.FencingToken, "evidence": evidence}, &map[string]any{})
}

func (c *CoordinatorClient) Fail(ctx context.Context, claim *Claim, state CommitState, class string) error {
	return c.call(ctx, http.MethodPost, "/v1/episodes/"+url.PathEscape(claim.EpisodeID)+"/fail", map[string]any{"attemptId": claim.AttemptID, "fencingToken": claim.FencingToken, "commitState": state, "errorClass": class}, &TaskRecord{})
}

func (c *CoordinatorClient) call(ctx context.Context, method, path string, body, target any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Content-Type", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, limits.MaxReportBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return err
	}
	if len(data) > limits.MaxReportBytes {
		return errors.New("coordinator response exceeds limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var envelope struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		_ = json.Unmarshal(data, &envelope)
		return &CoordinatorRequestError{StatusCode: response.StatusCode, Code: envelope.Error.Code}
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode coordinator response: %w", err)
	}
	return nil
}
