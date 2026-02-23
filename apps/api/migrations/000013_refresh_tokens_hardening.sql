DROP INDEX IF EXISTS refresh_tokens_token_hash_idx;

CREATE UNIQUE INDEX IF NOT EXISTS refresh_tokens_token_hash_uq
  ON refresh_tokens (token_hash);
