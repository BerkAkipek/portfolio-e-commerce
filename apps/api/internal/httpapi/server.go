package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	db "github.com/BerkAkipek/e-commerce-app/api/internal/db"
)

type ProductQuerier interface {
	ListActiveProducts(ctx context.Context, arg db.ListActiveProductsParams) ([]db.Product, error)
	GetProductBySlug(ctx context.Context, lower string) (db.Product, error)
	ListCategoriesByProductID(ctx context.Context, productID uuid.UUID) ([]db.Category, error)
	GetActiveCartByUserID(ctx context.Context, userID uuid.UUID) (db.Cart, error)
	GetActiveCartBySessionID(ctx context.Context, sessionID string) (db.Cart, error)
	CreateCart(ctx context.Context, arg db.CreateCartParams) (db.Cart, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error)
	GetUserByEmail(ctx context.Context, lower string) (db.User, error)
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	CreateRefreshToken(ctx context.Context, arg db.CreateRefreshTokenParams) (db.RefreshToken, error)
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (db.RefreshToken, error)
	RevokeRefreshTokenByHash(ctx context.Context, tokenHash string) (db.RefreshToken, error)
	RotateRefreshToken(ctx context.Context, arg db.RotateRefreshTokenParams) (db.RefreshToken, error)
	GetCartByID(ctx context.Context, id uuid.UUID) (db.Cart, error)
	GetProductByID(ctx context.Context, id uuid.UUID) (db.Product, error)
	GetCartItemByID(ctx context.Context, id uuid.UUID) (db.CartItem, error)
	GetCartItemByCartAndProduct(ctx context.Context, arg db.GetCartItemByCartAndProductParams) (db.CartItem, error)
	ListCartItemsByCartID(ctx context.Context, cartID uuid.UUID) ([]db.CartItem, error)
	CreateCartItem(ctx context.Context, arg db.CreateCartItemParams) (db.CartItem, error)
	UpdateCartItemQuantity(ctx context.Context, arg db.UpdateCartItemQuantityParams) (db.CartItem, error)
	RemoveCartItem(ctx context.Context, id uuid.UUID) error
	CreateOrder(ctx context.Context, arg db.CreateOrderParams) (db.Order, error)
	ListOrdersByUserID(ctx context.Context, arg db.ListOrdersByUserIDParams) ([]db.Order, error)
	CreateOrderItem(ctx context.Context, arg db.CreateOrderItemParams) (db.OrderItem, error)
	CreatePayment(ctx context.Context, arg db.CreatePaymentParams) (db.Payment, error)
	GetPaymentByCheckoutSessionID(ctx context.Context, checkoutSessionID string) (db.Payment, error)
	UpdateCartStatus(ctx context.Context, arg db.UpdateCartStatusParams) (db.Cart, error)
	GetOrderByID(ctx context.Context, id uuid.UUID) (db.Order, error)
	ListOrderItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]db.OrderItem, error)
}

type Server struct {
	products ProductQuerier
	carts    *CartService
	stripe   stripeGateway
	jwt      *JWTUtility
	http     *http.Client
}

const (
	defaultAccessTokenTTL  = 15 * time.Minute
	defaultRefreshTokenTTL = 30 * 24 * time.Hour
	defaultJWTIssuer       = "ecommerce-api"
	defaultBcryptCost      = 12
)

func NewServer(products ProductQuerier) *Server {
	var jwtUtil *JWTUtility
	if jwtSecret := os.Getenv("JWT_SECRET"); strings.TrimSpace(jwtSecret) != "" {
		issuer := strings.TrimSpace(os.Getenv("JWT_ISSUER"))
		if issuer == "" {
			issuer = defaultJWTIssuer
		}
		util, err := NewJWTUtility(jwtSecret, defaultAccessTokenTTL, issuer)
		if err == nil {
			jwtUtil = util
		}
	}

	return &Server{
		products: products,
		carts:    NewCartService(products),
		stripe:   newStripeGateway(http.DefaultClient),
		jwt:      jwtUtil,
		http:     http.DefaultClient,
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)
	r.Use(s.resolveAuthUser)
	r.Use(s.resolveGuestSession)

	s.registerRoutes(r)
	r.Route("/api", func(api chi.Router) {
		s.registerRoutes(api)
	})
	return r
}

