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
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	db "github.com/BerkAkipek/e-commerce-app/api/internal/db"
)

type ProductQuerier interface {
	ListActiveProducts(ctx context.Context, arg db.ListActiveProductsParams) ([]db.Product, error)
	GetProductBySlug(ctx context.Context, lower string) (db.Product, error)
	GetCartByID(ctx context.Context, id uuid.UUID) (db.Cart, error)
	GetProductByID(ctx context.Context, id uuid.UUID) (db.Product, error)
	GetCartItemByID(ctx context.Context, id uuid.UUID) (db.CartItem, error)
	GetCartItemByCartAndProduct(ctx context.Context, arg db.GetCartItemByCartAndProductParams) (db.CartItem, error)
	ListCartItemsByCartID(ctx context.Context, cartID uuid.UUID) ([]db.CartItem, error)
	CreateCartItem(ctx context.Context, arg db.CreateCartItemParams) (db.CartItem, error)
	UpdateCartItemQuantity(ctx context.Context, arg db.UpdateCartItemQuantityParams) (db.CartItem, error)
	RemoveCartItem(ctx context.Context, id uuid.UUID) error
	CreateOrder(ctx context.Context, arg db.CreateOrderParams) (db.Order, error)
	CreateOrderItem(ctx context.Context, arg db.CreateOrderItemParams) (db.OrderItem, error)
	CreatePayment(ctx context.Context, arg db.CreatePaymentParams) (db.Payment, error)
	GetPaymentByCheckoutSessionID(ctx context.Context, checkoutSessionID string) (db.Payment, error)
	UpdateCartStatus(ctx context.Context, arg db.UpdateCartStatusParams) (db.Cart, error)
}

type Server struct {
	products ProductQuerier
	stripe   stripeGateway
}

func NewServer(products ProductQuerier) *Server {
	return &Server{
		products: products,
		stripe:   newStripeGateway(http.DefaultClient),
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)

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
	r.Post("/cart/items", s.handleCreateCartItem)
	r.Patch("/cart/items/{id}", s.handlePatchCartItem)
	r.Delete("/cart/items/{id}", s.handleDeleteCartItem)
	r.Get("/cart/{id}/items", s.handleGetCartItems)
	r.Post("/checkout/session", s.handleCreateCheckoutSession)
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
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	PriceCents  int32   `json:"price_cents"`
	Currency    string  `json:"currency"`
	Stock       int32   `json:"stock"`
	IsActive    bool    `json:"is_active"`
	SKU         string  `json:"sku"`
	WeightGrams *int32  `json:"weight_grams,omitempty"`
	ImageURL    *string `json:"image_url,omitempty"`
}

type createCartItemRequest struct {
	CartID    string `json:"cart_id"`
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
	CartID     string `json:"cart_id"`
	SuccessURL string `json:"success_url"`
	CancelURL  string `json:"cancel_url"`
}

type checkoutSessionResponse struct {
	Data struct {
		SessionID string `json:"session_id"`
		URL       string `json:"url"`
	} `json:"data"`
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
		response.Data = append(response.Data, mapProduct(p))
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

	writeJSON(w, http.StatusOK, productResponse{Data: mapProduct(product)})
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

	cartID, err := uuid.Parse(req.CartID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "cart_id must be a valid uuid")
		return
	}
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "product_id must be a valid uuid")
		return
	}

	cart, err := s.products.GetCartByID(r.Context(), cartID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "cart not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get cart")
		return
	}
	if cart.Status != "active" {
		writeError(w, http.StatusConflict, "cart is not active")
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
	cartID, err := uuid.Parse(cartIDRaw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id must be a valid uuid")
		return
	}

	_, err = s.products.GetCartByID(r.Context(), cartID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "cart not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get cart")
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

	item, err := s.products.GetCartItemByID(r.Context(), cartItemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "cart item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get cart item")
		return
	}

	cart, err := s.products.GetCartByID(r.Context(), item.CartID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "cart not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get cart")
		return
	}
	if cart.Status != "active" {
		writeError(w, http.StatusConflict, "cart is not active")
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

	item, err := s.products.GetCartItemByID(r.Context(), cartItemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "cart item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get cart item")
		return
	}

	cart, err := s.products.GetCartByID(r.Context(), item.CartID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "cart not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get cart")
		return
	}
	if cart.Status != "active" {
		writeError(w, http.StatusConflict, "cart is not active")
		return
	}

	if err := s.products.RemoveCartItem(r.Context(), cartItemID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove cart item")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	var req createCheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cartID, err := uuid.Parse(req.CartID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "cart_id must be a valid uuid")
		return
	}

	cart, err := s.products.GetCartByID(r.Context(), cartID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "cart not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get cart")
		return
	}
	if cart.Status != "active" {
		writeError(w, http.StatusConflict, "cart is not active")
		return
	}
	if !cart.UserID.Valid {
		writeError(w, http.StatusBadRequest, "cart must be attached to a user for checkout")
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
			"user_id": cart.UserID.UUID.String(),
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

func mapProduct(p db.Product) productDTO {
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
