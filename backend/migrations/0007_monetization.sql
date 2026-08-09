-- +goose Up
-- +goose StatementBegin

-- ─── novels: monetisation, presentation and series placement ───────────────

ALTER TABLE novels
  -- Chapters in the original work, as opposed to chapters_count which counts
  -- what has actually been translated and published. Every "บทในต้นฉบับ"
  -- figure in the UI reads this.
  ADD COLUMN source_chapters_count INT      NOT NULL DEFAULT 0 CHECK (source_chapters_count >= 0),
  ADD COLUMN price_per_chapter     SMALLINT NOT NULL DEFAULT 0 CHECK (price_per_chapter >= 0),
  ADD COLUMN free_until_chapter    INT      NOT NULL DEFAULT 0 CHECK (free_until_chapter >= 0),
  ADD COLUMN sell_by_arc           BOOLEAN  NOT NULL DEFAULT false,
  ADD COLUMN tips_enabled          BOOLEAN  NOT NULL DEFAULT false,
  -- Hours a published chapter stays exclusive to auto-unlock subscribers.
  -- Capped at a week so a typo cannot hide a chapter for a year.
  ADD COLUMN early_access_hours    SMALLINT NOT NULL DEFAULT 0
             CHECK (early_access_hours BETWEEN 0 AND 168),
  -- Display copy only ("สัปดาห์ละ 2 บท จันทร์และพฤหัส"). Nothing reads it;
  -- if it ever drives scheduling, migrate to a structured JSONB spec then.
  ADD COLUMN release_schedule      TEXT,
  -- Cover is either an uploaded image (cover_url) or a generated template.
  ADD COLUMN cover_style           TEXT     NOT NULL DEFAULT 'image'
             CHECK (cover_style IN ('image','ink','seal','brush','plain')),
  ADD COLUMN cover_color           TEXT     CHECK (cover_color IS NULL OR cover_color ~ '^#[0-9A-Fa-f]{6}$'),
  ADD COLUMN cover_text            TEXT,
  -- Placement inside novels.series_id. A novel belongs to at most one series,
  -- so position and note live here rather than in a join table.
  ADD COLUMN series_position       SMALLINT,
  ADD COLUMN series_note           TEXT;

-- ซ่อนจากหน้าร้าน — hidden novels stay editable by their translator but
-- disappear from every reader-facing list, search result and ranking.
ALTER TABLE novels DROP CONSTRAINT novels_status_check;
ALTER TABLE novels ADD  CONSTRAINT novels_status_check
  CHECK (status IN ('ongoing','complete','hiatus','hidden'));

-- Two novels in one series cannot claim the same reading-order slot.
CREATE UNIQUE INDEX novels_series_position
  ON novels (series_id, series_position)
  WHERE series_id IS NOT NULL AND series_position IS NOT NULL;

-- ─── chapters: the early-access snapshot ───────────────────────────────────

-- When a published chapter becomes visible to everyone. Snapshotted at publish
-- time rather than derived from novels.early_access_hours at read time: a
-- derived expression would force a join to novels into the hottest queries in
-- the application, and would let a translator flipping the setting
-- retroactively un-publish the last day of chapters for every reader.
ALTER TABLE chapters ADD COLUMN public_at TIMESTAMPTZ;

UPDATE chapters SET public_at = published_at
 WHERE status = 'published' AND published_at IS NOT NULL;

CREATE INDEX chapters_novel_public ON chapters (novel_id, public_at)
  WHERE status = 'published';

-- ─── coin_ledger: the tip kind ────────────────────────────────────────────

