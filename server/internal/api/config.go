package api

import (
	"github.com/hpcsc/stash/internal/common/config"
	"github.com/hpcsc/stash/internal/storage"
)

type Config struct {
	Port           string   `env:"PORT" envDefault:"3333"`
	SessionSecret  string   `env:"SESSION_SECRET"`
	GoogleClientID string   `env:"GOOGLE_CLIENT_ID"`
	AllowedOrigins []string `env:"ALLOWED_ORIGINS" envSeparator:","`
	WebDir         string   `env:"WEB_DIR" envDefault:"/app/web"`
	Storage        storage.Config
	DB             config.DB
}
