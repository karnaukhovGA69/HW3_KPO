package main

import (
	"HW_KPO3/internal/analysis"
	"HW_KPO3/internal/config"
	"HW_KPO3/internal/storage"
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

const (
	envLocal = "local"
	envProd  = "production"
	envTest  = "test"
	envDev   = "development"
)

func main() {
	config := config.MustLoad()
	logger := setupLogger(config.Env)
	slog.SetDefault(logger)

	slog.Info("config loaded",
		"env", config.Env,
		"storage_path", config.StoragePath,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	dbAnalysis, err := storage.NewStorage(ctx, config.AnalysisDB.DSN)
	if err != nil {
		slog.Error("failed to connect to analysis db", "error", err)
		os.Exit(1)
	}
	defer dbAnalysis.Close()
	slog.Info("connected to analysis db")
	repo := analysis.NewRepository(dbAnalysis)
	handler := analysis.NewHandler(repo)
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)

	r.Route("/reports", func(r chi.Router) {
		r.Post("/", handler.CreateReport)
		r.Get("/{id}", handler.GetReport)
	})
	address := "localhost:8069"
	server := &http.Server{
		Addr:    address,
		Handler: r,
	}
	go func() {
		slog.Info("starting analysis http server", "address", address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()
	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	slog.Info("shutting down analysis http server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "err", err)
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	case envTest, envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return log
}