-- PRODUCTION CAVEAT: adding a CHECK validates every existing row under
-- ACCESS EXCLUSIVE, which on a large coin_ledger blocks all coin writes for
-- the duration of a full scan. At pre-launch size this inline swap is fine.
-- At scale, replace it with:
--     ALTER TABLE coin_ledger ADD CONSTRAINT ..._v2 CHECK (...) NOT VALID;
--     ALTER TABLE coin_ledger VALIDATE CONSTRAINT ..._v2;   -- weaker lock
--     ALTER TABLE coin_ledger DROP CONSTRAINT coin_ledger_kind_check;
-- VALIDATE only takes the weaker lock in its own transaction, and goose wraps
-- each migration in one, so that version needs goose's NO TRANSACTION
-- annotation. (Spelling that annotation out here would make goose parse this
-- comment as a directive, which is why it is described rather than quoted.)
ALTER TABLE coin_ledger DROP CONSTRAINT coin_ledger_kind_check;
ALTER TABLE coin_ledger ADD  CONSTRAINT coin_ledger_kind_check
  CHECK (kind IN ('topup','spend_unlock','refund',
                  'bonus_grant','bonus_expire','adjust','tip'));

-- ─── writer_earnings: unlock vs tip ───────────────────────────────────────

ALTER TABLE writer_earnings
  ADD COLUMN kind TEXT NOT NULL DEFAULT 'unlock' CHECK (kind IN ('unlock','tip'));

COMMENT ON COLUMN writer_earnings.unlock_ledger_id IS
  'The coin_ledger row that produced this earning: a chapter unlock, an arc bundle, or a tip.';

-- An arc bundle writes N earning rows sharing one ledger id, so look-ups by
-- that column are new. ListEarnings filters by writer_id and orders by id DESC.
CREATE INDEX writer_earnings_ledger ON writer_earnings (unlock_ledger_id);
CREATE INDEX writer_earnings_writer ON writer_earnings (writer_id, id DESC);

-- ─── auto-unlock ──────────────────────────────────────────────────────────

CREATE TABLE auto_unlock_subscriptions (
  user_id               BIGINT   NOT NULL REFERENCES users  ON DELETE CASCADE,
  novel_id              BIGINT   NOT NULL REFERENCES novels ON DELETE CASCADE,
  active                BOOLEAN  NOT NULL DEFAULT true,
  -- 0 means no cap. Protects a reader from a mid-series price rise.
  max_coins_per_chapter SMALLINT NOT NULL DEFAULT 0 CHECK (max_coins_per_chapter >= 0),
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, novel_id)
);

CREATE INDEX auto_unlock_subs_novel ON auto_unlock_subscriptions (novel_id) WHERE active;

-- One row per (reader, chapter) fan-out decision. The job's candidate query
-- keys off the absence of a chapter_unlocks row, so this table records only
-- *outcomes* — it is a log and a backoff record, never the source of truth.
CREATE TABLE auto_unlock_attempts (
  user_id      BIGINT   NOT NULL REFERENCES users    ON DELETE CASCADE,
  chapter_id   BIGINT   NOT NULL REFERENCES chapters ON DELETE CASCADE,
  outcome      TEXT     NOT NULL CHECK (outcome IN ('unlocked','insufficient','over_cap','skipped')),
  attempts     SMALLINT NOT NULL DEFAULT 1,
  ledger_id    BIGINT   REFERENCES coin_ledger,
  attempted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, chapter_id)
);

CREATE INDEX auto_unlock_attempts_retry
  ON auto_unlock_attempts (attempted_at) WHERE outcome = 'insufficient';

-- Same dedupe idiom as notifications_new_chapter_dedupe (migration 0006):
-- a retry must not tell the reader twice that their wallet was short.
CREATE UNIQUE INDEX notifications_auto_unlock_dedupe
    ON notifications (user_id, ((payload->>'chapter_id')))
 WHERE kind = 'auto_unlock_failed';

-- ─── series: ownership and a public slug ──────────────────────────────────

ALTER TABLE series ADD COLUMN owner_user_id BIGINT REFERENCES users ON DELETE SET NULL;

