package httpapi

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"

	db "github.com/BerkAkipek/e-commerce-app/api/internal/db"
)

func TestGoogleAuthStartRedirectsToProvider(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_OAUTH_REDIRECT_URL", "http://localhost:3000/api/auth/google/callback")

	server := NewServer(&fakeProductQuerier{})
	req := httptest.NewRequest(http.MethodGet, "/auth/google/start", nil)
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("expected status 302, got %d", rr.Code)
	}

	location := rr.Header().Get("Location")
	if location == "" {
		t.Fatalf("expected redirect location")
	}
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}
	if parsed.Host != "accounts.google.com" {
		t.Fatalf("expected google host, got %s", parsed.Host)
	}
	query := parsed.Query()
	if query.Get("client_id") != "client-id" {
		t.Fatalf("expected client id in query")
	}
	if query.Get("response_type") != "code" {
		t.Fatalf("expected code flow response_type")
	}
	if strings.TrimSpace(query.Get("state")) == "" {
		t.Fatalf("expected oauth state to be set")
	}
	hasStateCookie := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == googleOAuthStateCookieName && c.Value != "" {
			hasStateCookie = true
			break
		}
	}
	if !hasStateCookie {
		t.Fatalf("expected oauth state cookie to be set")
	}
}

func TestGoogleAuthStartRequiresConfiguration(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")

	server := NewServer(&fakeProductQuerier{})
	req := httptest.NewRequest(http.MethodGet, "/auth/google/start", nil)
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["error"] != "google oauth is not configured" {
		t.Fatalf("expected config error, got %+v", body)
	}
}

func TestGoogleAuthCallbackCreatesUserAndIssuesSession(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_OAUTH_REDIRECT_URL", "http://localhost:3000/api/auth/google/callback")
	t.Setenv("GOOGLE_OAUTH_POST_LOGIN_URL", "/catalog")
	t.Setenv("GOOGLE_OAUTH_AUTH_URL", "http://invalid.test/auth")

	t.Setenv("GOOGLE_OAUTH_TOKEN_URL", "https://oauth.example.test/token")
	t.Setenv("GOOGLE_OAUTH_USERINFO_URL", "https://oauth.example.test/userinfo")

	store := &fakeProductQuerier{
		userByEmailErr: sql.ErrNoRows,
		createdUser:    dbUser("00000000-0000-0000-0000-000000000991", "google-user@example.com"),
	}
	server := NewServer(store)
	server.http = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() == "https://oauth.example.test/token" {
				body, _ := io.ReadAll(req.Body)
				if !strings.Contains(string(body), "grant_type=authorization_code") {
					t.Fatalf("expected authorization_code grant, got body: %s", string(body))
				}
				return jsonHTTPResponse(http.StatusOK, map[string]any{
					"access_token": "google-access-token",
					"token_type":   "Bearer",
				}), nil
			}
			if req.URL.String() == "https://oauth.example.test/userinfo" {
				if got := req.Header.Get("Authorization"); got != "Bearer google-access-token" {
					t.Fatalf("expected bearer token, got %q", got)
				}
				return jsonHTTPResponse(http.StatusOK, map[string]any{
					"email":          "google-user@example.com",
					"email_verified": true,
					"name":           "Google User",
				}), nil
			}
			t.Fatalf("unexpected oauth request url: %s", req.URL.String())
			return nil, nil
		}),
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=test-code&state=state-1", nil)
	req.AddCookie(&http.Cookie{Name: googleOAuthStateCookieName, Value: "state-1"})
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("expected status 302, got %d", rr.Code)
	}
	if rr.Header().Get("Location") != "/catalog" {
		t.Fatalf("expected redirect to /catalog, got %s", rr.Header().Get("Location"))
	}
	if store.lastCreateUserArg.Email != "google-user@example.com" {
		t.Fatalf("expected user creation for google email, got %s", store.lastCreateUserArg.Email)
	}
	if store.lastCreateRefreshTokenArg.UserID == uuid.Nil {
		t.Fatalf("expected refresh token to be issued")
	}

	setCookies := strings.Join(rr.Header().Values("Set-Cookie"), "\n")
	if !strings.Contains(setCookies, authCookieName+"=") {
		t.Fatalf("expected auth cookie to be set")
	}
	if !strings.Contains(setCookies, refreshCookieName+"=") {
		t.Fatalf("expected refresh cookie to be set")
	}
}

func TestGoogleAuthCallbackRejectsStateMismatch(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	t.Setenv("GOOGLE_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_OAUTH_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_OAUTH_REDIRECT_URL", "http://localhost:3000/api/auth/google/callback")

	server := NewServer(&fakeProductQuerier{})

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=test-code&state=state-2", nil)
	req.AddCookie(&http.Cookie{Name: googleOAuthStateCookieName, Value: "state-1"})
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("expected status 302, got %d", rr.Code)
	}
	if rr.Header().Get("Location") != "/auth?error=google_state" {
		t.Fatalf("expected state error redirect, got %s", rr.Header().Get("Location"))
	}
}

func dbUser(id string, email string) db.User {
	return db.User{
		ID:    uuid.MustParse(id),
		Email: email,
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonHTTPResponse(status int, payload map[string]any) *http.Response {
	body, _ := json.Marshal(payload)
	return &http.Response{
		StatusCode: status,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(string(body))),
	}
}
