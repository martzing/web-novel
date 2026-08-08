-- +goose Up
-- +goose StatementBegin

-- POST /purchases creates a pending row and writes no coin_ledger entry, so the
-- ledger's UNIQUE (user_id, idempotency_key) cannot dedupe it. Give purchases
-- their own key so a retried request returns the original pending purchase
-- instead of creating a second one (I-COIN-01M).
ALTER TABLE purchases ADD COLUMN idempotency_key TEXT;

-- NULLs are distinct in Postgres, so rows without a key (admin-created,
-- historical) coexist freely under this index.
CREATE UNIQUE INDEX purchases_user_idem ON purchases (user_id, idempotency_key);

-- The table-of-contents bulk ownership lookup filters by user.
CREATE INDEX chapter_unlocks_user ON chapter_unlocks (user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS chapter_unlocks_user;
DROP INDEX IF EXISTS purchases_user_idem;
ALTER TABLE purchases DROP COLUMN IF EXISTS idempotency_key;
-- +goose StatementEnd
