package main

import (
	"HW_KPO3/internal/config"
	"HW_KPO3/internal/logger"
	"HW_KPO3/internal/storage"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.MustLoad()

	log := logger.SetupLogger(cfg.Env)
	slog.SetDefault(log)
	slog.Info("config loaded",
		"env", cfg.Env,
		"addr", cfg.HTTPServer.Address,
		"storage_path", cfg.StoragePath,
		"dsn", cfg.StorageDB.DSN,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	db, err := storage.NewStorage(ctx, cfg.StorageDB.DSN)
	if err != nil {
		slog.Error("failed to connect to storage db", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to storage db")
	defer db.Close()

	repo := storage.NewRepository(db)
	handler := storage.NewHandler(repo)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Route("/works", func(rt chi.Router) {
		rt.Post("/", handler.CreateWork)
		rt.Get("/{id}", handler.GetWork)
	})

	server := http.Server{
		Addr:    cfg.HTTPServer.Address,
		Handler: r,
	}
	go func() {
		slog.Info("starting http server", "addr", cfg.HTTPServer.Address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()
	<-ctx.Done()
	slog.Info("shutting down http server")

	if err := server.Shutdown(context.Background()); err != nil {
		slog.Error("http server shutdown error", "error", err)
	}
}