func (s *Server) registerRoutes(r chi.Router) {
	r.Get("/health", s.handleHealth)
	r.Get("/products", s.handleListProducts)
	r.Get("/products/{slug}", s.handleGetProductBySlug)
	r.With(s.requireAuth).Get("/orders", s.handleListOrdersByUser)
	r.With(s.requireAuth).Get("/orders/{id}", s.handleGetOrderByID)
	r.Post("/auth/register", s.handleRegister)
	r.Post("/auth/login", s.handleLogin)
	r.Get("/auth/google/start", s.handleGoogleAuthStart)
	r.Get("/auth/google/callback", s.handleGoogleAuthCallback)
	r.Post("/auth/dev-login", s.handleDevLogin)
	r.Post("/auth/logout", s.handleLogout)
	r.Post("/cart/items", s.handleCreateCartItem)
	r.Patch("/cart/items/{id}", s.handlePatchCartItem)
	r.Delete("/cart/items/{id}", s.handleDeleteCartItem)
	r.Get("/cart/{id}/items", s.handleGetCartItems)
	r.With(s.requireAuth).Post("/checkout/session", s.handleCreateCheckoutSession)
	r.Post("/webhooks/stripe", s.handleStripeWebhook)
}

type productsResponse struct {
	Data   []productDTO `json:"data"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type productResponse struct {
	Data productDTO `json:"data"`
}

type productDTO struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	PriceCents  int32    `json:"price_cents"`
	Currency    string   `json:"currency"`
	Stock       int32    `json:"stock"`
	IsActive    bool     `json:"is_active"`
	SKU         string   `json:"sku"`
	WeightGrams *int32   `json:"weight_grams,omitempty"`
	ImageURL    *string  `json:"image_url,omitempty"`
	Categories  []string `json:"categories,omitempty"`
}

type createCartItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

type patchCartItemRequest struct {
	Quantity int32 `json:"quantity"`
}

type cartItemDTO struct {
	ID                 string    `json:"id"`
	CartID             string    `json:"cart_id"`
	ProductID          string    `json:"product_id"`
	Quantity           int32     `json:"quantity"`
	PriceCentsSnapshot int32     `json:"price_cents_snapshot"`
	CreatedAt          time.Time `json:"created_at"`
}

type cartItemResponse struct {
	Data    cartItemDTO `json:"data"`
	Created bool        `json:"created"`
}

type cartItemsResponse struct {
	Data []cartItemDTO `json:"data"`
}

type createCheckoutSessionRequest struct {
	SuccessURL string `json:"success_url"`
	CancelURL  string `json:"cancel_url"`
}

type devLoginRequest struct {
	UserID string `json:"user_id"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type checkoutSessionResponse struct {
	Data struct {
		SessionID string `json:"session_id"`
		URL       string `json:"url"`
	} `json:"data"`
}

type ordersResponse struct {
	Data   []orderDTO `json:"data"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}

type orderDTO struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	Status        string    `json:"status"`
	Currency      string    `json:"currency"`
	SubtotalCents int32     `json:"subtotal_cents"`
	TotalCents    int32     `json:"total_cents"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type orderDetailResponse struct {
	Data orderDetailDTO `json:"data"`
}

type orderDetailDTO struct {
	Order orderDTO       `json:"order"`
	Items []orderItemDTO `json:"items"`
}

type orderItemDTO struct {
	ID                 string    `json:"id"`
	OrderID            string    `json:"order_id"`
	ProductID          string    `json:"product_id"`
	ProductName        string    `json:"product_name"`
	PriceCentsSnapshot int32     `json:"price_cents_snapshot"`
	Quantity           int32     `json:"quantity"`
	CreatedAt          time.Time `json:"created_at"`
}

type googleOAuthConfig struct {
	ClientID      string
	ClientSecret  string
	RedirectURL   string
	PostLoginPath string
	AuthURL       string
	TokenURL      string
	UserinfoURL   string
}

type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type googleUserinfoResponse struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	products, err := s.products.ListActiveProducts(r.Context(), db.ListActiveProductsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list products")
		return
	}

	response := productsResponse{Data: make([]productDTO, 0, len(products)), Limit: limit, Offset: offset}
	for _, p := range products {
		categories, categoryErr := s.products.ListCategoriesByProductID(r.Context(), p.ID)
		if categoryErr != nil {
			writeError(w, http.StatusInternalServerError, "failed to list product categories")
			return
		}
		response.Data = append(response.Data, mapProduct(p, categories))
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetProductBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "slug is required")
		return
	}

	product, err := s.products.GetProductBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get product")
		return
	}

	categories, err := s.products.ListCategoriesByProductID(r.Context(), product.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list product categories")
		return
	}

	writeJSON(w, http.StatusOK, productResponse{Data: mapProduct(product, categories)})
}

