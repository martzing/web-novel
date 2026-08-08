# Test Cases — หมอกจันทร์ (Mokchan)

Tiers:

- **U** — unit, no external deps.
- **I** — integration, real Postgres/Redis.
- **E** — end-to-end, browser + API.
- **L** — load / non-functional.

## Auth

| ID        | Tier | Scenario                   | Expected                                         |
| --------- | ---- | -------------------------- | ------------------------------------------------ |
| U-AUTH-01 | U    | Password hash              | Uses argon2id with configured params.            |
| I-AUTH-01 | I    | Register duplicate email   | 409, no user created.                            |
| I-AUTH-02 | I    | Login wrong password       | 401; message does not leak whether email exists. |
| I-AUTH-03 | I    | Refresh with revoked token | 401.                                             |

## Catalog

| ID       | Tier | Scenario          | Expected                                          |
| -------- | ---- | ----------------- | ------------------------------------------------- |
| I-CAT-01 | I    | Search "เซียนดาบ" | Matching novel ranked in top 3.                   |
| I-CAT-02 | I    | Genre filter      | Only novels linked in `novel_genres` returned.    |
| I-CAT-03 | I    | Novel detail      | Response has `arcs[]` and `rating_avg` populated. |

## Reader

| ID      | Tier | Scenario                          | Expected                                                           |
| ------- | ---- | --------------------------------- | ------------------------------------------------------------------ |
| I-RD-01 | I    | GET free chapter as anon          | 200 with `body_html`.                                              |
| I-RD-02 | I    | GET locked chapter without unlock | 200 with `locked=true, body_html=null`.                            |
| I-RD-03 | I    | GET locked chapter after unlock   | 200 with `body_html`.                                              |
| I-RD-04 | I    | PUT `progress` `para_anchor=42`   | Subsequent GET returns 42.                                         |
| E-RD-01 | E    | Cross-device resume               | Login on device B; "อ่านต่อ" opens the same chapter and paragraph. |
| E-RD-02 | E    | Theme persistence                 | Switch to sepia, reload — still sepia.                             |
| E-RD-03 | E    | Glossary popover                  | Tap `data-k="qi"` span — popover shows Thai + Chinese + body.      |

## Bookmarks / Library

| ID       | Tier | Scenario                       | Expected                       |
| -------- | ---- | ------------------------------ | ------------------------------ |
| I-BM-01  | I    | Bookmark isolation             | Only owner sees own bookmarks. |
| I-BM-02  | I    | Delete another user's bookmark | 403.                           |
| I-LIB-01 | I    | Move novel reading → done      | Sidebar count updates.         |

## Coins (highest-priority tier)

Invariant, per user: `sum(coin_ledger.delta) == wallet_balances.balance` and `sum(bonus_delta) == wallet_balances.bonus_balance` after bonus expiry has been applied.

| ID         | Tier | Scenario                                                      | Expected                                                                                                            |
| ---------- | ---- | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| U-COIN-01  | U    | Spend order                                                   | Bonus balance used before paid balance.                                                                             |
| U-COIN-02  | U    | Expired bonus at spend time                                   | `bonus_expire` row created before spend calculation.                                                                |
| I-COIN-01M | I    | `POST /purchases` twice with same `Idempotency-Key`           | One `pending` purchase row.                                                                                         |
| I-COIN-07  | I    | `mock-complete` twice                                         | Wallet credited once (unique on `coin_ledger.idempotency_key`).                                                     |
| I-COIN-08  | I    | `mock-complete` in prod build (`PAYMENTS_MOCK_ENABLED=false`) | 404.                                                                                                                |
| I-COIN-02  | I    | Concurrent unlock double-click                                | Exactly one 200; other `CHAPTER_ALREADY_UNLOCKED`; one debit.                                                       |
| I-COIN-03  | I    | Insufficient coins                                            | 402 `INSUFFICIENT_COINS`; no ledger row; balance unchanged.                                                         |
| I-COIN-04  | I    | Unlock of `price_coins=0`                                     | 400 `CHAPTER_NOT_FOR_SALE`.                                                                                         |
| I-COIN-05  | I    | Successful unlock                                             | `chapter_unlocks` row references the created `coin_ledger.id`.                                                      |
| I-COIN-06  | I    | Admin adjust                                                  | `coin_ledger.kind='adjust'` with `actor_user_id` and `reason`.                                                      |
| I-COIN-09  | I    | Nightly bonus expiry                                          | User with `bonus_balance=50, bonus_expires_at=yesterday` → `bonus_expire` row `bonus_delta=-50`; `bonus_balance=0`. |
| I-COIN-10  | I    | Spend after bonus expired but before cron ran                 | Bonus treated as 0; unlock uses `balance`; expiry job enqueued.                                                     |
| E-COIN-01  | E    | Buy 240-coin pack via mock                                    | Coins page shows 240 after `mock-complete`.                                                                         |

## Glossary re-render

| ID       | Tier | Scenario                           | Expected                                                                                                            |
| -------- | ---- | ---------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| I-GLO-01 | I    | Publish chapter with `{{ye}}`      | `chapter_bodies.body_html` contains `<span data-k="ye">`; `glossary_rev = novels.glossary_rev`.                     |
| I-GLO-02 | I    | Edit glossary entry `ye`           | `novels.glossary_rev` bumps; re-render worker updates `body_html` and lifts `chapter_bodies.glossary_rev` to match. |
| I-GLO-03 | I    | Reader GET on stale `glossary_rev` | Still returns valid old HTML; after worker completes, returns new HTML.                                             |

## Comments

| ID      | Tier | Scenario                | Expected                           |
| ------- | ---- | ----------------------- | ---------------------------------- |
| I-CM-01 | I    | Comment > 5000 chars    | 400.                               |
| I-CM-02 | I    | Like same comment twice | Count stays +1.                    |
| I-CM-03 | I    | Translator reply        | Serialized with `role=translator`. |

## Writer

| ID      | Tier | Scenario                           | Expected                                       |
| ------- | ---- | ---------------------------------- | ---------------------------------------------- |
| I-WR-01 | I    | Autosave revisions                 | Last 20 kept.                                  |
| I-WR-02 | I    | Publish with future `scheduled_at` | Hidden from readers until time.                |
| I-WR-03 | I    | Cross-writer edit                  | Writer A cannot edit writer B's chapter → 403. |
| I-WR-04 | I    | Stats aggregation                  | Totals match `chapter_daily_stats` fixture.    |

## Security & abuse

| ID       | Tier | Scenario                     | Expected                                                     |
| -------- | ---- | ---------------------------- | ------------------------------------------------------------ |
| I-SEC-01 | I    | Draft chapter body as reader | 404 or 403.                                                  |
| I-SEC-02 | I    | SQL injection in `q=`        | Parameterized; no error, no leak.                            |
| I-SEC-03 | I    | `/auth/*` rate limit         | 60/min per IP.                                               |
| I-SEC-04 | I    | Foreign progress/bookmarks   | Cannot list any other user's.                                |
| I-SEC-05 | I    | Body-fetch abuse             | No more than 20 distinct chapter bodies per user per minute. |

## Non-functional

| ID   | Tier | Scenario                                           | Expected                           |
| ---- | ---- | -------------------------------------------------- | ---------------------------------- |
| L-01 | L    | 500 concurrent chapter GETs (warm)                 | p95 ≤ 400 ms.                      |
| L-02 | L    | 100 concurrent unlocks on same chapter (100 users) | All succeed; no deadlocks in 60 s. |
