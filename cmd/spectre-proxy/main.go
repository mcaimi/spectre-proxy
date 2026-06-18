package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"codeberg.org/mcaimi/spectre-proxy/internal/api"
	"codeberg.org/mcaimi/spectre-proxy/internal/cert"
	"codeberg.org/mcaimi/spectre-proxy/internal/config"
	"codeberg.org/mcaimi/spectre-proxy/internal/proxy"
	"codeberg.org/mcaimi/spectre-proxy/internal/storage"
	"codeberg.org/mcaimi/spectre-proxy/internal/vhost"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	configPath string
	log        *logrus.Logger
)

func init() {
	log = logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	log.SetLevel(logrus.InfoLevel)
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "spectre",
		Short: "SPECTRE - HTTP/HTTPS interception proxy",
		Long:  "A transparent HTTP/HTTPS interception proxy for reverse engineers",
		Run:   run,
	}

	rootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to configuration file")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func run(cmd *cobra.Command, args []string) {
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err == nil {
		log.SetLevel(level)
	}

	if cfg.Logging.Format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{})
	}

	log.Info("Starting SPECTRE proxy...")

	db, err := storage.NewDatabase(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	log.Infof("Database initialized at %s", cfg.Database.Path)

	certRepo := storage.NewCertRepository(db)
	certManager, err := cert.NewManager(
		certRepo,
		cfg.TLS.CertCacheSize,
		cfg.TLS.CAKeySize,
		cfg.TLS.CertKeySize,
		cfg.TLS.CertValidity,
		log,
	)
	if err != nil {
		log.Fatalf("Failed to initialize certificate manager: %v", err)
	}
	log.Info("Certificate manager initialized")

	vhostRepo := storage.NewVHostRepository(db)
	router := vhost.NewRouter()
	if err := router.LoadFromRepository(vhostRepo); err != nil {
		log.Fatalf("Failed to load virtual hosts: %v", err)
	}
	log.Infof("Loaded virtual hosts into router")

	requestRepo := storage.NewRequestRepository(db)

	proxyConfig := &proxy.Config{
		HTTPAddr:     fmt.Sprintf("%s:%d", cfg.Proxy.BindAddr, cfg.Proxy.HTTPPort),
		HTTPSAddr:    fmt.Sprintf("%s:%d", cfg.Proxy.BindAddr, cfg.Proxy.HTTPSPort),
		ReadTimeout:  cfg.Proxy.ReadTimeout,
		WriteTimeout: cfg.Proxy.WriteTimeout,
		IdleTimeout:  cfg.Proxy.IdleTimeout,
	}

	proxyServer := proxy.NewProxy(proxyConfig, certManager, router, requestRepo, log)
	if err := proxyServer.Start(); err != nil {
		log.Fatalf("Failed to start proxy: %v", err)
	}

	var apiServer *api.Server
	if cfg.WebUI.Enabled {
		apiConfig := &api.Config{
			Addr:         fmt.Sprintf("%s:%d", cfg.WebUI.BindAddr, cfg.WebUI.Port),
			CAKeySize:    cfg.TLS.CAKeySize,
			CertKeySize:  cfg.TLS.CertKeySize,
			CertValidity: cfg.TLS.CertValidity,
		}
		apiServer = api.NewServer(apiConfig, certManager, router, db, log)
		go func() {
			if err := apiServer.Start(apiConfig); err != nil {
				log.Errorf("API server error: %v", err)
			}
		}()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Info("Received shutdown signal")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := proxyServer.Shutdown(shutdownCtx); err != nil {
		log.Errorf("Proxy shutdown error: %v", err)
	}

	if apiServer != nil {
		if err := apiServer.Shutdown(shutdownCtx); err != nil {
			log.Errorf("API server shutdown error: %v", err)
		}
	}

	log.Info("SPECTRE proxy stopped")
}
