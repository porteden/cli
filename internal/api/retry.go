package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/porteden/cli/internal/debug"
)

const (
	maxRetries     = 3
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
)

// isRetryable checks if the response status code is retryable
func isRetryable(statusCode int) bool {
	switch statusCode {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

// getRetryAfter parses the Retry-After header
func getRetryAfter(resp *http.Response) time.Duration {
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}

	// Try parsing as seconds
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP date
	if t, err := http.ParseTime(retryAfter); err == nil {
		return time.Until(t)
	}

	return 0
}

// doWithRetry executes a request with automatic retries for transient errors
// IMPORTANT: Accept []byte instead of io.Reader - io.Reader is consumed on first attempt
// and subsequent retries would send empty bodies!
func (c *Client) doWithRetry(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			debug.Log("Retry attempt %d/%d after %v", attempt, maxRetries, backoff)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		// Create fresh reader for each attempt
		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
		if err != nil {
			return nil, err
		}

		// Content-Type set here; Authorization handled by Transport
		req.Header.Set("Content-Type", "application/json")

		// Note: Transport handles Authorization and logging via RoundTrip
		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Network errors are retryable
			lastErr = err
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// Success or non-retryable error
		if !isRetryable(resp.StatusCode) {
			return resp, nil
		}

		// Retryable error - close body and prepare for retry
		resp.Body.Close()
		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)

		// Respect Retry-After header if present
		if retryAfter := getRetryAfter(resp); retryAfter > 0 {
			backoff = min(retryAfter, maxBackoff)
		} else {
			backoff = min(backoff*2, maxBackoff)
		}
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}
