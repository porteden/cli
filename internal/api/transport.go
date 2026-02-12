package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/porteden/cli/internal/config"
	"github.com/porteden/cli/internal/debug"
)

// Transport implements http.RoundTripper with automatic auth and logging
type Transport struct {
	Base   http.RoundTripper
	APIKey string
}

func NewTransport(apiKey string) *Transport {
	return &Transport{
		Base:   http.DefaultTransport,
		APIKey: apiKey,
	}
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+t.APIKey)

	// Add User-Agent header for version tracking
	// Format: PortEden-CLI/{version} ({os}; {arch})
	req.Header.Set("User-Agent", fmt.Sprintf("PortEden-CLI/%s (%s; %s)",
		config.Version, runtime.GOOS, runtime.GOARCH))

	// Add request ID for tracing
	requestID := randomHex(4)
	req.Header.Set("X-Request-ID", requestID)

	// Add content type if not set
	if req.Header.Get("Content-Type") == "" && req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Log request in verbose mode
	debug.LogRequest(req, requestID)
	start := time.Now()

	// Execute request
	resp, err := t.Base.RoundTrip(req)
	if err != nil {
		debug.Log("[%s] Request failed: %v", requestID, err)
		return nil, err
	}

	// Log response in verbose mode
	debug.LogResponse(resp, requestID, time.Since(start))

	return resp, nil
}

// NewHTTPClient creates an http.Client with the custom transport
func NewHTTPClient(apiKey string) *http.Client {
	return &http.Client{
		Transport: NewTransport(apiKey),
		Timeout:   30 * time.Second,
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
