package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"codeberg.org/mcaimi/spectre-proxy/internal/cert"
	"codeberg.org/mcaimi/spectre-proxy/internal/storage"
	"github.com/sirupsen/logrus"
)

type CertHandler struct {
	repo         *storage.CertRepository
	certManager  *cert.Manager
	log          *logrus.Logger
	caKeySize    int
	certKeySize  int
	certValidity time.Duration
}

func NewCertHandler(repo *storage.CertRepository, certManager *cert.Manager, caKeySize, certKeySize int, certValidity time.Duration, log *logrus.Logger) *CertHandler {
	return &CertHandler{
		repo:         repo,
		certManager:  certManager,
		log:          log,
		caKeySize:    caKeySize,
		certKeySize:  certKeySize,
		certValidity: certValidity,
	}
}

func (h *CertHandler) List(w http.ResponseWriter, r *http.Request) {
	certs, err := h.repo.ListCertificates()
	if err != nil {
		h.log.Errorf("Failed to list certificates: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to list certificates")
		return
	}

	type certResponse struct {
		ID        int64  `json:"id"`
		Hostname  string `json:"hostname"`
		CreatedAt string `json:"created_at"`
		ExpiresAt string `json:"expires_at"`
	}

	response := make([]certResponse, len(certs))
	for i, cert := range certs {
		response[i] = certResponse{
			ID:        cert.ID,
			Hostname:  cert.Hostname,
			CreatedAt: cert.CreatedAt.Format("2006-01-02 15:04:05"),
			ExpiresAt: cert.ExpiresAt.Format("2006-01-02 15:04:05"),
		}
	}

	respondJSON(w, http.StatusOK, response)
}

func (h *CertHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	if err := h.repo.DeleteCertificate(id); err != nil {
		h.log.Errorf("Failed to delete certificate: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to delete certificate")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CertHandler) DownloadCA(w http.ResponseWriter, r *http.Request) {
	caPEM := h.certManager.GetRootCACert()

	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", "attachment; filename=spectre-ca.crt")
	w.WriteHeader(http.StatusOK)
	w.Write(caPEM)
}

func (h *CertHandler) ReplaceCA(w http.ResponseWriter, r *http.Request) {
	type replaceCARequest struct {
		CommonName         string `json:"common_name"`
		Organization       string `json:"organization"`
		OrganizationalUnit string `json:"organizational_unit"`
		Country            string `json:"country"`
		Province           string `json:"province"`
		Locality           string `json:"locality"`
		ValidityYears      int    `json:"validity_years"`
	}

	var req replaceCARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.CommonName == "" {
		respondError(w, http.StatusBadRequest, "common_name is required")
		return
	}

	if req.ValidityYears <= 0 {
		req.ValidityYears = 10
	}

	fields := cert.CAFields{
		CommonName:         req.CommonName,
		Organization:       req.Organization,
		OrganizationalUnit: req.OrganizationalUnit,
		Country:            req.Country,
		Province:           req.Province,
		Locality:           req.Locality,
		ValidityYears:      req.ValidityYears,
	}

	if err := h.certManager.ReplaceCA(fields, h.caKeySize, h.certKeySize, h.certValidity); err != nil {
		h.log.Errorf("Failed to replace CA: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to replace CA certificate")
		return
	}

	h.log.Info("CA certificate replaced successfully via API")
	respondJSON(w, http.StatusOK, map[string]string{"message": "CA certificate replaced successfully"})
}
