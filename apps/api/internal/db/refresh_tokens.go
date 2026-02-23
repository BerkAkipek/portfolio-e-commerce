package db

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

const createRefreshToken = `
INSERT INTO refresh_tokens (
  user_id,
  token_hash,
  expires_at
) VALUES (
  $1, $2, $3
)
RETURNING id, user_id, token_hash, expires_at, revoked, created_at
`

type CreateRefreshTokenParams struct {
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
}

func (q *Queries) CreateRefreshToken(ctx context.Context, arg CreateRefreshTokenParams) (RefreshToken, error) {
	row := q.db.QueryRowContext(ctx, createRefreshToken, arg.UserID, arg.TokenHash, arg.ExpiresAt)
	var rt RefreshToken
	err := row.Scan(
		&rt.ID,
		&rt.UserID,
		&rt.TokenHash,
		&rt.ExpiresAt,
		&rt.Revoked,
		&rt.CreatedAt,
	)
	return rt, err
}

const revokeRefreshTokenByHash = `
UPDATE refresh_tokens
SET revoked = TRUE
WHERE token_hash = $1
  AND revoked = FALSE
RETURNING id, user_id, token_hash, expires_at, revoked, created_at
`

func (q *Queries) RevokeRefreshTokenByHash(ctx context.Context, tokenHash string) (RefreshToken, error) {
	row := q.db.QueryRowContext(ctx, revokeRefreshTokenByHash, tokenHash)
	var rt RefreshToken
	err := row.Scan(
		&rt.ID,
		&rt.UserID,
		&rt.TokenHash,
		&rt.ExpiresAt,
		&rt.Revoked,
		&rt.CreatedAt,
	)
	return rt, err
}

const getRefreshTokenByHash = `
SELECT id, user_id, token_hash, expires_at, revoked, created_at
FROM refresh_tokens
WHERE token_hash = $1
LIMIT 1
`

func (q *Queries) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (RefreshToken, error) {
	row := q.db.QueryRowContext(ctx, getRefreshTokenByHash, tokenHash)
	var rt RefreshToken
	err := row.Scan(
		&rt.ID,
		&rt.UserID,
		&rt.TokenHash,
		&rt.ExpiresAt,
		&rt.Revoked,
		&rt.CreatedAt,
	)
	return rt, err
}

const rotateRefreshToken = `
WITH revoked AS (
  UPDATE refresh_tokens
  SET revoked = TRUE
  WHERE token_hash = $1
    AND revoked = FALSE
    AND expires_at > NOW()
  RETURNING user_id
)
INSERT INTO refresh_tokens (
  user_id,
  token_hash,
  expires_at
)
SELECT user_id, $2, $3
FROM revoked
RETURNING id, user_id, token_hash, expires_at, revoked, created_at
`

type RotateRefreshTokenParams struct {
	OldTokenHash string
	NewTokenHash string
	ExpiresAt    time.Time
}

func (q *Queries) RotateRefreshToken(ctx context.Context, arg RotateRefreshTokenParams) (RefreshToken, error) {
	row := q.db.QueryRowContext(ctx, rotateRefreshToken, arg.OldTokenHash, arg.NewTokenHash, arg.ExpiresAt)
	var rt RefreshToken
	err := row.Scan(
		&rt.ID,
		&rt.UserID,
		&rt.TokenHash,
		&rt.ExpiresAt,
		&rt.Revoked,
		&rt.CreatedAt,
	)
	return rt, err
}
