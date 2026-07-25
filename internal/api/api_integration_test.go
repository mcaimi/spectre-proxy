package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mcaimi/spectre-proxy/internal/api/handlers"
	"github.com/mcaimi/spectre-proxy/internal/cert"
	"github.com/mcaimi/spectre-proxy/internal/storage"
	"github.com/mcaimi/spectre-proxy/internal/vhost"
	"github.com/sirupsen/logrus"
)

// Helper to create a test server with an in-memory database
func setupTestServer(t *testing.T) (*chi.Mux, *storage.Database, func()) {
	// Create in-memory database
	db, err := storage.NewDatabase(":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// Create repositories
	vhostRepo := storage.NewVHostRepository(db)
	certRepo := storage.NewCertRepository(db)
	requestRepo := storage.NewRequestRepository(db)

	// Create cert manager
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	certManager, err := cert.NewManager(certRepo, 10, 2048, 2048, 24*time.Hour, logger)
	if err != nil {
		t.Fatalf("failed to create cert manager: %v", err)
	}

	// Create vhost router
	router := vhost.NewRouter()
	if err := router.LoadFromRepository(vhostRepo); err != nil {
		t.Fatalf("failed to load router: %v", err)
	}

	// Create handlers
	vhostHandler := handlers.NewVHostHandler(vhostRepo, router, logger)
	certHandler := handlers.NewCertHandler(certRepo, certManager, 2048, 2048, 24*time.Hour, logger)
	requestHandler := handlers.NewRequestHandler(requestRepo, logger)

	// Setup routes
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
		})

		r.Route("/vhosts", func(r chi.Router) {
			r.Get("/", vhostHandler.List)
			r.Post("/", vhostHandler.Create)
			r.Get("/{id}", vhostHandler.Get)
			r.Put("/{id}", vhostHandler.Update)
			r.Delete("/{id}", vhostHandler.Delete)
		})

		r.Route("/certificates", func(r chi.Router) {
			r.Get("/", certHandler.List)
			r.Delete("/{id}", certHandler.Delete)
			r.Get("/ca", certHandler.DownloadCA)
			r.Post("/ca/replace", certHandler.ReplaceCA)
			r.Post("/ca/load", certHandler.LoadCA)
		})

		r.Route("/logs", func(r chi.Router) {
			r.Get("/", requestHandler.List)
			r.Get("/{id}", requestHandler.Get)
		})
	})

	cleanup := func() {
		db.Close()
	}

	return r, db, cleanup
}

func makeRequest(method, path string, body interface{}) (*http.Request, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, path, reqBody)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// Tests

func TestAPI_Health(t *testing.T) {
	r, _, cleanup := setupTestServer(t)
	defer cleanup()

	req, err := makeRequest(http.MethodGet, "/api/v1/health", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %v", response["status"])
	}
}

