package gateway

import (
	"bytes"
	"encoding/json"
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
		"status":     "done",
		"similarity": 0,
		"details":    "Plagiarism check completed",
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

	var workData *Work
	var reportData *Report
	hasWork, hasReport := false, false

	// Try to get work from storage
	storageURL := g.storageBaseURL + "/works/" + id
	stReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, storageURL, nil)
	if err != nil {
		slog.Warn("failed to create storage request", "err", err)
	} else {
		stResp, err := g.httpClient.Do(stReq)
		if err != nil {
			slog.Warn("storage request failed", "err", err)
		} else {
			if stResp != nil {
				defer func() { _ = stResp.Body.Close() }()
			}
			if stResp != nil && stResp.StatusCode == http.StatusOK {
				var ww Work
				if decErr := json.NewDecoder(stResp.Body).Decode(&ww); decErr == nil {
					workData = &ww
					hasWork = true
				} else {
					slog.Warn("failed to decode storage body", "err", decErr)
				}
			} else {
				status := 0
				if stResp != nil {
					status = stResp.StatusCode
				}
				slog.Warn("storage returned non-200", "status", status)
			}
		}
	}

	analysisURL := g.analysisBaseURL + "/reports/work/" + id
	anReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, analysisURL, nil)
	if err != nil {
		slog.Warn("failed to create analysis request", "err", err)
	} else {
		anResp, err := g.httpClient.Do(anReq)
		if err != nil {
			slog.Warn("analysis request failed", "err", err)
		} else {
			if anResp != nil {
				defer func() { _ = anResp.Body.Close() }()
			}
			if anResp != nil && anResp.StatusCode == http.StatusOK {
				var rr Report
				if decErr := json.NewDecoder(anResp.Body).Decode(&rr); decErr == nil {
					reportData = &rr
					hasReport = true
				} else {
					slog.Warn("failed to decode analysis body", "err", decErr)
				}
			} else {
				status := 0
				if anResp != nil {
					status = anResp.StatusCode
				}
				slog.Warn("analysis returned non-200", "status", status)
			}
		}
	}

	if !hasWork && !hasReport {
		http.Error(w, "both services unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if hasWork && hasReport {
		if err := json.NewEncoder(w).Encode(CombinedWorkResponse{Work: *workData, Report: *reportData}); err != nil {
			slog.Error("failed to encode gateway response", "err", err)
		}
		return
	}

	if hasWork {
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"work": workData, "message": "report service unavailable"}); err != nil {
			slog.Error("failed to encode gateway response", "err", err)
		}
		return
	}

	if hasReport {
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"report": reportData, "message": "storage service unavailable"}); err != nil {
			slog.Error("failed to encode gateway response", "err", err)
		}
		return
	}
}
