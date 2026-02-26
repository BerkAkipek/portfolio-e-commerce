package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	db "github.com/BerkAkipek/e-commerce-app/api/internal/db"
)

type fakeProductQuerier struct {
	products          []db.Product
	err               error
	lastArg           db.ListActiveProductsParams
	product           db.Product
	getErr            error
	lastSlug          string
	productCategories map[uuid.UUID][]db.Category

	cart                 db.Cart
	cartErr              error
	activeUserCart       db.Cart
	activeUserCartErr    error
	activeSessionCart    db.Cart
	activeSessionCartErr error
	createCart           db.Cart
	createCartErr        error
	productErr           error
	cartItem             db.CartItem
	cartItemErr          error
	createItem           db.CartItem
	createErr            error
	updateItem           db.CartItem
	updateErr            error
	removeErr            error
	listItems            []db.CartItem
	listErr              error
	order                db.Order
	orders               []db.Order
	orderErr             error
	ordersErr            error
	orderByID            db.Order
	orderByIDErr         error
	orderItems           []db.OrderItem
	orderItemsErr        error
	orderItem            db.OrderItem
	orderItemErr         error
	payment              db.Payment
	paymentErr           error
	paymentLookup        db.Payment
	paymentLookupErr     error
	cartStatusErr        error
	user                 db.User
	userErr              error
	userByEmail          db.User
	userByEmailErr       error
	createdUser          db.User
	createUserErr        error
	refreshToken         db.RefreshToken
	refreshTokenErr      error
	refreshLookup        db.RefreshToken
	refreshLookupErr     error
	rotatedRefreshToken  db.RefreshToken
	rotateRefreshErr     error
	revokedRefreshToken  db.RefreshToken
	revokeRefreshErr     error

	lastGetCartID                 uuid.UUID
	lastGetUserID                 uuid.UUID
	lastGetUserEmail              string
	lastCreateUserArg             db.CreateUserParams
	lastCreateRefreshTokenArg     db.CreateRefreshTokenParams
	lastGetRefreshTokenHash       string
	lastRotateOldRefreshTokenHash string
	lastRotateNewRefreshTokenHash string
	lastRotateRefreshExpiresAt    time.Time
	lastRevokeRefreshTokenHash    string
	lastGetActiveCartByUserID     uuid.UUID
	lastGetActiveCartBySessionID  string
	lastCreateCartArg             db.CreateCartParams
	lastGetProdID                 uuid.UUID
	lastGetItemID                 uuid.UUID
	lastCartItemArg               db.GetCartItemByCartAndProductParams
	lastCreateArg                 db.CreateCartItemParams
	lastUpdateArg                 db.UpdateCartItemQuantityParams
	lastRemoveID                  uuid.UUID
	lastListCartID                uuid.UUID
	lastCreateOrderArg            db.CreateOrderParams
	lastListOrdersArg             db.ListOrdersByUserIDParams
	lastGetOrderID                uuid.UUID
	lastListOrderItemsOrderID     uuid.UUID
	lastCreateOrderItemArg        db.CreateOrderItemParams
	lastCreatePaymentArg          db.CreatePaymentParams
	lastPaymentLookupSessionID    string
	lastUpdateCartStatusArg       db.UpdateCartStatusParams
}

func (f *fakeProductQuerier) ListActiveProducts(_ context.Context, arg db.ListActiveProductsParams) ([]db.Product, error) {
	f.lastArg = arg
	if f.err != nil {
		return nil, f.err
	}
	return f.products, nil
}

func (f *fakeProductQuerier) GetProductBySlug(_ context.Context, slug string) (db.Product, error) {
	f.lastSlug = slug
	if f.getErr != nil {
		return db.Product{}, f.getErr
	}
	return f.product, nil
}

func (f *fakeProductQuerier) ListCategoriesByProductID(_ context.Context, productID uuid.UUID) ([]db.Category, error) {
	if f.productCategories == nil {
		return []db.Category{}, nil
	}
	return f.productCategories[productID], nil
}

func (f *fakeProductQuerier) GetCartByID(_ context.Context, id uuid.UUID) (db.Cart, error) {
	f.lastGetCartID = id
	if f.cartErr != nil {
		return db.Cart{}, f.cartErr
	}
	return f.cart, nil
}

func (f *fakeProductQuerier) GetUserByID(_ context.Context, id uuid.UUID) (db.User, error) {
	f.lastGetUserID = id
	if f.userErr != nil {
		return db.User{}, f.userErr
	}
	return f.user, nil
}

func (f *fakeProductQuerier) GetUserByEmail(_ context.Context, email string) (db.User, error) {
	f.lastGetUserEmail = email
	if f.userByEmailErr != nil {
		return db.User{}, f.userByEmailErr
	}
	if f.userByEmail.ID != uuid.Nil {
		return f.userByEmail, nil
	}
	return f.user, f.userErr
}

func (f *fakeProductQuerier) CreateUser(_ context.Context, arg db.CreateUserParams) (db.User, error) {
	f.lastCreateUserArg = arg
	if f.createUserErr != nil {
		return db.User{}, f.createUserErr
	}
	if f.createdUser.ID != uuid.Nil {
		return f.createdUser, nil
	}
	return db.User{
		ID:           uuid.New(),
		Email:        arg.Email,
		PasswordHash: arg.PasswordHash,
		FullName:     arg.FullName,
		Role:         arg.Role,
	}, nil
}

