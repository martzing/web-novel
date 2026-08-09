# Database Schema — หมอกจันทร์ (Mokchan)

PostgreSQL 16 (`postgres:16-alpine` in `docker-compose.yml`). Conventions: PKs are `BIGINT GENERATED ALWAYS AS IDENTITY`; timestamps are `TIMESTAMPTZ NOT NULL DEFAULT now()`; enums are text with `CHECK`.

The authoritative DDL lives in [backend/migrations/0001_init.sql](../backend/migrations/0001_init.sql) and seed data in [backend/migrations/0002_seed.sql](../backend/migrations/0002_seed.sql). This document is the human-readable reference; keep the two in sync when things change.

## Extensions

```sql
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
```

## Domain overview

```
users ─┬─ writer_profiles
       ├─ user_prefs
       └─ user_genre_prefs ─ genres

series ─ novels ─┬─ novel_genres ─ genres
                 ├─ novel_relations ─ novels        (ภาคต่อ / ภาคแยก / โลกเดียวกัน)
                 ├─ arcs
                 ├─ chapters ─┬─ chapter_bodies
                 │            ├─ chapter_drafts
                 │            └─ chapter_glossary_refs ─ glossary_entries
                 └─ glossary_groups ─ glossary_entries

users ─┬─ library_entries ─ novels
       ├─ reading_progress ─ novels/chapters
       ├─ bookmarks ─ chapters
       └─ follows ─ novels

users ─┬─ wallet_balances
       ├─ coin_ledger ─┬─ chapter_unlocks ─ chapters
       │               ├─ purchases ─ coin_packs
       │               └─ writer_earnings ─ chapters
       ├─ auto_unlock_subscriptions ─ novels
       ├─ auto_unlock_attempts ─ chapters
       └─ payouts

users ─┬─ comments ─ chapters
       ├─ comment_likes ─ comments
       └─ reviews ─ novels

users ─ notifications
```

## Identity

### `users`

| Column                     | Type            | Notes                                |
| -------------------------- | --------------- | ------------------------------------ |
| `id`                       | `bigint` PK     |                                      |
| `username`                 | `citext` UNIQUE |                                      |
| `email`                    | `citext` UNIQUE |                                      |
| `password_hash`            | `text`          | argon2id                             |
| `display_name`             | `text`          |                                      |
| `avatar_url`               | `text` NULL     |                                      |
| `roles`                    | `text[]`        | default `{reader}`                   |
| `status`                   | `text`          | `active` \| `suspended` \| `deleted` |
| `created_at`, `updated_at` | `timestamptz`   |                                      |

### `writer_profiles`

Extends `users` for translators. `payout_info` is `jsonb` for provider-specific fields.

### `user_prefs`

Reader settings synced across devices.

| Column         | Constraint                           |
| -------------- | ------------------------------------ |
| `theme`        | `light` \| `sepia` \| `dark`         |
| `font`         | `loop` \| `serif` \| `sans`          |
| `font_size`    | `SMALLINT` `BETWEEN 14 AND 28`       |
| `line_height`  | `NUMERIC(3,2)` `BETWEEN 1.4 AND 2.4` |
| `column_width` | `narrow` \| `normal` \| `wide`       |

### `user_genre_prefs`

Composite PK `(user_id, genre_id)`; `weight` seeds home ranking personalisation.

## Catalog

### `series`

Multi-novel series (the design mentions `ดูทั้งชุด · 5 เรื่อง 8 ภาค`).

### `novels`

| Column                                                            | Notes                                                   |
| ----------------------------------------------------------------- | ------------------------------------------------------- |
| `slug`                                                            | UNIQUE, URL-safe                                        |
| `title_th`, `title_cn`, `author_name`, `description`              |                                                         |
| `status`                                                          | `ongoing` \| `complete` \| `hiatus`                     |
| `primary_translator_id`                                           | FK to `users`; **single translator per novel**          |
| `rating_avg`, `rating_count`, `followers_count`, `chapters_count` | Denormalised aggregates                                 |
| `glossary_rev`                                                    | Bumped by trigger on glossary changes; drives re-render |
| `search_tsv`                                                      | `tsvector` GIN-indexed                                  |

