package analysis

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{
		repo: repo,
	}
}

type createReportRequest struct {
	WorkID     int64   `json:"work_id"`
	Status     string  `json:"status"`
	Similarity float64 `json:"similarity"`
	Details    string  `json:"details"`
}

type reportResponse struct {
	ID         int64   `json:"id"`
	WorkID     int64   `json:"work_id"`
	Status     string  `json:"status"`
	Similarity float64 `json:"similarity"`
	Details    string  `json:"details"`
	CreatedAt  string  `json:"created_at"`
}

func (h *Handler) CreateReport(w http.ResponseWriter, r *http.Request) {
	var req createReportRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		slog.Error("failed to decode request", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.WorkID <= 0 || req.Status == "" {
		http.Error(w, "work_id and status are required", http.StatusBadRequest)
		return
	}

	switch req.Status {
	case "done":
		if req.Similarity < 0 || req.Similarity > 100 {
			http.Error(w, "similarity must be between 0 and 100", http.StatusBadRequest)
			return
		}
	default:
		if req.Similarity == 0 {
			req.Similarity = SimilarityUnknown // -1.0
		}
	}

	report := &Report{
		WorkID:     req.WorkID,
		Status:     req.Status,
		Details:    req.Details,
		Similarity: req.Similarity,
	}
	if err := h.repo.CreateReport(r.Context(), report); err != nil {
		slog.Error("failed to create report", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	response := &reportResponse{
		ID:         report.ID,
		WorkID:     report.WorkID,
		Status:     report.Status,
		Similarity: report.Similarity,
		Details:    report.Details,
		CreatedAt:  report.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, response)
}

func (h *Handler) GetReport(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "id parameter is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id parameter", http.StatusBadRequest)
		return
	}
	report, err := h.repo.GetReport(r.Context(), id)
	if err != nil {
		slog.Error("failed to get report", "err", err)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	response := &reportResponse{
		ID:         report.ID,
		WorkID:     report.WorkID,
		Status:     report.Status,
		Similarity: report.Similarity,
		Details:    report.Details,
		CreatedAt:  report.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}

func (h *Handler) GetReportByWorkID(w http.ResponseWriter, r *http.Request) {
	workIDStr := chi.URLParam(r, "work_id")
	if workIDStr == "" {
		http.Error(w, "work_id parameter is required", http.StatusBadRequest)
		return
	}

	workID, err := strconv.ParseInt(workIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid work_id parameter", http.StatusBadRequest)
		return
	}
	report, err := h.repo.GetReportByWorkID(r.Context(), workID)
	if err != nil {
		slog.Error("failed to get report by work_id", "err", err)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	response := &reportResponse{
		ID:         report.ID,
		WorkID:     report.WorkID,
		Status:     report.Status,
		Similarity: report.Similarity,
		Details:    report.Details,
		CreatedAt:  report.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}
