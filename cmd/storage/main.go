package main

import (
	"HW_KPO3/internal/analysis"
	"HW_KPO3/internal/config"
	"HW_KPO3/internal/storage"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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
	fmt.Println(config)

	log := setupLogger(config.Env)
	slog.SetDefault(log)
	slog.Info("config loaded",
		"env", config.Env,
		"addr", config.HTTPServer.Address,
		"storage_path", config.StoragePath,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	db, err := storage.NewStorage(ctx, config.StorageDB.DSN)
	if err != nil {
		slog.Error("failed to connect to storage db", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to db is successful")
	defer db.Close()

	db_analysis, err := storage.NewStorage(ctx, config.AnalysisDB.DSN)
	if err != nil {
		slog.Error("failed to connect to analysis db", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to db is successful")
	defer db_analysis.Close()

	repo := storage.NewRepository(db)
	repo_analysis := analysis.NewRepository(db_analysis)
	report := &analysis.Report{
		WorkID:  1,
		Status:  "done",
		Details: "no details",
	}
	if err := repo_analysis.CreateReport(ctx, report); err != nil {
		slog.Error("failed to create report", "error", err)
	} else {
		slog.Info("created report", "id", report.ID)
	}

	hand := storage.NewHandler(repo)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Route("/works", func(rt chi.Router) {
		rt.Post("/", hand.CreateWork)
		rt.Get("/{id}", hand.GetWork)
	})

	server := http.Server{
		Addr:    config.HTTPServer.Address,
		Handler: r,
	}
	go func() {
		slog.Info("starting http server", "addr", config.HTTPServer.Address)
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