Indexes: `GIN(search_tsv)`, `GIN(title_th gin_trgm_ops)`.

### `genres`, `novel_genres`

Static list + M:N.

### `arcs`

Per-novel arc (ภาค); `UNIQUE (novel_id, arc_no)`. Stores `from_chapter_no` / `to_chapter_no` for fast ToC grouping.

### `chapters`

| Column                         | Notes                                 |
| ------------------------------ | ------------------------------------- |
| `arc_id`                       | FK, `SET NULL` on arc delete          |
| `chapter_no`                   | `UNIQUE (novel_id, chapter_no)`       |
| `status`                       | `draft` \| `scheduled` \| `published` |
| `price_coins`                  | `SMALLINT`, 0 = free                  |
| `translator_id`                | Single FK, matches decision           |
| `published_at`, `scheduled_at` |                                       |

Partial index: `(novel_id, published_at DESC) WHERE status='published'` for hot ToC queries.

### `chapter_bodies`

Rendered `body_html` (with baked `<span data-k>`) + raw `body_source` for the editor. `glossary_rev` marks which revision of the parent novel's glossary the HTML was rendered against; the re-render worker updates HTML when `chapter_bodies.glossary_rev < novels.glossary_rev`.

### `chapter_drafts`

Autosave history. The last 20 rows are kept per chapter (application-enforced).

## Glossary

### `glossary_groups`, `glossary_entries`

Grouped by novel (e.g. `ลำดับขั้นการบำเพ็ญ`, `ศัพท์การบำเพ็ญ`, `สำนักและตระกูล`, `ตัวละคร`). `term_key` is what the inline `<span data-k>` uses.

### `chapter_glossary_refs`

Which glossary entries appear in which chapter — populated when the editor binds a term.

### Trigger — auto-bump

```sql
CREATE TRIGGER glossary_entries_bump
AFTER INSERT OR UPDATE OR DELETE ON glossary_entries
FOR EACH ROW EXECUTE FUNCTION bump_novel_glossary_rev();
```

`bump_novel_glossary_rev()` increments `novels.glossary_rev`. A background worker then reads `chapter_bodies` where `glossary_rev < novels.glossary_rev` and re-renders.

## Library, progress, bookmarks, follows

| Table              | PK                    | Purpose                                 |
| ------------------ | --------------------- | --------------------------------------- |
| `library_entries`  | `(user_id, novel_id)` | `reading` \| `saved` \| `done`          |
| `reading_progress` | `(user_id, novel_id)` | `last_chapter_id`, `para_anchor`, `pct` |
| `bookmarks`        | id                    | Per-paragraph bookmark with excerpt     |
| `follows`          | `(user_id, novel_id)` | Notification target for new chapters    |

## Coins (the ledger is APPEND-ONLY)

### `wallet_balances`

Cached balance per user; separate `balance` and `bonus_balance` with a shared `bonus_expires_at`. `CHECK (balance >= 0)` and `CHECK (bonus_balance >= 0)`.

Partial index: `(bonus_expires_at) WHERE bonus_balance > 0` so the nightly cron finds expiry candidates quickly.

### `coin_ledger`

| Column                                 | Notes                                                                                |
| -------------------------------------- | ------------------------------------------------------------------------------------ |
| `delta`                                | Signed, paid portion                                                                 |
| `bonus_delta`                          | Signed, bonus portion                                                                |
| `kind`                                 | `topup` \| `spend_unlock` \| `refund` \| `bonus_grant` \| `bonus_expire` \| `adjust` |
| `ref_type`, `ref_id`                   | e.g. `('chapter_unlock', 12345)`                                                     |
| `balance_after`, `bonus_balance_after` | Audit snapshots                                                                      |
| `actor_user_id`                        | Non-null for admin adjust                                                            |
| `idempotency_key`                      | `UNIQUE (user_id, idempotency_key)`                                                  |

