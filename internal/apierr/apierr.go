package apierr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// APIError represents an error response from the API.
// The backend uses two shapes:
//  1. {code, message}                     — most endpoints
//  2. {success:false, errorCode, errorMessage} — send/reply/forward
//
// GET on a single email/thread may also return 404 with {accessInfo}
// to signal a policy denial (vs. genuinely missing).
type APIError struct {
	StatusCode         int      `json:"-"`
	Code               string   `json:"code,omitempty"`
	ErrorCode          string   `json:"errorCode,omitempty"`
	ErrorMessage       string   `json:"errorMessage,omitempty"`
	LegacyError        string   `json:"error,omitempty"`
	Message            string   `json:"message,omitempty"`
	AccessInfo         string   `json:"accessInfo,omitempty"`
	Details            string   `json:"details,omitempty"`
	ConnectedProviders []string `json:"connectedProviders,omitempty"`
}

// EffectiveCode returns the backend error code from whichever shape the
// response used. Empty when neither field is populated.
func (e *APIError) EffectiveCode() string {
	if e.Code != "" {
		return e.Code
	}
	return e.ErrorCode
}

// EffectiveMessage returns the user-facing description from whichever shape
// the response used. The 422 shape carries it in `message`; the
// send/reply/forward/modify/delete shape carries it in `errorMessage`; the
// legacy Drive 404 shape (`{ "error": "..." }`) carries it in `error`. All
// three are user-formatted and include my.porteden.com deep links / actionable
// guidance — prefer any of them over our own generic fallbacks.
func (e *APIError) EffectiveMessage() string {
	if e.Message != "" {
		return e.Message
	}
	if e.ErrorMessage != "" {
		return e.ErrorMessage
	}
	return e.LegacyError
}

