package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	db "github.com/BerkAkipek/e-commerce-app/api/internal/db"
)

const (
	authCookieName         = "access_token"
	guestSessionCookieName = "guest_session_id"
	refreshCookieName      = "refresh_token"
)

type ActorType string

const (
	ActorTypeUser    ActorType = "user"
	ActorTypeSession ActorType = "session"
)

type Actor struct {
	ActorType ActorType
	UserID    uuid.UUID
	SessionID string
}

type actorContextKey struct{}
type authContextKey struct{}

type AuthContext struct {
	UserID          uuid.UUID
	IsAuthenticated bool
}

var (
	errInvalidJWTToken = errors.New("invalid jwt token")
	errInvalidJWTAlg   = errors.New("invalid jwt algorithm")
	errInvalidJWTIss   = errors.New("invalid jwt issuer")
	errExpiredJWTToken = errors.New("jwt token expired")
)

type jwtClaims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
	Iss string `json:"iss"`
}

type JWTUtility struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewJWTUtility(secret string, ttl time.Duration, issuer string) (*JWTUtility, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, errors.New("jwt secret cannot be empty")
	}
	if ttl <= 0 {
		return nil, errors.New("jwt ttl must be positive")
	}
	issuer = strings.TrimSpace(issuer)
	if issuer == "" {
		return nil, errors.New("jwt issuer cannot be empty")
	}
	return &JWTUtility{
		secret: []byte(secret),
		ttl:    ttl,
		issuer: issuer,
	}, nil
}

func (j *JWTUtility) CreateToken(userID uuid.UUID) (string, error) {
	if userID == uuid.Nil {
		return "", errors.New("user id cannot be empty")
	}

	headerJSON, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	claimsJSON, err := json.Marshal(jwtClaims{
		Sub: userID.String(),
		Iat: now.Unix(),
		Exp: now.Add(j.ttl).Unix(),
		Iss: j.issuer,
	})
	if err != nil {
		return "", err
	}

	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	signingInput := header + "." + payload
	signature := j.sign(signingInput)

	return signingInput + "." + signature, nil
}

func (j *JWTUtility) VerifyToken(token string) error {
	_, err := j.parseAndVerify(token)
	return err
}

func (j *JWTUtility) ExtractUserID(token string) (uuid.UUID, error) {
	claims, err := j.parseAndVerify(token)
	if err != nil {
		return uuid.Nil, err
	}
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		return uuid.Nil, errInvalidJWTToken
	}
	return userID, nil
}

func (j *JWTUtility) parseAndVerify(token string) (jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtClaims{}, errInvalidJWTToken
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return jwtClaims{}, errInvalidJWTToken
	}

	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return jwtClaims{}, errInvalidJWTToken
	}
	if header.Alg != "HS256" || header.Typ != "JWT" {
		return jwtClaims{}, errInvalidJWTAlg
	}

	signingInput := parts[0] + "." + parts[1]
	expectedSig := j.sign(signingInput)
	if !hmac.Equal([]byte(expectedSig), []byte(parts[2])) {
		return jwtClaims{}, errInvalidJWTToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtClaims{}, errInvalidJWTToken
	}

	var claimsMap map[string]json.RawMessage
	if err := json.Unmarshal(payloadBytes, &claimsMap); err != nil {
		return jwtClaims{}, errInvalidJWTToken
	}
	if len(claimsMap) != 4 {
		return jwtClaims{}, errInvalidJWTToken
	}
	if _, ok := claimsMap["sub"]; !ok {
		return jwtClaims{}, errInvalidJWTToken
	}
	if _, ok := claimsMap["exp"]; !ok {
		return jwtClaims{}, errInvalidJWTToken
	}
	if _, ok := claimsMap["iat"]; !ok {
		return jwtClaims{}, errInvalidJWTToken
	}
	if _, ok := claimsMap["iss"]; !ok {
		return jwtClaims{}, errInvalidJWTToken
	}

	var claims jwtClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return jwtClaims{}, errInvalidJWTToken
	}
	if claims.Sub == "" || claims.Exp <= 0 {
		return jwtClaims{}, errInvalidJWTToken
	}
	if claims.Iss != j.issuer {
		return jwtClaims{}, errInvalidJWTIss
	}
	if time.Now().UTC().Unix() >= claims.Exp {
		return jwtClaims{}, errExpiredJWTToken
	}

	return claims, nil
}

func (j *JWTUtility) sign(input string) string {
	mac := hmac.New(sha256.New, j.secret)
	_, _ = mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func setCookieWithTTL(w http.ResponseWriter, name, value string, ttl time.Duration) {
	maxAge := int(ttl.Seconds())
	if maxAge < 0 {
		maxAge = -1
	}
	expires := time.Time{}
	if maxAge == -1 {
		expires = time.Unix(0, 0)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureCookieEnabled(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		Expires:  expires,
	})
}

func SetAuthCookies(w http.ResponseWriter, accessToken string, accessTTL time.Duration, refreshToken string, refreshTTL time.Duration) {
	setCookieWithTTL(w, authCookieName, accessToken, accessTTL)
	setCookieWithTTL(w, refreshCookieName, refreshToken, refreshTTL)
}

func ClearAuthCookies(w http.ResponseWriter) {
	setCookieWithTTL(w, authCookieName, "", -1*time.Second)
	setCookieWithTTL(w, refreshCookieName, "", -1*time.Second)
}

func ReadCookie(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(cookie.Value)
	if value == "" {
		return "", http.ErrNoCookie
	}
	return value, nil
}

func ReadAuthCookie(r *http.Request) (string, error) {
	return ReadCookie(r, authCookieName)
}

func ReadRefreshCookie(r *http.Request) (string, error) {
	return ReadCookie(r, refreshCookieName)
}

func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func HashRefreshToken(rawToken string) (string, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return "", errors.New("refresh token cannot be empty")
	}
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:]), nil
}

