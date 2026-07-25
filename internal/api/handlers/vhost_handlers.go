package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mcaimi/spectre-proxy/internal/storage"
	"github.com/mcaimi/spectre-proxy/internal/vhost"
	"github.com/sirupsen/logrus"
)

type VHostHandler struct {
	repo   *storage.VHostRepository
	router *vhost.Router
	log    *logrus.Logger
}

func NewVHostHandler(repo *storage.VHostRepository, router *vhost.Router, log *logrus.Logger) *VHostHandler {
	return &VHostHandler{
		repo:   repo,
		router: router,
		log:    log,
	}
}

type createVHostRequest struct {
	Hostname  string `json:"hostname"`
	TargetURL string `json:"target_url"`
}

type updateVHostRequest struct {
	Hostname  string `json:"hostname"`
	TargetURL string `json:"target_url"`
	Enabled   bool   `json:"enabled"`
}

func (h *VHostHandler) List(w http.ResponseWriter, r *http.Request) {
	vhosts, err := h.repo.List()
	if err != nil {
		h.log.Errorf("Failed to list virtual hosts: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to list virtual hosts")
		return
	}

	respondJSON(w, http.StatusOK, vhosts)
}

func (h *VHostHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	vhost, err := h.repo.GetByID(id)
	if err != nil {
		h.log.Debugf("Virtual host not found: %v", err)
		respondError(w, http.StatusNotFound, "Virtual host not found")
		return
	}

	respondJSON(w, http.StatusOK, vhost)
}

func (h *VHostHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createVHostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Hostname == "" || req.TargetURL == "" {
		respondError(w, http.StatusBadRequest, "Hostname and target_url are required")
		return
	}

	vhost, err := h.repo.Create(req.Hostname, req.TargetURL)
	if err != nil {
		h.log.Errorf("Failed to create virtual host: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create virtual host")
		return
	}

	if err := h.router.Add(vhost); err != nil {
		h.log.Errorf("Failed to add virtual host to router: %v", err)
	}

	respondJSON(w, http.StatusCreated, vhost)
}

func (h *VHostHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	vhost, err := h.repo.GetByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Virtual host not found")
		return
	}

	var req updateVHostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Hostname != "" {
		vhost.Hostname = req.Hostname
	}
	if req.TargetURL != "" {
		vhost.TargetURL = req.TargetURL
	}
	vhost.Enabled = req.Enabled

	if err := h.repo.Update(vhost); err != nil {
		h.log.Errorf("Failed to update virtual host: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to update virtual host")
		return
	}

	if err := h.router.LoadFromRepository(h.repo); err != nil {
		h.log.Errorf("Failed to reload router: %v", err)
	}

	respondJSON(w, http.StatusOK, vhost)
}

func (h *VHostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	vhost, err := h.repo.GetByID(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Virtual host not found")
		return
	}

	if err := h.repo.Delete(id); err != nil {
		h.log.Errorf("Failed to delete virtual host: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to delete virtual host")
		return
	}

	h.router.Remove(vhost.Hostname)

	w.WriteHeader(http.StatusNoContent)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