func (e *APIError) Error() string {
	switch {
	case e.Message != "":
		return e.Message
	case e.ErrorMessage != "":
		return e.ErrorMessage
	case e.AccessInfo != "":
		return e.AccessInfo
	case e.LegacyError != "":
		return e.LegacyError
	}
	return fmt.Sprintf("HTTP %d", e.StatusCode)
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

// UserFriendlyError converts API errors to user-friendly messages.
// Backend codes win over status codes — they encode the precise denial
// reason (policy block vs. provider rejection vs. token gate).
func UserFriendlyError(err *APIError) string {
	// The backend's message strings are already user-facing and include
	// my.porteden.com deep links — prefer them verbatim when available.
	switch err.EffectiveCode() {
	case "EMAIL_NOT_ENABLED":
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "Email access is not enabled for this token. Enable it at https://my.porteden.com."
	case "NO_EMAIL_PROVIDER":
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "No mailbox connected. Connect Gmail or Outlook at https://my.porteden.com."
	case "ACCESS_RESTRICTED":
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "Blocked by policy: a participant on this message is on a block rule. Update the recipient list or block rules at https://my.porteden.com."
	case "BLOCKED":
		if err.AccessInfo != "" {
			return err.AccessInfo
		}
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "This resource is hidden by an active policy rule."
	case "OPERATION_NOT_ALLOWED":
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "The required operation flag is not enabled on this token. Enable it at https://my.porteden.com."
	case "PERMISSION_DENIED":
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "The connected mailbox lacks the upstream permission for this operation. Reconnect at https://my.porteden.com."
	case "INVALID_REQUEST", "VALIDATION_ERROR":
		// VALIDATION_ERROR is the controller-mapping alias; treat the two
		// codes identically. Backend always sends a user-formatted sentence.
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "Invalid request."
	case "NOT_FOUND":
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "Not found. The requested resource doesn't exist."
	case "AUTH_FAILED":
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "Not authenticated. Run 'porteden auth login' to authenticate."
	case "ACCESS_DENIED":
		// Board out of token scope, or assignee blocked by people/domain rules.
		// AccessInfo carries the backend's deep-linked guidance — prefer it.
		if err.AccessInfo != "" {
			return err.AccessInfo
		}
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "Access denied. The board may be out of token scope, or an access rule is blocking the item."
	case "INVALID_INPUT":
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		if err.AccessInfo != "" {
			return err.AccessInfo
		}
		return "Invalid input."
	case "RATE_LIMIT_EXCEEDED", "RATE_LIMITED":
		// RATE_LIMITED is the per-token sliding-window code; RATE_LIMIT_EXCEEDED
		// is the IP-bucket alias. Same remediation: back off (retry layer
		// honours Retry-After automatically).
		if err.AccessInfo != "" {
			return err.AccessInfo
		}
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "Rate limited. Please wait a moment and try again."
	case "QUOTA_EXCEEDED":
		// Monthly per-token cap. Carries a user-formatted message with a
		// my.porteden.com upgrade link plus a `usage` envelope on the wire.
		// Distinct from RATE_LIMITED — this is account-level, not transient.
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "Monthly API quota exceeded. Upgrade or wait for the quota reset at https://my.porteden.com."
	case "PROVIDER_ERROR":
		// Upstream Gmail / Microsoft Graph returned an error the firewall
		// couldn't promote to a typed code. Per spec the message is now a
		// user-formatted sentence (the raw upstream JSON lives in BE logs,
		// keyed by X-Request-ID). Do not retry — this is not a transient class.
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "The upstream provider returned an error. Check the connection status at https://my.porteden.com."
	case "TASK_NOT_ENABLED", "TASKS_NOT_ENABLED":
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "Task management is not enabled for this token. Enable it at https://my.porteden.com → Access Tokens → Task Management."
	case "NO_TASK_CONNECTION":
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "No task provider connected. Connect Monday/Asana/Jira/Linear/Notion at https://my.porteden.com → Connections."
	case "PROVIDER_REQUIRED":
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		if len(err.ConnectedProviders) > 0 {
			return "Multiple task providers connected; specify --provider. Connected: " + strings.Join(err.ConnectedProviders, ", ") + "."
		}
		return "Multiple task providers connected; specify --provider."
	case "BLOCKS_NOT_SUPPORTED":
		if err.AccessInfo != "" {
			return err.AccessInfo
		}
		return "Page-body blocks are only supported on Notion. Other providers don't have an equivalent surface."
	case "NO_WRITABLE_FIELDS":
		if err.AccessInfo != "" {
			return err.AccessInfo
		}
		return "All fields were rejected by the token's writability mask. Check token configuration at https://my.porteden.com."
	case "OPERATION_DENIED", "POLICY_DENIED":
		if err.AccessInfo != "" {
			return err.AccessInfo
		}
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "Operation denied by policy. Check token rules at https://my.porteden.com."
	case "CONNECTION_REVOKED":
		if err.AccessInfo != "" {
			return err.AccessInfo
		}
		return "Provider connection was revoked. Reconnect at https://my.porteden.com → Connections."
	}

	switch err.StatusCode {
	case 400:
		// Adapter validation rejections (BE returns e.g. errorCode=INVALID_FIELD_VALUE
		// with the actual upstream complaint in accessInfo: "body failed validation:
		// body.properties.Due.date.start should be a valid ISO 8601 date string…").
		// Prefer accessInfo over EffectiveMessage — it's the deepest, most specific
		// diagnostic the BE has plumbed through.
		if err.AccessInfo != "" {
			return err.AccessInfo
		}
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "Bad request — check the input format and values."
	case 401:
		return "Not authenticated. Run 'porteden auth login' to authenticate."
	case 403:
		// Backend messages are already user-facing — prefer them over the generic
		// fallback so the caller sees which flag/scope is missing.
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "Access denied. You don't have permission for this operation."
	case 404:
		// A populated accessInfo on 404 signals policy denial, not missing.
		if err.AccessInfo != "" {
			return err.AccessInfo
		}
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "Not found. The requested resource doesn't exist."
	case 422:
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "Request rejected. Check token configuration at https://my.porteden.com."
	case 429:
		return "Rate limited. Please wait a moment and try again."
	case 500, 502, 503:
		// Adapter errors (Notion / Monday / etc. rejecting the upstream call)
		// come back as 5xx with the actual provider message threaded through
		// accessInfo / message / errorMessage. Surface it instead of the
		// generic "server error" so the user sees what to fix
		// (e.g. "Property 'Due date' does not exist on database…").
		if err.AccessInfo != "" {
			return err.AccessInfo
		}
		if msg := err.EffectiveMessage(); msg != "" {
			return msg
		}
		return "Server error. Please try again later."
	}

	if msg := err.EffectiveMessage(); msg != "" {
		return msg
	}
	// Defence-in-depth: any future error code or status we haven't enumerated
	// still gets the backend's accessInfo (when present) instead of "Request
	// failed with status N". accessInfo is always user-formatted upstream.
	if err.AccessInfo != "" {
		return err.AccessInfo
	}
	if err.LegacyError != "" {
		return err.LegacyError
	}
	return fmt.Sprintf("Request failed with status %d", err.StatusCode)
}
