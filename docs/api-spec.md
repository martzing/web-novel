# API Specification — หมอกจันทร์ (Mokchan)

Base URL: `/api/v1`.
Auth: `Authorization: Bearer <jwt>` on protected routes.
All bodies JSON. IDs are string-encoded `int64`. Timestamps are ISO-8601. Mutating endpoints accept an `Idempotency-Key` header.

## Error envelope

```json
{ "error": { "code": "STRING", "message": "human-readable message" } }
```

## Pagination

Cursor-based: `?limit=<int>&cursor=<opaque>`. Responses shape:

```json
{ "data": [ ... ], "next_cursor": "..." }
```

---

## Auth

| Method | Path             | Description                                          |
| ------ | ---------------- | ---------------------------------------------------- |
| POST   | `/auth/register` | `{username, email, password}` → `{user, token}`      |
| POST   | `/auth/login`    | `{email, password}` → `{user, token, refresh_token}` |
| POST   | `/auth/refresh`  | `{refresh_token}` → `{token}`                        |
| POST   | `/auth/logout`   | Revoke the current token                             |
| GET    | `/auth/me`       | Current user + roles                                 |

## Me (user + prefs)

| Method     | Path                    |
| ---------- | ----------------------- |
| GET, PATCH | `/users/me`             |
| GET, PUT   | `/users/me/prefs`       |
| PUT        | `/users/me/genre-prefs` |

## Catalog

| Method | Path                    | Notes                                            |
| ------ | ----------------------- | ------------------------------------------------ |
| GET    | `/genres`               |                                                  |
| GET    | `/novels`               | `?q=&genre=&sort=popular\|latest&cursor=&limit=` |
| GET    | `/novels/{slug}`        | Detail with arcs summary + rating                |
| GET    | `/novels/{id}/chapters` | ToC rows only (no body)                          |
| GET    | `/novels/{id}/arcs`     |                                                  |
| GET    | `/novels/{id}/glossary` | Grouped                                          |
| GET    | `/novels/{id}/related`  | เรื่องเกี่ยวเนื่อง, both directions, with `kind_label` |
| GET    | `/series/{id}`          | ชุดหนังสือ by id *or* slug, books in reading order |

Every novel payload carries **both** chapter counts — `chapters_count`
(บทที่แปลแล้ว) and `source_chapters_count` (บทในต้นฉบับ) — plus the cover
template fields `cover_style`, `cover_color`, `cover_text`. `GET /novels/{id}`
additionally returns `sell_by_arc`, `tips_enabled`, `early_access_hours` and
`release_schedule`.

`GET /series/{id}` returns the series with `books[]` in `position` order, each
book a novel payload plus `position` and `note`, and the header totals
`chapters_count` / `source_chapters_count` summed across the visible books.

## Reading

| Method   | Path                          | Notes                                                            |
| -------- | ----------------------------- | ---------------------------------------------------------------- |
| GET      | `/chapters/{id}`              | Returns `{ ..., locked: true, body_html: null }` if not entitled |
| GET      | `/chapters/{id}/next` `/prev` |                                                                  |
| POST     | `/chapters/{id}/read-event`   | Fire-and-forget, `202`                                           |
| GET, PUT | `/me/progress/{novel_id}`     |                                                                  |

### Chapter response shape

```json
{
  "id": "182",
  "novel_id": "1",
  "chapter_no": 87,
  "title": "ดาบแรกใต้ฟ้าหมอก",
  "price_coins": 5,
  "locked": false,
  "locked_reason": "",
  "tips_enabled": true,
  "body_html": "<p>...</p>"
}
```

`locked_reason` tells the client which call to action to show:

| Value | Meaning | Client action |
| --- | --- | --- |
| `paywall` | Priced and unowned | Offer the unlock |
| `early_access` | Published, still inside its 24-hour window | Offer auto-unlock — **no amount of coins opens it yet** |

An early-access chapter is still listed in the table of contents for everyone,
with its metadata and a null body. Buying it is refused with
`403 EARLY_ACCESS_ONLY`.

