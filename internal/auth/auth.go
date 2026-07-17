package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/pkg/browser"
)

const defaultBaseURL = "https://cliv1b.porteden.com"

// baseURL honours PE_API_URL, same as api.NewClient. Without it the login flow can only ever be
// pointed at production, which is also why it had no tests.
func baseURL() string {
	if envURL := os.Getenv("PE_API_URL"); envURL != "" {
		return envURL
	}
	return defaultBaseURL
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

type LoginResponse struct {
	SessionToken string    `json:"sessionToken"`
	PollSecret   string    `json:"pollSecret"`
	LoginURL     string    `json:"loginUrl"`
	ExpiresAt    time.Time `json:"expiresAt"`
	Message      string    `json:"message"`

	// UserCode is the device-flow user code (RFC 8628 §3.3). The user types it in the browser to
	// confirm they're authorizing this terminal. Present only when we advertised supportsUserCode.
	//
	// Display it, and nothing more: it must reach the browser by the user reading it here and typing
	// it in. Never put it in a URL, and don't log it.
	UserCode string `json:"userCode"`
}

type PollResponse struct {
	Status string  `json:"status"`
	ApiKey *string `json:"apiKey,omitempty"`
	Error  *string `json:"error,omitempty"`
}

// LoginProgress reports login progress to the caller.
type LoginProgress struct {
	// OnUserCode is called with the code the user must type in the browser, before the browser is
	// opened. Only called when the server issued one. Implementations MUST display it — a user who
	// never sees it cannot complete the login.
	OnUserCode func(code string)
	// OnBrowserOpen is called when the browser is about to open, with the fallback URL.
	OnBrowserOpen func(loginURL string)
	// OnWaiting is called when polling starts.
	OnWaiting func()
	// OnServerMessage is called with the server's advisory message, if any. Used to reach CLI
	// versions we can't otherwise update (e.g. deprecation notices), so it should be shown.
	OnServerMessage func(message string)
}

// Login authenticates via browser and stores the API key for the given profile.
// If progress is nil, no progress messages are printed.
func Login(profile, operatorID, keyTitle string, progress *LoginProgress) (string, error) {
	if profile == "" {
		profile = "default"
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// 1. Initiate login session
	loginResp, err := initiateLogin(ctx, operatorID, keyTitle)
	if err != nil {
		return "", err
	}

	if progress != nil && progress.OnServerMessage != nil && loginResp.Message != "" {
		progress.OnServerMessage(loginResp.Message)
	}

	// 2. Show the user code, THEN open the browser — opening it first steals focus and the code
	// scrolls out of sight behind the window.
	if progress != nil && progress.OnUserCode != nil && loginResp.UserCode != "" {
		progress.OnUserCode(loginResp.UserCode)
	}

	// 3. Open browser
	if progress != nil && progress.OnBrowserOpen != nil {
		progress.OnBrowserOpen(loginResp.LoginURL)
	}
	_ = browser.OpenURL(loginResp.LoginURL)

	// 4. Poll for completion
	if progress != nil && progress.OnWaiting != nil {
		progress.OnWaiting()
	}
	apiKey, err := pollForCompletion(ctx, loginResp.SessionToken, loginResp.PollSecret, loginResp.ExpiresAt)
	if err != nil {
		return "", err
	}

	// 5. Store API key securely
	if err := StoreAPIKey(apiKey, profile); err != nil {
		return "", fmt.Errorf("failed to store API key: %w", err)
	}

	return apiKey, nil
}

// initiateLogin creates the login session and returns the server's response.
//
// Split out of Login so it can be tested without opening a browser or touching the credential store.
func initiateLogin(ctx context.Context, operatorID, keyTitle string) (*LoginResponse, error) {
	reqBody := map[string]interface{}{
		// We can display a user code, so the server issues one and requires it back before releasing
		// the key (RFC 8628 device authorization grant).
		"supportsUserCode": true,
	}
	if operatorID != "" {
		reqBody["operatorId"] = operatorID
	}
	if keyTitle != "" {
		reqBody["keyTitle"] = keyTitle
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL()+"/api/auth/token/login", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not connect to PortEden. Please check your internet connection and try again")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read server response. Please try again")
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("too many login attempts. Please wait a minute and try again")
	}
	if resp.StatusCode != http.StatusOK {
		// Prefer the server's explanation — it knows things we can't infer, and "try again later"
		// sends the user in circles when the answer was in the body all along.
		if msg := serverErrorMessage(body); msg != "" {
			return nil, errors.New(msg)
		}
		return nil, fmt.Errorf("could not start login session. Please try again later")
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &loginResp, nil
}

// serverErrorMessage extracts a human-readable message from an error body. The API uses a couple of
// shapes ({"message":…} from ApiError, {"error":…} from ad-hoc handlers), so try both rather than
// couple to one and silently fall back to a useless generic string.
func serverErrorMessage(body []byte) string {
	var payload struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if msg := strings.TrimSpace(payload.Message); msg != "" {
		return msg
	}
	return strings.TrimSpace(payload.Error)
}

func pollForCompletion(ctx context.Context, sessionToken, pollSecret string, expiresAt time.Time) (string, error) {
	// Build poll URL with proper encoding
	pollURL := fmt.Sprintf("%s/api/auth/token/poll/%s?secret=%s",
		baseURL(),
		url.PathEscape(sessionToken),
		url.QueryEscape(pollSecret))

	// Use server expiry or 90s minimum, whichever is longer
	timeout := time.Until(expiresAt)
	if timeout < 90*time.Second {
		timeout = 90 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// Initial delay — user needs time to login/signup in the browser
	initialDelay := time.NewTimer(10 * time.Second)
	defer initialDelay.Stop()
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("login cancelled by user")
	case <-timer.C:
		return "", fmt.Errorf("login timed out")
	case <-initialDelay.C:
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("login cancelled by user")
		case <-timer.C:
			return "", fmt.Errorf("login timed out")
		case <-ticker.C:
			// Bind each poll to ctx so an in-flight request is aborted the
			// instant the user hits Ctrl-C, instead of blocking up to the
			// client's 30s timeout (and swallowing repeated SIGINT).
			req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
			if reqErr != nil {
				return "", reqErr
			}
			resp, err := httpClient.Do(req)
			if err != nil {
				if ctx.Err() != nil {
					return "", fmt.Errorf("login cancelled by user")
				}
				continue // Retry on network errors
			}

			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()

			if readErr != nil {
				continue
			}

			if resp.StatusCode != http.StatusOK {
				if resp.StatusCode == http.StatusNotFound {
					return "", fmt.Errorf("login session expired. Please try again")
				}
				if resp.StatusCode == http.StatusTooManyRequests {
					return "", fmt.Errorf("too many login attempts. Please wait a minute and try again")
				}
				if resp.StatusCode >= 500 {
					continue // Retry server errors
				}
				if resp.StatusCode == http.StatusBadRequest {
					continue // Retry bad requests (transient)
				}
				continue
			}

			var pollResp PollResponse
			if err := json.Unmarshal(body, &pollResp); err != nil {
				continue
			}

			switch pollResp.Status {
			case "completed":
				if pollResp.ApiKey != nil {
					return *pollResp.ApiKey, nil
				}
				return "", fmt.Errorf("no API key in response")
			case "expired":
				return "", fmt.Errorf("login session expired")
			case "failed":
				msg := "authentication failed"
				if pollResp.Error != nil {
					msg = *pollResp.Error
				}
				return "", errors.New(msg)
			case "invalid_secret":
				return "", fmt.Errorf("invalid poll secret - session may be compromised")
			}
		}
	}
}
