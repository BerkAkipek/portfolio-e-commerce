package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	db "github.com/BerkAkipek/e-commerce-app/api/internal/db"
)

func cookieValueByName(resp *http.Response, name string) (string, bool) {
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c.Value, true
		}
	}
	return "", false
}

func TestSmokeRoutePrefixes(t *testing.T) {
	productID := uuid.MustParse("00000000-0000-0000-0000-000000001001")
	store := &fakeProductQuerier{
		products: []db.Product{{
			ID:       productID,
			Name:     "Basic Tee",
			Slug:     "basic-tee",
			IsActive: true,
			Currency: "USD",
		}},
		product: db.Product{
			ID:       productID,
			Name:     "Basic Tee",
			Slug:     "basic-tee",
			IsActive: true,
			Currency: "USD",
		},
	}
	server := NewServer(store)

	cases := []string{
		"/health",
		"/api/health",
		"/products",
		"/api/products",
		"/products/basic-tee",
		"/api/products/basic-tee",
	}

	for _, path := range cases {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		server.Router().ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected status 200 for %s, got %d", path, rr.Code)
		}
	}
}

func TestSmokeGuestCartLifecycle(t *testing.T) {
	sessionID := uuid.NewString()
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000001100")
	productID := uuid.MustParse("00000000-0000-0000-0000-000000001101")
	itemID := uuid.MustParse("00000000-0000-0000-0000-000000001102")
	createdAt := time.Now().UTC()

	store := &fakeProductQuerier{
		activeSessionCart: db.Cart{
			ID:        cartID,
			SessionID: sql.NullString{String: sessionID, Valid: true},
			Status:    "active",
		},
		product: db.Product{
			ID:         productID,
			IsActive:   true,
			PriceCents: 1999,
			Currency:   "USD",
			Name:       "Basic Tee",
		},
		cartItemErr: sql.ErrNoRows,
		createItem: db.CartItem{
			ID:                 itemID,
			CartID:             cartID,
			ProductID:          productID,
			Quantity:           2,
			PriceCentsSnapshot: 1999,
			CreatedAt:          createdAt,
		},
		updateItem: db.CartItem{
			ID:                 itemID,
			CartID:             cartID,
			ProductID:          productID,
			Quantity:           3,
			PriceCentsSnapshot: 1999,
			CreatedAt:          createdAt,
		},
	}

	server := NewServer(store)

	createReq := httptest.NewRequest(http.MethodPost, "/cart/items", strings.NewReader(`{"product_id":"00000000-0000-0000-0000-000000001101","quantity":2}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(&http.Cookie{Name: guestSessionCookieName, Value: sessionID})
	createRR := httptest.NewRecorder()
	server.Router().ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d", createRR.Code)
	}

	store.cartItem = store.createItem
	store.cartItemErr = nil
	store.listItems = []db.CartItem{store.createItem}

	getReq := httptest.NewRequest(http.MethodGet, "/cart/"+cartID.String()+"/items", nil)
	getReq.AddCookie(&http.Cookie{Name: guestSessionCookieName, Value: sessionID})
	getRR := httptest.NewRecorder()
	server.Router().ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d", getRR.Code)
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/cart/items/"+itemID.String(), strings.NewReader(`{"quantity":3}`))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.AddCookie(&http.Cookie{Name: guestSessionCookieName, Value: sessionID})
	patchRR := httptest.NewRecorder()
	server.Router().ServeHTTP(patchRR, patchReq)
	if patchRR.Code != http.StatusOK {
		t.Fatalf("expected patch status 200, got %d", patchRR.Code)
	}

	store.cartItem = store.updateItem
	deleteReq := httptest.NewRequest(http.MethodDelete, "/cart/items/"+itemID.String(), nil)
	deleteReq.AddCookie(&http.Cookie{Name: guestSessionCookieName, Value: sessionID})
	deleteRR := httptest.NewRecorder()
	server.Router().ServeHTTP(deleteRR, deleteReq)
	if deleteRR.Code != http.StatusNoContent {
		t.Fatalf("expected delete status 204, got %d", deleteRR.Code)
	}
}

func TestSmokeAuthLifecycle(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	t.Setenv("APP_ENV", "development")
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	t.Setenv("CHECKOUT_SUCCESS_URL", "https://example.com/success")
	t.Setenv("CHECKOUT_CANCEL_URL", "https://example.com/cancel")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000001200")
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000001201")
	productID := uuid.MustParse("00000000-0000-0000-0000-000000001202")
	itemID := uuid.MustParse("00000000-0000-0000-0000-000000001203")

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("P@ssw0rd!"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate bcrypt hash: %v", err)
	}

	store := &fakeProductQuerier{
		userByEmail: db.User{
			ID:           userID,
			Email:        "user@example.com",
			PasswordHash: sql.NullString{String: string(passwordHash), Valid: true},
		},
		activeUserCart: db.Cart{
			ID:     cartID,
			UserID: uuid.NullUUID{UUID: userID, Valid: true},
			Status: "active",
		},
		listItems: []db.CartItem{{
			ID:                 itemID,
			CartID:             cartID,
			ProductID:          productID,
			Quantity:           1,
			PriceCentsSnapshot: 1999,
			CreatedAt:          time.Now().UTC(),
		}},
		product: db.Product{
			ID:       productID,
			Name:     "Basic Tee",
			Currency: "USD",
			IsActive: true,
		},
		rotatedRefreshToken: db.RefreshToken{
			ID:        uuid.MustParse("00000000-0000-0000-0000-000000001204"),
			UserID:    userID,
			TokenHash: "rotated",
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
			Revoked:   false,
			CreatedAt: time.Now().UTC(),
		},
	}

	stripe := &fakeStripeGateway{
		createSession: stripeCheckoutSession{
			ID:  "cs_test_123",
			URL: "https://checkout.stripe.com/c/pay/cs_test_123",
		},
	}

	server := NewServer(store)
	server.stripe = stripe

	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"user@example.com","password":"P@ssw0rd!"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	server.Router().ServeHTTP(loginRR, loginReq)
	if loginRR.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", loginRR.Code)
	}
	accessToken, ok := cookieValueByName(loginRR.Result(), authCookieName)
	if !ok || accessToken == "" {
		t.Fatalf("expected access token cookie after login")
	}
	refreshToken, ok := cookieValueByName(loginRR.Result(), refreshCookieName)
	if !ok || refreshToken == "" {
		t.Fatalf("expected refresh token cookie after login")
	}

	checkoutReq := httptest.NewRequest(http.MethodPost, "/checkout/session", strings.NewReader(`{}`))
	checkoutReq.Header.Set("Content-Type", "application/json")
	checkoutReq.AddCookie(&http.Cookie{Name: authCookieName, Value: accessToken})
	checkoutReq.AddCookie(&http.Cookie{Name: refreshCookieName, Value: refreshToken})
	checkoutRR := httptest.NewRecorder()
	server.Router().ServeHTTP(checkoutRR, checkoutReq)
	if checkoutRR.Code != http.StatusOK {
		t.Fatalf("expected checkout status 200, got %d", checkoutRR.Code)
	}

	expiredAccess := mustTokenWithClaims(t, server.jwt, jwtClaims{
		Sub: userID.String(),
		Iat: time.Now().UTC().Add(-2 * time.Hour).Unix(),
		Exp: time.Now().UTC().Add(-time.Hour).Unix(),
		Iss: server.jwt.issuer,
	}, "HS256", "JWT")

	checkoutRefreshReq := httptest.NewRequest(http.MethodPost, "/checkout/session", strings.NewReader(`{}`))
	checkoutRefreshReq.Header.Set("Content-Type", "application/json")
	checkoutRefreshReq.AddCookie(&http.Cookie{Name: authCookieName, Value: expiredAccess})
	checkoutRefreshReq.AddCookie(&http.Cookie{Name: refreshCookieName, Value: refreshToken})
	checkoutRefreshRR := httptest.NewRecorder()
	server.Router().ServeHTTP(checkoutRefreshRR, checkoutRefreshReq)
	if checkoutRefreshRR.Code != http.StatusOK {
		t.Fatalf("expected refresh-checkout status 200, got %d", checkoutRefreshRR.Code)
	}

	rotatedRefreshRaw, ok := cookieValueByName(checkoutRefreshRR.Result(), refreshCookieName)
	if !ok || rotatedRefreshRaw == "" {
		t.Fatalf("expected rotated refresh token cookie")
	}

	store.lastRevokeRefreshTokenHash = ""
	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutReq.AddCookie(&http.Cookie{Name: refreshCookieName, Value: rotatedRefreshRaw})
	logoutRR := httptest.NewRecorder()
	server.Router().ServeHTTP(logoutRR, logoutReq)
	if logoutRR.Code != http.StatusOK {
		t.Fatalf("expected logout status 200, got %d", logoutRR.Code)
	}
	if store.lastRevokeRefreshTokenHash == "" {
		t.Fatalf("expected refresh token revoke on logout")
	}

	var body map[string]any
	if err := json.NewDecoder(logoutRR.Body).Decode(&body); err != nil {
		t.Fatalf("decode logout response: %v", err)
	}
	if body["success"] != true {
		t.Fatalf("expected logout success=true, got %+v", body)
	}
}