## Bookmarks / Library / Follows / Reviews

| Method       | Path                                   |
| ------------ | -------------------------------------- |
| GET          | `/me/bookmarks?novel_id=`              |
| POST         | `/me/bookmarks`                        |
| DELETE       | `/me/bookmarks/{id}`                   |
| GET          | `/me/library?tab=reading\|saved\|done` |
| PUT, DELETE  | `/me/library/{novel_id}`               |
| POST, DELETE | `/me/follows/{novel_id}`               |
| POST         | `/novels/{id}/reviews`                 |
| GET          | `/novels/{id}/reviews`                 |

## Comments

| Method       | Path                                                           |
| ------------ | -------------------------------------------------------------- |
| GET          | `/chapters/{id}/comments?sort=popular\|latest\|with_replies`   |
| POST         | `/chapters/{id}/comments`                                      |
| POST, DELETE | `/comments/{id}/like`                                          |
| DELETE       | `/comments/{id}` (author, translator of the chapter, or admin) |

## Coins & purchases (mock in Phase 2)

| Method | Path                            | Notes                                                                                                                                                                            |
| ------ | ------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| GET    | `/me/wallet`                    | `{balance, bonus_balance, bonus_expires_at}`                                                                                                                                     |
| GET    | `/me/wallet/ledger`             | Paginated                                                                                                                                                                        |
| GET    | `/coin-packs`                   |                                                                                                                                                                                  |
| POST   | `/purchases`                    | Body `{pack_id}`. Server sets `provider='mock'`, `status='pending'`. Returns `{purchase_id, mock_checkout_url}`                                                                  |
| POST   | `/purchases/{id}/mock-complete` | **Dev/admin only**, feature-flagged by `PAYMENTS_MOCK_ENABLED`. Requires `Idempotency-Key`. Flips to `succeeded`, credits wallet. Returns `{balance_after, bonus_balance_after}` |
| POST   | `/purchases/{id}/mock-fail`     | Dev/admin only. Flips to `failed`. No wallet change                                                                                                                              |
| POST   | `/chapters/{id}/unlock`         | Requires `Idempotency-Key`. Returns `{ledger_id, coins_spent, balance_after}`                                                                                                    |
| GET    | `/arcs/{id}/bundle`             | Quote for buying a whole arc: `{chapter_count, gross, discount_percent, discount, total, chapters[]}`. Excludes free and already-owned chapters                                  |
| POST   | `/arcs/{id}/unlock`             | Requires `Idempotency-Key`. One ledger row, N `chapter_unlocks`, one `writer_earnings` per chapter                                                                                |
| POST   | `/chapters/{id}/tip`            | Body `{coins}` (1–1000). Requires `Idempotency-Key` — repeat tipping is legitimate, so the key is the **only** dedupe                                                             |

Arc membership is resolved by **chapter-number range**, never `chapters.arc_id`,
which is NULL for chapters created before their arc existed.

`/arcs/{id}/unlock` recognises a retry *before* it re-quotes: once the first
attempt commits every chapter is owned, and a re-quote would raise
`ARC_ALREADY_OWNED` instead of returning the receipt.

