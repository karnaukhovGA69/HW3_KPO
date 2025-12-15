package main

import (
	"HW_KPO3/internal/analysis"
	"HW_KPO3/internal/config"
	"HW_KPO3/internal/logger"
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
	"github.com/go-chi/cors"
)

func main() {
	cfg := config.MustLoad()
	log := logger.SetupLogger(cfg.Env)
	slog.SetDefault(log)

	slog.Info("config loaded",
		"env", cfg.Env,
		"addr", cfg.AnalysisServer.Address,
		"storage_path", cfg.StoragePath,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	dbAnalysis, err := storage.NewStorage(ctx, cfg.AnalysisDB.DSN)
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
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Route("/reports", func(r chi.Router) {
		r.Post("/", handler.CreateReport)
		r.Get("/{id}", handler.GetReport)
		r.Get("/work/{work_id}", handler.GetReportByWorkID)
	})

	server := &http.Server{
		Addr:    cfg.AnalysisServer.Address,
		Handler: r,
	}
	go func() {
		slog.Info("starting analysis http server", "address", cfg.AnalysisServer.Address)
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
