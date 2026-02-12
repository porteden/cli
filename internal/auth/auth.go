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
	"time"

	"github.com/pkg/browser"
)

const (
	baseURL = "https://cliv1b.porteden.com"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

type LoginResponse struct {
	SessionToken string    `json:"sessionToken"`
	PollSecret   string    `json:"pollSecret"`
	LoginURL     string    `json:"loginUrl"`
	ExpiresAt    time.Time `json:"expiresAt"`
	Message      string    `json:"message"`
}

type PollResponse struct {
	Status string  `json:"status"`
	ApiKey *string `json:"apiKey,omitempty"`
	Error  *string `json:"error,omitempty"`
}

// LoginProgress reports login progress to the caller.
type LoginProgress struct {
	// OnBrowserOpen is called when the browser is about to open, with the fallback URL.
	OnBrowserOpen func(loginURL string)
	// OnWaiting is called when polling starts.
	OnWaiting func()
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
	reqBody := map[string]interface{}{}
	if operatorID != "" {
		reqBody["operatorId"] = operatorID
	}
	if keyTitle != "" {
		reqBody["keyTitle"] = keyTitle
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/auth/token/login", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not connect to PortEden. Please check your internet connection and try again")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read server response. Please try again")
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("too many login attempts. Please wait a minute and try again")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not start login session. Please try again later")
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// 2. Open browser
	if progress != nil && progress.OnBrowserOpen != nil {
		progress.OnBrowserOpen(loginResp.LoginURL)
	}
	_ = browser.OpenURL(loginResp.LoginURL)

	// 3. Poll for completion
	if progress != nil && progress.OnWaiting != nil {
		progress.OnWaiting()
	}
	apiKey, err := pollForCompletion(ctx, loginResp.SessionToken, loginResp.PollSecret, loginResp.ExpiresAt)
	if err != nil {
		return "", err
	}

	// 4. Store API key securely
	if err := StoreAPIKey(apiKey, profile); err != nil {
		return "", fmt.Errorf("failed to store API key: %w", err)
	}

	return apiKey, nil
}

func pollForCompletion(ctx context.Context, sessionToken, pollSecret string, expiresAt time.Time) (string, error) {
	// Build poll URL with proper encoding
	pollURL := fmt.Sprintf("%s/api/auth/token/poll/%s?secret=%s",
		baseURL,
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
			resp, err := httpClient.Get(pollURL)
			if err != nil {
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
