package event_browser

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

type Server struct {
	cfg        Config
	httpServer *http.Server
	logger     *slog.Logger
}

func New(cfg Config, logger *slog.Logger) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	ui, err := embeddedUI()
	if err != nil {
		return nil, err
	}

	session, err := NewSession(cfg)
	if err != nil {
		return nil, err
	}

	handler, err := NewHandler(cfg.Backend, session, ui, logger)
	if err != nil {
		return nil, err
	}

	return &Server{
		cfg: cfg,
		httpServer: &http.Server{
			Addr:              cfg.Addr,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		},
		logger: logger,
	}, nil
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.cfg.Addr, err)
	}

	s.logger.Info("event browser listening",
		"addr", "http://"+listener.Addr().String(),
		"backend", s.cfg.Backend,
	)

	if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("the event browser stopped: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
