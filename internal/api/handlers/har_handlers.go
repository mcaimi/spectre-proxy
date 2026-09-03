package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mcaimi/spectre-proxy/internal/har"
	"github.com/mcaimi/spectre-proxy/internal/storage"
	"github.com/sirupsen/logrus"
)

type HARHandler struct {
	requestRepo *storage.RequestRepository
	vhostRepo   *storage.VHostRepository
	log         *logrus.Logger
}

func NewHARHandler(requestRepo *storage.RequestRepository, vhostRepo *storage.VHostRepository, log *logrus.Logger) *HARHandler {
	return &HARHandler{
		requestRepo: requestRepo,
		vhostRepo:   vhostRepo,
		log:         log,
	}
}

func (h *HARHandler) ExportHAR(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	vhost, err := h.vhostRepo.GetByID(id)
	if err != nil {
		h.log.Debugf("Virtual host not found: %v", err)
		respondError(w, http.StatusNotFound, "Virtual host not found")
		return
	}

	logs, err := h.requestRepo.ListByVHostID(id)
	if err != nil {
		h.log.Errorf("Failed to list request logs for vhost %d: %v", id, err)
		respondError(w, http.StatusInternalServerError, "Failed to retrieve request logs")
		return
	}

	harFile, err := har.FromRequestLogs(logs)
	if err != nil {
		h.log.Errorf("Failed to convert logs to HAR: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to generate HAR file")
		return
	}

	filename := fmt.Sprintf("%s.har", vhost.Hostname)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(harFile)
}
