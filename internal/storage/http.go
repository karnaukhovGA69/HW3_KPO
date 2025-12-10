package storage

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type Handler struct {
	repo *Reposiroty
}

func NewHandler(repo *Reposiroty) *Handler {
	return &Handler{
		repo: repo,
	}
}

type createWorkRequest struct {
	Student  string `json:"student"`
	Task     string `json:"task"`
	FilePath string `json:"file_path"`
}

type workResponse struct {
	ID         int64  `json:"id"`
	Student    string `json:"student"`
	Task       string `json:"task"`
	FilePath   string `json:"file_path"`
	UploadedAt string `json:"uploaded_at"`
}

func (h *Handler) CreateWork(w http.ResponseWriter, r *http.Request) {
	var req createWorkRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		slog.Error("failed to decode request", "err", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Student == "" || req.Task == "" || req.FilePath == "" {
		http.Error(w, "student, task and file_path are required", http.StatusBadRequest)
		return
	}
	work := &Work{
		Student:  req.Student,
		Task:     req.Task,
		FilePath: req.FilePath,
	}
	if err := h.repo.CreateWork(r.Context(), work); err != nil {
		slog.Error("failed to create work", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	response := &workResponse{
		ID:         work.ID,
		Student:    work.Student,
		Task:       work.Task,
		FilePath:   work.FilePath,
		UploadedAt: work.UploadedAt.Format("2006-01-02 15:04:05"),
	}
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, response)
}

func (h *Handler) GetWork(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	work, err := h.repo.GetWork(r.Context(), id)
	if err != nil {
		slog.Error("failed to get work", "err", err)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	response := &workResponse{
		ID:         work.ID,
		Student:    work.Student,
		Task:       work.Task,
		FilePath:   work.FilePath,
		UploadedAt: work.UploadedAt.Format("2006-01-02 15:04:05"),
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}
