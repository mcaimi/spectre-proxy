package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"codeberg.org/mcaimi/spectre-proxy/internal/api/handlers"
	"codeberg.org/mcaimi/spectre-proxy/internal/cert"
	"codeberg.org/mcaimi/spectre-proxy/internal/storage"
	"codeberg.org/mcaimi/spectre-proxy/internal/vhost"
	"codeberg.org/mcaimi/spectre-proxy/internal/web"
	"github.com/sirupsen/logrus"
)

type Server struct {
	server       *http.Server
	log          *logrus.Logger
	certManager  *cert.Manager
	router       *vhost.Router
	db           *storage.Database
	caKeySize    int
	certKeySize  int
	certValidity time.Duration
}

type Config struct {
	Addr         string
	CAKeySize    int
	CertKeySize  int
	CertValidity time.Duration
}

func NewServer(cfg *Config, certManager *cert.Manager, router *vhost.Router, db *storage.Database, log *logrus.Logger) *Server {
	return &Server{
		log:          log,
		certManager:  certManager,
		router:       router,
		db:           db,
		caKeySize:    cfg.CAKeySize,
		certKeySize:  cfg.CertKeySize,
		certValidity: cfg.CertValidity,
	}
}

func (s *Server) Start(cfg *Config) error {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	vhostRepo := storage.NewVHostRepository(s.db)
	certRepo := storage.NewCertRepository(s.db)
	requestRepo := storage.NewRequestRepository(s.db)

	vhostHandler := handlers.NewVHostHandler(vhostRepo, s.router, s.log)
	certHandler := handlers.NewCertHandler(certRepo, s.certManager, s.caKeySize, s.certKeySize, s.certValidity, s.log)
	requestHandler := handlers.NewRequestHandler(requestRepo, s.log)

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

	// Serve static files from embedded React build
	staticFS, err := web.GetStaticFS()
	if err == nil {
		fileServer := http.FileServer(http.FS(staticFS))
		r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to serve the file
			if _, err := staticFS.Open(r.URL.Path[1:]); err == nil {
				fileServer.ServeHTTP(w, r)
			} else {
				// If file not found, serve index.html for client-side routing
				r.URL.Path = "/"
				fileServer.ServeHTTP(w, r)
			}
		}))
	} else {
		s.log.Warnf("Static files not embedded, UI will not be available: %v", err)
	}

	s.server = &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.log.Infof("API server listening on %s", cfg.Addr)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("Shutting down API server...")
	return s.server.Shutdown(ctx)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		fmt.Fprintf(w, "%v", data)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":"%s"}`, message)
}
