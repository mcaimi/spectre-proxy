package storage

import (
	"database/sql"
	"fmt"
	"time"
)

type CertRepository struct {
	db *Database
}

func NewCertRepository(db *Database) *CertRepository {
	return &CertRepository{db: db}
}

func (r *CertRepository) SaveRootCA(cert, key []byte, expiresAt time.Time) error {
	_, err := r.db.Exec(
		"INSERT INTO root_ca (certificate, private_key, created_at, expires_at, is_active) VALUES (?, ?, ?, ?, ?)",
		cert, key, time.Now(), expiresAt, true,
	)
	if err != nil {
		return fmt.Errorf("failed to save root CA: %w", err)
	}
	return nil
}

func (r *CertRepository) GetRootCA() (*RootCA, error) {
	ca := &RootCA{}
	err := r.db.QueryRow(
		"SELECT id, certificate, private_key, created_at, expires_at, is_active FROM root_ca WHERE is_active = 1 ORDER BY created_at DESC LIMIT 1",
	).Scan(&ca.ID, &ca.Certificate, &ca.PrivateKey, &ca.CreatedAt, &ca.ExpiresAt, &ca.IsActive)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("root CA not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get root CA: %w", err)
	}

	return ca, nil
}

func (r *CertRepository) SaveCertificate(hostname string, cert, key []byte, expiresAt time.Time) error {
	_, err := r.db.Exec(
		"INSERT OR REPLACE INTO certificates (hostname, certificate, private_key, created_at, expires_at, is_active) VALUES (?, ?, ?, ?, ?, ?)",
		hostname, cert, key, time.Now(), expiresAt, true,
	)
	if err != nil {
		return fmt.Errorf("failed to save certificate: %w", err)
	}
	return nil
}

func (r *CertRepository) GetCertificate(hostname string) (*Certificate, error) {
	cert := &Certificate{}
	err := r.db.QueryRow(
		"SELECT id, hostname, certificate, private_key, created_at, expires_at, is_active FROM certificates WHERE hostname = ? AND is_active = 1",
		hostname,
	).Scan(&cert.ID, &cert.Hostname, &cert.Certificate, &cert.PrivateKey, &cert.CreatedAt, &cert.ExpiresAt, &cert.IsActive)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("certificate not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}

	return cert, nil
}

func (r *CertRepository) ListCertificates() ([]*Certificate, error) {
	rows, err := r.db.Query(
		"SELECT id, hostname, certificate, private_key, created_at, expires_at, is_active FROM certificates WHERE is_active = 1 ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates: %w", err)
	}
	defer rows.Close()

	var certs []*Certificate
	for rows.Next() {
		cert := &Certificate{}
		if err := rows.Scan(&cert.ID, &cert.Hostname, &cert.Certificate, &cert.PrivateKey, &cert.CreatedAt, &cert.ExpiresAt, &cert.IsActive); err != nil {
			return nil, fmt.Errorf("failed to scan certificate: %w", err)
		}
		certs = append(certs, cert)
	}

	return certs, nil
}

func (r *CertRepository) DeleteCertificate(id int64) error {
	_, err := r.db.Exec("UPDATE certificates SET is_active = 0 WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete certificate: %w", err)
	}
	return nil
}

func (r *CertRepository) DeactivateCurrentCA() error {
	_, err := r.db.Exec("UPDATE root_ca SET is_active = 0 WHERE is_active = 1")
	if err != nil {
		return fmt.Errorf("failed to deactivate current CA: %w", err)
	}
	return nil
}

func (r *CertRepository) DeleteAllCertificates() error {
	_, err := r.db.Exec("UPDATE certificates SET is_active = 0 WHERE is_active = 1")
	if err != nil {
		return fmt.Errorf("failed to delete all certificates: %w", err)
	}
	return nil
}
