package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

func (p *Proxy) handleHTTPConnection(clientConn net.Conn) {
	defer clientConn.Close()

	clientConn.SetReadDeadline(time.Now().Add(p.config.ReadTimeout))
	clientConn.SetWriteDeadline(time.Now().Add(p.config.WriteTimeout))

	reader := bufio.NewReader(clientConn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		p.log.Debugf("Failed to read HTTP request: %v", err)
		return
	}

	hostname := req.Host
	if hostname == "" {
		p.log.Debug("No Host header in HTTP request")
		http.Error(newResponseWriter(clientConn), "Bad Request: No Host header", http.StatusBadRequest)
		return
	}

	vhost, ok := p.router.Match(hostname)
	if !ok {
		p.log.Debugf("No virtual host configured for %s", hostname)
		http.Error(newResponseWriter(clientConn), "Not Found", http.StatusNotFound)
		return
	}

	p.log.Debugf("Proxying HTTP request for %s to %s", hostname, vhost.TargetURL)

	start := time.Now()
	reqCapture, respCapture, err := p.forwardHTTPRequest(req, vhost.TargetURL, clientConn)
	duration := time.Since(start)

	if err != nil {
		p.log.Errorf("Failed to forward HTTP request: %v", err)
		http.Error(newResponseWriter(clientConn), "Bad Gateway", http.StatusBadGateway)
		return
	}

	go p.logRequest(reqCapture, respCapture, vhost.ID, duration, getClientIP(clientConn))
}

func (p *Proxy) forwardHTTPRequest(req *http.Request, targetURL string, clientConn net.Conn) (*RequestCapture, *ResponseCapture, error) {
	backendReq := req.Clone(req.Context())
	backendReq.RequestURI = ""
	backendReq.URL.Scheme = "http"
	backendReq.URL.Host = targetURL

	reqCapture := captureRequest(req)

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(backendReq)
	if err != nil {
		return reqCapture, nil, fmt.Errorf("backend request failed: %w", err)
	}
	defer resp.Body.Close()

	respCapture := captureResponse(resp)

	for key, values := range resp.Header {
		for _, value := range values {
			clientConn.Write([]byte(fmt.Sprintf("%s: %s\r\n", key, value)))
		}
	}
	clientConn.Write([]byte("\r\n"))

	io.Copy(clientConn, resp.Body)

	return reqCapture, respCapture, nil
}

type responseWriter struct {
	conn net.Conn
}

func newResponseWriter(conn net.Conn) *responseWriter {
	return &responseWriter{conn: conn}
}

func (rw *responseWriter) Header() http.Header {
	return make(http.Header)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.conn.Write(b)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.conn.Write([]byte(fmt.Sprintf("HTTP/1.1 %d %s\r\n\r\n", statusCode, http.StatusText(statusCode))))
}

func getClientIP(conn net.Conn) string {
	if addr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		return addr.IP.String()
	}
	return conn.RemoteAddr().String()
}