func (s *Server) handleCreateCartItem(w http.ResponseWriter, r *http.Request) {
	var req createCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Quantity <= 0 {
		writeError(w, http.StatusBadRequest, "quantity must be greater than 0")
		return
	}

	cartID, err := s.carts.ResolveCart(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve cart")
		return
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "product_id must be a valid uuid")
		return
	}

	product, err := s.products.GetProductByID(r.Context(), productID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get product")
		return
	}
	if !product.IsActive {
		writeError(w, http.StatusBadRequest, "product is not active")
		return
	}

	existing, err := s.products.GetCartItemByCartAndProduct(r.Context(), db.GetCartItemByCartAndProductParams{
		CartID:    cartID,
		ProductID: productID,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "failed to get cart item")
		return
	}

	if errors.Is(err, sql.ErrNoRows) {
		createdItem, createErr := s.products.CreateCartItem(r.Context(), db.CreateCartItemParams{
			CartID:             cartID,
			ProductID:          productID,
			Quantity:           req.Quantity,
			PriceCentsSnapshot: product.PriceCents,
		})
		if createErr != nil {
			writeError(w, http.StatusInternalServerError, "failed to create cart item")
			return
		}
		writeJSON(w, http.StatusCreated, cartItemResponse{
			Data:    mapCartItem(createdItem),
			Created: true,
		})
		return
	}

	updatedQty := existing.Quantity + req.Quantity
	updatedItem, updateErr := s.products.UpdateCartItemQuantity(r.Context(), db.UpdateCartItemQuantityParams{
		ID:       existing.ID,
		Quantity: updatedQty,
	})
	if updateErr != nil {
		writeError(w, http.StatusInternalServerError, "failed to update cart item")
		return
	}

	writeJSON(w, http.StatusOK, cartItemResponse{
		Data:    mapCartItem(updatedItem),
		Created: false,
	})
}

