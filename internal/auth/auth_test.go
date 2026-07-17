package auth

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// serveLogin stands up a fake /api/auth/token/login and points the client at it. It hands back the
// request body the client sent, so tests can assert on what we ask the server for.
func serveLogin(t *testing.T, status int, responseBody string) *map[string]interface{} {
	t.Helper()

	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/token/login" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, responseBody)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("PE_API_URL", srv.URL)

	return &got
}

func TestInitiateLogin_AdvertisesUserCodeSupport(t *testing.T) {
	sent := serveLogin(t, http.StatusOK, `{"sessionToken":"s","pollSecret":"p","loginUrl":"https://example/cli-auth?session=s"}`)

	if _, err := initiateLogin(context.Background(), "", ""); err != nil {
		t.Fatalf("initiateLogin: %v", err)
	}

	// Without this the server can't issue a code, and once the cutoff is on this client is refused.
	if (*sent)["supportsUserCode"] != true {
		t.Fatalf("supportsUserCode not advertised; sent %v", *sent)
	}
}

func TestInitiateLogin_ReturnsUserCode(t *testing.T) {
	serveLogin(t, http.StatusOK,
		`{"sessionToken":"s","pollSecret":"p","loginUrl":"https://example/cli-auth?session=s","userCode":"WDJB-MJHT"}`)

	resp, err := initiateLogin(context.Background(), "", "")
	if err != nil {
		t.Fatalf("initiateLogin: %v", err)
	}

	if resp.UserCode != "WDJB-MJHT" {
		t.Fatalf("UserCode = %q, want WDJB-MJHT", resp.UserCode)
	}
	// The code must reach the browser by the user typing it, never by riding along in the URL.
	if strings.Contains(resp.LoginURL, resp.UserCode) {
		t.Fatalf("login URL carries the user code: %q", resp.LoginURL)
	}
}

func TestInitiateLogin_OmittedUserCodeIsEmptyNotAnError(t *testing.T) {
	// Older servers don't return one. That's a normal login, not a failure.
	serveLogin(t, http.StatusOK, `{"sessionToken":"s","pollSecret":"p","loginUrl":"https://example/x"}`)

	resp, err := initiateLogin(context.Background(), "", "")
	if err != nil {
		t.Fatalf("initiateLogin: %v", err)
	}
	if resp.UserCode != "" {
		t.Fatalf("UserCode = %q, want empty", resp.UserCode)
	}
}

func TestInitiateLogin_SurfacesServerErrorMessage(t *testing.T) {
	// The server uses this to tell the user something only it knows — e.g. that this build can no
	// longer sign in and they should run `porteden update`. Swallowing it strands them.
	serveLogin(t, http.StatusBadRequest,
		`{"statusCode":400,"message":"Your PortEden CLI is out of date. Run 'porteden update'."}`)

	_, err := initiateLogin(context.Background(), "", "")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "porteden update") {
		t.Fatalf("server message not surfaced; got %q", err)
	}
}

func TestInitiateLogin_FallsBackWhenErrorBodyIsUnhelpful(t *testing.T) {
	serveLogin(t, http.StatusInternalServerError, `<html>502 upstream</html>`)

	_, err := initiateLogin(context.Background(), "", "")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "could not start login session") {
		t.Fatalf("want generic fallback, got %q", err)
	}
}

func TestInitiateLogin_RateLimitKeepsItsOwnMessage(t *testing.T) {
	serveLogin(t, http.StatusTooManyRequests, `{"message":"slow down"}`)

	_, err := initiateLogin(context.Background(), "", "")
	if err == nil || !strings.Contains(err.Error(), "too many login attempts") {
		t.Fatalf("want rate-limit message, got %v", err)
	}
}

func TestInitiateLogin_PassesThroughOptionalFields(t *testing.T) {
	sent := serveLogin(t, http.StatusOK, `{"sessionToken":"s","pollSecret":"p","loginUrl":"u"}`)

	if _, err := initiateLogin(context.Background(), "op-1", "my key"); err != nil {
		t.Fatalf("initiateLogin: %v", err)
	}
	if (*sent)["operatorId"] != "op-1" || (*sent)["keyTitle"] != "my key" {
		t.Fatalf("optional fields not sent: %v", *sent)
	}
}

func TestInitiateLogin_OmitsEmptyOptionalFields(t *testing.T) {
	sent := serveLogin(t, http.StatusOK, `{"sessionToken":"s","pollSecret":"p","loginUrl":"u"}`)

	if _, err := initiateLogin(context.Background(), "", ""); err != nil {
		t.Fatalf("initiateLogin: %v", err)
	}
	if _, ok := (*sent)["operatorId"]; ok {
		t.Error("operatorId sent when empty")
	}
	if _, ok := (*sent)["keyTitle"]; ok {
		t.Error("keyTitle sent when empty")
	}
}

func TestServerErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"ApiError shape", `{"statusCode":400,"message":"nope"}`, "nope"},
		{"ad-hoc error shape", `{"error":"nope"}`, "nope"},
		{"message wins over error", `{"error":"generic","message":"specific"}`, "specific"},
		{"blank message falls back to error", `{"error":"nope","message":"   "}`, "nope"},
		{"not json", `<html>`, ""},
		{"empty", ``, ""},
		{"json without either field", `{"foo":"bar"}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := serverErrorMessage([]byte(tt.body)); got != tt.want {
				t.Fatalf("serverErrorMessage(%s) = %q, want %q", tt.body, got, tt.want)
			}
		})
	}
}

func TestBaseURL_HonoursEnvOverride(t *testing.T) {
	t.Setenv("PE_API_URL", "https://staging.example")
	if got := baseURL(); got != "https://staging.example" {
		t.Fatalf("baseURL() = %q", got)
	}

	t.Setenv("PE_API_URL", "")
	if got := baseURL(); got != defaultBaseURL {
		t.Fatalf("baseURL() = %q, want default", got)
	}
}
