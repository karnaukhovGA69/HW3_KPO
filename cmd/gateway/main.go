package main

import (
	"HW_KPO3/internal/config"
	"HW_KPO3/internal/gateway"
	"HW_KPO3/internal/logger"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.MustLoad()

	log := logger.SetupLogger(cfg.Env)
	slog.SetDefault(log)

	slog.Info("config loaded",
		"env", cfg.Env,
		"gateway_addr", cfg.Gateway.Address,
		"storage_url", cfg.Gateway.StorageBaseURL,
		"analysis_url", cfg.Gateway.AnalysisBaseURL,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	gw := gateway.NewGateway(cfg.Gateway.StorageBaseURL, cfg.Gateway.AnalysisBaseURL)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// high-level API
	r.Post("/works", gw.CreateWorkAndReport)
	r.Get("/works/{id}", gw.GetWorkProxy)

	srv := &http.Server{
		Addr:    cfg.Gateway.Address,
		Handler: r,
	}

	go func() {
		slog.Info("starting gateway http server", "addr", cfg.Gateway.Address)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("gateway server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down gateway server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("gateway shutdown error", "err", err)
	}
}