func GenerateGuestSessionID() string {
	return uuid.NewString()
}

func SetGuestSessionCookie(w http.ResponseWriter, sessionID string) {
	setCookieWithTTL(w, guestSessionCookieName, sessionID, 30*24*time.Hour)
}

func ReadGuestSessionCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(guestSessionCookieName)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(cookie.Value)
	if value == "" {
		return "", http.ErrNoCookie
	}
	if _, err := uuid.Parse(value); err != nil {
		return "", err
	}
	return value, nil
}

func ActorFromContext(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(actorContextKey{}).(Actor)
	return actor, ok
}

func AuthUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	authCtx, ok := ctx.Value(authContextKey{}).(AuthContext)
	if !ok || !authCtx.IsAuthenticated || authCtx.UserID == uuid.Nil {
		return uuid.Nil, false
	}
	return authCtx.UserID, true
}

func IsAuthenticatedFromContext(ctx context.Context) bool {
	authCtx, ok := ctx.Value(authContextKey{}).(AuthContext)
	return ok && authCtx.IsAuthenticated
}

func AuthContextFromContext(ctx context.Context) (AuthContext, bool) {
	authCtx, ok := ctx.Value(authContextKey{}).(AuthContext)
	return authCtx, ok
}

func (s *Server) resolveAuthUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.jwt != nil {
			if token, err := ReadAuthCookie(r); err == nil {
				userID, parseErr := s.jwt.ExtractUserID(token)
				if parseErr == nil {
					actor := Actor{
						ActorType: ActorTypeUser,
						UserID:    userID,
					}
					ctx := context.WithValue(r.Context(), actorContextKey{}, actor)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) resolveGuestSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if actor, ok := ActorFromContext(r.Context()); ok && actor.ActorType == ActorTypeUser && actor.UserID != uuid.Nil {
			next.ServeHTTP(w, r)
			return
		}

		sessionID, err := ReadGuestSessionCookie(r)
		if err != nil {
			sessionID = GenerateGuestSessionID()
			SetGuestSessionCookie(w, sessionID)
		}

		actor := Actor{
			ActorType: ActorTypeSession,
			SessionID: sessionID,
		}
		ctx := context.WithValue(r.Context(), actorContextKey{}, actor)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) resolveActor(next http.Handler) http.Handler {
	return s.resolveAuthUser(s.resolveGuestSession(next))
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.jwt == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		token, err := ReadAuthCookie(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		userID, err := s.jwt.ExtractUserID(token)
		if err == nil {
			authCtx := AuthContext{
				UserID:          userID,
				IsAuthenticated: true,
			}
			ctx := context.WithValue(r.Context(), authContextKey{}, authCtx)
			ctx = context.WithValue(ctx, actorContextKey{}, Actor{
				ActorType: ActorTypeUser,
				UserID:    userID,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		if !errors.Is(err, errExpiredJWTToken) {
			ClearAuthCookies(w)
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		refreshRaw, err := ReadRefreshCookie(r)
		if err != nil {
			ClearAuthCookies(w)
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		refreshHash, err := HashRefreshToken(refreshRaw)
		if err != nil {
			ClearAuthCookies(w)
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		newRefreshRaw, err := GenerateRefreshToken()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to refresh session")
			return
		}
		newRefreshHash, err := HashRefreshToken(newRefreshRaw)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to refresh session")
			return
		}

		rotated, err := s.products.RotateRefreshToken(r.Context(), db.RotateRefreshTokenParams{
			OldTokenHash: refreshHash,
			NewTokenHash: newRefreshHash,
			ExpiresAt:    time.Now().UTC().Add(defaultRefreshTokenTTL),
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				ClearAuthCookies(w)
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to refresh session")
			return
		}

		newAccessToken, err := s.jwt.CreateToken(rotated.UserID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to refresh session")
			return
		}

		SetAuthCookies(w, newAccessToken, defaultAccessTokenTTL, newRefreshRaw, defaultRefreshTokenTTL)

		if rotated.UserID == uuid.Nil {
			ClearAuthCookies(w)
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		authCtx := AuthContext{
			UserID:          rotated.UserID,
			IsAuthenticated: true,
		}
		ctx := context.WithValue(r.Context(), authContextKey{}, authCtx)
		ctx = context.WithValue(ctx, actorContextKey{}, Actor{
			ActorType: ActorTypeUser,
			UserID:    rotated.UserID,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isSecureCookieEnabled() bool {
	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		return true
	}
	return strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true")
}

func isDevelopmentAuthEnabled() bool {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	return env == "development" || env == "dev" || env == "local"
}

func SetAuthCookie(w http.ResponseWriter, token string) {
	setCookieWithTTL(w, authCookieName, token, 7*24*time.Hour)
}

func SetRefreshCookie(w http.ResponseWriter, token string, ttl time.Duration) {
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	setCookieWithTTL(w, refreshCookieName, token, ttl)
}

func ClearAuthCookie(w http.ResponseWriter) {
	setCookieWithTTL(w, authCookieName, "", -1*time.Second)
}

func ClearRefreshCookie(w http.ResponseWriter) {
	setCookieWithTTL(w, refreshCookieName, "", -1*time.Second)
}
