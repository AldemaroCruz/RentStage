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

	"github.com/rentstage/rentstage/apps/api/internal/config"
	"github.com/rentstage/rentstage/apps/api/internal/database"
	"github.com/rentstage/rentstage/apps/api/internal/httpapi"
)

var version = "dev"

const localHealthcheckURL = "http://127.0.0.1:8080/healthz"

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "healthcheck" {
		healthcheck()
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool, cfg.SeedDemoData); err != nil {
		logger.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	server, err := httpapi.New(ctx, cfg, pool, logger)
	if err != nil {
		logger.Error("http server initialization failed", "error", err)
		os.Exit(1)
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("rentstage api started", "addr", cfg.HTTPAddr, "environment", cfg.AppEnv, "version", version)
		serverErrors <- server.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case signalValue := <-signals:
		logger.Info("shutdown signal received", "signal", signalValue.String())
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server stopped unexpectedly", "error", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}

func healthcheck() {
	client := http.Client{
		Timeout: 2 * time.Second,
	}

	response, err := client.Get(localHealthcheckURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		fmt.Fprintln(os.Stderr, response.Status)
		os.Exit(1)
	}
}