func (f *fakeProductQuerier) CreateRefreshToken(_ context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error) {
	f.lastCreateRefreshTokenArg = arg
	if f.refreshTokenErr != nil {
		return db.RefreshToken{}, f.refreshTokenErr
	}
	if f.refreshToken.ID != uuid.Nil {
		return f.refreshToken, nil
	}
	return db.RefreshToken{
		ID:        uuid.New(),
		UserID:    arg.UserID,
		TokenHash: arg.TokenHash,
		ExpiresAt: arg.ExpiresAt,
		Revoked:   false,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (f *fakeProductQuerier) GetRefreshTokenByHash(_ context.Context, tokenHash string) (db.RefreshToken, error) {
	f.lastGetRefreshTokenHash = tokenHash
	if f.refreshLookupErr != nil {
		return db.RefreshToken{}, f.refreshLookupErr
	}
	if f.refreshLookup.ID != uuid.Nil {
		return f.refreshLookup, nil
	}
	return db.RefreshToken{}, sql.ErrNoRows
}

func (f *fakeProductQuerier) RevokeRefreshTokenByHash(_ context.Context, tokenHash string) (db.RefreshToken, error) {
	f.lastRevokeRefreshTokenHash = tokenHash
	if f.revokeRefreshErr != nil {
		return db.RefreshToken{}, f.revokeRefreshErr
	}
	if f.revokedRefreshToken.ID != uuid.Nil {
		return f.revokedRefreshToken, nil
	}
	return db.RefreshToken{
		ID:        uuid.New(),
		TokenHash: tokenHash,
		Revoked:   true,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (f *fakeProductQuerier) RotateRefreshToken(_ context.Context, arg db.RotateRefreshTokenParams) (db.RefreshToken, error) {
	f.lastRotateOldRefreshTokenHash = arg.OldTokenHash
	f.lastRotateNewRefreshTokenHash = arg.NewTokenHash
	f.lastRotateRefreshExpiresAt = arg.ExpiresAt
	if f.rotateRefreshErr != nil {
		return db.RefreshToken{}, f.rotateRefreshErr
	}
	if f.rotatedRefreshToken.ID != uuid.Nil {
		return f.rotatedRefreshToken, nil
	}
	return db.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		TokenHash: arg.NewTokenHash,
		ExpiresAt: arg.ExpiresAt,
		Revoked:   false,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (f *fakeProductQuerier) GetActiveCartByUserID(_ context.Context, userID uuid.UUID) (db.Cart, error) {
	f.lastGetActiveCartByUserID = userID
	if f.activeUserCartErr != nil {
		return db.Cart{}, f.activeUserCartErr
	}
	if f.activeUserCart.ID != uuid.Nil {
		return f.activeUserCart, nil
	}
	if f.cartErr != nil {
		return db.Cart{}, f.cartErr
	}
	if f.cart.ID != uuid.Nil && f.cart.Status == "active" && f.cart.UserID.Valid && f.cart.UserID.UUID == userID {
		return f.cart, nil
	}
	return db.Cart{}, sql.ErrNoRows
}

func (f *fakeProductQuerier) GetActiveCartBySessionID(_ context.Context, sessionID string) (db.Cart, error) {
	f.lastGetActiveCartBySessionID = sessionID
	if f.activeSessionCartErr != nil {
		return db.Cart{}, f.activeSessionCartErr
	}
	if f.activeSessionCart.ID != uuid.Nil {
		return f.activeSessionCart, nil
	}
	if f.cartErr != nil {
		return db.Cart{}, f.cartErr
	}
	if f.cart.ID != uuid.Nil && f.cart.Status == "active" && f.cart.SessionID.Valid && f.cart.SessionID.String == sessionID {
		return f.cart, nil
	}
	return db.Cart{}, sql.ErrNoRows
}

func (f *fakeProductQuerier) CreateCart(_ context.Context, arg db.CreateCartParams) (db.Cart, error) {
	f.lastCreateCartArg = arg
	if f.createCartErr != nil {
		return db.Cart{}, f.createCartErr
	}
	if f.createCart.ID != uuid.Nil {
		return f.createCart, nil
	}
	return f.cart, f.cartErr
}

func (f *fakeProductQuerier) GetProductByID(_ context.Context, id uuid.UUID) (db.Product, error) {
	f.lastGetProdID = id
	if f.productErr != nil {
		return db.Product{}, f.productErr
	}
	return f.product, nil
}

func (f *fakeProductQuerier) GetCartItemByID(_ context.Context, id uuid.UUID) (db.CartItem, error) {
	f.lastGetItemID = id
	if f.cartItemErr != nil {
		return db.CartItem{}, f.cartItemErr
	}
	return f.cartItem, nil
}

func (f *fakeProductQuerier) GetCartItemByCartAndProduct(_ context.Context, arg db.GetCartItemByCartAndProductParams) (db.CartItem, error) {
	f.lastCartItemArg = arg
	if f.cartItemErr != nil {
		return db.CartItem{}, f.cartItemErr
	}
	return f.cartItem, nil
}

func (f *fakeProductQuerier) CreateCartItem(_ context.Context, arg db.CreateCartItemParams) (db.CartItem, error) {
	f.lastCreateArg = arg
	if f.createErr != nil {
		return db.CartItem{}, f.createErr
	}
	return f.createItem, nil
}

func (f *fakeProductQuerier) UpdateCartItemQuantity(_ context.Context, arg db.UpdateCartItemQuantityParams) (db.CartItem, error) {
	f.lastUpdateArg = arg
	if f.updateErr != nil {
		return db.CartItem{}, f.updateErr
	}
	return f.updateItem, nil
}

func (f *fakeProductQuerier) ListCartItemsByCartID(_ context.Context, cartID uuid.UUID) ([]db.CartItem, error) {
	f.lastListCartID = cartID
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listItems, nil
}

func (f *fakeProductQuerier) RemoveCartItem(_ context.Context, id uuid.UUID) error {
	f.lastRemoveID = id
	return f.removeErr
}

func (f *fakeProductQuerier) CreateOrder(_ context.Context, arg db.CreateOrderParams) (db.Order, error) {
	f.lastCreateOrderArg = arg
	if f.orderErr != nil {
		return db.Order{}, f.orderErr
	}
	return f.order, nil
}

func (f *fakeProductQuerier) ListOrdersByUserID(_ context.Context, arg db.ListOrdersByUserIDParams) ([]db.Order, error) {
	f.lastListOrdersArg = arg
	if f.ordersErr != nil {
		return nil, f.ordersErr
	}
	return f.orders, nil
}

func (f *fakeProductQuerier) GetOrderByID(_ context.Context, id uuid.UUID) (db.Order, error) {
	f.lastGetOrderID = id
	if f.orderByIDErr != nil {
		return db.Order{}, f.orderByIDErr
	}
	if f.orderByID.ID != uuid.Nil {
		return f.orderByID, nil
	}
	return db.Order{}, sql.ErrNoRows
}

func (f *fakeProductQuerier) ListOrderItemsByOrderID(_ context.Context, orderID uuid.UUID) ([]db.OrderItem, error) {
	f.lastListOrderItemsOrderID = orderID
	if f.orderItemsErr != nil {
		return nil, f.orderItemsErr
	}
	return f.orderItems, nil
}

func (f *fakeProductQuerier) CreateOrderItem(_ context.Context, arg db.CreateOrderItemParams) (db.OrderItem, error) {
	f.lastCreateOrderItemArg = arg
	if f.orderItemErr != nil {
		return db.OrderItem{}, f.orderItemErr
	}
	return f.orderItem, nil
}

func (f *fakeProductQuerier) CreatePayment(_ context.Context, arg db.CreatePaymentParams) (db.Payment, error) {
	f.lastCreatePaymentArg = arg
	if f.paymentErr != nil {
		return db.Payment{}, f.paymentErr
	}
	return f.payment, nil
}

func (f *fakeProductQuerier) GetPaymentByCheckoutSessionID(_ context.Context, checkoutSessionID string) (db.Payment, error) {
	f.lastPaymentLookupSessionID = checkoutSessionID
	if f.paymentLookupErr != nil {
		return db.Payment{}, f.paymentLookupErr
	}
	return f.paymentLookup, nil
}

func (f *fakeProductQuerier) UpdateCartStatus(_ context.Context, arg db.UpdateCartStatusParams) (db.Cart, error) {
	f.lastUpdateCartStatusArg = arg
	if f.cartStatusErr != nil {
		return db.Cart{}, f.cartStatusErr
	}
	return f.cart, nil
}

type fakeStripeGateway struct {
	createSession    stripeCheckoutSession
	createSessionErr error
	lastCreateInput  stripeCreateCheckoutSessionInput
	webhookEvent     stripeEvent
	webhookErr       error
}

func (f *fakeStripeGateway) CreateCheckoutSession(_ context.Context, input stripeCreateCheckoutSessionInput) (stripeCheckoutSession, error) {
	f.lastCreateInput = input
	if f.createSessionErr != nil {
		return stripeCheckoutSession{}, f.createSessionErr
	}
	return f.createSession, nil
}

func (f *fakeStripeGateway) ParseWebhook(_ []byte, _ string, _ string, _ time.Duration) (stripeEvent, error) {
	if f.webhookErr != nil {
		return stripeEvent{}, f.webhookErr
	}
	return f.webhookEvent, nil
}

func TestGetProductsSuccess(t *testing.T) {
	store := &fakeProductQuerier{
		products: []db.Product{
			{
				ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Name:       "Basic Tee",
				Slug:       "basic-tee",
				PriceCents: 1999,
				Currency:   "USD",
				Stock:      12,
				IsActive:   true,
				Sku:        "TEE-001",
			},
		},
	}

	server := NewServer(store)
	req := httptest.NewRequest(http.MethodGet, "/products?limit=10&offset=5", nil)
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if store.lastArg.Limit != 10 || store.lastArg.Offset != 5 {
		t.Fatalf("unexpected pagination arg: %+v", store.lastArg)
	}

	var body struct {
		Data []struct {
			Name string `json:"name"`
			Slug string `json:"slug"`
		} `json:"data"`
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Limit != 10 || body.Offset != 5 {
		t.Fatalf("unexpected pagination response: %+v", body)
	}
	if len(body.Data) != 1 || body.Data[0].Slug != "basic-tee" {
		t.Fatalf("unexpected data response: %+v", body.Data)
	}
}

func TestGetProductsBadLimit(t *testing.T) {
	server := NewServer(&fakeProductQuerier{})
	req := httptest.NewRequest(http.MethodGet, "/products?limit=0", nil)
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetProductsStoreError(t *testing.T) {
	server := NewServer(&fakeProductQuerier{err: errors.New("db down")})
	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestGetProductBySlugSuccess(t *testing.T) {
	store := &fakeProductQuerier{
		product: db.Product{
			ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			Name:       "Running Shoe",
			Slug:       "running-shoe",
			PriceCents: 7999,
			Currency:   "USD",
			Stock:      4,
			IsActive:   true,
			Sku:        "RUN-002",
		},
	}

	server := NewServer(store)
	req := httptest.NewRequest(http.MethodGet, "/products/running-shoe", nil)
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if store.lastSlug != "running-shoe" {
		t.Fatalf("expected slug lookup to be running-shoe, got %q", store.lastSlug)
	}
}

func TestGetProductBySlugNotFound(t *testing.T) {
	server := NewServer(&fakeProductQuerier{getErr: sql.ErrNoRows})
	req := httptest.NewRequest(http.MethodGet, "/products/missing", nil)
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestGetProductBySlugStoreError(t *testing.T) {
	server := NewServer(&fakeProductQuerier{getErr: errors.New("db down")})
	req := httptest.NewRequest(http.MethodGet, "/products/running-shoe", nil)
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestDevLoginJWTNotConfigured(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	req := httptest.NewRequest(http.MethodPost, "/auth/dev-login", strings.NewReader(`{"user_id":"00000000-0000-0000-0000-000000000900"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(&fakeProductQuerier{}).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}
}

func TestDevLoginDisabledOutsideDevelopment(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "resolve-secret")

	req := httptest.NewRequest(http.MethodPost, "/auth/dev-login", strings.NewReader(`{"user_id":"00000000-0000-0000-0000-000000000900"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(&fakeProductQuerier{}).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestRegisterSuccessHashesPasswordAndLogsIn(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000905")
	store := &fakeProductQuerier{
		createdUser: db.User{
			ID: userID,
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"email":"new@example.com","password":"P@ssw0rd!","full_name":"New User"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if store.lastCreateUserArg.Email != "new@example.com" {
		t.Fatalf("expected email new@example.com, got %q", store.lastCreateUserArg.Email)
	}
	if store.lastCreateUserArg.FullName != "New User" {
		t.Fatalf("expected full name New User, got %q", store.lastCreateUserArg.FullName)
	}
	if store.lastCreateUserArg.Role != "user" {
		t.Fatalf("expected role user, got %q", store.lastCreateUserArg.Role)
	}
	if !store.lastCreateUserArg.PasswordHash.Valid || store.lastCreateUserArg.PasswordHash.String == "" {
		t.Fatalf("expected password hash to be persisted")
	}
	if store.lastCreateUserArg.PasswordHash.String == "P@ssw0rd!" {
		t.Fatalf("expected stored password to be hashed")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(store.lastCreateUserArg.PasswordHash.String), []byte("P@ssw0rd!")); err != nil {
		t.Fatalf("expected stored hash to match password: %v", err)
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
		t.Fatalf("expected both auth cookies to be set")
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["success"] != true {
		t.Fatalf("expected success=true response, got %+v", body)
	}
	if _, ok := body["access_token"]; ok {
		t.Fatalf("expected access_token to be absent from response body")
	}
	if _, ok := body["refresh_token"]; ok {
		t.Fatalf("expected refresh_token to be absent from response body")
	}
}

func TestRegisterValidation(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	tests := []string{
		`{"email":"bad","password":"P@ssw0rd!","full_name":"Name"}`,
		`{"email":"ok@example.com","password":"short","full_name":"Name"}`,
		`{"email":"ok@example.com","password":"P@ssw0rd!","full_name":""}`,
	}

	for _, body := range tests {
		req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		NewServer(&fakeProductQuerier{}).Router().ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d for body %s", rr.Code, body)
		}
	}
}

func TestRegisterDuplicateEmailReturnsConflict(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	store := &fakeProductQuerier{
		createUserErr: &pgconn.PgError{Code: "23505"},
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"email":"dup@example.com","password":"P@ssw0rd!","full_name":"Dup User"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rr.Code)
	}
}

func TestRegisterPasswordPreservesWhitespace(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	store := &fakeProductQuerier{
		createdUser: db.User{
			ID: uuid.MustParse("00000000-0000-0000-0000-000000000906"),
		},
	}

	rawPassword := "  P@ssw0rd!  "
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"email":"spaces@example.com","password":"  P@ssw0rd!  ","full_name":"Space User"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(store.lastCreateUserArg.PasswordHash.String), []byte(rawPassword)); err != nil {
		t.Fatalf("expected hash to preserve password whitespace: %v", err)
	}
}

func TestLoginSuccessSetsCookiesAndNoTokensInBody(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000910")
	hash, err := bcrypt.GenerateFromPassword([]byte("P@ssw0rd!"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate bcrypt hash: %v", err)
	}
	store := &fakeProductQuerier{
		userByEmail: db.User{
			ID:           userID,
			Email:        "user@example.com",
			PasswordHash: sql.NullString{String: string(hash), Valid: true},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"user@example.com","password":"P@ssw0rd!"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if store.lastGetUserEmail != "user@example.com" {
		t.Fatalf("expected user lookup by email user@example.com, got %q", store.lastGetUserEmail)
	}
	if store.lastCreateRefreshTokenArg.UserID != userID {
		t.Fatalf("expected refresh token created for user %s, got %s", userID, store.lastCreateRefreshTokenArg.UserID)
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
		t.Fatalf("expected both auth cookies to be set")
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["success"] != true {
		t.Fatalf("expected success=true response, got %+v", body)
	}
	if _, ok := body["access_token"]; ok {
		t.Fatalf("expected access_token to be absent from response body")
	}
	if _, ok := body["refresh_token"]; ok {
		t.Fatalf("expected refresh_token to be absent from response body")
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate bcrypt hash: %v", err)
	}
	store := &fakeProductQuerier{
		userByEmail: db.User{
			ID:           uuid.MustParse("00000000-0000-0000-0000-000000000911"),
			Email:        "user@example.com",
			PasswordHash: sql.NullString{String: string(hash), Valid: true},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"user@example.com","password":"wrong-password"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestLoginUserNotFound(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	store := &fakeProductQuerier{userByEmailErr: sql.ErrNoRows}

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"missing@example.com","password":"any"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestLoginMissingPasswordHash(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	store := &fakeProductQuerier{
		userByEmail: db.User{
			ID:    uuid.MustParse("00000000-0000-0000-0000-000000000912"),
			Email: "user@example.com",
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"user@example.com","password":"any"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestDevLoginSuccessSetsAuthCookie(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_SECRET", "resolve-secret")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000901")
	store := &fakeProductQuerier{
		user: db.User{ID: userID},
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/dev-login", strings.NewReader(`{"user_id":"00000000-0000-0000-0000-000000000901"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if store.lastGetUserID != userID {
		t.Fatalf("expected user lookup by %s, got %s", userID, store.lastGetUserID)
	}

	var foundAuth bool
	var foundRefresh bool
	for _, c := range rr.Result().Cookies() {
		if c.Name == authCookieName && c.Value != "" {
			foundAuth = true
		}
		if c.Name == refreshCookieName && c.Value != "" {
			foundRefresh = true
		}
	}
	if !foundAuth {
		t.Fatalf("expected auth cookie to be set")
	}
	if !foundRefresh {
		t.Fatalf("expected refresh cookie to be set")
	}
	if store.lastCreateRefreshTokenArg.UserID != userID {
		t.Fatalf("expected refresh token to be created for user %s, got %s", userID, store.lastCreateRefreshTokenArg.UserID)
	}
	if store.lastCreateRefreshTokenArg.TokenHash == "" {
		t.Fatalf("expected hashed refresh token to be persisted")
	}
	if len(store.lastCreateRefreshTokenArg.TokenHash) != 64 {
		t.Fatalf("expected sha256 hex hash length 64, got %d", len(store.lastCreateRefreshTokenArg.TokenHash))
	}
}

func TestLogoutClearsAuthCookie(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rr := httptest.NewRecorder()

	NewServer(&fakeProductQuerier{}).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var foundAuth bool
	var foundRefresh bool
	for _, c := range rr.Result().Cookies() {
		if c.Name == authCookieName && c.MaxAge == -1 {
			foundAuth = true
		}
		if c.Name == refreshCookieName && c.MaxAge == -1 {
			foundRefresh = true
		}
	}
	if !foundAuth {
		t.Fatalf("expected cleared auth cookie to be set")
	}
	if !foundRefresh {
		t.Fatalf("expected cleared refresh cookie to be set")
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["success"] != true {
		t.Fatalf("expected success=true response, got %+v", body)
	}
}

func TestLogoutRevokesRefreshTokenByHash(t *testing.T) {
	store := &fakeProductQuerier{}
	rawRefresh := "raw-refresh-token-123"
	expectedHash, err := HashRefreshToken(rawRefresh)
	if err != nil {
		t.Fatalf("hash refresh token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: rawRefresh})
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if store.lastRevokeRefreshTokenHash != expectedHash {
		t.Fatalf("expected revoke hash %s, got %s", expectedHash, store.lastRevokeRefreshTokenHash)
	}
}

func TestPostCartItemsCreateSuccess(t *testing.T) {
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	productID := uuid.MustParse("00000000-0000-0000-0000-000000000011")

	store := &fakeProductQuerier{
		cart: db.Cart{ID: cartID, Status: "active"},
		product: db.Product{
			ID:         productID,
			IsActive:   true,
			PriceCents: 1999,
		},
		cartItemErr: sql.ErrNoRows,
		createItem: db.CartItem{
			ID:                 uuid.MustParse("00000000-0000-0000-0000-000000000012"),
			CartID:             cartID,
			ProductID:          productID,
			Quantity:           2,
			PriceCentsSnapshot: 1999,
			CreatedAt:          time.Now().UTC(),
		},
	}

	body := `{"cart_id":"00000000-0000-0000-0000-000000000010","product_id":"00000000-0000-0000-0000-000000000011","quantity":2}`
	req := httptest.NewRequest(http.MethodPost, "/cart/items", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}
	if store.lastCreateArg.Quantity != 2 || store.lastCreateArg.PriceCentsSnapshot != 1999 {
		t.Fatalf("unexpected create args: %+v", store.lastCreateArg)
	}
}

func TestPostCartItemsIncrementSuccess(t *testing.T) {
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	productID := uuid.MustParse("00000000-0000-0000-0000-000000000021")
	itemID := uuid.MustParse("00000000-0000-0000-0000-000000000022")

	store := &fakeProductQuerier{
		cart:    db.Cart{ID: cartID, Status: "active"},
		product: db.Product{ID: productID, IsActive: true, PriceCents: 999},
		cartItem: db.CartItem{
			ID:        itemID,
			CartID:    cartID,
			ProductID: productID,
			Quantity:  3,
		},
		updateItem: db.CartItem{
			ID:                 itemID,
			CartID:             cartID,
			ProductID:          productID,
			Quantity:           5,
			PriceCentsSnapshot: 999,
			CreatedAt:          time.Now().UTC(),
		},
	}

	body := `{"cart_id":"00000000-0000-0000-0000-000000000020","product_id":"00000000-0000-0000-0000-000000000021","quantity":2}`
	req := httptest.NewRequest(http.MethodPost, "/cart/items", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if store.lastUpdateArg.ID != itemID || store.lastUpdateArg.Quantity != 5 {
		t.Fatalf("unexpected update args: %+v", store.lastUpdateArg)
	}
}

func TestPostCartItemsUsesUserActorResolvedCart(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")

	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000023")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000024")
	productID := uuid.MustParse("00000000-0000-0000-0000-000000000025")

	store := &fakeProductQuerier{
		activeUserCart: db.Cart{
			ID:     cartID,
			UserID: uuid.NullUUID{UUID: userID, Valid: true},
			Status: "active",
		},
		product:     db.Product{ID: productID, IsActive: true, PriceCents: 1499},
		cartItemErr: sql.ErrNoRows,
		createItem: db.CartItem{
			ID:                 uuid.MustParse("00000000-0000-0000-0000-000000000026"),
			CartID:             cartID,
			ProductID:          productID,
			Quantity:           1,
			PriceCentsSnapshot: 1499,
			CreatedAt:          time.Now().UTC(),
		},
	}

	server := NewServer(store)
	token, err := server.jwt.CreateToken(userID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	body := `{"product_id":"00000000-0000-0000-0000-000000000025","quantity":1}`
	req := httptest.NewRequest(http.MethodPost, "/cart/items", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}
	if store.lastGetActiveCartByUserID != userID {
		t.Fatalf("expected user cart lookup by %s, got %s", userID, store.lastGetActiveCartByUserID)
	}
	if store.lastCreateArg.CartID != cartID {
		t.Fatalf("expected cart id %s, got %s", cartID, store.lastCreateArg.CartID)
	}
}

func TestPostCartItemsBadRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/cart/items", strings.NewReader(`{"cart_id":"bad","product_id":"bad","quantity":0}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(&fakeProductQuerier{}).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestPostCartItemsTrailingSlash(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/cart/items/", strings.NewReader(`{"quantity":0}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(&fakeProductQuerier{}).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestPostCartItemsAPIPrefix(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/cart/items", strings.NewReader(`{"quantity":0}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(&fakeProductQuerier{}).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestPatchCartItemsSuccess(t *testing.T) {
	itemID := uuid.MustParse("00000000-0000-0000-0000-000000000060")
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000061")

	store := &fakeProductQuerier{
		activeSessionCart: db.Cart{ID: cartID, Status: "active"},
		cartItem: db.CartItem{
			ID:                 itemID,
			CartID:             cartID,
			ProductID:          uuid.MustParse("00000000-0000-0000-0000-000000000062"),
			Quantity:           1,
			PriceCentsSnapshot: 1999,
			CreatedAt:          time.Now().UTC(),
		},
		updateItem: db.CartItem{
			ID:                 itemID,
			CartID:             cartID,
			ProductID:          uuid.MustParse("00000000-0000-0000-0000-000000000062"),
			Quantity:           4,
			PriceCentsSnapshot: 1999,
			CreatedAt:          time.Now().UTC(),
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/cart/items/00000000-0000-0000-0000-000000000060", strings.NewReader(`{"quantity":4}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if store.lastGetItemID != itemID {
		t.Fatalf("expected get cart item id %s, got %s", itemID, store.lastGetItemID)
	}
	if store.lastUpdateArg.ID != itemID || store.lastUpdateArg.Quantity != 4 {
		t.Fatalf("unexpected update args: %+v", store.lastUpdateArg)
	}
}

func TestPatchCartItemsBadID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/cart/items/not-a-uuid", strings.NewReader(`{"quantity":1}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(&fakeProductQuerier{}).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestPatchCartItemsBadRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPatch, "/cart/items/00000000-0000-0000-0000-000000000070", strings.NewReader(`{"quantity":0}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(&fakeProductQuerier{}).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestPatchCartItemsNotFound(t *testing.T) {
	store := &fakeProductQuerier{cartItemErr: sql.ErrNoRows}
	req := httptest.NewRequest(http.MethodPatch, "/cart/items/00000000-0000-0000-0000-000000000080", strings.NewReader(`{"quantity":2}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestDeleteCartItemsSuccess(t *testing.T) {
	itemID := uuid.MustParse("00000000-0000-0000-0000-000000000090")
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000091")
	store := &fakeProductQuerier{
		activeSessionCart: db.Cart{ID: cartID, Status: "active"},
		cartItem: db.CartItem{
			ID:     itemID,
			CartID: cartID,
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/cart/items/00000000-0000-0000-0000-000000000090", nil)
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}
	if store.lastRemoveID != itemID {
		t.Fatalf("expected remove cart item id %s, got %s", itemID, store.lastRemoveID)
	}
}

func TestDeleteCartItemsBadID(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/cart/items/not-a-uuid", nil)
	rr := httptest.NewRecorder()

	NewServer(&fakeProductQuerier{}).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestDeleteCartItemsNotFound(t *testing.T) {
	store := &fakeProductQuerier{cartItemErr: sql.ErrNoRows}
	req := httptest.NewRequest(http.MethodDelete, "/cart/items/00000000-0000-0000-0000-000000000092", nil)
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestDeleteCartItemsInactiveCart(t *testing.T) {
	activeCartID := uuid.MustParse("00000000-0000-0000-0000-000000000194")
	store := &fakeProductQuerier{
		createCart: db.Cart{
			ID:        activeCartID,
			Status:    "active",
			SessionID: sql.NullString{String: "sess_1", Valid: true},
		},
		cartItem: db.CartItem{
			ID:     uuid.MustParse("00000000-0000-0000-0000-000000000093"),
			CartID: uuid.MustParse("00000000-0000-0000-0000-000000000094"),
		},
	}
	req := httptest.NewRequest(http.MethodDelete, "/cart/items/00000000-0000-0000-0000-000000000093", nil)
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestPostCartItemsResolveCartError(t *testing.T) {
	store := &fakeProductQuerier{activeSessionCartErr: errors.New("db down")}
	body := `{"cart_id":"00000000-0000-0000-0000-000000000030","product_id":"00000000-0000-0000-0000-000000000031","quantity":1}`
	req := httptest.NewRequest(http.MethodPost, "/cart/items", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestPostCartItemsProductInactive(t *testing.T) {
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000095")
	body := `{"cart_id":"00000000-0000-0000-0000-000000000095","product_id":"00000000-0000-0000-0000-000000000096","quantity":1}`
	store := &fakeProductQuerier{
		cart:    db.Cart{ID: cartID, Status: "active"},
		product: db.Product{ID: uuid.MustParse("00000000-0000-0000-0000-000000000096"), IsActive: false},
	}
	req := httptest.NewRequest(http.MethodPost, "/cart/items", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetCartItemsSuccess(t *testing.T) {
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000040")
	store := &fakeProductQuerier{
		activeSessionCart: db.Cart{ID: cartID, Status: "active"},
		listItems: []db.CartItem{
			{
				ID:                 uuid.MustParse("00000000-0000-0000-0000-000000000041"),
				CartID:             cartID,
				ProductID:          uuid.MustParse("00000000-0000-0000-0000-000000000042"),
				Quantity:           2,
				PriceCentsSnapshot: 1999,
				CreatedAt:          time.Now().UTC(),
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/cart/00000000-0000-0000-0000-000000000040/items", nil)
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if store.lastListCartID != cartID {
		t.Fatalf("expected list cart id %s, got %s", cartID, store.lastListCartID)
	}
}

func TestGetCartItemsBadID(t *testing.T) {
	store := &fakeProductQuerier{}
	req := httptest.NewRequest(http.MethodGet, "/cart/not-a-uuid/items", nil)
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
	if store.lastCreateCartArg.Status != "" {
		t.Fatalf("expected no cart creation for invalid read request")
	}
}

func TestGetCartItemsNotFound(t *testing.T) {
	store := &fakeProductQuerier{
		createCart: db.Cart{
			ID:        uuid.MustParse("00000000-0000-0000-0000-000000000151"),
			Status:    "active",
			SessionID: sql.NullString{String: "sess_2", Valid: true},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/cart/00000000-0000-0000-0000-000000000050/items", nil)
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestGetCartItemsResolveCartError(t *testing.T) {
	store := &fakeProductQuerier{activeSessionCartErr: errors.New("db down")}
	req := httptest.NewRequest(http.MethodGet, "/cart/00000000-0000-0000-0000-000000000050/items", nil)
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestPatchCartItemsInactiveCart(t *testing.T) {
	activeCartID := uuid.MustParse("00000000-0000-0000-0000-000000000197")
	store := &fakeProductQuerier{
		createCart: db.Cart{
			ID:        activeCartID,
			Status:    "active",
			SessionID: sql.NullString{String: "sess_3", Valid: true},
		},
		cartItem: db.CartItem{
			ID:     uuid.MustParse("00000000-0000-0000-0000-000000000098"),
			CartID: uuid.MustParse("00000000-0000-0000-0000-000000000097"),
		},
	}
	req := httptest.NewRequest(http.MethodPatch, "/cart/items/00000000-0000-0000-0000-000000000098", strings.NewReader(`{"quantity":2}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestPatchCartItemsResolveCartError(t *testing.T) {
	store := &fakeProductQuerier{activeSessionCartErr: errors.New("db down")}
	req := httptest.NewRequest(http.MethodPatch, "/cart/items/00000000-0000-0000-0000-000000000098", strings.NewReader(`{"quantity":2}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestDeleteCartItemsRemoveError(t *testing.T) {
	itemID := uuid.MustParse("00000000-0000-0000-0000-000000000099")
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000100")
	store := &fakeProductQuerier{
		activeSessionCart: db.Cart{ID: cartID, Status: "active"},
		removeErr:         errors.New("delete failed"),
		cartItem: db.CartItem{
			ID:     itemID,
			CartID: cartID,
		},
	}
	req := httptest.NewRequest(http.MethodDelete, "/cart/items/00000000-0000-0000-0000-000000000099", nil)
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestDeleteCartItemsResolveCartError(t *testing.T) {
	store := &fakeProductQuerier{activeSessionCartErr: errors.New("db down")}
	req := httptest.NewRequest(http.MethodDelete, "/cart/items/00000000-0000-0000-0000-000000000099", nil)
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestGetCartItemsListError(t *testing.T) {
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000101")
	store := &fakeProductQuerier{
		activeSessionCart: db.Cart{ID: cartID, Status: "active"},
		listErr:           errors.New("list failed"),
	}
	req := httptest.NewRequest(http.MethodGet, "/cart/00000000-0000-0000-0000-000000000101/items", nil)
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestCreateCheckoutSessionSuccess(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	t.Setenv("CHECKOUT_SUCCESS_URL", "https://example.com/success")
	t.Setenv("CHECKOUT_CANCEL_URL", "https://example.com/cancel")
	t.Setenv("JWT_SECRET", "resolve-secret")

	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000110")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000111")
	productID := uuid.MustParse("00000000-0000-0000-0000-000000000112")
	store := &fakeProductQuerier{
		cart: db.Cart{
			ID:     cartID,
			UserID: uuid.NullUUID{UUID: userID, Valid: true},
			Status: "active",
		},
		listItems: []db.CartItem{{
			ID:                 uuid.MustParse("00000000-0000-0000-0000-000000000113"),
			CartID:             cartID,
			ProductID:          productID,
			Quantity:           2,
			PriceCentsSnapshot: 1999,
		}},
		product: db.Product{
			ID:       productID,
			Name:     "Basic Tee",
			Currency: "USD",
			IsActive: true,
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

	token, err := server.jwt.CreateToken(userID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/checkout/session", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if stripe.lastCreateInput.Metadata["cart_id"] != cartID.String() {
		t.Fatalf("expected cart_id metadata to be %s, got %s", cartID.String(), stripe.lastCreateInput.Metadata["cart_id"])
	}
	if len(stripe.lastCreateInput.LineItems) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(stripe.lastCreateInput.LineItems))
	}
}

func TestCreateCheckoutSessionCartWithoutUser(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	t.Setenv("CHECKOUT_SUCCESS_URL", "https://example.com/success")
	t.Setenv("CHECKOUT_CANCEL_URL", "https://example.com/cancel")
	t.Setenv("JWT_SECRET", "resolve-secret")

	req := httptest.NewRequest(http.MethodPost, "/checkout/session", strings.NewReader(`{"cart_id":"00000000-0000-0000-0000-000000000114"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(&fakeProductQuerier{}).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestCreateCheckoutSessionEmptyCart(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	t.Setenv("CHECKOUT_SUCCESS_URL", "https://example.com/success")
	t.Setenv("CHECKOUT_CANCEL_URL", "https://example.com/cancel")
	t.Setenv("JWT_SECRET", "resolve-secret")

	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000115")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000116")
	store := &fakeProductQuerier{
		cart: db.Cart{
			ID:     cartID,
			UserID: uuid.NullUUID{UUID: userID, Valid: true},
			Status: "active",
		},
	}

	server := NewServer(store)
	token, err := server.jwt.CreateToken(userID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/checkout/session", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestCreateCheckoutSessionResolveCartError(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	t.Setenv("CHECKOUT_SUCCESS_URL", "https://example.com/success")
	t.Setenv("CHECKOUT_CANCEL_URL", "https://example.com/cancel")
	t.Setenv("JWT_SECRET", "resolve-secret")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000117")
	store := &fakeProductQuerier{
		activeUserCartErr: errors.New("db down"),
	}
	server := NewServer(store)
	token, err := server.jwt.CreateToken(userID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/checkout/session", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	rr := httptest.NewRecorder()

	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestListOrdersByUserSuccess(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000201")
	store := &fakeProductQuerier{
		orders: []db.Order{
			{
				ID:            uuid.MustParse("00000000-0000-0000-0000-000000000202"),
				UserID:        userID,
				Status:        "paid",
				Currency:      "USD",
				SubtotalCents: 1999,
				TotalCents:    1999,
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
			},
		},
	}
	server := NewServer(store)
	token, err := server.jwt.CreateToken(userID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/orders?limit=20&offset=0", nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	rr := httptest.NewRecorder()
	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if store.lastListOrdersArg.UserID != userID {
		t.Fatalf("expected user id %s, got %s", userID, store.lastListOrdersArg.UserID)
	}
	if store.lastListOrdersArg.Limit != 20 || store.lastListOrdersArg.Offset != 0 {
		t.Fatalf("unexpected pagination: %+v", store.lastListOrdersArg)
	}
}

func TestListOrdersByUserUnauthorized(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	rr := httptest.NewRecorder()

	NewServer(&fakeProductQuerier{}).Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestGetOrderByIDSuccess(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000301")
	orderID := uuid.MustParse("00000000-0000-0000-0000-000000000302")
	productID := uuid.MustParse("00000000-0000-0000-0000-000000000303")
	store := &fakeProductQuerier{
		orderByID: db.Order{
			ID:            orderID,
			UserID:        userID,
			Status:        "paid",
			Currency:      "USD",
			SubtotalCents: 3000,
			TotalCents:    3300,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		},
		orderItems: []db.OrderItem{
			{
				ID:                  uuid.MustParse("00000000-0000-0000-0000-000000000304"),
				OrderID:             orderID,
				ProductID:           productID,
				ProductNameSnapshot: "Pro Tee",
				PriceCentsSnapshot:  1500,
				Quantity:            2,
				CreatedAt:           time.Now().UTC(),
			},
		},
	}
	server := NewServer(store)
	token, err := server.jwt.CreateToken(userID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String(), nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	rr := httptest.NewRecorder()
	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if store.lastGetOrderID != orderID {
		t.Fatalf("expected get order id %s, got %s", orderID, store.lastGetOrderID)
	}
	if store.lastListOrderItemsOrderID != orderID {
		t.Fatalf("expected list order items id %s, got %s", orderID, store.lastListOrderItemsOrderID)
	}
}

func TestGetOrderByIDOwnershipDenied(t *testing.T) {
	t.Setenv("JWT_SECRET", "resolve-secret")
	authUserID := uuid.MustParse("00000000-0000-0000-0000-000000000311")
	ownerUserID := uuid.MustParse("00000000-0000-0000-0000-000000000312")
	orderID := uuid.MustParse("00000000-0000-0000-0000-000000000313")
	store := &fakeProductQuerier{
		orderByID: db.Order{
			ID:            orderID,
			UserID:        ownerUserID,
			Status:        "paid",
			Currency:      "USD",
			SubtotalCents: 3000,
			TotalCents:    3300,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		},
	}
	server := NewServer(store)
	token, err := server.jwt.CreateToken(authUserID)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/orders/"+orderID.String(), nil)
	req.AddCookie(&http.Cookie{Name: authCookieName, Value: token})
	rr := httptest.NewRecorder()
	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestStripeWebhookCheckoutCompletedSuccess(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_123")

	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000120")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000121")
	productID := uuid.MustParse("00000000-0000-0000-0000-000000000122")
	orderID := uuid.MustParse("00000000-0000-0000-0000-000000000123")
	cartItemID := uuid.MustParse("00000000-0000-0000-0000-000000000124")

	store := &fakeProductQuerier{
		paymentLookupErr: sql.ErrNoRows,
		cart: db.Cart{
			ID:     cartID,
			UserID: uuid.NullUUID{UUID: userID, Valid: true},
			Status: "active",
		},
		listItems: []db.CartItem{{
			ID:                 cartItemID,
			CartID:             cartID,
			ProductID:          productID,
			Quantity:           2,
			PriceCentsSnapshot: 1999,
		}},
		product: db.Product{
			ID:       productID,
			Name:     "Basic Tee",
			Currency: "USD",
			IsActive: true,
		},
		order: db.Order{ID: orderID},
	}

	rawObject := []byte(`{"id":"cs_test_123","payment_intent":"pi_test_123","amount_total":3998,"currency":"usd","metadata":{"cart_id":"00000000-0000-0000-0000-000000000120"}}`)
	var evt stripeEvent
	evt.Type = "checkout.session.completed"
	evt.Data.Object = rawObject

	stripe := &fakeStripeGateway{webhookEvent: evt}
	server := NewServer(store)
	server.stripe = stripe

	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(`{}`))
	req.Header.Set("Stripe-Signature", "t=1,v1=sig")
	rr := httptest.NewRecorder()
	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if store.lastCreateOrderArg.UserID != userID {
		t.Fatalf("expected order user id %s, got %s", userID, store.lastCreateOrderArg.UserID)
	}
	if store.lastCreatePaymentArg.CheckoutSessionID.String != "cs_test_123" {
		t.Fatalf("expected checkout session id cs_test_123, got %q", store.lastCreatePaymentArg.CheckoutSessionID.String)
	}
	if store.lastUpdateCartStatusArg.Status != "converted" {
		t.Fatalf("expected cart status converted, got %q", store.lastUpdateCartStatusArg.Status)
	}
	if store.lastRemoveID != cartItemID {
		t.Fatalf("expected removed cart item id %s, got %s", cartItemID, store.lastRemoveID)
	}
}

func TestStripeWebhookCheckoutCompletedIdempotent(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_123")

	store := &fakeProductQuerier{
		paymentLookup: db.Payment{ID: uuid.MustParse("00000000-0000-0000-0000-000000000130")},
	}

	rawObject := []byte(`{"id":"cs_test_existing","metadata":{"cart_id":"00000000-0000-0000-0000-000000000120"}}`)
	var evt stripeEvent
	evt.Type = "checkout.session.completed"
	evt.Data.Object = rawObject

	stripe := &fakeStripeGateway{webhookEvent: evt}
	server := NewServer(store)
	server.stripe = stripe

	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(`{}`))
	req.Header.Set("Stripe-Signature", "t=1,v1=sig")
	rr := httptest.NewRecorder()
	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if store.lastCreateOrderArg.Status != "" {
		t.Fatalf("expected no order creation for duplicate webhook")
	}
	if store.lastPaymentLookupSessionID != "cs_test_existing" {
		t.Fatalf("expected payment lookup by cs_test_existing, got %q", store.lastPaymentLookupSessionID)
	}
}

func TestStripeWebhookInvalidSignature(t *testing.T) {
	_ = os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_123")
	t.Cleanup(func() { _ = os.Unsetenv("STRIPE_WEBHOOK_SECRET") })
	server := NewServer(&fakeProductQuerier{})
	server.stripe = &fakeStripeGateway{webhookErr: errors.New("invalid")}

	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestStripeWebhookMissingSecret(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SECRET", "")
	server := NewServer(&fakeProductQuerier{})
	server.stripe = &fakeStripeGateway{}

	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestStripeWebhookInvalidCheckoutSessionPayload(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_123")
	var evt stripeEvent
	evt.Type = "checkout.session.completed"
	evt.Data.Object = []byte(`{"id":1`)

	server := NewServer(&fakeProductQuerier{})
	server.stripe = &fakeStripeGateway{webhookEvent: evt}

	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(`{}`))
	req.Header.Set("Stripe-Signature", "t=1,v1=sig")
	rr := httptest.NewRecorder()
	server.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}
