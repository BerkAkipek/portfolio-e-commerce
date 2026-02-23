package httpapi

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	db "github.com/BerkAkipek/e-commerce-app/api/internal/db"
)

func mustCreateJWTUtility(t *testing.T) *JWTUtility {
	t.Helper()
	util, err := NewJWTUtility("test-secret", time.Hour, "test-issuer")
	if err != nil {
		t.Fatalf("new jwt utility: %v", err)
	}
	return util
}

func mustTokenWithClaims(t *testing.T, util *JWTUtility, claims jwtClaims, alg string, typ string) string {
	t.Helper()
	headerJSON, err := json.Marshal(map[string]string{
		"alg": alg,
		"typ": typ,
	})
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := header + "." + payload
	return signingInput + "." + util.sign(signingInput)
}

func TestNewJWTUtilityValidation(t *testing.T) {
	if _, err := NewJWTUtility("", time.Hour, "issuer"); err == nil {
		t.Fatalf("expected empty secret to fail")
	}
	if _, err := NewJWTUtility("secret", 0, "issuer"); err == nil {
		t.Fatalf("expected non-positive ttl to fail")
	}
	if _, err := NewJWTUtility("secret", time.Hour, ""); err == nil {
		t.Fatalf("expected empty issuer to fail")
	}
}

func TestJWTUtilityCreateVerifyExtractUserID(t *testing.T) {
	util := mustCreateJWTUtility(t)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000201")
	token, err := util.CreateToken(userID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	if err := util.VerifyToken(token); err != nil {
		t.Fatalf("verify token: %v", err)
	}

	gotUserID, err := util.ExtractUserID(token)
	if err != nil {
		t.Fatalf("extract user id: %v", err)
	}
	if gotUserID != userID {
		t.Fatalf("expected user id %s, got %s", userID, gotUserID)
	}
}

func TestJWTUtilityVerifyRejectsTamperedToken(t *testing.T) {
	util := mustCreateJWTUtility(t)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000202")
	token, err := util.CreateToken(userID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	tampered := token + "x"
	if err := util.VerifyToken(tampered); err == nil {
		t.Fatalf("expected tampered token to fail verification")
	}
}

func TestJWTUtilityCreateTokenRejectsNilUserID(t *testing.T) {
	util := mustCreateJWTUtility(t)
	if _, err := util.CreateToken(uuid.Nil); err == nil {
		t.Fatalf("expected nil user id to fail")
	}
}

func TestJWTUtilityVerifyRejectsMalformedAndInvalidTokens(t *testing.T) {
	util := mustCreateJWTUtility(t)
	now := time.Now().UTC()

	tests := []struct {
		name           string
		token          string
		verifyShouldOK bool
	}{
		{
			name:           "missing parts",
			token:          "only-one-part",
			verifyShouldOK: false,
		},
		{
			name: "invalid alg",
			token: mustTokenWithClaims(t, util, jwtClaims{
				Sub: uuid.NewString(),
				Iat: now.Unix(),
				Exp: now.Add(time.Hour).Unix(),
				Iss: "test-issuer",
			}, "HS512", "JWT"),
			verifyShouldOK: false,
		},
		{
			name: "expired",
			token: mustTokenWithClaims(t, util, jwtClaims{
				Sub: uuid.NewString(),
				Iat: now.Add(-2 * time.Hour).Unix(),
				Exp: now.Add(-time.Hour).Unix(),
				Iss: "test-issuer",
			}, "HS256", "JWT"),
			verifyShouldOK: false,
		},
		{
			name: "missing sub",
			token: mustTokenWithClaims(t, util, jwtClaims{
				Sub: "",
				Iat: now.Unix(),
				Exp: now.Add(time.Hour).Unix(),
				Iss: "test-issuer",
			}, "HS256", "JWT"),
			verifyShouldOK: false,
		},
		{
			name: "invalid sub uuid",
			token: mustTokenWithClaims(t, util, jwtClaims{
				Sub: "not-a-uuid",
				Iat: now.Unix(),
				Exp: now.Add(time.Hour).Unix(),
				Iss: "test-issuer",
			}, "HS256", "JWT"),
			verifyShouldOK: true,
		},
		{
			name: "invalid issuer",
			token: mustTokenWithClaims(t, util, jwtClaims{
				Sub: uuid.NewString(),
				Iat: now.Unix(),
				Exp: now.Add(time.Hour).Unix(),
				Iss: "other-issuer",
			}, "HS256", "JWT"),
			verifyShouldOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := util.VerifyToken(tc.token)
			if tc.verifyShouldOK && err != nil {
				t.Fatalf("expected verify to pass, got %v", err)
			}
			if !tc.verifyShouldOK && err == nil {
				t.Fatalf("expected verify to fail")
			}
			if _, err := util.ExtractUserID(tc.token); err == nil {
				t.Fatalf("expected extract user id to fail")
			}
		})
	}
}

func TestJWTUtilityCreateTokenUsesOnlyRequiredClaims(t *testing.T) {
	util := mustCreateJWTUtility(t)
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000204")
	token, err := util.CreateToken(userID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 token parts, got %d", len(parts))
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if len(payload) != 4 {
		t.Fatalf("expected 4 claims, got %d", len(payload))
	}
	for _, key := range []string{"sub", "exp", "iat", "iss"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("expected claim %q to be present", key)
		}
	}
}

func TestAuthCookieHelpers(t *testing.T) {
	rr := httptest.NewRecorder()
	SetAuthCookie(rr, "jwt-token")
	resp := rr.Result()

	var authCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == authCookieName {
			authCookie = c
			break
		}
	}
	if authCookie == nil {
		t.Fatalf("expected auth cookie to be set")
	}
	if !authCookie.HttpOnly {
		t.Fatalf("expected auth cookie to be httpOnly")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(authCookie)
	value, err := ReadAuthCookie(req)
	if err != nil {
		t.Fatalf("read auth cookie: %v", err)
	}
	if value != "jwt-token" {
		t.Fatalf("expected jwt-token, got %q", value)
	}

	rr = httptest.NewRecorder()
	ClearAuthCookie(rr)
	resp = rr.Result()
	var cleared *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == authCookieName {
			cleared = c
			break
		}
	}
	if cleared == nil {
		t.Fatalf("expected cleared auth cookie to be set")
	}
	if cleared.MaxAge != -1 {
		t.Fatalf("expected cleared cookie max-age -1, got %d", cleared.MaxAge)
	}
}

