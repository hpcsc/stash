//go:build unit

package main

import (
	"testing"

	"github.com/hpcsc/stash/internal/event_browser"
	"github.com/stretchr/testify/require"
)

func TestParseFlags(t *testing.T) {
	t.Run("token", func(t *testing.T) {
		t.Run("reads the session token from the environment", func(t *testing.T) {
			t.Setenv(event_browser.TokenEnvName, "from-env")

			cfg, err := parseFlags([]string{"--url", "https://stash.example.com"})

			require.NoError(t, err)
			require.Equal(t, "from-env", cfg.Token)
		})

		t.Run("prefers the flag over the environment", func(t *testing.T) {
			t.Setenv(event_browser.TokenEnvName, "from-env")

			cfg, err := parseFlags([]string{"--url", "https://stash.example.com", "--token", "from-flag"})

			require.NoError(t, err)
			require.Equal(t, "from-flag", cfg.Token)
		})

		t.Run("fails when neither the flag nor the environment has a token", func(t *testing.T) {
			t.Setenv(event_browser.TokenEnvName, "")

			_, err := parseFlags([]string{"--url", "https://stash.example.com"})

			require.ErrorContains(t, err, "session token is required")
		})
	})
}
