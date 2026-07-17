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

// isIdempotent reports whether re-sending the method is safe. POST and PATCH
// are side-effecting (send email, create event, append rows, post comment): a
// blind retry after an ambiguous failure — a dropped connection or a gateway
// 5xx that the origin may already have acted on — can duplicate the operation.
// 429 is handled separately: the server explicitly refused the request, so it
// was not processed and is safe to retry regardless of method.
func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
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
			// A transport error is ambiguous for a non-idempotent method — the
			// request may have reached and been acted on by the origin before
			// the connection failed. Don't silently re-send a POST/PATCH and
			// duplicate the side effect; surface the error to the caller.
			if !isIdempotent(method) {
				return nil, err
			}
			// Network errors are retryable for idempotent methods.
			lastErr = err
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		// Success or non-retryable error
		if !isRetryable(resp.StatusCode) {
			return resp, nil
		}

		// A 429 with a monthly-quota body is not transient — its Retry-After
		// points at the billing-period reset, not a bucket refill. Peek the
		// (small) body, restore it for the caller's error parsing, and bail
		// out instead of burning retries that cannot succeed.
		if resp.StatusCode == 429 {
			quotaBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
			resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(quotaBody))
			if bytes.Contains(quotaBody, []byte("QUOTA_EXCEEDED")) {
				return resp, nil
			}
		}

		// A 5xx is ambiguous for a non-idempotent method: a gateway 502/504 can
		// arrive after the origin already processed the write. Retry only 429
		// (demonstrably not processed) for POST/PATCH; return the 5xx as-is so
		// the caller sees the error instead of us duplicating the operation.
		if resp.StatusCode != 429 && !isIdempotent(method) {
			return resp, nil
		}

		// Last attempt — return the response even though it's retryable so the
		// caller can parse the body for accessInfo / errorMessage. Otherwise
		// we'd swallow the backend's diagnostic on retry exhaustion and surface
		// only "HTTP 502", which is the opposite of what the user needs.
		if attempt == maxRetries {
			return resp, nil
		}

		// Mid-loop retry: discard body and prepare for next attempt
		resp.Body.Close()
		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)

		// Respect Retry-After header if present
		if retryAfter := getRetryAfter(resp); retryAfter > 0 {
			backoff = min(retryAfter, maxBackoff)
		} else {
			backoff = min(backoff*2, maxBackoff)
		}
	}

	// All attempts hit a network error (no response). The retryable-status
	// case is handled inside the loop above.
	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}
