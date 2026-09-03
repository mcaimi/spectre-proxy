package har

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/mcaimi/spectre-proxy/internal/storage"
)

func TestFromRequestLogs_Empty(t *testing.T) {
	h, err := FromRequestLogs([]*storage.RequestLog{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.Log.Version != "1.2" {
		t.Errorf("expected version 1.2, got %s", h.Log.Version)
	}
	if len(h.Log.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(h.Log.Entries))
	}
}

func TestFromRequestLogs_SingleGET(t *testing.T) {
	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	vhostID := int64(1)
	logs := []*storage.RequestLog{
		{
			ID:              1,
			VHostID:         &vhostID,
			Method:          "GET",
			URL:             "https://example.com/api/users",
			RequestHeaders:  `{"Accept":["application/json"],"Host":["example.com"]}`,
			ResponseStatus:  200,
			ResponseHeaders: `{"Content-Type":["application/json"]}`,
			ResponseBody:    []byte(`{"users":[]}`),
			Timestamp:       ts,
			DurationMs:      45,
			ClientIP:        "10.0.0.1",
		},
	}

	h, err := FromRequestLogs(logs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(h.Log.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(h.Log.Entries))
	}

	entry := h.Log.Entries[0]
	if entry.Request.Method != "GET" {
		t.Errorf("expected method GET, got %s", entry.Request.Method)
	}
	if entry.Request.URL != "https://example.com/api/users" {
		t.Errorf("expected URL https://example.com/api/users, got %s", entry.Request.URL)
	}
	if entry.Time != 45 {
		t.Errorf("expected time 45, got %f", entry.Time)
	}
	if entry.Response.Status != 200 {
		t.Errorf("expected status 200, got %d", entry.Response.Status)
	}
	if entry.Response.StatusText != "OK" {
		t.Errorf("expected status text OK, got %s", entry.Response.StatusText)
	}
	if entry.Response.Content.Text != `{"users":[]}` {
		t.Errorf("unexpected response content text: %s", entry.Response.Content.Text)
	}
	if entry.Request.PostData != nil {
		t.Error("GET request should not have PostData")
	}
}

func TestFromRequestLogs_POSTWithBody(t *testing.T) {
	vhostID := int64(1)
	logs := []*storage.RequestLog{
		{
			ID:              1,
			VHostID:         &vhostID,
			Method:          "POST",
			URL:             "https://example.com/api/users",
			RequestHeaders:  `{"Content-Type":["application/json"]}`,
			RequestBody:     []byte(`{"name":"Alice"}`),
			ResponseStatus:  201,
			ResponseHeaders: `{"Content-Type":["application/json"]}`,
			ResponseBody:    []byte(`{"id":1,"name":"Alice"}`),
			Timestamp:       time.Now(),
			DurationMs:      100,
			ClientIP:        "10.0.0.1",
		},
	}

	h, err := FromRequestLogs(logs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry := h.Log.Entries[0]
	if entry.Request.PostData == nil {
		t.Fatal("POST request should have PostData")
	}
	if entry.Request.PostData.MimeType != "application/json" {
		t.Errorf("expected MIME type application/json, got %s", entry.Request.PostData.MimeType)
	}
	if entry.Request.PostData.Text != `{"name":"Alice"}` {
		t.Errorf("unexpected PostData text: %s", entry.Request.PostData.Text)
	}
	if entry.Request.BodySize != len(`{"name":"Alice"}`) {
		t.Errorf("expected body size %d, got %d", len(`{"name":"Alice"}`), entry.Request.BodySize)
	}
}

func TestFromRequestLogs_BinaryResponseBody(t *testing.T) {
	vhostID := int64(1)
	binaryBody := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10}
	logs := []*storage.RequestLog{
		{
			ID:              1,
			VHostID:         &vhostID,
			Method:          "GET",
			URL:             "https://example.com/image.jpg",
			RequestHeaders:  `{}`,
			ResponseStatus:  200,
			ResponseHeaders: `{"Content-Type":["image/jpeg"]}`,
			ResponseBody:    binaryBody,
			Timestamp:       time.Now(),
			DurationMs:      200,
			ClientIP:        "10.0.0.1",
		},
	}

	h, err := FromRequestLogs(logs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := h.Log.Entries[0].Response.Content
	if content.Encoding != "base64" {
		t.Errorf("expected encoding base64, got %s", content.Encoding)
	}
	if content.MimeType != "image/jpeg" {
		t.Errorf("expected MIME type image/jpeg, got %s", content.MimeType)
	}

	decoded, err := base64.StdEncoding.DecodeString(content.Text)
	if err != nil {
		t.Fatalf("failed to decode base64 content: %v", err)
	}
	if len(decoded) != len(binaryBody) {
		t.Errorf("expected decoded length %d, got %d", len(binaryBody), len(decoded))
	}
}

func TestFromRequestLogs_QueryString(t *testing.T) {
	vhostID := int64(1)
	logs := []*storage.RequestLog{
		{
			ID:              1,
			VHostID:         &vhostID,
			Method:          "GET",
			URL:             "https://example.com/search?q=test&page=2&limit=10",
			RequestHeaders:  `{}`,
			ResponseStatus:  200,
			ResponseHeaders: `{}`,
			Timestamp:       time.Now(),
			DurationMs:      30,
			ClientIP:        "10.0.0.1",
		},
	}

	h, err := FromRequestLogs(logs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	qs := h.Log.Entries[0].Request.QueryString
	if len(qs) != 3 {
		t.Fatalf("expected 3 query parameters, got %d", len(qs))
	}

	qsMap := make(map[string]string)
	for _, nv := range qs {
		qsMap[nv.Name] = nv.Value
	}
	if qsMap["q"] != "test" {
		t.Errorf("expected q=test, got q=%s", qsMap["q"])
	}
	if qsMap["page"] != "2" {
		t.Errorf("expected page=2, got page=%s", qsMap["page"])
	}
}

func TestConvertHeaders(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectCount int
		expectErr   bool
	}{
		{
			name:        "empty string",
			input:       "",
			expectCount: 0,
		},
		{
			name:        "single header",
			input:       `{"Content-Type":["application/json"]}`,
			expectCount: 1,
		},
		{
			name:        "multi-valued header",
			input:       `{"Accept":["text/html","application/json"]}`,
			expectCount: 2,
		},
		{
			name:        "multiple headers",
			input:       `{"Content-Type":["application/json"],"Accept":["*/*"],"Host":["example.com"]}`,
			expectCount: 3,
		},
		{
			name:      "invalid JSON",
			input:     `not-json`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertHeaders(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != tt.expectCount {
				t.Errorf("expected %d headers, got %d", tt.expectCount, len(result))
			}
		})
	}
}

func TestExtractContentType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard Content-Type",
			input:    `{"Content-Type":["application/json"]}`,
			expected: "application/json",
		},
		{
			name:     "lowercase content-type",
			input:    `{"content-type":["text/html"]}`,
			expected: "text/html",
		},
		{
			name:     "missing Content-Type",
			input:    `{"Accept":["*/*"]}`,
			expected: "application/octet-stream",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "application/octet-stream",
		},
		{
			name:     "invalid JSON",
			input:    "not-json",
			expected: "application/octet-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractContentType(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestFromRequestLogs_NilVHostID(t *testing.T) {
	logs := []*storage.RequestLog{
		{
			ID:              1,
			VHostID:         nil,
			Method:          "GET",
			URL:             "https://example.com/",
			RequestHeaders:  `{}`,
			ResponseStatus:  200,
			ResponseHeaders: `{}`,
			Timestamp:       time.Now(),
			DurationMs:      10,
			ClientIP:        "127.0.0.1",
		},
	}

	h, err := FromRequestLogs(logs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(h.Log.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(h.Log.Entries))
	}
}