func (s *Server) handleGetCartItems(w http.ResponseWriter, r *http.Request) {
	cartIDRaw := chi.URLParam(r, "id")
	requestCartID, err := uuid.Parse(cartIDRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a valid uuid")
		return
	}

	cartID, err := s.carts.ResolveExistingCart(r.Context())
	if err != nil {
		if errors.Is(err, ErrActiveCartNotFound) {
			writeError(w, http.StatusNotFound, "cart not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to resolve cart")
		return
	}
	if requestCartID != cartID {
		writeError(w, http.StatusNotFound, "cart not found")
		return
	}

	items, err := s.products.ListCartItemsByCartID(r.Context(), cartID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list cart items")
		return
	}

	response := cartItemsResponse{Data: make([]cartItemDTO, 0, len(items))}
	for _, item := range items {
		response.Data = append(response.Data, mapCartItem(item))
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handlePatchCartItem(w http.ResponseWriter, r *http.Request) {
	cartItemIDRaw := chi.URLParam(r, "id")
	cartItemID, err := uuid.Parse(cartItemIDRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a valid uuid")
		return
	}

	var req patchCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Quantity <= 0 {
		writeError(w, http.StatusBadRequest, "quantity must be greater than 0")
		return
	}

	resolvedCartID, err := s.carts.ResolveExistingCart(r.Context())
	if err != nil {
		if errors.Is(err, ErrActiveCartNotFound) {
			writeError(w, http.StatusNotFound, "cart item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to resolve cart")
		return
	}

	item, err := s.products.GetCartItemByID(r.Context(), cartItemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "cart item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get cart item")
		return
	}

	if item.CartID != resolvedCartID {
		writeError(w, http.StatusNotFound, "cart item not found")
		return
	}

	updatedItem, err := s.products.UpdateCartItemQuantity(r.Context(), db.UpdateCartItemQuantityParams{
		ID:       cartItemID,
		Quantity: req.Quantity,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update cart item")
		return
	}

	writeJSON(w, http.StatusOK, map[string]cartItemDTO{
		"data": mapCartItem(updatedItem),
	})
}

func (s *Server) handleDeleteCartItem(w http.ResponseWriter, r *http.Request) {
	cartItemIDRaw := chi.URLParam(r, "id")
	cartItemID, err := uuid.Parse(cartItemIDRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a valid uuid")
		return
	}

	resolvedCartID, err := s.carts.ResolveExistingCart(r.Context())
	if err != nil {
		if errors.Is(err, ErrActiveCartNotFound) {
			writeError(w, http.StatusNotFound, "cart item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to resolve cart")
		return
	}

	item, err := s.products.GetCartItemByID(r.Context(), cartItemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "cart item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get cart item")
		return
	}

	if item.CartID != resolvedCartID {
		writeError(w, http.StatusNotFound, "cart item not found")
		return
	}

	if err := s.products.RemoveCartItem(r.Context(), cartItemID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove cart item")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDevLogin(w http.ResponseWriter, r *http.Request) {
	if !isDevelopmentAuthEnabled() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if s.jwt == nil {
		writeError(w, http.StatusServiceUnavailable, "jwt auth is not configured")
		return
	}

	var req devLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "user_id must be a valid uuid")
		return
	}

	_, err = s.products.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	if err := s.issueAuthSession(r.Context(), w, userID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"user_id": userID.String()})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.jwt == nil {
		writeError(w, http.StatusServiceUnavailable, "jwt auth is not configured")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email := strings.TrimSpace(req.Email)
	if email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := s.products.GetUserByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}
	if !user.PasswordHash.Valid || strings.TrimSpace(user.PasswordHash.String) == "" {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash.String), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := s.issueAuthSession(r.Context(), w, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if s.jwt == nil {
		writeError(w, http.StatusServiceUnavailable, "jwt auth is not configured")
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email := strings.TrimSpace(req.Email)
	password := req.Password
	fullName := strings.TrimSpace(req.FullName)
	if _, err := mail.ParseAddress(email); err != nil {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}
	if len(password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if fullName == "" {
		writeError(w, http.StatusBadRequest, "full_name is required")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), defaultBcryptCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := s.products.CreateUser(r.Context(), db.CreateUserParams{
		Email:        email,
		PasswordHash: sql.NullString{String: string(passwordHash), Valid: true},
		Phone:        sql.NullString{},
		FullName:     fullName,
		Role:         "user",
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	if err := s.issueAuthSession(r.Context(), w, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func readGoogleOAuthConfig() (googleOAuthConfig, bool) {
	cfg := googleOAuthConfig{
		ClientID:      strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_ID")),
		ClientSecret:  strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET")),
		RedirectURL:   strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_REDIRECT_URL")),
		PostLoginPath: strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_POST_LOGIN_URL")),
		AuthURL:       strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_AUTH_URL")),
		TokenURL:      strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_TOKEN_URL")),
		UserinfoURL:   strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_USERINFO_URL")),
	}
	if cfg.PostLoginPath == "" {
		cfg.PostLoginPath = "/catalog"
	}
	if cfg.AuthURL == "" {
		cfg.AuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
	}
	if cfg.TokenURL == "" {
		cfg.TokenURL = "https://oauth2.googleapis.com/token"
	}
	if cfg.UserinfoURL == "" {
		cfg.UserinfoURL = "https://openidconnect.googleapis.com/v1/userinfo"
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RedirectURL == "" {
		return cfg, false
	}
	return cfg, true
}

func (s *Server) handleGoogleAuthStart(w http.ResponseWriter, r *http.Request) {
	if s.jwt == nil {
		writeError(w, http.StatusServiceUnavailable, "jwt auth is not configured")
		return
	}
	cfg, ok := readGoogleOAuthConfig()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "google oauth is not configured")
		return
	}

	state, err := GenerateRefreshToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to initialize google oauth")
		return
	}
	setCookieWithTTL(w, googleOAuthStateCookieName, state, 10*time.Minute)

	authURL, err := url.Parse(cfg.AuthURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid google oauth auth url")
		return
	}
	query := authURL.Query()
	query.Set("client_id", cfg.ClientID)
	query.Set("redirect_uri", cfg.RedirectURL)
	query.Set("response_type", "code")
	query.Set("scope", "openid email profile")
	query.Set("state", state)
	query.Set("access_type", "online")
	query.Set("prompt", "select_account")
	authURL.RawQuery = query.Encode()

	http.Redirect(w, r, authURL.String(), http.StatusFound)
}

func (s *Server) handleGoogleAuthCallback(w http.ResponseWriter, r *http.Request) {
	if s.jwt == nil {
		writeError(w, http.StatusServiceUnavailable, "jwt auth is not configured")
		return
	}
	cfg, ok := readGoogleOAuthConfig()
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "google oauth is not configured")
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if code == "" || state == "" {
		http.Redirect(w, r, "/auth?error=google_oauth", http.StatusFound)
		return
	}

	stateCookie, err := ReadCookie(r, googleOAuthStateCookieName)
	if err != nil || stateCookie != state {
		setCookieWithTTL(w, googleOAuthStateCookieName, "", -1*time.Second)
		http.Redirect(w, r, "/auth?error=google_state", http.StatusFound)
		return
	}
	setCookieWithTTL(w, googleOAuthStateCookieName, "", -1*time.Second)

	tokenReqBody := url.Values{}
	tokenReqBody.Set("code", code)
	tokenReqBody.Set("client_id", cfg.ClientID)
	tokenReqBody.Set("client_secret", cfg.ClientSecret)
	tokenReqBody.Set("redirect_uri", cfg.RedirectURL)
	tokenReqBody.Set("grant_type", "authorization_code")

	tokenReq, err := http.NewRequestWithContext(
		r.Context(),
		http.MethodPost,
		cfg.TokenURL,
		strings.NewReader(tokenReqBody.Encode()),
	)
	if err != nil {
		http.Redirect(w, r, "/auth?error=google_token_request", http.StatusFound)
		return
	}
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenReq.Header.Set("Accept", "application/json")

	tokenResp, err := s.http.Do(tokenReq)
	if err != nil {
		http.Redirect(w, r, "/auth?error=google_token_exchange", http.StatusFound)
		return
	}
	defer tokenResp.Body.Close()
	if tokenResp.StatusCode != http.StatusOK {
		http.Redirect(w, r, "/auth?error=google_token_exchange", http.StatusFound)
		return
	}

	var tokenPayload googleTokenResponse
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenPayload); err != nil {
		http.Redirect(w, r, "/auth?error=google_token_response", http.StatusFound)
		return
	}
	if strings.TrimSpace(tokenPayload.AccessToken) == "" {
		http.Redirect(w, r, "/auth?error=google_access_token", http.StatusFound)
		return
	}

	userinfoReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, cfg.UserinfoURL, nil)
	if err != nil {
		http.Redirect(w, r, "/auth?error=google_userinfo_request", http.StatusFound)
		return
	}
	userinfoReq.Header.Set("Authorization", "Bearer "+tokenPayload.AccessToken)
	userinfoReq.Header.Set("Accept", "application/json")

	userinfoResp, err := s.http.Do(userinfoReq)
	if err != nil {
		http.Redirect(w, r, "/auth?error=google_userinfo", http.StatusFound)
		return
	}
	defer userinfoResp.Body.Close()
	if userinfoResp.StatusCode != http.StatusOK {
		http.Redirect(w, r, "/auth?error=google_userinfo", http.StatusFound)
		return
	}

	var userinfo googleUserinfoResponse
	if err := json.NewDecoder(userinfoResp.Body).Decode(&userinfo); err != nil {
		http.Redirect(w, r, "/auth?error=google_userinfo_parse", http.StatusFound)
		return
	}
	email := strings.TrimSpace(userinfo.Email)
	if email == "" {
		http.Redirect(w, r, "/auth?error=google_email_missing", http.StatusFound)
		return
	}

	user, err := s.products.GetUserByEmail(r.Context(), email)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/auth?error=user_lookup", http.StatusFound)
			return
		}

		fullName := strings.TrimSpace(userinfo.Name)
		if fullName == "" {
			fullName = strings.Split(email, "@")[0]
		}
		user, err = s.products.CreateUser(r.Context(), db.CreateUserParams{
			Email:        email,
			PasswordHash: sql.NullString{},
			Phone:        sql.NullString{},
			FullName:     fullName,
			Role:         "user",
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				user, err = s.products.GetUserByEmail(r.Context(), email)
			}
			if err != nil {
				http.Redirect(w, r, "/auth?error=user_create", http.StatusFound)
				return
			}
		}
	}

	if err := s.issueAuthSession(r.Context(), w, user.ID); err != nil {
		http.Redirect(w, r, "/auth?error=session_issue", http.StatusFound)
		return
	}

	http.Redirect(w, r, cfg.PostLoginPath, http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if rawToken, err := ReadRefreshCookie(r); err == nil {
		if tokenHash, hashErr := HashRefreshToken(rawToken); hashErr == nil {
			_, _ = s.products.RevokeRefreshTokenByHash(r.Context(), tokenHash)
		}
	}
	ClearAuthCookies(w)
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) issueAuthSession(ctx context.Context, w http.ResponseWriter, userID uuid.UUID) error {
	token, err := s.jwt.CreateToken(userID)
	if err != nil {
		return errors.New("failed to create auth token")
	}

	refreshTokenRaw, err := GenerateRefreshToken()
	if err != nil {
		return errors.New("failed to create refresh token")
	}
	refreshTokenHash, err := HashRefreshToken(refreshTokenRaw)
	if err != nil {
		return errors.New("failed to hash refresh token")
	}
	if _, err := s.products.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: refreshTokenHash,
		ExpiresAt: time.Now().UTC().Add(defaultRefreshTokenTTL),
	}); err != nil {
		return errors.New("failed to persist refresh token")
	}
	SetAuthCookies(w, token, defaultAccessTokenTTL, refreshTokenRaw, defaultRefreshTokenTTL)
	return nil
}

func (s *Server) handleCreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	var req createCheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userID, ok := AuthUserIDFromContext(r.Context())
	if !ok || userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	cartID, err := s.carts.ResolveExistingCart(r.Context())
	if err != nil {
		if errors.Is(err, ErrActiveCartNotFound) {
			writeError(w, http.StatusBadRequest, "cart is empty")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to resolve cart")
		return
	}

	items, err := s.products.ListCartItemsByCartID(r.Context(), cartID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list cart items")
		return
	}
	if len(items) == 0 {
		writeError(w, http.StatusBadRequest, "cart is empty")
		return
	}

	lineItems := make([]stripeLineItem, 0, len(items))
	currency := ""
	for _, item := range items {
		product, productErr := s.products.GetProductByID(r.Context(), item.ProductID)
		if productErr != nil {
			if errors.Is(productErr, sql.ErrNoRows) {
				writeError(w, http.StatusNotFound, "product not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to get product")
			return
		}
		if !product.IsActive {
			writeError(w, http.StatusBadRequest, "product is not active")
			return
		}

		itemCurrency := strings.ToLower(product.Currency)
		if currency == "" {
			currency = itemCurrency
		} else if currency != itemCurrency {
			writeError(w, http.StatusBadRequest, "all cart items must have the same currency")
			return
		}

		lineItems = append(lineItems, stripeLineItem{
			Name:       product.Name,
			UnitAmount: item.PriceCentsSnapshot,
			Quantity:   item.Quantity,
			Currency:   itemCurrency,
		})
	}

	successURL := req.SuccessURL
	if successURL == "" {
		successURL = os.Getenv("CHECKOUT_SUCCESS_URL")
	}
	cancelURL := req.CancelURL
	if cancelURL == "" {
		cancelURL = os.Getenv("CHECKOUT_CANCEL_URL")
	}
	if successURL == "" || cancelURL == "" {
		writeError(w, http.StatusInternalServerError, "checkout URLs are not configured")
		return
	}

	stripeSecretKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeSecretKey == "" {
		writeError(w, http.StatusInternalServerError, "stripe secret key is not configured")
		return
	}

	session, err := s.stripe.CreateCheckoutSession(r.Context(), stripeCreateCheckoutSessionInput{
		SecretKey:  stripeSecretKey,
		SuccessURL: successURL,
		CancelURL:  cancelURL,
		LineItems:  lineItems,
		Metadata: map[string]string{
			"cart_id": cartID.String(),
			"user_id": userID.String(),
		},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create checkout session")
		return
	}

	var response checkoutSessionResponse
	response.Data.SessionID = session.ID
	response.Data.URL = session.URL
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleListOrdersByUser(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	userID, ok := AuthUserIDFromContext(r.Context())
	if !ok || userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	orders, err := s.products.ListOrdersByUserID(r.Context(), db.ListOrdersByUserIDParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list orders")
		return
	}

	response := ordersResponse{
		Data:   make([]orderDTO, 0, len(orders)),
		Limit:  limit,
		Offset: offset,
	}
	for _, order := range orders {
		response.Data = append(response.Data, mapOrder(order))
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetOrderByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := AuthUserIDFromContext(r.Context())
	if !ok || userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	orderIDRaw := chi.URLParam(r, "id")
	orderID, err := uuid.Parse(orderIDRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	order, err := s.products.GetOrderByID(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get order")
		return
	}
	if order.UserID != userID {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	items, err := s.products.ListOrderItemsByOrderID(r.Context(), orderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list order items")
		return
	}

	response := orderDetailResponse{
		Data: orderDetailDTO{
			Order: mapOrder(order),
			Items: make([]orderItemDTO, 0, len(items)),
		},
	}
	for _, item := range items {
		response.Data.Items = append(response.Data.Items, mapOrderItem(item))
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if webhookSecret == "" {
		writeError(w, http.StatusInternalServerError, "stripe webhook secret is not configured")
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	event, err := s.stripe.ParseWebhook(payload, r.Header.Get("Stripe-Signature"), webhookSecret, 5*time.Minute)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid stripe signature")
		return
	}

	if event.Type == "checkout.session.completed" {
		var session stripeCheckoutSession
		if err := json.Unmarshal(event.Data.Object, &session); err != nil {
			writeError(w, http.StatusBadRequest, "invalid checkout session payload")
			return
		}
		if err := s.processCheckoutSessionCompleted(r.Context(), session); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to process checkout session")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]bool{"received": true})
}

func (s *Server) processCheckoutSessionCompleted(ctx context.Context, session stripeCheckoutSession) error {
	if session.ID == "" {
		return errors.New("checkout session id is required")
	}

	_, err := s.products.GetPaymentByCheckoutSessionID(ctx, session.ID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	cartIDRaw := session.Metadata["cart_id"]
	cartID, err := uuid.Parse(cartIDRaw)
	if err != nil {
		return fmt.Errorf("invalid cart id in metadata: %w", err)
	}

	cart, err := s.products.GetCartByID(ctx, cartID)
	if err != nil {
		return err
	}
	if cart.Status != "active" {
		return nil
	}
	if !cart.UserID.Valid {
		return errors.New("cart is not attached to a user")
	}

	cartItems, err := s.products.ListCartItemsByCartID(ctx, cartID)
	if err != nil {
		return err
	}
	if len(cartItems) == 0 {
		return errors.New("cart is empty")
	}

	var subtotal int64
	currency := ""
	for _, item := range cartItems {
		product, productErr := s.products.GetProductByID(ctx, item.ProductID)
		if productErr != nil {
			return productErr
		}
		if currency == "" {
			currency = strings.ToUpper(product.Currency)
		}
		subtotal += int64(item.PriceCentsSnapshot) * int64(item.Quantity)
	}

	order, err := s.products.CreateOrder(ctx, db.CreateOrderParams{
		UserID:        cart.UserID.UUID,
		Status:        "paid",
		Currency:      currency,
		SubtotalCents: int32(subtotal),
		TotalCents:    int32(subtotal),
	})
	if err != nil {
		return err
	}

	for _, item := range cartItems {
		product, productErr := s.products.GetProductByID(ctx, item.ProductID)
		if productErr != nil {
			return productErr
		}
		if _, err := s.products.CreateOrderItem(ctx, db.CreateOrderItemParams{
			OrderID:             order.ID,
			ProductID:           item.ProductID,
			ProductNameSnapshot: product.Name,
			PriceCentsSnapshot:  item.PriceCentsSnapshot,
			Quantity:            item.Quantity,
		}); err != nil {
			return err
		}
	}

	providerPaymentID := session.PaymentIntent
	if providerPaymentID == "" {
		providerPaymentID = session.ID
	}

	amountCents := int32(session.AmountTotal)
	if amountCents <= 0 {
		amountCents = int32(subtotal)
	}
	paymentCurrency := strings.ToUpper(session.Currency)
	if paymentCurrency == "" {
		paymentCurrency = currency
	}

	if _, err := s.products.CreatePayment(ctx, db.CreatePaymentParams{
		OrderID:           order.ID,
		Provider:          "stripe",
		ProviderPaymentID: sql.NullString{String: providerPaymentID, Valid: true},
		AmountCents:       amountCents,
		Currency:          paymentCurrency,
		Status:            "succeeded",
		CheckoutSessionID: sql.NullString{String: session.ID, Valid: true},
		PaymentIntentID:   sql.NullString{String: session.PaymentIntent, Valid: session.PaymentIntent != ""},
	}); err != nil {
		return err
	}

	if _, err := s.products.UpdateCartStatus(ctx, db.UpdateCartStatusParams{
		ID:     cartID,
		Status: "converted",
	}); err != nil {
		return err
	}

	for _, item := range cartItems {
		if err := s.products.RemoveCartItem(ctx, item.ID); err != nil {
			return err
		}
	}

	return nil
}

type stripeGateway interface {
	CreateCheckoutSession(ctx context.Context, input stripeCreateCheckoutSessionInput) (stripeCheckoutSession, error)
	ParseWebhook(payload []byte, signatureHeader, secret string, tolerance time.Duration) (stripeEvent, error)
}

type stripeClient struct {
	httpClient *http.Client
}

func newStripeGateway(httpClient *http.Client) stripeGateway {
	return &stripeClient{httpClient: httpClient}
}

type stripeLineItem struct {
	Name       string
	UnitAmount int32
	Quantity   int32
	Currency   string
}

type stripeCreateCheckoutSessionInput struct {
	SecretKey  string
	SuccessURL string
	CancelURL  string
	LineItems  []stripeLineItem
	Metadata   map[string]string
}

type stripeEvent struct {
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

type stripeCheckoutSession struct {
	ID            string            `json:"id"`
	URL           string            `json:"url"`
	PaymentIntent string            `json:"payment_intent"`
	AmountTotal   int64             `json:"amount_total"`
	Currency      string            `json:"currency"`
	Metadata      map[string]string `json:"metadata"`
}

func (s *stripeClient) CreateCheckoutSession(ctx context.Context, input stripeCreateCheckoutSessionInput) (stripeCheckoutSession, error) {
	form := url.Values{}
	form.Set("mode", "payment")
	form.Set("success_url", input.SuccessURL)
	form.Set("cancel_url", input.CancelURL)

	for idx, item := range input.LineItems {
		form.Set(fmt.Sprintf("line_items[%d][price_data][currency]", idx), item.Currency)
		form.Set(fmt.Sprintf("line_items[%d][price_data][product_data][name]", idx), item.Name)
		form.Set(fmt.Sprintf("line_items[%d][price_data][unit_amount]", idx), strconv.Itoa(int(item.UnitAmount)))
		form.Set(fmt.Sprintf("line_items[%d][quantity]", idx), strconv.Itoa(int(item.Quantity)))
	}

	for k, v := range input.Metadata {
		form.Set(fmt.Sprintf("metadata[%s]", k), v)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.stripe.com/v1/checkout/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		return stripeCheckoutSession{}, err
	}
	req.Header.Set("Authorization", "Bearer "+input.SecretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return stripeCheckoutSession{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return stripeCheckoutSession{}, err
	}
	if resp.StatusCode >= 300 {
		return stripeCheckoutSession{}, fmt.Errorf("stripe checkout session request failed with status %d", resp.StatusCode)
	}

	var session stripeCheckoutSession
	if err := json.Unmarshal(body, &session); err != nil {
		return stripeCheckoutSession{}, err
	}

	return session, nil
}

func (s *stripeClient) ParseWebhook(payload []byte, signatureHeader, secret string, tolerance time.Duration) (stripeEvent, error) {
	parts := strings.Split(signatureHeader, ",")
	var timestamp string
	var signature string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "t=") {
			timestamp = strings.TrimPrefix(part, "t=")
		} else if strings.HasPrefix(part, "v1=") {
			signature = strings.TrimPrefix(part, "v1=")
		}
	}
	if timestamp == "" || signature == "" {
		return stripeEvent{}, errors.New("missing signature parts")
	}

	tsInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return stripeEvent{}, err
	}
	now := time.Now().Unix()
	if tolerance > 0 {
		diff := now - tsInt
		if diff < 0 {
			diff = -diff
		}
		if time.Duration(diff)*time.Second > tolerance {
			return stripeEvent{}, errors.New("signature timestamp outside tolerance")
		}
	}

	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return stripeEvent{}, errors.New("invalid signature")
	}

	var event stripeEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return stripeEvent{}, err
	}
	return event, nil
}

func mapProduct(p db.Product, categories []db.Category) productDTO {
	var weight *int32
	if p.WeightGrams.Valid {
		value := p.WeightGrams.Int32
		weight = &value
	}

	var imageURL *string
	if p.ImageUrl.Valid {
		value := p.ImageUrl.String
		imageURL = &value
	}

	categoryNames := make([]string, 0, len(categories))
	for _, category := range categories {
		categoryNames = append(categoryNames, category.Name)
	}

	return productDTO{
		ID:          p.ID.String(),
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		PriceCents:  p.PriceCents,
		Currency:    p.Currency,
		Stock:       p.Stock,
		IsActive:    p.IsActive,
		SKU:         p.Sku,
		WeightGrams: weight,
		ImageURL:    imageURL,
		Categories:  categoryNames,
	}
}

func mapOrder(order db.Order) orderDTO {
	return orderDTO{
		ID:            order.ID.String(),
		UserID:        order.UserID.String(),
		Status:        order.Status,
		Currency:      order.Currency,
		SubtotalCents: order.SubtotalCents,
		TotalCents:    order.TotalCents,
		CreatedAt:     order.CreatedAt,
		UpdatedAt:     order.UpdatedAt,
	}
}

func mapOrderItem(item db.OrderItem) orderItemDTO {
	return orderItemDTO{
		ID:                 item.ID.String(),
		OrderID:            item.OrderID.String(),
		ProductID:          item.ProductID.String(),
		ProductName:        item.ProductNameSnapshot,
		PriceCentsSnapshot: item.PriceCentsSnapshot,
		Quantity:           item.Quantity,
		CreatedAt:          item.CreatedAt,
	}
}

func mapCartItem(item db.CartItem) cartItemDTO {
	return cartItemDTO{
		ID:                 item.ID.String(),
		CartID:             item.CartID.String(),
		ProductID:          item.ProductID.String(),
		Quantity:           item.Quantity,
		PriceCentsSnapshot: item.PriceCentsSnapshot,
		CreatedAt:          item.CreatedAt,
	}
}

func parsePagination(r *http.Request) (int, int, error) {
	limit := 20
	offset := 0

	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			return 0, 0, errors.New("limit must be a positive integer")
		}
		if value > 100 {
			return 0, 0, errors.New("limit must be <= 100")
		}
		limit = value
	}

	if raw := r.URL.Query().Get("offset"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return 0, 0, errors.New("offset must be a non-negative integer")
		}
		offset = value
	}

	return limit, offset, nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

var _ ProductQuerier = (*db.Queries)(nil)