func TestRefreshCookieHelpers(t *testing.T) {
	rr := httptest.NewRecorder()
	SetRefreshCookie(rr, "refresh-token", time.Hour)
	resp := rr.Result()

	var refreshCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == refreshCookieName {
			refreshCookie = c
			break
		}
	}
	if refreshCookie == nil {
		t.Fatalf("expected refresh cookie to be set")
	}
	if !refreshCookie.HttpOnly {
		t.Fatalf("expected refresh cookie to be httpOnly")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(refreshCookie)
	value, err := ReadRefreshCookie(req)
	if err != nil {
		t.Fatalf("read refresh cookie: %v", err)
	}
	if value != "refresh-token" {
		t.Fatalf("expected refresh-token, got %q", value)
	}

	rr = httptest.NewRecorder()
	ClearRefreshCookie(rr)
	resp = rr.Result()
	var cleared *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == refreshCookieName {
			cleared = c
			break
		}
	}
	if cleared == nil {
		t.Fatalf("expected cleared refresh cookie to be set")
	}
	if cleared.MaxAge != -1 {
		t.Fatalf("expected cleared refresh cookie max-age -1, got %d", cleared.MaxAge)
	}
}

func TestReadRefreshCookieErrors(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, err := ReadRefreshCookie(req); err == nil {
		t.Fatalf("expected missing refresh cookie to fail")
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "   "})
	if _, err := ReadRefreshCookie(req); err == nil {
		t.Fatalf("expected blank refresh cookie to fail")
	}
}

func TestGenerateAndHashRefreshToken(t *testing.T) {
	token, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty refresh token")
	}

	token2, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("generate second refresh token: %v", err)
	}
	if token == token2 {
		t.Fatalf("expected refresh tokens to be random and unique")
	}

	hash1, err := HashRefreshToken(token)
	if err != nil {
		t.Fatalf("hash refresh token: %v", err)
	}
	if len(hash1) != 64 {
		t.Fatalf("expected sha256 hex length 64, got %d", len(hash1))
	}
	hash2, err := HashRefreshToken(token)
	if err != nil {
		t.Fatalf("hash refresh token second time: %v", err)
	}
	if hash1 != hash2 {
		t.Fatalf("expected deterministic hash for same token")
	}
	if _, err := HashRefreshToken(" "); err == nil {
		t.Fatalf("expected blank refresh token hashing to fail")
	}
}

