package cert

import (
	"crypto/tls"
	"fmt"
	"time"

	"codeberg.org/mcaimi/spectre-proxy/internal/storage"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	ca        *CA
	generator *Generator
	cache     *Cache
	repo      *storage.CertRepository
	log       *logrus.Logger
}

func NewManager(repo *storage.CertRepository, cacheSize, caKeySize, certKeySize int, certValidity time.Duration, log *logrus.Logger) (*Manager, error) {
	cache := NewCache(cacheSize)

	ca, err := loadOrCreateCA(repo, caKeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize CA: %w", err)
	}

	generator := NewGenerator(ca, certKeySize, certValidity)

	return &Manager{
		ca:        ca,
		generator: generator,
		cache:     cache,
		repo:      repo,
		log:       log,
	}, nil
}

func loadOrCreateCA(repo *storage.CertRepository, keySize int) (*CA, error) {
	storedCA, err := repo.GetRootCA()
	if err == nil {
		ca, err := LoadCA(storedCA.Certificate, storedCA.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load stored CA: %w", err)
		}
		return ca, nil
	}

	ca, err := NewCA(keySize)
	if err != nil {
		return nil, fmt.Errorf("failed to create new CA: %w", err)
	}

	if err := repo.SaveRootCA(ca.GetCertPEM(), ca.GetKeyPEM(), ca.GetExpiresAt()); err != nil {
		return nil, fmt.Errorf("failed to save CA: %w", err)
	}

	return ca, nil
}

func (m *Manager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	hostname := hello.ServerName
	if hostname == "" {
		return nil, fmt.Errorf("no SNI hostname provided")
	}

	if cert, ok := m.cache.Get(hostname); ok {
		m.log.Debugf("Certificate cache hit for %s", hostname)
		return cert, nil
	}

	storedCert, err := m.repo.GetCertificate(hostname)
	if err == nil {
		cert, err := tls.X509KeyPair(storedCert.Certificate, storedCert.PrivateKey)
		if err == nil {
			m.cache.Set(hostname, &cert, storedCert.ExpiresAt)
			m.log.Infof("Loaded certificate for %s from database", hostname)
			return &cert, nil
		}
	}

	cert, certPEM, keyPEM, expiresAt, err := m.generator.GenerateCertificate(hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate for %s: %w", hostname, err)
	}

	if err := m.repo.SaveCertificate(hostname, certPEM, keyPEM, expiresAt); err != nil {
		m.log.Warnf("Failed to persist certificate for %s: %v", hostname, err)
	}

	m.cache.Set(hostname, cert, expiresAt)
	m.log.Infof("Generated new certificate for %s", hostname)

	return cert, nil
}

func (m *Manager) GetRootCACert() []byte {
	return m.ca.GetCertPEM()
}

func (m *Manager) CleanupExpiredCache() int {
	count := m.cache.CleanupExpired()
	if count > 0 {
		m.log.Infof("Cleaned up %d expired certificates from cache", count)
	}
	return count
}

func (m *Manager) ReplaceCA(fields CAFields, caKeySize, certKeySize int, certValidity time.Duration) error {
	newCA, err := NewCAWithFields(caKeySize, fields)
	if err != nil {
		return fmt.Errorf("failed to create new CA: %w", err)
	}

	if err := m.repo.DeactivateCurrentCA(); err != nil {
		return fmt.Errorf("failed to deactivate current CA: %w", err)
	}

	if err := m.repo.SaveRootCA(newCA.GetCertPEM(), newCA.GetKeyPEM(), newCA.GetExpiresAt()); err != nil {
		return fmt.Errorf("failed to save new CA: %w", err)
	}

	if err := m.repo.DeleteAllCertificates(); err != nil {
		m.log.Warnf("Failed to delete old certificates: %v", err)
	}

	m.ca = newCA
	m.generator = NewGenerator(newCA, certKeySize, certValidity)
	m.cache.Clear()

	m.log.Info("CA certificate replaced successfully")
	return nil
}

func (m *Manager) LoadCustomCA(certPEM, keyPEM []byte, certKeySize int, certValidity time.Duration) error {
	newCA, err := LoadCA(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("failed to load custom CA: %w", err)
	}

	if err := m.repo.DeactivateCurrentCA(); err != nil {
		return fmt.Errorf("failed to deactivate current CA: %w", err)
	}

	if err := m.repo.SaveRootCA(newCA.GetCertPEM(), newCA.GetKeyPEM(), newCA.GetExpiresAt()); err != nil {
		return fmt.Errorf("failed to save custom CA: %w", err)
	}

	if err := m.repo.DeleteAllCertificates(); err != nil {
		m.log.Warnf("Failed to delete old certificates: %v", err)
	}

	m.ca = newCA
	m.generator = NewGenerator(newCA, certKeySize, certValidity)
	m.cache.Clear()

	m.log.Info("Custom CA certificate loaded successfully")
	return nil
}

func (m *Manager) RegenerateAllCertificates() error {
	certs, err := m.repo.ListCertificates()
	if err != nil {
		return fmt.Errorf("failed to list certificates: %w", err)
	}

	if err := m.repo.DeleteAllCertificates(); err != nil {
		return fmt.Errorf("failed to delete old certificates: %w", err)
	}

	m.cache.Clear()

	for _, cert := range certs {
		_, certPEM, keyPEM, expiresAt, err := m.generator.GenerateCertificate(cert.Hostname)
		if err != nil {
			m.log.Errorf("Failed to regenerate certificate for %s: %v", cert.Hostname, err)
			continue
		}

		if err := m.repo.SaveCertificate(cert.Hostname, certPEM, keyPEM, expiresAt); err != nil {
			m.log.Errorf("Failed to save regenerated certificate for %s: %v", cert.Hostname, err)
		}
	}

	m.log.Info("All certificates regenerated successfully")
	return nil
}
