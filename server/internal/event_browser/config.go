package event_browser

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const (
	DefaultAddr  = "127.0.0.1:7777"
	TokenEnvName = "STASH_DEBUG_TOKEN"
)

type Config struct {
	Addr    string
	Backend string
	Token   string
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Backend) == "" {
		return errors.New("the backend URL is required, for example --url https://stash.example.com")
	}

	parsed, err := url.Parse(c.Backend)
	if err != nil {
		return fmt.Errorf("the backend URL is not valid: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("the backend URL must start with http:// or https://")
	}
	if parsed.Host == "" {
		return errors.New("the backend URL must include a host")
	}

	if c.Token == "" {
		return fmt.Errorf("the debug token is required: pass --token or set %s", TokenEnvName)
	}

	if c.Addr == "" {
		return errors.New("the listen address must not be empty")
	}
	return nil
}