func TestReadAuthCookieErrors(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, err := ReadAuthCookie(req); err == nil {
		t.Fatalf("expected missing cookie to fail")
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: "   "})
	if _, err := ReadAuthCookie(req); err == nil {
		t.Fatalf("expected blank auth cookie to fail")
	}
}

func TestSetAndClearAuthCookies(t *testing.T) {
	rr := httptest.NewRecorder()
	SetAuthCookies(rr, "access-1", 15*time.Minute, "refresh-1", 30*24*time.Hour)
	resp := rr.Result()

	var accessCookie *http.Cookie
	var refreshCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == authCookieName {
			accessCookie = c
		}
		if c.Name == refreshCookieName {
			refreshCookie = c
		}
	}
	if accessCookie == nil || refreshCookie == nil {
		t.Fatalf("expected both access and refresh cookies to be set")
	}
	if !accessCookie.HttpOnly || !refreshCookie.HttpOnly {
		t.Fatalf("expected auth cookies to be httpOnly")
	}
	if accessCookie.Path != "/" || refreshCookie.Path != "/" {
		t.Fatalf("expected auth cookie path to be /")
	}
	if accessCookie.SameSite != http.SameSiteLaxMode || refreshCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected auth cookies to use SameSite=Lax")
	}

	rr = httptest.NewRecorder()
	ClearAuthCookies(rr)
	resp = rr.Result()
	var clearedAccess bool
	var clearedRefresh bool
	for _, c := range resp.Cookies() {
		if c.Name == authCookieName && c.MaxAge == -1 {
			clearedAccess = true
		}
		if c.Name == refreshCookieName && c.MaxAge == -1 {
			clearedRefresh = true
		}
	}
	if !clearedAccess || !clearedRefresh {
		t.Fatalf("expected both auth cookies to be cleared")
	}
}

func TestReadCookieUtility(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "sample", Value: "value-1"})

	got, err := ReadCookie(req, "sample")
	if err != nil {
		t.Fatalf("read cookie: %v", err)
	}
	if got != "value-1" {
		t.Fatalf("expected value-1, got %q", got)
	}

	if _, err := ReadCookie(req, "missing"); err == nil {
		t.Fatalf("expected missing cookie to fail")
	}
}

func TestGuestSessionCookieHelpers(t *testing.T) {
	sessionID := GenerateGuestSessionID()
	if _, err := uuid.Parse(sessionID); err != nil {
		t.Fatalf("generated session id is not a valid uuid: %v", err)
	}

	rr := httptest.NewRecorder()
	SetGuestSessionCookie(rr, sessionID)
	resp := rr.Result()

	var guestCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == guestSessionCookieName {
			guestCookie = c
			break
		}
	}
	if guestCookie == nil {
		t.Fatalf("expected guest session cookie to be set")
	}
	if !guestCookie.HttpOnly {
		t.Fatalf("expected guest session cookie to be httpOnly")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(guestCookie)
	value, err := ReadGuestSessionCookie(req)
	if err != nil {
		t.Fatalf("read guest session cookie: %v", err)
	}
	if value != sessionID {
		t.Fatalf("expected %s, got %s", sessionID, value)
	}
}

func TestReadGuestSessionCookieErrors(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, err := ReadGuestSessionCookie(req); err == nil {
		t.Fatalf("expected missing guest cookie to fail")
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: guestSessionCookieName, Value: "   "})
	if _, err := ReadGuestSessionCookie(req); err == nil {
		t.Fatalf("expected blank guest cookie to fail")
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: guestSessionCookieName, Value: "not-a-uuid"})
	if _, err := ReadGuestSessionCookie(req); err == nil {
		t.Fatalf("expected invalid uuid guest cookie to fail")
	}
}

