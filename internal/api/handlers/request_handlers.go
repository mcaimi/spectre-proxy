package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mcaimi/spectre-proxy/internal/storage"
	"github.com/sirupsen/logrus"
)

type RequestHandler struct {
	repo *storage.RequestRepository
	log  *logrus.Logger
}

func NewRequestHandler(repo *storage.RequestRepository, log *logrus.Logger) *RequestHandler {
	return &RequestHandler{
		repo: repo,
		log:  log,
	}
}

func (h *RequestHandler) List(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 100
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	logs, err := h.repo.List(limit, offset)
	if err != nil {
		h.log.Errorf("Failed to list request logs: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to list request logs")
		return
	}

	total, _ := h.repo.Count()

	response := map[string]interface{}{
		"logs":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	respondJSON(w, http.StatusOK, response)
}

func (h *RequestHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	log, err := h.repo.GetByID(id)
	if err != nil {
		h.log.Debugf("Request log not found: %v", err)
		respondError(w, http.StatusNotFound, "Request log not found")
		return
	}

	respondJSON(w, http.StatusOK, log)
}