func TestAPI_VHost_CreateListGetUpdateDelete(t *testing.T) {
	r, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Test CREATE
	createBody := map[string]string{
		"hostname":   "api.example.com",
		"target_url": "https://backend.example.com",
	}

	req, err := makeRequest(http.MethodPost, "/api/v1/vhosts", createBody)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var created map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if created["id"] == nil {
		t.Fatalf("response missing 'id' field. Response body: %s", rr.Body.String())
	}

	vhostID := int(created["id"].(float64))

	// Test LIST
	req, _ = makeRequest(http.MethodGet, "/api/v1/vhosts", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var vhosts []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &vhosts); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(vhosts) != 1 {
		t.Errorf("expected 1 vhost, got %d", len(vhosts))
	}

	// Test GET
	req, _ = makeRequest(http.MethodGet, fmt.Sprintf("/api/v1/vhosts/%d", vhostID), nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Test UPDATE
	updateBody := map[string]interface{}{
		"hostname":   "updated.example.com",
		"target_url": "https://new-backend.example.com",
		"enabled":    false,
	}

	req, _ = makeRequest(http.MethodPut, fmt.Sprintf("/api/v1/vhosts/%d", vhostID), updateBody)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var updated map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if updated["hostname"] != "updated.example.com" {
		t.Errorf("expected hostname 'updated.example.com', got %v", updated["hostname"])
	}

	// Test DELETE
	req, _ = makeRequest(http.MethodDelete, fmt.Sprintf("/api/v1/vhosts/%d", vhostID), nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}

	// Verify deletion
	req, _ = makeRequest(http.MethodGet, fmt.Sprintf("/api/v1/vhosts/%d", vhostID), nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status %d after deletion, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestAPI_VHost_ValidationErrors(t *testing.T) {
	r, _, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           map[string]string
		expectedStatus int
	}{
		{
			name:           "missing hostname",
			body:           map[string]string{"target_url": "https://backend.com"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing target_url",
			body:           map[string]string{"hostname": "example.com"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty body",
			body:           map[string]string{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := makeRequest(http.MethodPost, "/api/v1/vhosts", tt.body)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestAPI_Certificates_List(t *testing.T) {
	r, db, cleanup := setupTestServer(t)
	defer cleanup()

	certRepo := storage.NewCertRepository(db)

	// Add some certificates
	certRepo.SaveCertificate("example.com", []byte("cert1"), []byte("key1"), time.Now().Add(24*time.Hour))
	certRepo.SaveCertificate("test.com", []byte("cert2"), []byte("key2"), time.Now().Add(48*time.Hour))

	req, _ := makeRequest(http.MethodGet, "/api/v1/certificates", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var certs []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &certs); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(certs) != 2 {
		t.Errorf("expected 2 certificates, got %d", len(certs))
	}

	// Verify sensitive data is not exposed
	for _, cert := range certs {
		if _, ok := cert["certificate"]; ok {
			t.Error("certificate data should not be exposed in list response")
		}
		if _, ok := cert["private_key"]; ok {
			t.Error("private key data should not be exposed in list response")
		}
	}
}

func TestAPI_Certificates_DownloadCA(t *testing.T) {
	r, _, cleanup := setupTestServer(t)
	defer cleanup()

	req, _ := makeRequest(http.MethodGet, "/api/v1/certificates/ca", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/x-pem-file" {
		t.Errorf("expected Content-Type 'application/x-pem-file', got %s", contentType)
	}

	disposition := rr.Header().Get("Content-Disposition")
	if disposition != "attachment; filename=spectre-ca.crt" {
		t.Errorf("expected Content-Disposition 'attachment; filename=spectre-ca.crt', got %s", disposition)
	}

	if rr.Body.Len() == 0 {
		t.Error("CA certificate body should not be empty")
	}
}

func TestAPI_RequestLogs_Pagination(t *testing.T) {
	r, db, cleanup := setupTestServer(t)
	defer cleanup()

	requestRepo := storage.NewRequestRepository(db)

	// Create 50 request logs
	for i := 0; i < 50; i++ {
		requestRepo.Save(&storage.RequestLog{
			Method:         "GET",
			URL:            fmt.Sprintf("https://example.com/api/%d", i),
			ResponseStatus: 200,
			ClientIP:       "127.0.0.1",
		})
	}

	tests := []struct {
		name          string
		queryParams   string
		expectedCount int
	}{
		{
			name:          "default pagination",
			queryParams:   "",
			expectedCount: 50,
		},
		{
			name:          "limit 10",
			queryParams:   "?limit=10",
			expectedCount: 10,
		},
		{
			name:          "limit 20 offset 10",
			queryParams:   "?limit=20&offset=10",
			expectedCount: 20,
		},
		{
			name:          "offset beyond records",
			queryParams:   "?limit=10&offset=100",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := makeRequest(http.MethodGet, "/api/v1/logs"+tt.queryParams, nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			var logs []interface{}
			if response["logs"] != nil {
				logs = response["logs"].([]interface{})
			}

			if len(logs) != tt.expectedCount {
				t.Errorf("expected %d logs, got %d", tt.expectedCount, len(logs))
			}

			total := int64(response["total"].(float64))
			if total != 50 {
				t.Errorf("expected total 50, got %d", total)
			}
		})
	}
}

func TestAPI_RequestLogs_GetByID(t *testing.T) {
	r, db, cleanup := setupTestServer(t)
	defer cleanup()

	requestRepo := storage.NewRequestRepository(db)

	// Create a request log
	vhostID := int64(1)
	requestRepo.Save(&storage.RequestLog{
		VHostID:         &vhostID,
		Method:          "POST",
		URL:             "https://api.example.com/users",
		RequestHeaders:  `{"Content-Type": "application/json"}`,
		RequestBody:     []byte(`{"name": "John"}`),
		ResponseStatus:  201,
		ResponseHeaders: `{"Content-Type": "application/json"}`,
		ResponseBody:    []byte(`{"id": 1}`),
		ClientIP:        "192.168.1.100",
		DurationMs:      125,
	})

	req, _ := makeRequest(http.MethodGet, "/api/v1/logs/1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var log map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &log); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if log["method"] != "POST" {
		t.Errorf("expected method POST, got %v", log["method"])
	}
	if log["url"] != "https://api.example.com/users" {
		t.Errorf("expected URL https://api.example.com/users, got %v", log["url"])
	}
	if log["status_code"] != float64(201) {
		t.Errorf("expected response status 201, got %v", log["status_code"])
	}
}

func TestAPI_ReplaceCA(t *testing.T) {
	r, db, cleanup := setupTestServer(t)
	defer cleanup()

	certRepo := storage.NewCertRepository(db)

	// Get the initial CA
	initialCA, err := certRepo.GetRootCA()
	if err != nil {
		t.Fatalf("failed to get initial CA: %v", err)
	}

	// Create some certificates to verify regeneration
	vhostRepo := storage.NewVHostRepository(db)
	vhostRepo.Create("test.example.com", "http://localhost:8080")

	tests := []struct {
		name           string
		request        interface{}
		expectedStatus int
	}{
		{
			name: "replace with custom CA",
			request: map[string]interface{}{
				"common_name":         "My Custom CA",
				"organization":        "My Organization",
				"organizational_unit": "Security Team",
				"country":             "US",
				"province":            "California",
				"locality":            "San Francisco",
				"validity_years":      5,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "replace with minimal fields",
			request: map[string]interface{}{
				"common_name": "Another Custom CA",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing common_name",
			request: map[string]interface{}{
				"organization": "Test Org",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := makeRequest(http.MethodPost, "/api/v1/certificates/ca/replace", tt.request)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			if tt.expectedStatus == http.StatusOK {
				// Verify the CA was actually replaced
				newCA, err := certRepo.GetRootCA()
				if err != nil {
					t.Fatalf("failed to get new CA: %v", err)
				}

				// Verify it's different from the initial CA
				if newCA.ID == initialCA.ID {
					t.Error("CA ID should have changed")
				}

				// Update initialCA for next iteration
				initialCA = newCA
			}
		})
	}
}

func TestAPI_InvalidEndpoints(t *testing.T) {
	r, _, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "invalid vhost ID",
			method:         http.MethodGet,
			path:           "/api/v1/vhosts/invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "non-existent vhost",
			method:         http.MethodGet,
			path:           "/api/v1/vhosts/999",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid log ID",
			method:         http.MethodGet,
			path:           "/api/v1/logs/invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "non-existent log",
			method:         http.MethodGet,
			path:           "/api/v1/logs/999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := makeRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}
