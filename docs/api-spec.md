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
| GET    | `/series/{id}`          |                                                  |

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
  "body_html": "<p>...</p>"
}
```

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

Unlock-specific error codes: `CHAPTER_ALREADY_UNLOCKED`, `INSUFFICIENT_COINS`, `CHAPTER_NOT_FOR_SALE`.

## Writer

| Method      | Path                                             |
| ----------- | ------------------------------------------------ |
| POST, PATCH | `/writer/novels`, `/writer/novels/{id}`          |
| POST        | `/writer/novels/{id}/cover` (multipart)          |
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

## Phase-1 endpoints (implemented in this repo)

The endpoints below are already wired end-to-end in the current backend:

- `GET /health`
- `GET /api/v1/genres`
- `GET /api/v1/novels?q=&genre=&sort=popular|latest`
- `GET /api/v1/novels/{slug}`
- `GET /api/v1/novels/{id}/chapters`
- `GET /api/v1/chapters/{id}`
