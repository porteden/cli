package apierr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError represents an error response from the API
type APIError struct {
	StatusCode   int    `json:"-"`
	Code         string `json:"code,omitempty"`         // Backend error code (ACCESS_DENIED, NOT_FOUND, etc.)
	ErrorMessage string `json:"error,omitempty"`        // Legacy error field
	Message      string `json:"message,omitempty"`      // Detailed error message
	Details      string `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.ErrorMessage
}

// ParseAPIError extracts error details from an HTTP response.
// NOTE: This function does NOT close resp.Body - caller is responsible for closing.
// This allows the caller to use defer resp.Body.Close() consistently.
func ParseAPIError(resp *http.Response) *APIError {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			StatusCode:   resp.StatusCode,
			ErrorMessage: fmt.Sprintf("HTTP %d", resp.StatusCode),
		}
	}

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return &APIError{
			StatusCode:   resp.StatusCode,
			ErrorMessage: string(body),
		}
	}

	apiErr.StatusCode = resp.StatusCode
	return &apiErr
}

// UserFriendlyError converts API errors to user-friendly messages
func UserFriendlyError(err *APIError) string {
	switch err.StatusCode {
	case 401:
		return "Not authenticated. Run 'porteden auth login' to authenticate."
	case 403:
		return "Access denied. You don't have permission for this operation."
	case 404:
		return "Not found. The requested resource doesn't exist."
	case 429:
		return "Rate limited. Please wait a moment and try again."
	case 500, 502, 503:
		return "Server error. Please try again later."
	default:
		if err.Message != "" {
			return err.Message
		}
		if err.ErrorMessage != "" {
			return err.ErrorMessage
		}
		return fmt.Sprintf("Request failed with status %d", err.StatusCode)
	}
}
