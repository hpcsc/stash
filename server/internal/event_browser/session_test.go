//go:build unit

package event_browser_test

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hpcsc/stash/internal/event_browser"
	"github.com/stretchr/testify/require"
)

func TestSession(t *testing.T) {
	t.Run("load", func(t *testing.T) {
		t.Run("keeps a stored token that has not expired", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "session")
			token := tokenExpiringAt(time.Now().Add(time.Hour))
			require.NoError(t, os.WriteFile(path, []byte(token), 0o600))

			session, err := event_browser.NewSession(event_browser.Config{SessionFile: path})

			require.NoError(t, err)
			require.Equal(t, token, session.Token())
			require.False(t, session.Fixed())
		})

		t.Run("discards an expired token", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "session")
			require.NoError(t, os.WriteFile(path, []byte(tokenExpiringAt(time.Now().Add(-time.Minute))), 0o600))

			session, err := event_browser.NewSession(event_browser.Config{SessionFile: path})

			require.NoError(t, err)
			require.Empty(t, session.Token())
			require.NoFileExists(t, path)
		})

		t.Run("discards a malformed token", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "session")
			require.NoError(t, os.WriteFile(path, []byte("not-a-token"), 0o600))

			session, err := event_browser.NewSession(event_browser.Config{SessionFile: path})

			require.NoError(t, err)
			require.Empty(t, session.Token())
			require.NoFileExists(t, path)
		})

		t.Run("starts signed out without a file", func(t *testing.T) {
			session, err := event_browser.NewSession(event_browser.Config{SessionFile: filepath.Join(t.TempDir(), "missing")})

			require.NoError(t, err)
			require.Empty(t, session.Token())
		})

		t.Run("uses a token from the command line and ignores the file", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "session")
			require.NoError(t, os.WriteFile(path, []byte(tokenExpiringAt(time.Now().Add(time.Hour))), 0o600))

			session, err := event_browser.NewSession(event_browser.Config{Token: "from-command-line", SessionFile: path})

			require.NoError(t, err)
			require.Equal(t, "from-command-line", session.Token())
			require.True(t, session.Fixed())
		})
	})

	t.Run("save", func(t *testing.T) {
		t.Run("writes the token with owner-only permissions", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "nested", "session")
			session, err := event_browser.NewSession(event_browser.Config{SessionFile: path})
			require.NoError(t, err)

			require.NoError(t, session.Set("new-token"))

			contents, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, "new-token", string(contents))
			info, err := os.Stat(path)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
			require.Equal(t, "new-token", session.Token())
		})

		t.Run("refuses to replace a token from the command line", func(t *testing.T) {
			session, err := event_browser.NewSession(event_browser.Config{Token: "from-command-line"})
			require.NoError(t, err)

			require.Error(t, session.Set("new-token"))
			require.Equal(t, "from-command-line", session.Token())
		})
	})

	t.Run("clear", func(t *testing.T) {
		t.Run("removes the stored token", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "session")
			session, err := event_browser.NewSession(event_browser.Config{SessionFile: path})
			require.NoError(t, err)
			require.NoError(t, session.Set("new-token"))

			require.NoError(t, session.Clear())

			require.Empty(t, session.Token())
			require.NoFileExists(t, path)
		})

		t.Run("keeps a token from the command line", func(t *testing.T) {
			session, err := event_browser.NewSession(event_browser.Config{Token: "from-command-line"})
			require.NoError(t, err)

			require.NoError(t, session.Clear())

			require.Equal(t, "from-command-line", session.Token())
		})
	})

	t.Run("per backend", func(t *testing.T) {
		t.Run("keeps a separate token for each backend", func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("XDG_CONFIG_HOME", "")

			firstToken := tokenExpiringAt(time.Now().Add(time.Hour))
			secondToken := tokenExpiringAt(time.Now().Add(2 * time.Hour))

			first, err := event_browser.NewSession(event_browser.Config{Backend: "https://one.example.com/"})
			require.NoError(t, err)
			require.NoError(t, first.Set(firstToken))

			second, err := event_browser.NewSession(event_browser.Config{Backend: "https://two.example.com"})
			require.NoError(t, err)
			require.NoError(t, second.Set(secondToken))

			reloaded, err := event_browser.NewSession(event_browser.Config{Backend: "https://one.example.com"})
			require.NoError(t, err)
			require.Equal(t, firstToken, reloaded.Token())
		})
	})
}

func tokenExpiringAt(expiry time.Time) string {
	payload, _ := json.Marshal(map[string]int64{"exp": expiry.Unix()})
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}
