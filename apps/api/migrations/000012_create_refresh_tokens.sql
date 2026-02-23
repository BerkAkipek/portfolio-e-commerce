CREATE TABLE IF NOT EXISTS refresh_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT refresh_tokens_token_hash_not_blank_chk CHECK (length(btrim(token_hash)) > 0)
);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx
  ON refresh_tokens (user_id);

CREATE INDEX IF NOT EXISTS refresh_tokens_token_hash_idx
  ON refresh_tokens (token_hash);

CREATE INDEX IF NOT EXISTS refresh_tokens_expires_at_idx
  ON refresh_tokens (expires_at);

CREATE INDEX IF NOT EXISTS refresh_tokens_revoked_idx
  ON refresh_tokens (revoked);
