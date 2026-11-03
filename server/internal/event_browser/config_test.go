//go:build unit

package event_browser_test

import (
	"testing"

	"github.com/hpcsc/stash/internal/event_browser"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	t.Run("validate", func(t *testing.T) {
		t.Run("accepts a complete configuration", func(t *testing.T) {
			err := event_browser.Config{Backend: "https://stash.example.com", Token: "secret", Addr: "127.0.0.1:7777"}.Validate()

			require.NoError(t, err)
		})

		t.Run("rejects a missing backend URL", func(t *testing.T) {
			err := event_browser.Config{Token: "secret", Addr: "127.0.0.1:7777"}.Validate()

			require.ErrorContains(t, err, "backend URL is required")
		})

		t.Run("rejects a backend URL without a scheme", func(t *testing.T) {
			err := event_browser.Config{Backend: "stash.example.com", Token: "secret", Addr: "127.0.0.1:7777"}.Validate()

			require.ErrorContains(t, err, "http://")
		})

		t.Run("rejects a missing token", func(t *testing.T) {
			err := event_browser.Config{Backend: "https://stash.example.com", Addr: "127.0.0.1:7777"}.Validate()

			require.ErrorContains(t, err, "session token is required")
		})
	})
}