**Invariant enforced in application:** every mutation goes through one Go helper that opens a tx, `SELECT ... FOR UPDATE` on `wallet_balances`, inserts one `coin_ledger` row, updates `wallet_balances`, and writes any related child row (`chapter_unlocks`, `purchases`, `writer_earnings`) in the same transaction.

### `coin_packs`

Fixed catalogue. `is_best_value` drives the "คุ้มที่สุด" badge on the mock.

### `purchases`

| Column         | Notes                                                    |
| -------------- | -------------------------------------------------------- |
| `provider`     | Phase 2 only allows `'mock'`; real providers wired later |
| `provider_ref` | `UNIQUE (provider, provider_ref)` → webhook idempotency  |
| `status`       | `pending` \| `succeeded` \| `failed` \| `refunded`       |
| `ledger_id`    | FK to the successful credit row                          |

### `chapter_unlocks`

Composite PK `(user_id, chapter_id)`. `coins_spent` snapshot at time of unlock (chapter price may change later).

### `writer_earnings`

Row per successful unlock crediting the translator. `net_coins = gross_coins - platform_fee` computed in Go.

### `payouts`

Fiat payout requests. `status` = `requested` \| `approved` \| `paid` \| `rejected`.

## Social

### `comments`

Chapter-scoped. `parent_id` supports one-level replies (application-enforced). `is_spoiler_hidden` toggles the "ซ่อนสปอยล์" pill. `is_translator=true` for translator replies (rendered with the red badge). Soft-delete via `deleted_at`.

### `comment_likes`

Composite PK `(user_id, comment_id)`. `comments.likes_count` denormalised via triggers or app writes.

### `reviews`

`UNIQUE (novel_id, user_id)` — one review per user per novel; drives `novels.rating_avg` / `rating_count`.

## Ranking & stats

### `ranking_snapshots`

Weekly leaderboard snapshot (`novel_id`, `period`, `rank`, `score`). `jobs.RankingJob` writes a snapshot every Monday at 04:00 Asia/Bangkok; the read path falls back to live popularity while no snapshot exists yet. There is no separate cache tier — see "Open items" below.

### `chapter_read_events` (partitioned)

```sql
CREATE TABLE chapter_read_events (...)
  PARTITION BY RANGE (occurred_at);
CREATE TABLE chapter_read_events_default PARTITION OF chapter_read_events DEFAULT;
```

Monthly partitions created by a job.

### `chapter_daily_stats`, `novel_daily_stats`

Aggregates powering the writer stats page (ยอดอ่าน, ผู้ติดตามใหม่, เหรียญที่ได้รับ, อ่านจบต่อบท) with `+X.X%` trend calculated at read time.

`completions` counts reads that reached the end of a chapter, and rolls up
chapter → novel like `reads` does. The signal is `chapter_read_events.completed`
rather than something derived from `reading_progress`, which is per novel and
mutable and so cannot say how many of a given day's reads finished. The KPI is
`completions ÷ reads` over the whole window, not an average of per-chapter
rates, so a chapter with a single read cannot swing it.

## Notifications

| Column    | Notes                                                      |
| --------- | ---------------------------------------------------------- |
| `kind`    | `new_chapter` \| `reply` \| `bonus_expiring`               |
| `payload` | `jsonb` — kind-specific                                    |
| `read_at` | Partial index `WHERE read_at IS NULL` for the unread count |

## Migration & seed layout