func TestResolveActorUsesUserFromAuthCookie(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")

	server := NewServer(&fakeProductQuerier{})
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000203")
	token, err := server.jwt.CreateToken(userID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	handler := server.resolveActor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := ActorFromContext(r.Context())
		if !ok {
			t.Fatalf("expected actor in context")
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"actor_type": string(actor.ActorType),
			"user_id":    actor.UserID.String(),
			"session_id": actor.SessionID,
		})
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["actor_type"] != string(ActorTypeUser) {
		t.Fatalf("expected actor_type user, got %q", body["actor_type"])
	}
	if body["user_id"] != userID.String() {
		t.Fatalf("expected user id %s, got %s", userID, body["user_id"])
	}
	if body["session_id"] != "" {
		t.Fatalf("expected empty session id, got %q", body["session_id"])
	}
}

func TestResolveActorFallsBackToSessionAndSetsCookie(t *testing.T) {
	server := NewServer(&fakeProductQuerier{})

	handler := server.resolveActor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := ActorFromContext(r.Context())
		if !ok {
			t.Fatalf("expected actor in context")
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"actor_type": string(actor.ActorType),
			"user_id":    actor.UserID.String(),
			"session_id": actor.SessionID,
		})
	}))

	req := httptest.NewRequest(http.MethodGet, "/cart/00000000-0000-0000-0000-000000000401/items", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["actor_type"] != string(ActorTypeSession) {
		t.Fatalf("expected actor_type session, got %q", body["actor_type"])
	}
	if body["user_id"] != uuid.Nil.String() {
		t.Fatalf("expected empty user id, got %q", body["user_id"])
	}
	if _, err := uuid.Parse(body["session_id"]); err != nil {
		t.Fatalf("expected valid session id, got %q", body["session_id"])
	}

	rawSetCookie := rr.Header().Get("Set-Cookie")
	if !strings.Contains(rawSetCookie, guestSessionCookieName+"=") {
		t.Fatalf("expected guest session cookie to be set, got %q", rawSetCookie)
	}
}