-- slug cannot be added as UNIQUE NOT NULL in one statement: existing rows
-- would violate NOT NULL before the backfill runs.
ALTER TABLE series ADD COLUMN slug TEXT;
UPDATE series SET slug = 'series-' || id WHERE slug IS NULL;
ALTER TABLE series ALTER COLUMN slug SET NOT NULL;

CREATE UNIQUE INDEX series_slug_key ON series (slug);
CREATE INDEX series_owner ON series (owner_user_id);

-- ─── เรื่องเกี่ยวเนื่อง — related works ────────────────────────────────────

-- Stored directional: the kind is stated from novel_id's point of view.
-- 'same_world' is the one symmetric kind and is mirrored when read.
CREATE TABLE novel_relations (
  novel_id         BIGINT   NOT NULL REFERENCES novels ON DELETE CASCADE,
  related_novel_id BIGINT   NOT NULL REFERENCES novels ON DELETE CASCADE,
  kind             TEXT     NOT NULL CHECK (kind IN
                     ('sequel','prequel','spinoff','side_story','same_world')),
  note             TEXT,
  sort_no          SMALLINT NOT NULL DEFAULT 0,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (novel_id, related_novel_id),
  CHECK (novel_id <> related_novel_id)
);

CREATE INDEX novel_relations_related ON novel_relations (related_novel_id);

-- ─── seed: give the featured novel a source count and settings ────────────

UPDATE novels
   SET source_chapters_count = 214,
       price_per_chapter     = 5,
       free_until_chapter    = 48,
       sell_by_arc           = true,
       tips_enabled          = true,
       early_access_hours    = 24
 WHERE slug = 'nine-streams-sword-immortal';

UPDATE novels
   SET source_chapters_count = 143,
       price_per_chapter     = 5,
       free_until_chapter    = 20
 WHERE slug = 'return-to-nineteen';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS novel_relations;

DROP INDEX IF EXISTS series_owner;
DROP INDEX IF EXISTS series_slug_key;
ALTER TABLE series DROP COLUMN IF EXISTS slug;
ALTER TABLE series DROP COLUMN IF EXISTS owner_user_id;

DROP INDEX IF EXISTS notifications_auto_unlock_dedupe;
DROP TABLE IF EXISTS auto_unlock_attempts;
DROP TABLE IF EXISTS auto_unlock_subscriptions;

DROP INDEX IF EXISTS writer_earnings_writer;
DROP INDEX IF EXISTS writer_earnings_ledger;
ALTER TABLE writer_earnings DROP COLUMN IF EXISTS kind;

ALTER TABLE coin_ledger DROP CONSTRAINT coin_ledger_kind_check;
ALTER TABLE coin_ledger ADD  CONSTRAINT coin_ledger_kind_check
  CHECK (kind IN ('topup','spend_unlock','refund','bonus_grant','bonus_expire','adjust'));

DROP INDEX IF EXISTS chapters_novel_public;
ALTER TABLE chapters DROP COLUMN IF EXISTS public_at;

DROP INDEX IF EXISTS novels_series_position;
ALTER TABLE novels DROP CONSTRAINT novels_status_check;
ALTER TABLE novels ADD  CONSTRAINT novels_status_check
  CHECK (status IN ('ongoing','complete','hiatus'));

ALTER TABLE novels
  DROP COLUMN IF EXISTS series_note,
  DROP COLUMN IF EXISTS series_position,
  DROP COLUMN IF EXISTS cover_text,
  DROP COLUMN IF EXISTS cover_color,
  DROP COLUMN IF EXISTS cover_style,
  DROP COLUMN IF EXISTS release_schedule,
  DROP COLUMN IF EXISTS early_access_hours,
  DROP COLUMN IF EXISTS tips_enabled,
  DROP COLUMN IF EXISTS sell_by_arc,
  DROP COLUMN IF EXISTS free_until_chapter,
  DROP COLUMN IF EXISTS price_per_chapter,
  DROP COLUMN IF EXISTS source_chapters_count;

-- +goose StatementEnd
