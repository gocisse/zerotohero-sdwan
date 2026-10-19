package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"velocloud-academy/backend/internal/config"
	"velocloud-academy/backend/internal/content"
	"velocloud-academy/backend/internal/db"
	apihttp "velocloud-academy/backend/internal/http"
	"velocloud-academy/backend/internal/store"
	"velocloud-academy/backend/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(cfg.LogLevel)}))
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool, migrations.FS, log); err != nil {
		return err
	}

	st := store.New(pool)
	courseContent := content.NewProvider(cfg.ContentDir, log)
	if _, err := courseContent.Current(); err != nil {
		content.LogErrors(log, err)
		return fmt.Errorf("course content in %s is invalid", cfg.ContentDir)
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           apihttp.NewRouter(apihttp.Deps{Health: st, Content: courseContent, Progress: st, Log: log}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("server listening", "addr", cfg.HTTPAddr, "content_dir", cfg.ContentDir)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
	return nil
}

func parseLevel(s string) slog.Level {
	var l slog.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return slog.LevelInfo
	}
	return l
}
