package proxy

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

func (p *Proxy) handleHTTPSConnection(clientConn net.Conn) {
	defer clientConn.Close()

	tlsConn, ok := clientConn.(*tls.Conn)
	if !ok {
		p.log.Error("Not a TLS connection")
		return
	}

	if err := tlsConn.Handshake(); err != nil {
		p.log.Debugf("TLS handshake failed: %v", err)
		return
	}

	state := tlsConn.ConnectionState()
	hostname := state.ServerName

	if hostname == "" {
		p.log.Debug("No SNI hostname in TLS connection")
		return
	}

	vhost, ok := p.router.Match(hostname)
	if !ok {
		p.log.Debugf("No virtual host configured for %s", hostname)
		return
	}

	p.log.Debugf("Proxying HTTPS request for %s to %s", hostname, vhost.TargetURL)

	reader := bufio.NewReader(tlsConn)

	for {
		tlsConn.SetReadDeadline(time.Now().Add(p.config.ReadTimeout))

		req, err := http.ReadRequest(reader)
		if err != nil {
			if err != io.EOF {
				p.log.Debugf("Failed to read HTTPS request: %v", err)
			}
			return
		}

		start := time.Now()
		reqCapture, respCapture, err := p.forwardHTTPSRequest(req, vhost.TargetURL, tlsConn)
		duration := time.Since(start)

		if err != nil {
			p.log.Errorf("Failed to forward HTTPS request: %v", err)
			return
		}

		go p.logRequest(reqCapture, respCapture, vhost.ID, duration, getClientIP(clientConn))

		if req.Close || req.Header.Get("Connection") == "close" {
			return
		}
	}
}

func (p *Proxy) forwardHTTPSRequest(req *http.Request, targetURL string, clientConn *tls.Conn) (*RequestCapture, *ResponseCapture, error) {
	parsedTarget, err := url.Parse(targetURL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid target URL: %w", err)
	}

	backendReq := req.Clone(req.Context())
	backendReq.RequestURI = ""

	if parsedTarget.Scheme == "https" {
		backendReq.URL.Scheme = "https"
	} else {
		backendReq.URL.Scheme = "http"
	}
	backendReq.URL.Host = parsedTarget.Host

	reqCapture := captureRequest(req)

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
	}

	resp, err := client.Do(backendReq)
	if err != nil {
		return reqCapture, nil, fmt.Errorf("backend request failed: %w", err)
	}
	defer resp.Body.Close()

	respCapture := captureResponse(resp)

	if err := resp.Write(clientConn); err != nil {
		return reqCapture, respCapture, fmt.Errorf("failed to write response: %w", err)
	}

	return reqCapture, respCapture, nil
}
