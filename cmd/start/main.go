package main

import (
	"HW_KPO3/internal/config"
	"HW_KPO3/internal/storage"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
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

	db, err := storage.NewStorage(ctx, config.StorageDB.DNS)
	if err != nil {
		slog.Error("failed to connect to storage db", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	slog.Info("connected to db is successful")
	//TODO: init router chi, render

	//TODO: run server
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
