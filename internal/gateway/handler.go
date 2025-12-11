package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (g *Gateway) CreateWorkAndReport(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to decode createWork request", "err", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

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

	var createdWork Work
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

	var createdReport Report
	if err := json.NewDecoder(anResp.Body).Decode(&createdReport); err != nil {
		slog.Error("failed to decode analysis response", "err", err)
		http.Error(w, "invalid response from analysis", http.StatusBadGateway)
		return
	}

	combined := CombinedWorkResponse{
		Work:   createdWork,
		Report: createdReport,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(combined); err != nil {
		slog.Error("failed to encode gateway response", "err", err)
	}
}

func (g *Gateway) GetWorkProxy(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	storageURL := g.storageBaseURL + "/works/" + id
	stReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, storageURL, nil)
	if err != nil {
		slog.Error("failed to create storage get request", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	stResp, err := g.httpClient.Do(stReq)
	if err != nil {
		slog.Error("storage get request failed", "err", err)
		http.Error(w, "storage service unavailable", http.StatusBadGateway)
		return
	}
	defer stResp.Body.Close()

	if stResp.StatusCode != http.StatusOK {
		slog.Error("storage returned non-200", "status", stResp.StatusCode)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(stResp.StatusCode)
		_, _ = io.Copy(w, stResp.Body)
		return
	}

	var workData Work
	if err := json.NewDecoder(stResp.Body).Decode(&workData); err != nil {
		slog.Error("failed to decode storage response", "err", err)
		http.Error(w, "invalid response from storage", http.StatusBadGateway)
		return
	}

	analysisURL := g.analysisBaseURL + "/reports/work/" + id
	anReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, analysisURL, nil)
	if err != nil {
		slog.Error("failed to create analysis get request", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	anResp, err := g.httpClient.Do(anReq)
	if err != nil {
		slog.Error("analysis get request failed", "err", err)
		slog.Warn("analysis service unavailable, returning work without report")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(workData); err != nil {
			slog.Error("failed to encode work response", "err", err)
		}
		return
	}
	defer anResp.Body.Close()

	var reportData Report
	if anResp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(anResp.Body).Decode(&reportData); err != nil {
			slog.Warn("failed to decode analysis response, returning work without report", "err", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(workData); err != nil {
				slog.Error("failed to encode work response", "err", err)
			}
			return
		}
	} else if anResp.StatusCode == http.StatusNotFound {
		slog.Info("report not found for work_id, returning work without report", "work_id", id)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(workData); err != nil {
			slog.Error("failed to encode work response", "err", err)
		}
		return
	} else {
		slog.Error("analysis service returned error, returning work without report",
			"status", anResp.StatusCode,
			"work_id", id)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(workData); err != nil {
			slog.Error("failed to encode work response", "err", err)
		}
		return
	}

	combined := CombinedWorkResponse{
		Work:   workData,
		Report: reportData,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(combined); err != nil {
		slog.Error("failed to encode gateway response", "err", err)
	}
}
