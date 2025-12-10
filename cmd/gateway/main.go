package main

import (
	"HW_KPO3/internal/config"
	"bytes"
	"context"
	"encoding/json"
	"io"
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

// DTO из storage-сервиса
type work struct {
	ID         int64  `json:"id"`
	Student    string `json:"student"`
	Task       string `json:"task"`
	FilePath   string `json:"file_path"`
	UploadedAt string `json:"uploaded_at"`
}

// DTO из analysis-сервиса
type report struct {
	ID         int64   `json:"id"`
	WorkID     int64   `json:"work_id"`
	Status     string  `json:"status"`
	Similarity float64 `json:"similarity"`
	Details    string  `json:"details"`
	CreatedAt  string  `json:"created_at"`
}

// тело запроса на создание работы через gateway
type createWorkRequest struct {
	Student  string `json:"student"`
	Task     string `json:"task"`
	FilePath string `json:"file_path"`
}

// объединённый ответ: работа + отчёт
type combinedWorkResponse struct {
	Work   work   `json:"work"`
	Report report `json:"report"`
}

type Gateway struct {
	storageBaseURL  string
	analysisBaseURL string
	httpClient      *http.Client
}

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)
	slog.SetDefault(log)

	slog.Info("config loaded",
		"env", cfg.Env,
		"gateway_addr", cfg.Gateway.Address,
		"storage_url", cfg.Gateway.StorageBaseURL,
		"analysis_url", cfg.Gateway.AnalysisBaseURL,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	gw := &Gateway{
		storageBaseURL:  cfg.Gateway.StorageBaseURL,
		analysisBaseURL: cfg.Gateway.AnalysisBaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// high-level API
	r.Post("/works", gw.createWorkAndReport)
	r.Get("/works/{id}", gw.getWorkProxy) // пока простое проксирование в storage

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

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envTest, envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	return log
}

// POST /works на gateway:
// 1) создаёт работу в storage
// 2) создаёт для неё pending-отчёт в analysis
// 3) возвращает объединённый ответ
func (g *Gateway) createWorkAndReport(w http.ResponseWriter, r *http.Request) {
	var req createWorkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to decode createWork request", "err", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// --- 1. создаём работу в storage ---
	storageURL := g.storageBaseURL + "/works"

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		slog.Error("failed to marshal request to storage", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	stReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, storageURL, bytes.NewReader(bodyBytes))
	if err != nil {
		slog.Error("failed to create storage request", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	stReq.Header.Set("Content-Type", "application/json")

	stResp, err := g.httpClient.Do(stReq)
	if err != nil {
		slog.Error("storage request failed", "err", err)
		http.Error(w, "storage service unavailable", http.StatusBadGateway)
		return
	}
	defer stResp.Body.Close()

	if stResp.StatusCode != http.StatusCreated {
		slog.Error("storage returned non-201", "status", stResp.StatusCode)
		http.Error(w, "failed to create work", http.StatusBadGateway)
		return
	}

	var createdWork work
	if err := json.NewDecoder(stResp.Body).Decode(&createdWork); err != nil {
		slog.Error("failed to decode storage response", "err", err)
		http.Error(w, "invalid response from storage", http.StatusBadGateway)
		return
	}

	analysisURL := g.analysisBaseURL + "/reports"

	createReportPayload := map[string]interface{}{
		"work_id":    createdWork.ID,
		"status":     "pending",
		"similarity": -1.0,
		"details":    "pending",
	}

	reportBody, err := json.Marshal(createReportPayload)
	if err != nil {
		slog.Error("failed to marshal analysis request", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	anReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, analysisURL, bytes.NewReader(reportBody))
	if err != nil {
		slog.Error("failed to create analysis request", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	anReq.Header.Set("Content-Type", "application/json")

	anResp, err := g.httpClient.Do(anReq)
	if err != nil {
		slog.Error("analysis request failed", "err", err)
		http.Error(w, "analysis service unavailable", http.StatusBadGateway)
		return
	}
	defer anResp.Body.Close()

	if anResp.StatusCode != http.StatusCreated {
		slog.Error("analysis returned non-201", "status", anResp.StatusCode)
		http.Error(w, "failed to create report", http.StatusBadGateway)
		return
	}

	var createdReport report
	if err := json.NewDecoder(anResp.Body).Decode(&createdReport); err != nil {
		slog.Error("failed to decode analysis response", "err", err)
		http.Error(w, "invalid response from analysis", http.StatusBadGateway)
		return
	}

	combined := combinedWorkResponse{
		Work:   createdWork,
		Report: createdReport,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(combined); err != nil {
		slog.Error("failed to encode gateway response", "err", err)
	}
}

// GET /works/{id} на gateway — пока просто проксирует запрос в storage
func (g *Gateway) getWorkProxy(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	url := g.storageBaseURL + "/works/" + id

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		slog.Error("failed to create storage get request", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		slog.Error("storage get request failed", "err", err)
		http.Error(w, "storage service unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
