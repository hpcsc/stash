package event_browser

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Session interface {
	Token() string
	Set(token string) error
	Clear() error
	Fixed() bool
}

type storedSession struct {
	mu    sync.RWMutex
	token string
	path  string
	fixed bool
}

func NewSession(cfg Config) (Session, error) {
	if cfg.Token != "" {
		return &storedSession{token: cfg.Token, fixed: true}, nil
	}

	path := cfg.SessionFile
	if path == "" {
		var err error
		path, err = defaultSessionFile(cfg.Backend)
		if err != nil {
			return nil, err
		}
	}

	session := &storedSession{path: path}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return session, nil
		}
		return nil, fmt.Errorf("failed to read the session file: %w", err)
	}

	token := strings.TrimSpace(string(raw))
	if token == "" || hasExpired(token, time.Now()) {
		_ = os.Remove(path)
		return session, nil
	}

	session.token = token
	return session, nil
}

func (s *storedSession) Token() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token
}

func (s *storedSession) Set(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fixed {
		return errors.New("the session token came from the command line and cannot be replaced")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("failed to create the session directory: %w", err)
	}
	if err := writePrivateFile(s.path, token); err != nil {
		return err
	}
	s.token = token
	return nil
}

func (s *storedSession) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fixed {
		return nil
	}
	s.token = ""
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to remove the session file: %w", err)
	}
	return nil
}

func (s *storedSession) Fixed() bool {
	return s.fixed
}

func defaultSessionFile(backend string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to find the user config directory: %w", err)
	}
	return filepath.Join(dir, "stash", sessionFileName(backend)), nil
}

func sessionFileName(backend string) string {
	sum := sha256.Sum256([]byte(strings.TrimRight(backend, "/")))
	return fmt.Sprintf("event-browser-session-%x", sum[:6])
}

func writePrivateFile(path string, contents string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open the session file: %w", err)
	}
	defer file.Close()

	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("failed to restrict the session file: %w", err)
	}
	if _, err := file.WriteString(contents); err != nil {
		return fmt.Errorf("failed to write the session file: %w", err)
	}
	return nil
}

func hasExpired(token string, now time.Time) bool {
	expiry, ok := tokenExpiry(token)
	if !ok {
		return true
	}
	return !now.Before(expiry)
}

func tokenExpiry(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, false
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}

	var claims struct {
		ExpiresAt int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.ExpiresAt == 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.ExpiresAt, 0), true
}