| File                                                                                                        | Purpose                                                                                                                            |
| ----------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| [`0001_init.sql`](../backend/migrations/0001_init.sql)                                                       | All tables above, plus the glossary trigger and the wallet expiry partial index.                                                   |
| [`0002_seed.sql`](../backend/migrations/0002_seed.sql)                                                       | Genres, the translator account, **two** novels (`เซียนดาบเก้าสายธาร` and `คืนกลับสู่ปีที่สิบเก้า`), the first one's 4 arcs, chapters 86–88, glossary entries, and 4 coin packs. |
| [`0003_auth.sql`](../backend/migrations/0003_auth.sql)                                                       | `refresh_tokens` and its indexes; replaces the seeded placeholder password hash with a real argon2id hash.                         |
| [`0004_search.sql`](../backend/migrations/0004_search.sql)                                                   | `search_tsv` refresh trigger and backfill, plus trigram indexes for the blended search ranking.                                    |
| [`0005_purchases_idempotency.sql`](../backend/migrations/0005_purchases_idempotency.sql)                     | `purchases.idempotency_key` + unique index; `chapter_unlocks (user_id)` index.                                                     |
| [`0006_notifications.sql`](../backend/migrations/0006_notifications.sql)                                     | Notification dedupe indexes, `comments (parent_id)`, `chapter_read_events (chapter_id, occurred_at)`.                              |
| [`0007_monetization.sql`](../backend/migrations/0007_monetization.sql)                                       | Twelve new `novels` columns, `novels.status='hidden'`, the partial unique `novels_series_position`, `chapters.public_at`, the `tip` ledger kind, `writer_earnings.kind`, `series.owner_user_id`/`slug`, and three new tables. |
| [`0008_seed_fixes.sql`](../backend/migrations/0008_seed_fixes.sql)                                           | Seed corrections: the featured novel's `chapters_count` becomes 88 against its 214 source chapters, and the coin packs match the design.                                    |
| [`0009_read_completion.sql`](../backend/migrations/0009_read_completion.sql)                                 | `chapter_read_events.completed` and `chapter_daily_stats.completions`, backing the `อ่านจบต่อบท` writer KPI.                       |

### `refresh_tokens` (0003)

| Column        | Notes                                                             |
| ------------- | ----------------------------------------------------------------- |
| `token_hash`  | `BYTEA UNIQUE` — only `sha256(token)` is stored                   |
| `family_id`   | `UUID`; a rotation chain. Replaying a revoked token revokes the whole family |
| `replaced_by` | Self-FK to the successor token                                    |
| `revoked_at`  | Set on rotation, logout, or reuse detection                       |

The seeded translator's development password is `mokchan-dev`.

### Search ranking (0004)

The trigger keeps `search_tsv` current, but **tsvector alone cannot serve Thai
search**: `เซียนดาบ` is a substring of the single lexeme
`เซียนดาบเก้าสายธาร`, so `plainto_tsquery` misses it. The repository blends
trigram `similarity()`, `ILIKE` and `ts_rank`; user input is escaped with an
explicit `ESCAPE '\'` so `%` and `_` are literal. Do not reduce this to a pure
tsvector match — test I-CAT-01 covers exactly this case.

### Idempotency (0005)

`POST /purchases` creates a `pending` row and writes no ledger entry, so
`coin_ledger`'s unique key cannot dedupe it. `purchases.idempotency_key` with
`UNIQUE (user_id, idempotency_key)` fills that gap (test I-COIN-01M). NULLs are
distinct in Postgres, so rows without a key coexist freely under both indexes —
which is also what lets every key-less `bonus_expire` ledger row coexist.

### Notification dedupe (0006)

Unique partial indexes on `(user_id, payload->>'chapter_id') WHERE
kind='new_chapter'` and the equivalent for replies. Unpublishing and
republishing a chapter is a normal editorial action, so fan-out relies on these
indexes plus `ON CONFLICT DO NOTHING` rather than on the caller remembering not
to repeat itself.

### Monetisation, series and covers (0007)

The largest migration since the initial schema. It carries every schema change
for works management and advanced monetisation.

**`novels`** gains twelve columns: `source_chapters_count` (บทในต้นฉบับ),
`price_per_chapter`, `free_until_chapter`, `sell_by_arc`, `tips_enabled`,
`early_access_hours` (0–168), `release_schedule`, the cover template trio
`cover_style` / `cover_color` / `cover_text`, and the series placement pair
`series_position` / `series_note`. `novels.status` gains `'hidden'`.

