package httpapi

import (
	"crypto/hmac"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type fixedWindowState struct {
	count   int
	resetAt time.Time
}

type fixedWindowLimiter struct {
	limit  int
	window time.Duration

	mu    sync.Mutex
	state map[string]fixedWindowState
}

func newFixedWindowLimiter(limit int, window time.Duration) *fixedWindowLimiter {
	if limit <= 0 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &fixedWindowLimiter{
		limit:  limit,
		window: window,
		state:  map[string]fixedWindowState{},
	}
}

func (l *fixedWindowLimiter) allow(key string) bool {
	if l == nil {
		return true
	}
	now := time.Now().UTC()

	l.mu.Lock()
	defer l.mu.Unlock()

	item, ok := l.state[key]
	if !ok || now.After(item.resetAt) {
		l.state[key] = fixedWindowState{
			count:   1,
			resetAt: now.Add(l.window),
		}
		return true
	}
	if item.count >= l.limit {
		return false
	}
	item.count++
	l.state[key] = item
	return true
}

func clientIPFromRequest(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	if remote := strings.TrimSpace(r.RemoteAddr); remote != "" {
		return remote
	}
	return "unknown"
}

func (s *Server) rateLimitMiddleware(scope string, limiter *fixedWindowLimiter, keyFunc func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil {
				next.ServeHTTP(w, r)
				return
			}
			key := scope + ":" + keyFunc(r)
			if !limiter.allow(key) {
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func webhookRateLimitKey(r *http.Request) string {
	signature := strings.TrimSpace(r.Header.Get("Stripe-Signature"))
	if signature != "" {
		return signature
	}
	return clientIPFromRequest(r)
}

func isCSRFCookieModeEnabled() bool {
	// Token mode is needed when cookies are intentionally sent cross-site.
	// Keeping this env-gated avoids breaking API clients in same-origin setups.
	return strings.EqualFold(strings.TrimSpace(os.Getenv("CSRF_PROTECTION_MODE")), "token")
}

func shouldSkipCSRFPath(path string) bool {
	return path == "/webhooks/stripe" || path == "/api/webhooks/stripe"
}

func isStateChangingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func csrfGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isCSRFCookieModeEnabled() || !isStateChangingMethod(r.Method) || shouldSkipCSRFPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		cookieToken, err := ReadCookie(r, csrfCookieName)
		headerToken := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
		if err != nil || headerToken == "" || !hmac.Equal([]byte(cookieToken), []byte(headerToken)) {
			writeError(w, http.StatusForbidden, "csrf validation failed")
			return
		}

		next.ServeHTTP(w, r)
	})
}
