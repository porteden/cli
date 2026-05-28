package debug

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var Verbose bool

// maxBodyLogBytes caps the response body dump so a giant Gmail message body
// doesn't flood the terminal in verbose mode.
const maxBodyLogBytes = 4096

// Log prints debug messages when verbose mode is enabled
func Log(format string, args ...interface{}) {
	if Verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// LogRequest logs HTTP request details in verbose mode
// IMPORTANT: Authorization header is redacted for security
func LogRequest(req *http.Request, requestID string) {
	if !Verbose {
		return
	}

	Log("[%s] Request: %s %s", requestID, req.Method, req.URL.String())

	// Log headers with Authorization redacted
	for name, values := range req.Header {
		if strings.EqualFold(name, "Authorization") {
			Log("[%s]   Header: %s: [REDACTED]", requestID, name)
		} else {
			Log("[%s]   Header: %s: %s", requestID, name, strings.Join(values, ", "))
		}
	}
}

// LogResponse logs HTTP response details in verbose mode. For error
// responses (4xx/5xx) it also dumps the body — buffering it back onto
// resp.Body so downstream consumers (retry layer, apierr.ParseAPIError)
// see the same bytes. Without this, BE bug correlation is blind: we know
// the status but not which `errorCode` / `message` came back.
func LogResponse(resp *http.Response, requestID string, duration time.Duration) {
	if !Verbose {
		return
	}

	Log("[%s] Response: %s (took %v)", requestID, resp.Status, duration)

	// Log rate limit headers if present
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		Log("[%s] Rate limit remaining: %s", requestID, remaining)
	}

	if resp.StatusCode >= 400 && resp.Body != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			Log("[%s] Response body: <read error: %v>", requestID, err)
			return
		}
		// Restore the body for downstream consumers — they expect to read it.
		resp.Body = io.NopCloser(bytes.NewReader(body))

		if len(body) == 0 {
			Log("[%s] Response body: <empty>", requestID)
			return
		}
		if len(body) > maxBodyLogBytes {
			Log("[%s] Response body (%d bytes, truncated to %d): %s",
				requestID, len(body), maxBodyLogBytes, string(body[:maxBodyLogBytes]))
		} else {
			Log("[%s] Response body: %s", requestID, string(body))
		}
	}
}