func TestResolveActorUsesExistingSessionCookie(t *testing.T) {
	server := NewServer(&fakeProductQuerier{})
	existingSessionID := uuid.NewString()

	handler := server.resolveActor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := ActorFromContext(r.Context())
		if !ok {
			t.Fatalf("expected actor in context")
		}
		if actor.ActorType != ActorTypeSession {
			t.Fatalf("expected session actor, got %s", actor.ActorType)
		}
		if actor.SessionID != existingSessionID {
			t.Fatalf("expected session id %s, got %s", existingSessionID, actor.SessionID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/cart/00000000-0000-0000-0000-000000000402/items", nil)
	req.AddCookie(&http.Cookie{Name: guestSessionCookieName, Value: existingSessionID})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if strings.Contains(rr.Header().Get("Set-Cookie"), guestSessionCookieName+"=") {
		t.Fatalf("did not expect guest session cookie to be reset")
	}
}

func TestResolveActorInvalidAuthFallsBackToSession(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	server := NewServer(&fakeProductQuerier{})

	handler := server.resolveActor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := ActorFromContext(r.Context())
		if !ok {
			t.Fatalf("expected actor in context")
		}
		if actor.ActorType != ActorTypeSession {
			t.Fatalf("expected session actor, got %s", actor.ActorType)
		}
		if _, err := uuid.Parse(actor.SessionID); err != nil {
			t.Fatalf("expected generated session id, got %q", actor.SessionID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/cart/00000000-0000-0000-0000-000000000403/items", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: "invalid.jwt.token"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	rawSetCookie := rr.Header().Values("Set-Cookie")
	joined := strings.Join(rawSetCookie, ";")
	if !strings.Contains(joined, guestSessionCookieName+"=") {
		t.Fatalf("expected guest session cookie to be set")
	}
}

func TestResolveActorPublicRouteSetsGuestCookieWhenUnauthenticated(t *testing.T) {
	server := NewServer(&fakeProductQuerier{})

	handler := server.resolveActor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, ok := ActorFromContext(r.Context())
		if !ok {
			t.Fatalf("expected actor in context")
		}
		if actor.ActorType != ActorTypeSession {
			t.Fatalf("expected session actor, got %s", actor.ActorType)
		}
		if _, err := uuid.Parse(actor.SessionID); err != nil {
			t.Fatalf("expected valid session id for public route, got %q", actor.SessionID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Set-Cookie"), guestSessionCookieName+"=") {
		t.Fatalf("expected guest session cookie on public route")
	}
}

func TestRequireAuthSetsUserIDInContext(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	server := NewServer(&fakeProductQuerier{})
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000205")
	token, err := server.jwt.CreateToken(userID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	handler := server.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, ok := AuthUserIDFromContext(r.Context())
		if !ok {
			t.Fatalf("expected user id in context")
		}
		if ctxUserID != userID {
			t.Fatalf("expected user id %s, got %s", userID, ctxUserID)
		}
		if !IsAuthenticatedFromContext(r.Context()) {
			t.Fatalf("expected isAuthenticated=true in context")
		}
		authCtx, ok := AuthContextFromContext(r.Context())
		if !ok {
			t.Fatalf("expected full auth context in request context")
		}
		if !authCtx.IsAuthenticated || authCtx.UserID != userID {
			t.Fatalf("unexpected auth context: %+v", authCtx)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/checkout/session", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}

func TestRequireAuthRejectsMissingOrInvalidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	server := NewServer(&fakeProductQuerier{})

	handler := server.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/checkout/session", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing token status 401, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/checkout/session", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: "bad.token.value"})
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid token status 401, got %d", rr.Code)
	}
}

func TestRequireAuthRefreshesExpiredAccessTokenAndContinues(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	server := NewServer(&fakeProductQuerier{})
	store := server.products.(*fakeProductQuerier)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000206")
	expiredAccess := mustTokenWithClaims(t, server.jwt, jwtClaims{
		Sub: userID.String(),
		Iat: time.Now().UTC().Add(-2 * time.Hour).Unix(),
		Exp: time.Now().UTC().Add(-time.Hour).Unix(),
		Iss: server.jwt.issuer,
	}, "HS256", "JWT")

	refreshRaw := "refresh-raw-token-xyz"
	refreshHash, err := HashRefreshToken(refreshRaw)
	if err != nil {
		t.Fatalf("hash refresh token: %v", err)
	}
	store.refreshLookup = db.RefreshToken{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000207"),
		UserID:    userID,
		TokenHash: refreshHash,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now().UTC(),
	}
	store.rotatedRefreshToken = db.RefreshToken{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000299"),
		UserID:    userID,
		TokenHash: "new-hash",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		Revoked:   false,
		CreatedAt: time.Now().UTC(),
	}

	handler := server.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxUserID, ok := AuthUserIDFromContext(r.Context())
		if !ok {
			t.Fatalf("expected user id in context")
		}
		if ctxUserID != userID {
			t.Fatalf("expected user id %s, got %s", userID, ctxUserID)
		}
		if !IsAuthenticatedFromContext(r.Context()) {
			t.Fatalf("expected isAuthenticated=true in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/checkout/session", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: expiredAccess})
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: refreshRaw})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if store.lastRotateOldRefreshTokenHash != refreshHash {
		t.Fatalf("expected refresh rotate old hash %s, got %s", refreshHash, store.lastRotateOldRefreshTokenHash)
	}
	if store.lastRotateNewRefreshTokenHash == "" {
		t.Fatalf("expected refresh rotate new hash to be set")
	}
	if !store.lastRotateRefreshExpiresAt.After(time.Now().UTC()) {
		t.Fatalf("expected refresh rotate expiry in future")
	}

	var hasAccessCookie bool
	var hasRefreshCookie bool
	for _, c := range rr.Result().Cookies() {
		if c.Name == authCookieName && c.Value != "" {
			hasAccessCookie = true
		}
		if c.Name == refreshCookieName && c.Value != "" {
			hasRefreshCookie = true
		}
	}
	if !hasAccessCookie || !hasRefreshCookie {
		t.Fatalf("expected rotated auth cookies to be set")
	}
}

func TestRequireAuthRejectsExpiredAccessWithInvalidRefresh(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	server := NewServer(&fakeProductQuerier{rotateRefreshErr: sql.ErrNoRows})

	expiredAccess := mustTokenWithClaims(t, server.jwt, jwtClaims{
		Sub: uuid.MustParse("00000000-0000-0000-0000-000000000208").String(),
		Iat: time.Now().UTC().Add(-2 * time.Hour).Unix(),
		Exp: time.Now().UTC().Add(-time.Hour).Unix(),
		Iss: server.jwt.issuer,
	}, "HS256", "JWT")

	handler := server.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/checkout/session", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: expiredAccess})
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "refresh-raw-token-invalid"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}
