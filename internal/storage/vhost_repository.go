package storage

import (
	"database/sql"
	"fmt"
	"time"
)

type VHostRepository struct {
	db *Database
}

func NewVHostRepository(db *Database) *VHostRepository {
	return &VHostRepository{db: db}
}

func (r *VHostRepository) Create(hostname, targetURL string) (*VirtualHost, error) {
	result, err := r.db.Exec(
		"INSERT INTO virtual_hosts (hostname, target_url, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		hostname, targetURL, true, time.Now(), time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual host: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return r.GetByID(id)
}

func (r *VHostRepository) GetByID(id int64) (*VirtualHost, error) {
	vh := &VirtualHost{}
	err := r.db.QueryRow(
		"SELECT id, hostname, target_url, enabled, created_at, updated_at FROM virtual_hosts WHERE id = ?",
		id,
	).Scan(&vh.ID, &vh.Hostname, &vh.TargetURL, &vh.Enabled, &vh.CreatedAt, &vh.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("virtual host not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual host: %w", err)
	}

	return vh, nil
}

func (r *VHostRepository) GetByHostname(hostname string) (*VirtualHost, error) {
	vh := &VirtualHost{}
	err := r.db.QueryRow(
		"SELECT id, hostname, target_url, enabled, created_at, updated_at FROM virtual_hosts WHERE hostname = ?",
		hostname,
	).Scan(&vh.ID, &vh.Hostname, &vh.TargetURL, &vh.Enabled, &vh.CreatedAt, &vh.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("virtual host not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual host: %w", err)
	}

	return vh, nil
}

func (r *VHostRepository) List() ([]*VirtualHost, error) {
	rows, err := r.db.Query(
		"SELECT id, hostname, target_url, enabled, created_at, updated_at FROM virtual_hosts ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual hosts: %w", err)
	}
	defer rows.Close()

	var vhosts []*VirtualHost
	for rows.Next() {
		vh := &VirtualHost{}
		if err := rows.Scan(&vh.ID, &vh.Hostname, &vh.TargetURL, &vh.Enabled, &vh.CreatedAt, &vh.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan virtual host: %w", err)
		}
		vhosts = append(vhosts, vh)
	}

	return vhosts, nil
}

func (r *VHostRepository) Update(vh *VirtualHost) error {
	vh.UpdatedAt = time.Now()
	_, err := r.db.Exec(
		"UPDATE virtual_hosts SET hostname = ?, target_url = ?, enabled = ?, updated_at = ? WHERE id = ?",
		vh.Hostname, vh.TargetURL, vh.Enabled, vh.UpdatedAt, vh.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update virtual host: %w", err)
	}
	return nil
}

func (r *VHostRepository) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM virtual_hosts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete virtual host: %w", err)
	}
	return nil
}
