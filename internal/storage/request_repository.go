package storage

import (
	"fmt"
	"time"
)

type RequestRepository struct {
	db *Database
}

func NewRequestRepository(db *Database) *RequestRepository {
	return &RequestRepository{db: db}
}

func (r *RequestRepository) Save(req *RequestLog) error {
	result, err := r.db.Exec(
		`INSERT INTO request_logs
		(vhost_id, method, url, request_headers, request_body, response_status, response_headers, response_body, timestamp, duration_ms, client_ip)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.VHostID, req.Method, req.URL, req.RequestHeaders, req.RequestBody,
		req.ResponseStatus, req.ResponseHeaders, req.ResponseBody,
		req.Timestamp, req.DurationMs, req.ClientIP,
	)
	if err != nil {
		return fmt.Errorf("failed to save request log: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	req.ID = id

	return nil
}

func (r *RequestRepository) GetByID(id int64) (*RequestLog, error) {
	req := &RequestLog{}
	err := r.db.QueryRow(
		`SELECT id, vhost_id, method, url, request_headers, request_body, response_status, response_headers, response_body, timestamp, duration_ms, client_ip
		FROM request_logs WHERE id = ?`,
		id,
	).Scan(
		&req.ID, &req.VHostID, &req.Method, &req.URL, &req.RequestHeaders, &req.RequestBody,
		&req.ResponseStatus, &req.ResponseHeaders, &req.ResponseBody,
		&req.Timestamp, &req.DurationMs, &req.ClientIP,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get request log: %w", err)
	}

	return req, nil
}

func (r *RequestRepository) List(limit, offset int) ([]*RequestLog, error) {
	rows, err := r.db.Query(
		`SELECT id, vhost_id, method, url, request_headers, request_body, response_status, response_headers, response_body, timestamp, duration_ms, client_ip
		FROM request_logs ORDER BY timestamp DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list request logs: %w", err)
	}
	defer rows.Close()

	var logs []*RequestLog
	for rows.Next() {
		req := &RequestLog{}
		if err := rows.Scan(
			&req.ID, &req.VHostID, &req.Method, &req.URL, &req.RequestHeaders, &req.RequestBody,
			&req.ResponseStatus, &req.ResponseHeaders, &req.ResponseBody,
			&req.Timestamp, &req.DurationMs, &req.ClientIP,
		); err != nil {
			return nil, fmt.Errorf("failed to scan request log: %w", err)
		}
		logs = append(logs, req)
	}

	return logs, nil
}

func (r *RequestRepository) ListByVHostID(vhostID int64) ([]*RequestLog, error) {
	rows, err := r.db.Query(
		`SELECT id, vhost_id, method, url, request_headers, request_body, response_status, response_headers, response_body, timestamp, duration_ms, client_ip
		FROM request_logs WHERE vhost_id = ? ORDER BY timestamp ASC`,
		vhostID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list request logs by vhost: %w", err)
	}
	defer rows.Close()

	var logs []*RequestLog
	for rows.Next() {
		req := &RequestLog{}
		if err := rows.Scan(
			&req.ID, &req.VHostID, &req.Method, &req.URL, &req.RequestHeaders, &req.RequestBody,
			&req.ResponseStatus, &req.ResponseHeaders, &req.ResponseBody,
			&req.Timestamp, &req.DurationMs, &req.ClientIP,
		); err != nil {
			return nil, fmt.Errorf("failed to scan request log: %w", err)
		}
		logs = append(logs, req)
	}

	return logs, nil
}

func (r *RequestRepository) DeleteOlderThan(duration time.Duration) error {
	cutoff := time.Now().Add(-duration)
	_, err := r.db.Exec("DELETE FROM request_logs WHERE timestamp < ?", cutoff)
	if err != nil {
		return fmt.Errorf("failed to delete old request logs: %w", err)
	}
	return nil
}

func (r *RequestRepository) Count() (int64, error) {
	var count int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM request_logs").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count request logs: %w", err)
	}
	return count, nil
}