A **partial unique index** `novels_series_position` on
`(series_id, series_position)` keeps two books in one series from claiming the
same slot. Being partial, it cannot be deferred — Postgres enforces it row by
row within a statement — so renumbering a reading order permutes through
negatives first (see `SetSeriesOrder`); a single `UPDATE ... CASE` collides
halfway through on any rotation.

**`chapters.public_at`** is when a published chapter becomes visible to
non-subscribers. It is **snapshotted at publish time** (`published_at +
novels.early_access_hours`), not derived on read: deriving it would force a join
to `novels` into the hottest queries in the app, and would let a translator
flipping the setting retroactively un-publish the last day of chapters. Backfilled
to `published_at` for existing rows, so nothing already public becomes private.

**`coin_ledger.kind`** gains `'tip'` and **`writer_earnings.kind`** distinguishes
`'unlock'` from `'tip'`, so a tip is not filed as a sale and unlock-derived
statistics stay meaningful.

**`series`** gains `owner_user_id` and `slug` (added → backfilled → `SET NOT
NULL` → unique index; it cannot be one statement).

Three new tables:

| Table | Purpose |
| --- | --- |
| `auto_unlock_subscriptions` | `(user_id, novel_id)` opt-in with `active` and `max_coins_per_chapter`. A per-chapter cap only — a monthly cap would need a mutable counter locked after `wallet_balances`, doubling the lock story for a guardrail the balance already provides. |
| `auto_unlock_attempts` | `(user_id, chapter_id)` outcome log driving retry backoff: `unlocked`, `insufficient`, `over_cap`, `skipped`. |
| `novel_relations` | `(novel_id, related_novel_id)` with `kind`, `note`, `sort_no` and `CHECK (novel_id <> related_novel_id)`. Stored directional; read in both directions with the inverse kind applied to the mirrored side. |

> **Production caveat.** Swapping the `coin_ledger` and `novels` CHECK
> constraints validates every existing row under `ACCESS EXCLUSIVE`. That is
> fine at pre-launch size; at scale it needs `ADD CONSTRAINT ... NOT VALID`
> followed by a separate `VALIDATE CONSTRAINT` in a migration marked as running
> outside a transaction. The same applies to any future index on the hot coin
> tables, which needs `CREATE INDEX CONCURRENTLY`.

Applied via `goose`:

```bash
cd backend
go run ./cmd/migrate -cmd up
go run ./cmd/migrate -cmd status
```

## Open items

Resolved:

- ~~Add a `revoked_refresh_tokens` table (or Redis set) once auth is wired.~~
  Done in 0003 as `refresh_tokens` with family-based revocation. Postgres rather
  than Redis: rotation needs a compare-and-swap, and reuse detection needs the
  revoked row to survive — a TTL cache would evict exactly the evidence that
  matters.
- ~~Introduce monthly partition creation cron for `chapter_read_events`.~~ Done:
  `jobs.PartitionJob` creates the current and next month's partitions monthly.
- ~~Consider a materialized view for the weekly ranking.~~ Done without Redis:
  `jobs.RankingJob` writes `ranking_snapshots` every Monday, and the read path
  falls back to live popularity when no snapshot exists yet.

Still open:

- `chapter_daily_stats.coins_earned` attributes an unlock to the day it happened
  rather than to the chapter's own read events; revisit if per-chapter revenue
  reporting needs to be exact.
- No table backs `GET /admin/reports`; add `comment_reports` when comment
  moderation is built.
- `novels.chapters_count` is a stored counter, which is what makes an
  early-access chapter a *teaser* rather than a hidden row: a viewer-dependent
  table of contents would disagree with the stored count for 24 hours.
- `auto_unlock_attempts` grows one row per subscriber per paid chapter. It is
  small at current scale but has no retention policy; add one before the
  subscriber count makes it interesting.
