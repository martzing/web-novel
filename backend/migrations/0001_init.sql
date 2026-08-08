-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================
-- Identity
-- ============================================================
CREATE TABLE users (
  id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  username      CITEXT UNIQUE NOT NULL,
  email         CITEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  display_name  TEXT NOT NULL,
  avatar_url    TEXT,
  roles         TEXT[] NOT NULL DEFAULT ARRAY['reader'],
  status        TEXT NOT NULL DEFAULT 'active'
                 CHECK (status IN ('active','suspended','deleted')),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE writer_profiles (
  user_id     BIGINT PRIMARY KEY REFERENCES users ON DELETE CASCADE,
  pen_name    TEXT NOT NULL,
  bio         TEXT,
  sect_name   TEXT,
  payout_info JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_prefs (
  user_id      BIGINT PRIMARY KEY REFERENCES users ON DELETE CASCADE,
  theme        TEXT NOT NULL DEFAULT 'light'
                CHECK (theme IN ('light','sepia','dark')),
  font         TEXT NOT NULL DEFAULT 'loop'
                CHECK (font IN ('loop','serif','sans')),
  font_size    SMALLINT NOT NULL DEFAULT 20 CHECK (font_size BETWEEN 14 AND 28),
  line_height  NUMERIC(3,2) NOT NULL DEFAULT 2.0
                CHECK (line_height BETWEEN 1.4 AND 2.4),
  column_width TEXT NOT NULL DEFAULT 'normal'
                CHECK (column_width IN ('narrow','normal','wide')),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Catalog
-- ============================================================
CREATE TABLE series (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  title       TEXT NOT NULL,
  description TEXT,
  cover_url   TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE novels (
  id                    BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  series_id             BIGINT REFERENCES series ON DELETE SET NULL,
  slug                  TEXT UNIQUE NOT NULL,
  title_th              TEXT NOT NULL,
  title_cn              TEXT,
  author_name           TEXT,
  description           TEXT,
  cover_url             TEXT,
  status                TEXT NOT NULL DEFAULT 'ongoing'
                         CHECK (status IN ('ongoing','complete','hiatus')),
  primary_translator_id BIGINT REFERENCES users ON DELETE SET NULL,
  rating_avg            NUMERIC(3,2) NOT NULL DEFAULT 0,
  rating_count          INT NOT NULL DEFAULT 0,
  followers_count       INT NOT NULL DEFAULT 0,
  chapters_count        INT NOT NULL DEFAULT 0,
  glossary_rev          INT NOT NULL DEFAULT 0,
  search_tsv            TSVECTOR,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX novels_search_idx ON novels USING GIN (search_tsv);
CREATE INDEX novels_title_trgm ON novels USING GIN (title_th gin_trgm_ops);

CREATE TABLE genres (
  id      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  slug    TEXT UNIQUE NOT NULL,
  name_th TEXT NOT NULL
);

CREATE TABLE novel_genres (
  novel_id BIGINT REFERENCES novels ON DELETE CASCADE,
  genre_id BIGINT REFERENCES genres ON DELETE CASCADE,
  PRIMARY KEY (novel_id, genre_id)
);

CREATE TABLE user_genre_prefs (
  user_id  BIGINT REFERENCES users  ON DELETE CASCADE,
  genre_id BIGINT REFERENCES genres ON DELETE CASCADE,
  weight   SMALLINT NOT NULL DEFAULT 1,
  PRIMARY KEY (user_id, genre_id)
);

CREATE TABLE arcs (
  id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  novel_id        BIGINT NOT NULL REFERENCES novels ON DELETE CASCADE,
  arc_no          SMALLINT NOT NULL,
  name            TEXT NOT NULL,
  from_chapter_no INT NOT NULL,
  to_chapter_no   INT NOT NULL,
  UNIQUE (novel_id, arc_no)
);

CREATE TABLE chapters (
  id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  novel_id      BIGINT NOT NULL REFERENCES novels ON DELETE CASCADE,
  arc_id        BIGINT REFERENCES arcs ON DELETE SET NULL,
  chapter_no    INT NOT NULL,
  title         TEXT NOT NULL,
  status        TEXT NOT NULL DEFAULT 'draft'
                 CHECK (status IN ('draft','scheduled','published')),
  price_coins   SMALLINT NOT NULL DEFAULT 0 CHECK (price_coins >= 0),
  word_count    INT NOT NULL DEFAULT 0,
  translator_id BIGINT REFERENCES users ON DELETE SET NULL,
  published_at  TIMESTAMPTZ,
  scheduled_at  TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (novel_id, chapter_no)
);
CREATE INDEX chapters_novel_pub ON chapters (novel_id, published_at DESC)
  WHERE status = 'published';

CREATE TABLE chapter_bodies (
  chapter_id   BIGINT PRIMARY KEY REFERENCES chapters ON DELETE CASCADE,
  body_html    TEXT NOT NULL,
  body_source  TEXT NOT NULL,
  revision     INT NOT NULL DEFAULT 1,
  glossary_rev INT NOT NULL DEFAULT 0,
  rendered_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE chapter_drafts (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  chapter_id  BIGINT NOT NULL REFERENCES chapters ON DELETE CASCADE,
  author_id   BIGINT NOT NULL REFERENCES users,
  body_source TEXT NOT NULL,
  saved_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX chapter_drafts_ch_idx ON chapter_drafts (chapter_id, saved_at DESC);

-- ============================================================
-- Glossary
-- ============================================================
CREATE TABLE glossary_groups (
  id       BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  novel_id BIGINT NOT NULL REFERENCES novels ON DELETE CASCADE,
  name     TEXT NOT NULL,
  sort_no  SMALLINT NOT NULL DEFAULT 0
);

CREATE TABLE glossary_entries (
  id       BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  group_id BIGINT NOT NULL REFERENCES glossary_groups ON DELETE CASCADE,
  term_key TEXT NOT NULL,
  title_th TEXT NOT NULL,
  title_cn TEXT,
  body     TEXT NOT NULL,
  kind     TEXT,
  UNIQUE (group_id, term_key)
);

CREATE TABLE chapter_glossary_refs (
  chapter_id BIGINT REFERENCES chapters         ON DELETE CASCADE,
  entry_id   BIGINT REFERENCES glossary_entries ON DELETE CASCADE,
  PRIMARY KEY (chapter_id, entry_id)
);

-- Bump novels.glossary_rev whenever a glossary entry changes; re-render worker uses it.
CREATE OR REPLACE FUNCTION bump_novel_glossary_rev() RETURNS TRIGGER AS $$
BEGIN
  UPDATE novels n
     SET glossary_rev = glossary_rev + 1,
         updated_at   = now()
    FROM glossary_groups g
   WHERE g.id = COALESCE(NEW.group_id, OLD.group_id)
     AND n.id = g.novel_id;
  RETURN NULL;
END $$ LANGUAGE plpgsql;

CREATE TRIGGER glossary_entries_bump
AFTER INSERT OR UPDATE OR DELETE ON glossary_entries
FOR EACH ROW EXECUTE FUNCTION bump_novel_glossary_rev();

-- ============================================================
-- Library / progress / bookmarks / follows
-- ============================================================
CREATE TABLE library_entries (
  user_id  BIGINT REFERENCES users  ON DELETE CASCADE,
  novel_id BIGINT REFERENCES novels ON DELETE CASCADE,
  status   TEXT NOT NULL CHECK (status IN ('reading','saved','done')),
  added_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, novel_id)
);

CREATE TABLE reading_progress (
  user_id         BIGINT REFERENCES users  ON DELETE CASCADE,
  novel_id        BIGINT REFERENCES novels ON DELETE CASCADE,
  last_chapter_id BIGINT REFERENCES chapters,
  last_chapter_no INT,
  para_anchor     INT NOT NULL DEFAULT 0,
  pct             NUMERIC(5,2) NOT NULL DEFAULT 0,
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, novel_id)
);

CREATE TABLE bookmarks (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id     BIGINT NOT NULL REFERENCES users    ON DELETE CASCADE,
  novel_id    BIGINT NOT NULL REFERENCES novels   ON DELETE CASCADE,
  chapter_id  BIGINT NOT NULL REFERENCES chapters ON DELETE CASCADE,
  para_anchor INT NOT NULL,
  excerpt     TEXT NOT NULL,
  note        TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX bookmarks_user_idx ON bookmarks (user_id, created_at DESC);

CREATE TABLE follows (
  user_id  BIGINT REFERENCES users  ON DELETE CASCADE,
  novel_id BIGINT REFERENCES novels ON DELETE CASCADE,
  since    TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, novel_id)
);

-- ============================================================
-- Coins (ledger is APPEND-ONLY)
-- ============================================================
CREATE TABLE wallet_balances (
  user_id          BIGINT PRIMARY KEY REFERENCES users ON DELETE CASCADE,
  balance          INT NOT NULL DEFAULT 0 CHECK (balance >= 0),
  bonus_balance    INT NOT NULL DEFAULT 0 CHECK (bonus_balance >= 0),
  bonus_expires_at TIMESTAMPTZ,
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX wallet_bonus_expiry_idx
  ON wallet_balances (bonus_expires_at)
  WHERE bonus_balance > 0;

CREATE TABLE coin_ledger (
  id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id             BIGINT NOT NULL REFERENCES users,
  delta               INT NOT NULL,
  bonus_delta         INT NOT NULL DEFAULT 0,
  kind                TEXT NOT NULL CHECK (kind IN
                        ('topup','spend_unlock','refund',
                         'bonus_grant','bonus_expire','adjust')),
  ref_type            TEXT,
  ref_id              BIGINT,
  balance_after       INT NOT NULL,
  bonus_balance_after INT NOT NULL,
  reason              TEXT,
  actor_user_id       BIGINT REFERENCES users,
  idempotency_key     TEXT,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, idempotency_key)
);
CREATE INDEX coin_ledger_user_time ON coin_ledger (user_id, created_at DESC);

CREATE TABLE coin_packs (
  id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  coins         INT NOT NULL,
  bonus_coins   INT NOT NULL DEFAULT 0,
  price_satang  INT NOT NULL,
  currency      CHAR(3) NOT NULL DEFAULT 'THB',
  is_best_value BOOLEAN NOT NULL DEFAULT false,
  active        BOOLEAN NOT NULL DEFAULT true,
  sort_no       SMALLINT NOT NULL DEFAULT 0
);

CREATE TABLE purchases (
  id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id       BIGINT NOT NULL REFERENCES users,
  pack_id       BIGINT NOT NULL REFERENCES coin_packs,
  provider      TEXT NOT NULL,
  provider_ref  TEXT NOT NULL,
  amount_satang INT NOT NULL,
  currency      CHAR(3) NOT NULL,
  status        TEXT NOT NULL CHECK (status IN
                  ('pending','succeeded','failed','refunded')),
  ledger_id     BIGINT REFERENCES coin_ledger,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at  TIMESTAMPTZ,
  UNIQUE (provider, provider_ref)
);

CREATE TABLE chapter_unlocks (
  user_id     BIGINT REFERENCES users    ON DELETE CASCADE,
  chapter_id  BIGINT REFERENCES chapters ON DELETE CASCADE,
  coins_spent SMALLINT NOT NULL,
  ledger_id   BIGINT NOT NULL REFERENCES coin_ledger,
  unlocked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, chapter_id)
);

CREATE TABLE writer_earnings (
  id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  writer_id        BIGINT NOT NULL REFERENCES users,
  chapter_id       BIGINT NOT NULL REFERENCES chapters,
  unlock_ledger_id BIGINT NOT NULL REFERENCES coin_ledger,
  gross_coins      INT NOT NULL,
  net_coins        INT NOT NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE payouts (
  id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  writer_id     BIGINT NOT NULL REFERENCES users,
  amount_satang INT NOT NULL,
  status        TEXT NOT NULL CHECK (status IN
                  ('requested','approved','paid','rejected')),
  requested_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  paid_at       TIMESTAMPTZ
);

-- ============================================================
-- Social
-- ============================================================
CREATE TABLE comments (
  id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  chapter_id        BIGINT NOT NULL REFERENCES chapters ON DELETE CASCADE,
  user_id           BIGINT NOT NULL REFERENCES users,
  parent_id         BIGINT REFERENCES comments ON DELETE CASCADE,
  body              TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 5000),
  is_spoiler_hidden BOOLEAN NOT NULL DEFAULT false,
  likes_count       INT NOT NULL DEFAULT 0,
  is_translator     BOOLEAN NOT NULL DEFAULT false,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  deleted_at        TIMESTAMPTZ
);
CREATE INDEX comments_chapter_time ON comments (chapter_id, created_at DESC);

CREATE TABLE comment_likes (
  user_id    BIGINT REFERENCES users    ON DELETE CASCADE,
  comment_id BIGINT REFERENCES comments ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, comment_id)
);

CREATE TABLE reviews (
  id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  novel_id   BIGINT NOT NULL REFERENCES novels ON DELETE CASCADE,
  user_id    BIGINT NOT NULL REFERENCES users,
  rating     SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
  body       TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (novel_id, user_id)
);

-- ============================================================
-- Ranking / stats
-- ============================================================
CREATE TABLE ranking_snapshots (
  novel_id BIGINT REFERENCES novels ON DELETE CASCADE,
  period   DATE   NOT NULL,
  rank     INT    NOT NULL,
  score    NUMERIC NOT NULL,
  PRIMARY KEY (novel_id, period)
);

CREATE TABLE chapter_read_events (
  id          BIGSERIAL,
  chapter_id  BIGINT NOT NULL,
  user_id     BIGINT,
  session_id  UUID,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (id, occurred_at)
) PARTITION BY RANGE (occurred_at);

-- Default catch-all partition; a job creates monthly partitions later.
CREATE TABLE chapter_read_events_default PARTITION OF chapter_read_events DEFAULT;

CREATE TABLE chapter_daily_stats (
  chapter_id     BIGINT REFERENCES chapters ON DELETE CASCADE,
  day            DATE   NOT NULL,
  reads          INT NOT NULL DEFAULT 0,
  unique_readers INT NOT NULL DEFAULT 0,
  coins_earned   INT NOT NULL DEFAULT 0,
  PRIMARY KEY (chapter_id, day)
);

CREATE TABLE novel_daily_stats (
  novel_id         BIGINT REFERENCES novels ON DELETE CASCADE,
  day              DATE   NOT NULL,
  reads            INT NOT NULL DEFAULT 0,
  followers_gained INT NOT NULL DEFAULT 0,
  coins_earned     INT NOT NULL DEFAULT 0,
  PRIMARY KEY (novel_id, day)
);

-- ============================================================
-- Notifications
-- ============================================================
CREATE TABLE notifications (
  id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id    BIGINT NOT NULL REFERENCES users ON DELETE CASCADE,
  kind       TEXT NOT NULL,
  payload    JSONB NOT NULL,
  read_at    TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX notif_user_unread ON notifications (user_id, created_at DESC)
  WHERE read_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS novel_daily_stats CASCADE;
DROP TABLE IF EXISTS chapter_daily_stats CASCADE;
DROP TABLE IF EXISTS chapter_read_events CASCADE;
DROP TABLE IF EXISTS ranking_snapshots CASCADE;
DROP TABLE IF EXISTS reviews CASCADE;
DROP TABLE IF EXISTS comment_likes CASCADE;
DROP TABLE IF EXISTS comments CASCADE;
DROP TABLE IF EXISTS payouts CASCADE;
DROP TABLE IF EXISTS writer_earnings CASCADE;
DROP TABLE IF EXISTS chapter_unlocks CASCADE;
DROP TABLE IF EXISTS purchases CASCADE;
DROP TABLE IF EXISTS coin_packs CASCADE;
DROP TABLE IF EXISTS coin_ledger CASCADE;
DROP TABLE IF EXISTS wallet_balances CASCADE;
DROP TABLE IF EXISTS follows CASCADE;
DROP TABLE IF EXISTS bookmarks CASCADE;
DROP TABLE IF EXISTS reading_progress CASCADE;
DROP TABLE IF EXISTS library_entries CASCADE;
DROP TRIGGER  IF EXISTS glossary_entries_bump ON glossary_entries;
DROP FUNCTION IF EXISTS bump_novel_glossary_rev();
DROP TABLE IF EXISTS chapter_glossary_refs CASCADE;
DROP TABLE IF EXISTS glossary_entries CASCADE;
DROP TABLE IF EXISTS glossary_groups CASCADE;
DROP TABLE IF EXISTS chapter_drafts CASCADE;
DROP TABLE IF EXISTS chapter_bodies CASCADE;
DROP TABLE IF EXISTS chapters CASCADE;
DROP TABLE IF EXISTS arcs CASCADE;
DROP TABLE IF EXISTS user_genre_prefs CASCADE;
DROP TABLE IF EXISTS novel_genres CASCADE;
DROP TABLE IF EXISTS genres CASCADE;
DROP TABLE IF EXISTS novels CASCADE;
DROP TABLE IF EXISTS series CASCADE;
DROP TABLE IF EXISTS user_prefs CASCADE;
DROP TABLE IF EXISTS writer_profiles CASCADE;
DROP TABLE IF EXISTS users CASCADE;
-- +goose StatementEnd
