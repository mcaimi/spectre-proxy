package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mcaimi/spectre-proxy/internal/cert"
	"github.com/mcaimi/spectre-proxy/internal/storage"
	"github.com/mcaimi/spectre-proxy/internal/vhost"
	"github.com/sirupsen/logrus"
)

type Proxy struct {
	httpListener  net.Listener
	httpsListener net.Listener
	certManager   *cert.Manager
	router        *vhost.Router
	requestRepo   *storage.RequestRepository
	log           *logrus.Logger
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	config        *Config
}

type Config struct {
	HTTPAddr     string
	HTTPSAddr    string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func NewProxy(cfg *Config, certManager *cert.Manager, router *vhost.Router, requestRepo *storage.RequestRepository, log *logrus.Logger) *Proxy {
	ctx, cancel := context.WithCancel(context.Background())

	return &Proxy{
		certManager: certManager,
		router:      router,
		requestRepo: requestRepo,
		log:         log,
		ctx:         ctx,
		cancel:      cancel,
		config:      cfg,
	}
}

func (p *Proxy) Start() error {
	httpListener, err := net.Listen("tcp", p.config.HTTPAddr)
	if err != nil {
		return fmt.Errorf("failed to start HTTP listener: %w", err)
	}
	p.httpListener = httpListener
	p.log.Infof("HTTP proxy listening on %s", p.config.HTTPAddr)

	tlsConfig := &tls.Config{
		GetCertificate: p.certManager.GetCertificate,
		MinVersion:     tls.VersionTLS12,
	}

	httpsListener, err := tls.Listen("tcp", p.config.HTTPSAddr, tlsConfig)
	if err != nil {
		httpListener.Close()
		return fmt.Errorf("failed to start HTTPS listener: %w", err)
	}
	p.httpsListener = httpsListener
	p.log.Infof("HTTPS proxy listening on %s", p.config.HTTPSAddr)

	p.wg.Add(2)
	go p.acceptHTTP()
	go p.acceptHTTPS()

	return nil
}

func (p *Proxy) Shutdown(ctx context.Context) error {
	p.log.Info("Shutting down proxy...")
	p.cancel()

	if p.httpListener != nil {
		p.httpListener.Close()
	}
	if p.httpsListener != nil {
		p.httpsListener.Close()
	}

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.log.Info("Proxy shutdown complete")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout")
	}
}

func (p *Proxy) acceptHTTP() {
	defer p.wg.Done()

	for {
		conn, err := p.httpListener.Accept()
		if err != nil {
			select {
			case <-p.ctx.Done():
				return
			default:
				p.log.Warnf("HTTP accept error: %v", err)
				continue
			}
		}

		go p.handleHTTPConnection(conn)
	}
}

func (p *Proxy) acceptHTTPS() {
	defer p.wg.Done()

	for {
		conn, err := p.httpsListener.Accept()
		if err != nil {
			select {
			case <-p.ctx.Done():
				return
			default:
				p.log.Warnf("HTTPS accept error: %v", err)
				continue
			}
		}

		go p.handleHTTPSConnection(conn)
	}
}
