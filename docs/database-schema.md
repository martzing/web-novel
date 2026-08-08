# Database Schema — หมอกจันทร์ (Mokchan)

PostgreSQL 15+. Conventions: PKs are `BIGINT GENERATED ALWAYS AS IDENTITY`; timestamps are `TIMESTAMPTZ NOT NULL DEFAULT now()`; enums are text with `CHECK`.

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

Weekly leaderboard snapshot (`novel_id`, `period`, `rank`, `score`). Live ranking is computed in Redis (`ZSET`) and snapshotted here for history.

### `chapter_read_events` (partitioned)

```sql
CREATE TABLE chapter_read_events (...)
  PARTITION BY RANGE (occurred_at);
CREATE TABLE chapter_read_events_default PARTITION OF chapter_read_events DEFAULT;
```

Monthly partitions created by a job.

### `chapter_daily_stats`, `novel_daily_stats`

Aggregates powering the writer stats page (ยอดอ่าน, ผู้ติดตามใหม่, เหรียญที่ได้รับ) with `+X.X%` trend calculated at read time.

## Notifications

| Column    | Notes                                                      |
| --------- | ---------------------------------------------------------- |
| `kind`    | `new_chapter` \| `reply` \| `bonus_expiring`               |
| `payload` | `jsonb` — kind-specific                                    |
| `read_at` | Partial index `WHERE read_at IS NULL` for the unread count |

## Migration & seed layout

| File                                                                      | Purpose                                                                                                                            |
| ------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| [`backend/migrations/0001_init.sql`](../backend/migrations/0001_init.sql) | All tables above, plus the glossary trigger and the wallet expiry partial index.                                                   |
| [`backend/migrations/0002_seed.sql`](../backend/migrations/0002_seed.sql) | Genres, translator account, the featured novel `เซียนดาบเก้าสายธาร`, its arcs, chapters 86–88, glossary entries, and 4 coin packs. |

Applied via `goose`:

```bash
cd backend
go run ./cmd/migrate -cmd up
go run ./cmd/migrate -cmd status
```

## Open items to watch after Phase 1

- Add a `revoked_refresh_tokens` table (or Redis set) once auth is wired.
- Introduce monthly partition creation cron for `chapter_read_events` before Phase 4.
- Consider a materialized view for the weekly ranking if Redis is unavailable in production.
