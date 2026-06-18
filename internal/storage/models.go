package storage

import "time"

type RootCA struct {
	ID          int64     `json:"id"`
	Certificate []byte    `json:"-"`
	PrivateKey  []byte    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	IsActive    bool      `json:"is_active"`
}

type VirtualHost struct {
	ID         int64     `json:"id"`
	Hostname   string    `json:"hostname"`
	TargetURL  string    `json:"target_url"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Certificate struct {
	ID          int64     `json:"id"`
	Hostname    string    `json:"hostname"`
	Certificate []byte    `json:"-"`
	PrivateKey  []byte    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	IsActive    bool      `json:"is_active"`
}

type RequestLog struct {
	ID              int64     `json:"id"`
	VHostID         *int64    `json:"vhost_id"`
	Method          string    `json:"method"`
	URL             string    `json:"url"`
	RequestHeaders  string    `json:"request_headers"`
	RequestBody     []byte    `json:"request_body"`
	ResponseStatus  int       `json:"status_code"`
	ResponseHeaders string    `json:"response_headers"`
	ResponseBody    []byte    `json:"response_body"`
	Timestamp       time.Time `json:"timestamp"`
	DurationMs      int64     `json:"duration_ms"`
	ClientIP        string    `json:"client_ip"`
}