### Auto-unlock subscriptions

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/me/auto-unlock` | The caller's opt-ins |
| PUT | `/me/auto-unlock/{novel_id}` | Body `{active, max_coins_per_chapter}`. Omitting `active` means "turn it on" |
| DELETE | `/me/auto-unlock/{novel_id}` | `204` |

Subscribers are debited by a background job that scans for *missing* unlocks, so
a repeated run is a no-op and a reader who bought manually is never charged
again. The idempotency key is derived (`autounlock:<user>:<chapter>`) and never
client-supplied.

Unlock-specific error codes: `CHAPTER_ALREADY_UNLOCKED`, `INSUFFICIENT_COINS`,
`CHAPTER_NOT_FOR_SALE`, `EARLY_ACCESS_ONLY` (403).

Bundle codes: `ARC_NOT_FOR_SALE` (400), `ARC_ALREADY_OWNED` (409),
`ARC_BUNDLE_STALE` (409 — the arc's contents changed between quote and purchase).

Tip codes: `INSUFFICIENT_PAID_COINS` (402 — **distinct** from
`INSUFFICIENT_COINS`, because bonus coins cannot fund a tip), `TIPS_DISABLED`
(400), `CANNOT_TIP_SELF` (400), `INVALID_AMOUNT` (400).

## Writer

| Method      | Path                                             |
| ----------- | ------------------------------------------------ |
| POST, PATCH | `/writer/novels`, `/writer/novels/{id}`          |
| POST        | `/writer/novels/{id}/cover` (multipart)          |
| GET, POST   | `/writer/series`                                 |
| PATCH, DELETE | `/writer/series/{id}`                          |
| GET         | `/writer/series/{id}/books`                      |
| PUT         | `/writer/series/{id}/order` (`{novel_ids: []}`)  |
| PUT         | `/writer/novels/{id}/series-note` (`{note}`)     |
| GET, POST   | `/writer/novels/{id}/relations`                  |
| DELETE      | `/writer/novels/{id}/relations/{related_id}`     |
| POST, PATCH | `/writer/novels/{id}/arcs`, `/writer/arcs/{id}`  |
| GET, POST   | `/writer/novels/{id}/chapters`                   |
| GET, PUT    | `/writer/chapters/{id}` (autosave)               |
| POST        | `/writer/chapters/{id}/publish`                  |
| POST        | `/writer/chapters/{id}/unpublish`                |
| GET, POST   | `/writer/novels/{id}/glossary`                   |
| PATCH       | `/writer/glossary-entries/{id}`                  |
| GET         | `/writer/stats/novels/{id}?period=14d\|30d\|all` |
| GET         | `/writer/earnings`                               |
| POST        | `/writer/payouts`                                |

### The novel patch

`PATCH /writer/novels/{id}` is a partial patch, and the settings fields are
**presence-sensitive**: an omitted key leaves the field alone, and an explicitly
sent `false` or `0` applies. This matters — turning arc sales off and setting
"free until chapter 0" are exactly the edits a translator makes, and a
non-zero test would silently drop them.

`series_id` is three-valued: omitted leaves the membership alone, a string joins
that series (which the caller must own, or `403`), and an explicit `null` leaves
the series and clears the reading-order slot and note.

Settings fields: `source_chapters_count`, `price_per_chapter` (0–999),
`free_until_chapter`, `sell_by_arc`, `tips_enabled`, `early_access_hours`
(0–168), `release_schedule`, `cover_style`, `cover_color` (`#RRGGBB`),
`cover_text`, `series_position`, `series_note`.

`POST /writer/novels/{id}/chapters` defaults `price_coins` from the novel's
`price_per_chapter`, and forces 0 at or below `free_until_chapter` even when the
request supplies a price.

Relation kinds: `sequel`, `prequel`, `spinoff`, `side_story`, `same_world`. A
link is stored once and listed from both novels; the far side is returned with
the inverse kind and `mirrored: true`, and can only be unlinked from the novel
that declared it.

## Admin

| Method      | Path                                          |
| ----------- | --------------------------------------------- |
| GET         | `/admin/reports`                              |
| POST        | `/admin/comments/{id}/moderate`               |
| POST        | `/admin/wallet-adjust`                        |
| POST, PATCH | `/admin/coin-packs`, `/admin/coin-packs/{id}` |
| POST        | `/admin/payouts/{id}/approve` `/reject`       |

## Discovery utility

| Method | Path                                        |
| ------ | ------------------------------------------- |
| GET    | `/search?q=&type=novel\|chapter\|character` |
| GET    | `/ranking/weekly?limit=`                    |

## Notifications (Phase 4)

Not in the original spec; added because Phase 4 and R-17 require notifying
followers of new chapters.

| Method | Path                                  |
| ------ | ------------------------------------- |
| GET    | `/me/notifications?unread=true`       |
| GET    | `/me/notifications/unread-count`      |
| POST   | `/me/notifications/read` (`{ids: []}` — empty means all) |

