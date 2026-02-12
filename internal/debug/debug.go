package debug

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

var Verbose bool

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

// LogResponse logs HTTP response details in verbose mode
func LogResponse(resp *http.Response, requestID string, duration time.Duration) {
	if !Verbose {
		return
	}

	Log("[%s] Response: %s (took %v)", requestID, resp.Status, duration)

	// Log rate limit headers if present
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		Log("[%s] Rate limit remaining: %s", requestID, remaining)
	}
}
