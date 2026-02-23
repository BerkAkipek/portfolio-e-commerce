package httpapi

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	db "github.com/BerkAkipek/e-commerce-app/api/internal/db"
)

var ErrActiveCartNotFound = errors.New("active cart not found")

type CartService struct {
	store ProductQuerier
}

func NewCartService(store ProductQuerier) *CartService {
	return &CartService{store: store}
}

func (s *CartService) ResolveCart(ctx context.Context) (uuid.UUID, error) {
	return s.resolveCart(ctx, true)
}

func (s *CartService) ResolveExistingCart(ctx context.Context) (uuid.UUID, error) {
	return s.resolveCart(ctx, false)
}

func (s *CartService) resolveCart(ctx context.Context, createIfMissing bool) (uuid.UUID, error) {
	actor, ok := ActorFromContext(ctx)
	if !ok {
		return uuid.Nil, errors.New("actor is required")
	}

	switch actor.ActorType {
	case ActorTypeUser:
		if actor.UserID == uuid.Nil {
			return uuid.Nil, errors.New("user actor requires user id")
		}
		cart, err := s.store.GetActiveCartByUserID(ctx, actor.UserID)
		if err == nil {
			return cart.ID, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, err
		}
		if !createIfMissing {
			return uuid.Nil, ErrActiveCartNotFound
		}

		created, createErr := s.store.CreateCart(ctx, db.CreateCartParams{
			UserID:    uuid.NullUUID{UUID: actor.UserID, Valid: true},
			SessionID: sql.NullString{},
			Status:    "active",
		})
		if createErr == nil {
			return created.ID, nil
		}

		cart, err = s.store.GetActiveCartByUserID(ctx, actor.UserID)
		if err == nil {
			return cart.ID, nil
		}
		return uuid.Nil, createErr

	case ActorTypeSession:
		if actor.SessionID == "" {
			return uuid.Nil, errors.New("session actor requires session id")
		}
		cart, err := s.store.GetActiveCartBySessionID(ctx, actor.SessionID)
		if err == nil {
			return cart.ID, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, err
		}
		if !createIfMissing {
			return uuid.Nil, ErrActiveCartNotFound
		}

		created, createErr := s.store.CreateCart(ctx, db.CreateCartParams{
			UserID:    uuid.NullUUID{},
			SessionID: sql.NullString{String: actor.SessionID, Valid: true},
			Status:    "active",
		})
		if createErr == nil {
			return created.ID, nil
		}

		cart, err = s.store.GetActiveCartBySessionID(ctx, actor.SessionID)
		if err == nil {
			return cart.ID, nil
		}
		return uuid.Nil, createErr
	}

	return uuid.Nil, errors.New("invalid actor type")
}