---

## Implementation status

All of the above are implemented except where noted below.

### Deviations from this document

- **`/novels/{id}` accepts an id *or* a slug.** Gin panics at startup when two
  different wildcard names occupy the same path segment, so `/novels/{slug}`
  and `/novels/{id}/chapters` cannot coexist. Every `/novels/...` route uses one
  `:id` parameter and the service resolves numeric ids and slugs alike. Both
  spellings in this document therefore work.
- **`GET /novels/{id}/chapters` accepts optional auth** and adds an `unlocked`
  boolean per row, resolved in one bulk query, so the table of contents can show
  ownership. Additive.
- **`GET /novels/{id}/reviews` adds `my_review`** for an authenticated caller,
  so the review form opens pre-filled.
- **`GET /writer/earnings` adds `available_satang`**, the amount still
  withdrawable after pending payouts.
- **Cursor pagination is keyset, never `OFFSET`.** A cursor is bound to the sort
  order it was minted under; replaying it against a different `?sort=` is
  `400 BAD_CURSOR` rather than a silently mixed ordering.
- **`POST /auth/logout` revokes the refresh-token family only.** The access
  token stays valid until it expires (15 minutes by default); there is no shared
  denylist. Clients must discard it.
- **`POST /purchases` requires `Idempotency-Key`** as well, deduplicated through
  `purchases.idempotency_key` (migration 0005) since it writes no ledger row.
- **Idempotency keys are namespaced per operation.** The stored key is
  `<operation>:<client key>`, so reusing one key across a chapter unlock and an
  arc purchase performs both rather than replaying the first. `Apply` also
  compares `ref_type` as well as `ref_id`, so the guarantee survives a caller
  that derives its own key.
- **`GET /novels/{id}` takes optional auth.** A novel with status `hidden` is
  `404` for everyone except its own translator, who can still open it from
  จัดการผลงาน. `hidden` novels are likewise excluded from `/novels`, `/search`
  and `/ranking/weekly`.

### Not implemented

- `GET /novels/{id}/glossary` is implemented; `GET /search` is an alias of
  `GET /novels` with the same blended ranking.
- Most of **Admin**. Only `POST /admin/wallet-adjust` exists, because the coin
  test matrix requires it. `/admin/reports`, comment moderation, coin-pack CRUD
  and payout approval are not built, and no tables back the report queue.
- Real payment providers (**Phase 7**, the last phase). `provider` is always
  `mock`.

### Error codes

Beyond the unlock codes above: `INVALID_CREDENTIALS`, `EMAIL_TAKEN`,
`USERNAME_TAKEN`, `INVALID_USERNAME`, `INVALID_EMAIL`, `WEAK_PASSWORD`,
`INVALID_REFRESH_TOKEN`, `INVALID_PREFS`, `INVALID_STATUS`, `BAD_ID`,
`BAD_CURSOR`, `IDEMPOTENCY_KEY_REQUIRED`, `INVALID_IDEMPOTENCY_KEY`,
`IDEMPOTENCY_KEY_CONFLICT`, `PURCHASE_NOT_PENDING`, `COMMENT_EMPTY`,
`COMMENT_TOO_LONG`, `REPLY_TOO_DEEP`, `INVALID_RATING`, `SLUG_TAKEN`,
`CHAPTER_NO_TAKEN`, `INVALID_PRICE`, `UNSUPPORTED_FILE`, `FILE_TOO_LARGE`,
`RATE_LIMITED`, `FORBIDDEN`, `NOT_FOUND`, `INTERNAL`, `INVALID_BODY`,
`INVALID_AMOUNT`, `INSUFFICIENT_PAID_COINS`, `TIPS_DISABLED`, `CANNOT_TIP_SELF`,
`ARC_NOT_FOR_SALE`, `ARC_ALREADY_OWNED`, `ARC_BUNDLE_STALE`,
`EARLY_ACCESS_ONLY`.
