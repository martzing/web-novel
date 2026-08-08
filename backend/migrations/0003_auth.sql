-- +goose Up
-- +goose StatementBegin

-- Refresh-token sessions.
--
-- Only sha256(token) is stored, so a database dump grants no sessions.
-- family_id ties a rotation chain together: replaying an already-revoked token
-- is treated as theft and revokes the whole family.
CREATE TABLE refresh_tokens (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id     BIGINT NOT NULL REFERENCES users ON DELETE CASCADE,
  family_id   UUID NOT NULL,
  token_hash  BYTEA NOT NULL UNIQUE,
  user_agent  TEXT,
  expires_at  TIMESTAMPTZ NOT NULL,
  revoked_at  TIMESTAMPTZ,
  replaced_by BIGINT REFERENCES refresh_tokens ON DELETE SET NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX refresh_tokens_user_active ON refresh_tokens (user_id) WHERE revoked_at IS NULL;
CREATE INDEX refresh_tokens_family      ON refresh_tokens (family_id);
CREATE INDEX refresh_tokens_expiry      ON refresh_tokens (expires_at) WHERE revoked_at IS NULL;

-- The seeded translator carried a bcrypt-shaped placeholder that argon2id
-- cannot parse. Replace it with a real argon2id hash of the documented
-- development password so the account can actually sign in.
--   password: mokchan-dev
UPDATE users
   SET password_hash = '$argon2id$v=19$m=65536,t=3,p=2$BdzA4vlHtgkNrKJeLrcuyg$4qg5SQkH2hUoEtLtOcblMEXOygdlBPZHopxXz0Homqk'
 WHERE username = 'mokchan';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS refresh_tokens;
-- +goose StatementEnd
