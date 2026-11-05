package event_browser

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hpcsc/stash/internal/api/response"
)

type sessionHandler struct {
	backend  *url.URL
	session  Session
	client   *http.Client
	logger   *slog.Logger
	mu       sync.Mutex
	clientID string
}

func newSessionHandler(backend *url.URL, session Session, logger *slog.Logger) *sessionHandler {
	return &sessionHandler{
		backend: backend,
		session: session,
		client:  &http.Client{Timeout: 15 * time.Second},
		logger:  logger,
	}
}

func (h *sessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.read(w, r)
	case http.MethodPost:
		h.signIn(w, r)
	case http.MethodDelete:
		h.signOut(w, r)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		writeJSON(w, http.StatusMethodNotAllowed, response.Fail("method not allowed"))
	}
}

func (h *sessionHandler) read(w http.ResponseWriter, r *http.Request) {
	clientID, err := h.googleClientID(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, response.Fail("the backend did not answer: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, response.SucceedWithData(map[string]any{
		"signedIn":       h.session.Token() != "",
		"googleClientId": clientID,
		"canSignOut":     !h.session.Fixed(),
	}))
}

func (h *sessionHandler) signIn(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDToken string `json:"idToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.IDToken == "" {
		writeJSON(w, http.StatusBadRequest, response.Fail("an ID token is required"))
		return
	}

	payload, err := json.Marshal(map[string]string{"provider": "google", "idToken": body.IDToken})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response.Fail("failed to build the sign-in request"))
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, h.backendURL("/api/auth/signin"), bytes.NewReader(payload))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response.Fail("failed to build the sign-in request"))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Warn("failed to reach the backend to sign in", "error", err)
		writeJSON(w, http.StatusBadGateway, response.Fail("the backend did not answer: "+err.Error()))
		return
	}
	defer resp.Body.Close()

	var result response.Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		writeJSON(w, http.StatusBadGateway, response.Fail("the backend returned an unexpected response"))
		return
	}
	if !result.Successful {
		h.logger.Warn("backend rejected sign-in", "status", resp.StatusCode, "messages", result.Messages)
		writeJSON(w, resp.StatusCode, result)
		return
	}

	resultData, _ := result.Data.(map[string]any)
	token, _ := resultData["token"].(string)
	if token == "" {
		writeJSON(w, http.StatusBadGateway, response.Fail("the backend did not return a session token"))
		return
	}
	if err := h.session.Set(token); err != nil {
		h.logger.Warn("failed to save the session", "error", err)
		writeJSON(w, http.StatusInternalServerError, response.Fail("failed to save the session: "+err.Error()))
		return
	}

	clientID, err := h.googleClientID(r.Context())
	if err != nil {
		h.logger.Warn("failed to read the Google client id", "error", err)
	}

	writeJSON(w, http.StatusOK, response.SucceedWithData(map[string]any{
		"signedIn":       true,
		"googleClientId": clientID,
		"canSignOut":     !h.session.Fixed(),
	}))
}

func (h *sessionHandler) signOut(w http.ResponseWriter, r *http.Request) {
	if !h.session.Fixed() {
		h.signOutBackend(r.Context())
	}
	if err := h.session.Clear(); err != nil {
		h.logger.Warn("failed to clear the session", "error", err)
		writeJSON(w, http.StatusInternalServerError, response.Fail("failed to clear the session: "+err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, response.Succeed())
}

func (h *sessionHandler) signOutBackend(ctx context.Context) {
	token := h.session.Token()
	if token == "" {
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.backendURL("/api/auth/signout"), nil)
	if err != nil {
		h.logger.Warn("failed to build the sign-out request", "error", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Warn("failed to reach the backend to sign out", "error", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
}

func (h *sessionHandler) googleClientID(ctx context.Context) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clientID != "" {
		return h.clientID, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.backendURL("/api/auth/config"), nil)
	if err != nil {
		return "", err
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result response.Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("unexpected response: %w", err)
	}
	if !result.Successful {
		return "", errors.New(strings.Join(result.Messages, "; "))
	}

	clientData, _ := result.Data.(map[string]any)
	clientID, _ := clientData["googleClientId"].(string)
	if clientID == "" {
		return "", errors.New("no Google client id in the response")
	}

	h.clientID = clientID
	return clientID, nil
}

func (h *sessionHandler) backendURL(path string) string {
	target := *h.backend
	target.Path, target.RawPath = joinPath(h.backend, &url.URL{Path: path})
	return target.String()
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
