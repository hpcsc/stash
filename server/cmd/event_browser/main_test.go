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

		t.Run("accepts no token so the page can sign in", func(t *testing.T) {
			t.Setenv(event_browser.TokenEnvName, "")

			cfg, err := parseFlags([]string{"--url", "https://stash.example.com"})

			require.NoError(t, err)
			require.Empty(t, cfg.Token)
		})
	})
}
