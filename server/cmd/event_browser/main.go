package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hpcsc/stash/internal/event_browser"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg, err := parseFlags(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		logger.Error(err.Error())
		os.Exit(2)
	}

	server, err := event_browser.New(cfg, logger)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		s := <-sig
		logger.Info(fmt.Sprintf("received %v signal", s))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Warn("failed to shut down", "error", err)
		}
	}()

	if err := server.Start(); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	logger.Info("exit")
}

func parseFlags(args []string) (event_browser.Config, error) {
	cfg := event_browser.Config{}

	flags := flag.NewFlagSet("eventbrowser", flag.ContinueOnError)
	flags.StringVar(&cfg.Backend, "url", "", "base URL of the Stash API, for example https://stash.example.com")
	flags.StringVar(&cfg.Token, "token", "", "session token to use without signing in; defaults to $"+event_browser.TokenEnvName)
	flags.StringVar(&cfg.Addr, "addr", event_browser.DefaultAddr, "address for the local UI")

	flags.Usage = func() {
		out := flags.Output()
		fmt.Fprintf(out, "Open the Stash event browser in a local web server.\n\n")
		fmt.Fprintf(out, "Usage:\n  eventbrowser --url <backend> [--token <token>] [--addr <address>]\n\n")
		fmt.Fprintf(out, "Without --token, the page asks you to sign in with Google. The session token stays in this process, so the browser never sees it.\n\n")
		fmt.Fprintf(out, "Flags:\n")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return cfg, err
	}

	if flags.NArg() > 0 {
		return cfg, fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}

	if cfg.Token == "" {
		cfg.Token = os.Getenv(event_browser.TokenEnvName)
	}

	return cfg, cfg.Validate()
}
