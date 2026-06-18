package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"codeberg.org/mcaimi/spectre-proxy/internal/storage"
)

type RequestCapture struct {
	Method  string
	URL     string
	Headers map[string][]string
	Body    []byte
}

type ResponseCapture struct {
	Status  int
	Headers map[string][]string
	Body    []byte
}

const maxBodySize = 1024 * 1024

func captureRequest(req *http.Request) *RequestCapture {
	capture := &RequestCapture{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: make(map[string][]string),
	}

	for key, values := range req.Header {
		capture.Headers[key] = values
	}

	if req.Body != nil {
		bodyBytes, err := io.ReadAll(io.LimitReader(req.Body, maxBodySize))
		if err == nil {
			capture.Body = bodyBytes
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	}

	return capture
}

func captureResponse(resp *http.Response) *ResponseCapture {
	capture := &ResponseCapture{
		Status:  resp.StatusCode,
		Headers: make(map[string][]string),
	}

	for key, values := range resp.Header {
		capture.Headers[key] = values
	}

	if resp.Body != nil {
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
		if err == nil {
			capture.Body = bodyBytes
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	}

	return capture
}

func (p *Proxy) logRequest(reqCapture *RequestCapture, respCapture *ResponseCapture, vhostID int64, duration time.Duration, clientIP string) {
	reqHeadersJSON, _ := json.Marshal(reqCapture.Headers)
	respHeadersJSON, _ := json.Marshal(respCapture.Headers)

	log := &storage.RequestLog{
		VHostID:         &vhostID,
		Method:          reqCapture.Method,
		URL:             reqCapture.URL,
		RequestHeaders:  string(reqHeadersJSON),
		RequestBody:     reqCapture.Body,
		ResponseStatus:  respCapture.Status,
		ResponseHeaders: string(respHeadersJSON),
		ResponseBody:    respCapture.Body,
		Timestamp:       time.Now(),
		DurationMs:      duration.Milliseconds(),
		ClientIP:        clientIP,
	}

	if err := p.requestRepo.Save(log); err != nil {
		p.log.Errorf("Failed to save request log: %v", err)
	}
}
