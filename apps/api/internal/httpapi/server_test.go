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

	db "github.com/BerkAkipek/e-commerce-app/api/internal/db"
)

type fakeProductQuerier struct {
	products []db.Product
	err      error
	lastArg  db.ListActiveProductsParams
	product  db.Product
	getErr   error
	lastSlug string

	cart        db.Cart
	cartErr     error
	productErr  error
	cartItem    db.CartItem
	cartItemErr error
	createItem  db.CartItem
	createErr   error
	updateItem  db.CartItem
	updateErr   error
	removeErr   error
	listItems   []db.CartItem
	listErr     error
	order       db.Order
	orderErr    error
	orderItem   db.OrderItem
	orderItemErr error
	payment     db.Payment
	paymentErr  error
	paymentLookup db.Payment
	paymentLookupErr error
	cartStatusErr error

	lastGetCartID   uuid.UUID
	lastGetProdID   uuid.UUID
	lastGetItemID   uuid.UUID
	lastCartItemArg db.GetCartItemByCartAndProductParams
	lastCreateArg   db.CreateCartItemParams
	lastUpdateArg   db.UpdateCartItemQuantityParams
	lastRemoveID    uuid.UUID
	lastListCartID  uuid.UUID
	lastCreateOrderArg db.CreateOrderParams
	lastCreateOrderItemArg db.CreateOrderItemParams
	lastCreatePaymentArg db.CreatePaymentParams
	lastPaymentLookupSessionID string
	lastUpdateCartStatusArg db.UpdateCartStatusParams
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

func (f *fakeProductQuerier) GetCartByID(_ context.Context, id uuid.UUID) (db.Cart, error) {
	f.lastGetCartID = id
	if f.cartErr != nil {
		return db.Cart{}, f.cartErr
	}
	return f.cart, nil
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
		cart: db.Cart{ID: cartID, Status: "active"},
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
		cart: db.Cart{ID: cartID, Status: "active"},
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
	store := &fakeProductQuerier{
		cart: db.Cart{ID: uuid.MustParse("00000000-0000-0000-0000-000000000094"), Status: "converted"},
		cartItem: db.CartItem{
			ID:     uuid.MustParse("00000000-0000-0000-0000-000000000093"),
			CartID: uuid.MustParse("00000000-0000-0000-0000-000000000094"),
		},
	}
	req := httptest.NewRequest(http.MethodDelete, "/cart/items/00000000-0000-0000-0000-000000000093", nil)
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rr.Code)
	}
}

func TestPostCartItemsCartNotFound(t *testing.T) {
	store := &fakeProductQuerier{cartErr: sql.ErrNoRows}
	body := `{"cart_id":"00000000-0000-0000-0000-000000000030","product_id":"00000000-0000-0000-0000-000000000031","quantity":1}`
	req := httptest.NewRequest(http.MethodPost, "/cart/items", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
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
		cart: db.Cart{ID: cartID, Status: "active"},
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
	req := httptest.NewRequest(http.MethodGet, "/cart/not-a-uuid/items", nil)
	rr := httptest.NewRecorder()

	NewServer(&fakeProductQuerier{}).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestGetCartItemsNotFound(t *testing.T) {
	store := &fakeProductQuerier{cartErr: sql.ErrNoRows}
	req := httptest.NewRequest(http.MethodGet, "/cart/00000000-0000-0000-0000-000000000050/items", nil)
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

func TestPatchCartItemsInactiveCart(t *testing.T) {
	store := &fakeProductQuerier{
		cart: db.Cart{ID: uuid.MustParse("00000000-0000-0000-0000-000000000097"), Status: "converted"},
		cartItem: db.CartItem{
			ID:     uuid.MustParse("00000000-0000-0000-0000-000000000098"),
			CartID: uuid.MustParse("00000000-0000-0000-0000-000000000097"),
		},
	}
	req := httptest.NewRequest(http.MethodPatch, "/cart/items/00000000-0000-0000-0000-000000000098", strings.NewReader(`{"quantity":2}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rr.Code)
	}
}

func TestDeleteCartItemsRemoveError(t *testing.T) {
	itemID := uuid.MustParse("00000000-0000-0000-0000-000000000099")
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000100")
	store := &fakeProductQuerier{
		cart:      db.Cart{ID: cartID, Status: "active"},
		removeErr: errors.New("delete failed"),
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

func TestGetCartItemsListError(t *testing.T) {
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000101")
	store := &fakeProductQuerier{
		cart:    db.Cart{ID: cartID, Status: "active"},
		listErr: errors.New("list failed"),
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

	req := httptest.NewRequest(http.MethodPost, "/checkout/session", strings.NewReader(`{"cart_id":"00000000-0000-0000-0000-000000000110"}`))
	req.Header.Set("Content-Type", "application/json")
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

	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000114")
	store := &fakeProductQuerier{
		cart: db.Cart{
			ID:     cartID,
			Status: "active",
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/checkout/session", strings.NewReader(`{"cart_id":"00000000-0000-0000-0000-000000000114"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestCreateCheckoutSessionEmptyCart(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	t.Setenv("CHECKOUT_SUCCESS_URL", "https://example.com/success")
	t.Setenv("CHECKOUT_CANCEL_URL", "https://example.com/cancel")

	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000115")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000116")
	store := &fakeProductQuerier{
		cart: db.Cart{
			ID:     cartID,
			UserID: uuid.NullUUID{UUID: userID, Valid: true},
			Status: "active",
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/checkout/session", strings.NewReader(`{"cart_id":"00000000-0000-0000-0000-000000000115"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	NewServer(store).Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
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
