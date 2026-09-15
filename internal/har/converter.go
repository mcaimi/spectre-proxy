package har

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mcaimi/spectre-proxy/internal/storage"
)

func FromRequestLogs(logs []*storage.RequestLog) (*HAR, error) {
	entries := make([]Entry, 0, len(logs))

	for _, log := range logs {
		entry, err := convertEntry(log)
		if err != nil {
			return nil, fmt.Errorf("failed to convert log %d: %w", log.ID, err)
		}
		entries = append(entries, entry)
	}

	return &HAR{
		Log: Log{
			Version: "1.2",
			Creator: Creator{
				Name:    "spectre-proxy",
				Version: "1.0.0",
			},
			Entries: entries,
		},
	}, nil
}

func convertEntry(log *storage.RequestLog) (Entry, error) {
	reqHeaders, err := convertHeaders(log.RequestHeaders)
	if err != nil {
		return Entry{}, fmt.Errorf("request headers: %w", err)
	}

	respHeaders, err := convertHeaders(log.ResponseHeaders)
	if err != nil {
		return Entry{}, fmt.Errorf("response headers: %w", err)
	}

	queryString := buildQueryString(log.URL)

	req := Request{
		Method:      log.Method,
		URL:         log.URL,
		HTTPVersion: "HTTP/1.1",
		Cookies:     []Cookie{},
		Headers:     reqHeaders,
		QueryString: queryString,
		HeadersSize: -1,
		BodySize:    len(log.RequestBody),
	}

	if len(log.RequestBody) > 0 {
		mimeType := extractContentType(log.RequestHeaders)
		req.PostData = &PostData{
			MimeType: mimeType,
			Text:     encodeBody(log.RequestBody),
		}
	}

	respMimeType := extractContentType(log.ResponseHeaders)
	content := Content{
		Size:     len(log.ResponseBody),
		MimeType: respMimeType,
	}
	if len(log.ResponseBody) > 0 {
		if utf8.Valid(log.ResponseBody) {
			content.Text = string(log.ResponseBody)
		} else {
			content.Text = base64.StdEncoding.EncodeToString(log.ResponseBody)
			content.Encoding = "base64"
		}
	}

	statusText := http.StatusText(log.ResponseStatus)

	resp := Response{
		Status:      log.ResponseStatus,
		StatusText:  statusText,
		HTTPVersion: "HTTP/1.1",
		Cookies:     []Cookie{},
		Headers:     respHeaders,
		Content:     content,
		RedirectURL: "",
		HeadersSize: -1,
		BodySize:    len(log.ResponseBody),
	}

	durationMs := float64(log.DurationMs)

	return Entry{
		StartedDateTime: log.Timestamp.Format(time.RFC3339Nano),
		Time:            durationMs,
		Request:         req,
		Response:        resp,
		Cache:           Cache{},
		Timings: Timings{
			Send:    0,
			Wait:    durationMs,
			Receive: 0,
		},
	}, nil
}

func convertHeaders(headersJSON string) ([]NameValue, error) {
	if headersJSON == "" {
		return []NameValue{}, nil
	}

	var headerMap map[string][]string
	if err := json.Unmarshal([]byte(headersJSON), &headerMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
	}

	var result []NameValue
	for name, values := range headerMap {
		for _, value := range values {
			result = append(result, NameValue{Name: name, Value: value})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].Value < result[j].Value
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}

func buildQueryString(rawURL string) []NameValue {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return []NameValue{}
	}

	var result []NameValue
	for name, values := range parsed.Query() {
		for _, value := range values {
			result = append(result, NameValue{Name: name, Value: value})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

func extractContentType(headersJSON string) string {
	if headersJSON == "" {
		return "application/octet-stream"
	}

	var headerMap map[string][]string
	if err := json.Unmarshal([]byte(headersJSON), &headerMap); err != nil {
		return "application/octet-stream"
	}

	for name, values := range headerMap {
		if strings.EqualFold(name, "content-type") && len(values) > 0 {
			return values[0]
		}
	}

	return "application/octet-stream"
}

func encodeBody(body []byte) string {
	if utf8.Valid(body) {
		return string(body)
	}
	return base64.StdEncoding.EncodeToString(body)
}
