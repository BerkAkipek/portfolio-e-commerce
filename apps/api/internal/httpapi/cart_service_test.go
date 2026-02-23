package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"

	db "github.com/BerkAkipek/e-commerce-app/api/internal/db"
)

func TestCartServiceResolveCartUserExisting(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000301")
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000302")
	store := &fakeProductQuerier{
		activeUserCart: db.Cart{
			ID:     cartID,
			UserID: uuid.NullUUID{UUID: userID, Valid: true},
			Status: "active",
		},
	}

	svc := NewCartService(store)
	ctx := context.WithValue(context.Background(), actorContextKey{}, Actor{
		ActorType: ActorTypeUser,
		UserID:    userID,
	})
	got, err := svc.ResolveCart(ctx)
	if err != nil {
		t.Fatalf("resolve cart: %v", err)
	}
	if got != cartID {
		t.Fatalf("expected cart id %s, got %s", cartID, got)
	}
	if store.lastCreateCartArg.Status != "" {
		t.Fatalf("expected no cart creation when active cart exists")
	}
}

func TestCartServiceResolveCartUserCreatesWhenMissing(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000303")
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000304")
	store := &fakeProductQuerier{
		activeUserCartErr: sql.ErrNoRows,
		createCart: db.Cart{
			ID:     cartID,
			UserID: uuid.NullUUID{UUID: userID, Valid: true},
			Status: "active",
		},
	}

	svc := NewCartService(store)
	ctx := context.WithValue(context.Background(), actorContextKey{}, Actor{
		ActorType: ActorTypeUser,
		UserID:    userID,
	})
	got, err := svc.ResolveCart(ctx)
	if err != nil {
		t.Fatalf("resolve cart: %v", err)
	}
	if got != cartID {
		t.Fatalf("expected cart id %s, got %s", cartID, got)
	}
	if !store.lastCreateCartArg.UserID.Valid || store.lastCreateCartArg.UserID.UUID != userID {
		t.Fatalf("expected user cart creation for user %s, got %+v", userID, store.lastCreateCartArg)
	}
}

func TestCartServiceResolveCartSessionExisting(t *testing.T) {
	sessionID := uuid.NewString()
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000305")
	store := &fakeProductQuerier{
		activeSessionCart: db.Cart{
			ID:        cartID,
			SessionID: sql.NullString{String: sessionID, Valid: true},
			Status:    "active",
		},
	}

	svc := NewCartService(store)
	ctx := context.WithValue(context.Background(), actorContextKey{}, Actor{
		ActorType: ActorTypeSession,
		SessionID: sessionID,
	})
	got, err := svc.ResolveCart(ctx)
	if err != nil {
		t.Fatalf("resolve cart: %v", err)
	}
	if got != cartID {
		t.Fatalf("expected cart id %s, got %s", cartID, got)
	}
}

func TestCartServiceResolveCartSessionCreatesWhenMissing(t *testing.T) {
	sessionID := uuid.NewString()
	cartID := uuid.MustParse("00000000-0000-0000-0000-000000000306")
	store := &fakeProductQuerier{
		activeSessionCartErr: sql.ErrNoRows,
		createCart: db.Cart{
			ID:        cartID,
			SessionID: sql.NullString{String: sessionID, Valid: true},
			Status:    "active",
		},
	}

	svc := NewCartService(store)
	ctx := context.WithValue(context.Background(), actorContextKey{}, Actor{
		ActorType: ActorTypeSession,
		SessionID: sessionID,
	})
	got, err := svc.ResolveCart(ctx)
	if err != nil {
		t.Fatalf("resolve cart: %v", err)
	}
	if got != cartID {
		t.Fatalf("expected cart id %s, got %s", cartID, got)
	}
	if !store.lastCreateCartArg.SessionID.Valid || store.lastCreateCartArg.SessionID.String != sessionID {
		t.Fatalf("expected session cart creation for session %s, got %+v", sessionID, store.lastCreateCartArg)
	}
}

func TestCartServiceResolveCartValidationErrors(t *testing.T) {
	svc := NewCartService(&fakeProductQuerier{})

	if _, err := svc.ResolveCart(context.Background()); err == nil {
		t.Fatalf("expected missing actor to fail")
	}

	ctxBadUser := context.WithValue(context.Background(), actorContextKey{}, Actor{
		ActorType: ActorTypeUser,
	})
	if _, err := svc.ResolveCart(ctxBadUser); err == nil {
		t.Fatalf("expected missing user id to fail")
	}

	ctxBadSession := context.WithValue(context.Background(), actorContextKey{}, Actor{
		ActorType: ActorTypeSession,
	})
	if _, err := svc.ResolveCart(ctxBadSession); err == nil {
		t.Fatalf("expected missing session id to fail")
	}

	ctxBadActor := context.WithValue(context.Background(), actorContextKey{}, Actor{
		ActorType: ActorType("unknown"),
	})
	if _, err := svc.ResolveCart(ctxBadActor); err == nil {
		t.Fatalf("expected invalid actor type to fail")
	}
}

func TestCartServiceResolveCartUserCreateFailureReturnsError(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000307")
	store := &fakeProductQuerier{
		activeUserCartErr: sql.ErrNoRows,
		createCartErr:     errors.New("insert failed"),
	}
	svc := NewCartService(store)
	ctx := context.WithValue(context.Background(), actorContextKey{}, Actor{
		ActorType: ActorTypeUser,
		UserID:    userID,
	})

	if _, err := svc.ResolveCart(ctx); err == nil {
		t.Fatalf("expected create error to bubble up")
	}
}

func TestCartServiceResolveCartUserLookupFailureReturnsError(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000308")
	store := &fakeProductQuerier{
		activeUserCartErr: errors.New("lookup failed"),
	}
	svc := NewCartService(store)
	ctx := context.WithValue(context.Background(), actorContextKey{}, Actor{
		ActorType: ActorTypeUser,
		UserID:    userID,
	})

	if _, err := svc.ResolveCart(ctx); err == nil {
		t.Fatalf("expected lookup error to bubble up")
	}
}

func TestCartServiceResolveCartSessionLookupFailureReturnsError(t *testing.T) {
	sessionID := uuid.NewString()
	store := &fakeProductQuerier{
		activeSessionCartErr: errors.New("lookup failed"),
	}
	svc := NewCartService(store)
	ctx := context.WithValue(context.Background(), actorContextKey{}, Actor{
		ActorType: ActorTypeSession,
		SessionID: sessionID,
	})

	if _, err := svc.ResolveCart(ctx); err == nil {
		t.Fatalf("expected lookup error to bubble up")
	}
}

func TestCartServiceResolveExistingCartDoesNotCreate(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000309")
	store := &fakeProductQuerier{
		activeUserCartErr: sql.ErrNoRows,
		createCart: db.Cart{
			ID:     uuid.MustParse("00000000-0000-0000-0000-000000000310"),
			UserID: uuid.NullUUID{UUID: userID, Valid: true},
			Status: "active",
		},
	}
	svc := NewCartService(store)
	ctx := context.WithValue(context.Background(), actorContextKey{}, Actor{
		ActorType: ActorTypeUser,
		UserID:    userID,
	})

	_, err := svc.ResolveExistingCart(ctx)
	if !errors.Is(err, ErrActiveCartNotFound) {
		t.Fatalf("expected ErrActiveCartNotFound, got %v", err)
	}
	if store.lastCreateCartArg.Status != "" {
		t.Fatalf("expected ResolveExistingCart not to create cart")
	}
}
