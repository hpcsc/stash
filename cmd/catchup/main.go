package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/hpcsc/stash/internal/common/config"
	"github.com/hpcsc/stash/internal/domain/consumer"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DB config.DB
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load[Config]()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DB.ConnectionString())
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	ctx := runShutdownSignalHandler(context.Background(), logger)
	failed := run(ctx, consumers(), logger)

	pool.Close()
	if failed {
		os.Exit(1)
	}
}

func consumers() []*consumer.Consumer {
	return []*consumer.Consumer{}
}

func run(parentCtx context.Context, consumers []*consumer.Consumer, logger *slog.Logger) bool {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	var failed atomic.Bool
	var wg sync.WaitGroup
	for _, c := range consumers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if err := recover(); err != nil {
					logger.Error(fmt.Sprintf("consumer %s received unhandled error: %v", c.Name(), err))
					failed.Store(true)
					cancel()
				}
			}()

			if err := c.Start(ctx); err != nil {
				logger.Error(fmt.Sprintf("consumer %s failed: %v", c.Name(), err))
				failed.Store(true)
				cancel()
				return
			}
			logger.Info(fmt.Sprintf("consumer %s stopped", c.Name()))
		}()
	}

	wg.Wait()
	return failed.Load()
}

func runShutdownSignalHandler(parentCtx context.Context, logger *slog.Logger) context.Context {
	ctx, cancel := context.WithCancel(parentCtx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		s := <-sig
		logger.Info(fmt.Sprintf("received %v signal", s))
		cancel()
	}()

	return ctx
}
