# Architecture — หมอกจันทร์ backend

The backend is organised as a **hexagonal (ports & adapters)** application.
Business rules live in a dependency-free **domain** layer. The rest of the code
is adapters plugged into ports the domain defines.

```
                    ┌──────────────────────────┐
     HTTP request → │    Handler (Gin)         │  driving adapter
                    │  internal/handler/...    │
                    └────────────┬─────────────┘
                                 │  (calls)
                    ┌────────────▼─────────────┐
                    │    Service               │  application layer
                    │  internal/service/...    │
                    └────────────┬─────────────┘
                                 │  (uses port)
                    ┌────────────▼─────────────┐
                    │    Domain                │  business core
                    │  internal/domain/...     │
                    │  – types                 │
                    │  – Repository interface  │
                    └────────────▲─────────────┘
                                 │  (implements port)
                    ┌────────────┴─────────────┐
                    │    Repository (GORM)     │  driven adapter
                    │  internal/repository/... │
                    └────────────┬─────────────┘
                                 │  (SQL)
                    ┌────────────▼─────────────┐
                    │    PostgreSQL            │
                    └──────────────────────────┘
```

## Layer responsibilities

### `internal/domain/<bounded-context>/`

- Pure Go types describing the business model (`Novel`, `Chapter`, `Arc`, …).
- Domain errors (`ErrNotFound`).
- **Ports** — Go interfaces that the domain needs the outside world to provide
  (e.g. `Repository`).
- MUST NOT import anything from `service`, `repository`, `handler`,
  `entities`, or any third-party framework (Gin, GORM, pgx).

### `internal/service/<bounded-context>/`

- Application use cases (business orchestration).
- Depends on domain ports only, never on GORM or Gin.
- Owns policies like default paging, chapter-locking (Phase 1: free vs paid),
  and future access-control checks.
- Unit-tested with hand-written fakes for the port interface — no database
  required.

### `internal/repository/<bounded-context>/`

- Concrete adapter that satisfies the domain port using GORM + the
  `internal/entities` persistence models.
- Contains all SQL / GORM code. Maps persistence structs to and from domain
  types.
- Depends on `domain` (to implement the port) and `entities` (for tables).
- Integration-tested with `makeme` (real PostgreSQL through testcontainers).

### `internal/handler/<bounded-context>/`

- Gin HTTP adapter. Parses the request, calls the service, maps the domain
  result into a **response DTO** with `json` tags, and writes the response.
- Owns the wire contract; the domain never carries JSON tags.
- Depends on `service` and `domain`.

### Supporting packages

| Package             | Purpose                                                                                    |
| ------------------- | ------------------------------------------------------------------------------------------ |
| `internal/config`   | Environment loader.                                                                        |
| `internal/db`       | GORM connection factory.                                                                   |
| `internal/httpx`    | Shared JSON error helper for Gin.                                                          |
| `internal/entities` | GORM persistence models. Only the repository layer imports them.                           |
| `internal/server`   | Composition root — builds config, DB, repositories, services, handlers, and mounts routes. |
| `migrations`        | Embedded goose SQL migrations.                                                             |
| `test/makeme`       | Test-data builders + testcontainers Postgres for integration tests.                        |

## Composition root

`internal/server/server.go` wires the layers, top-down:

```go
catalogRepo    := catalogrepo.New(db)         // adapter
catalogService := catalogsvc.New(catalogRepo) // uses the port
cataloghandler.New(catalogService).Register(v1)
```

Only this file knows about all three layers of a bounded context. Every other
package sees at most one direction of the dependency arrow.

## Dependency rule

Arrows go inward. If a package imports something outside its allowed set, the
architecture is broken.

| Layer            | May import                                |
| ---------------- | ----------------------------------------- |
| `domain/...`     | standard library only                     |
| `service/...`    | `domain/...`                              |
| `repository/...` | `domain/...`, `entities`, `gorm.io/...`   |
| `handler/...`    | `service/...`, `domain/...`, `httpx`, Gin |
| `server`         | everything above                          |

## Testing pyramid

| Layer            | Preferred test type                | Tool                       |
| ---------------- | ---------------------------------- | -------------------------- |
| Domain / Service | Unit tests with hand-written fakes | plain `testing`            |
| Repository       | Integration against real Postgres  | `test/makeme`              |
| Handler          | Integration through the Gin engine | `test/makeme` + `httptest` |

The `internal/service/catalog/service_test.go` file is the reference for the
service pattern: define a `fakeRepo` implementing the port, wire per-test
callbacks for the methods that particular test uses, then assert behaviour.

## Adding a new bounded context (checklist)

When you add e.g. `wallet`:

1. `internal/domain/wallet/` — types + `Repository` interface + errors.
2. `internal/service/wallet/` — use cases + `service_test.go` with fake repo.
3. `internal/repository/wallet/` — GORM adapter + entity mapping.
4. `internal/handler/wallet/` — Gin handler + DTOs.
5. Register the wiring in `internal/server/server.go`.
6. Add builders to `test/makeme/wallet_builders.go`.
7. Add rows/tables in a new migration if needed.

## The bounded contexts

| Context | Owns | Notes |
| --- | --- | --- |
| `identity` | users, writer_profiles, user_prefs, user_genre_prefs, refresh_tokens | Auth, profile and preferences all mutate one aggregate. |
| `catalog` | genres, novels, novel_genres, arcs, chapters (read), glossary (read), ranking_snapshots (read) | Anonymous and cacheable. |
| `reading` | chapter_bodies (read), reading_progress, chapter_read_events | Owns the entitlement decision — the one place `locked` is computed. |
| `library` | library_entries, bookmarks, follows | Three per-user↔novel relations with one ownership rule. |
| `wallet` | wallet_balances, coin_ledger, coin_packs, purchases, chapter_unlocks, writer_earnings, payouts | Must be one context: they commit together. |
| `social` | comments, comment_likes, reviews | Shared soft-delete and denormalised counters. |
| `writer` | chapters (write), chapter_bodies (write), chapter_drafts, chapter_glossary_refs, glossary CRUD, daily stats (read) | One persona, one authorization predicate. |
| `notification` | notifications | Thin on purpose, so `writer` and `social` depend on it and not on each other. |

Contexts never import one another. Where one needs another's data it declares a
narrow port that the composition root satisfies — `reading.Entitlements` and
`catalog.Entitlements` are both implemented by the wallet repository,
`notification.Followers` by the library repository.

## Shared vocabulary packages

`domain/page` (cursor paging) and `domain/roles` (role names) are stdlib-only
and dependency-free, so every domain package may import them. This is a
deliberate, narrow exception to the dependency rule: the alternative is copying
an identical `Page` struct into eight packages.

## Transactions

Two mechanisms, and the domain never sees `*gorm.DB`:

1. **Repository-owned transactions** — the default. A use case whose writes stay
   inside one context puts the whole transaction inside one repository method:
   `wallet.Apply`, `writer.PublishChapter`, `social.UpsertReview`.
2. **Ambient transactions** — every repository method starts
   `db := dbctx.From(ctx, r.db)`, which returns the transaction published on the
   context when there is one and the pooled handle otherwise. That makes
   repositories transparently composable inside a larger transaction (the
   background jobs use this) without a transaction type appearing in any port
   signature.

## The single coin write path

Every coin movement in the system — top-up, unlock, refund, bonus grant, bonus
expiry, admin adjustment — is expressed as a `wallet.Command` and applied by
`wallet.Repository.Apply`, in one transaction:

1. ensure a `wallet_balances` row exists, then lock it `FOR UPDATE`;
2. replay check on `(user_id, idempotency_key)` — a hit returns the stored
   result and writes nothing;
3. child preconditions (existing unlock, purchase still pending);
4. call the pure planner (`PlanSpend` / `PlanCredit` / `PlanAdjust` /
   `PlanBonusExpiry`), writing any `bonus_expire` row before the main row so the
   ledger stays a valid running total;
5. update the balance and write the child row referencing the new ledger id.

The wallet row is the only lock any coin operation takes first, so there is
exactly one lock-acquisition order and deadlock is structurally impossible.
A concurrent double-unlock therefore resolves to one `200`, one
`409 CHAPTER_ALREADY_UNLOCKED`, and exactly one debit.

The policy is pure and lives in `domain/wallet/spend.go`; only the locking and
row writes live in the repository. That is what lets U-COIN-01 and U-COIN-02 be
plain unit tests.

## Background jobs

`internal/jobs` holds a scheduler and eight jobs. Every job takes `now` as a
parameter instead of calling `time.Now`, so each is deterministic under test,
and each wraps its work in a **transaction-scoped** advisory lock
(`pg_try_advisory_xact_lock`) so multiple runners are safe. A session-scoped
lock would be unsafe here: GORM hands out pooled connections, so the lock could
be taken and released on different ones.

Jobs run inside `cmd/api` when `RUN_JOBS_IN_API=true` (the default outside
production) and in `cmd/worker` otherwise.

## Known risks

- `chapter_bodies.body_source` is rendered into `body_html` without
  sanitisation, so a translator can inject HTML. Acceptable while translator
  accounts are admin-provisioned; close it with an allowlist sanitiser before
  opening translator signup.
- Access tokens remain valid until they expire (15 minutes by default) even
  after logout, because there is no shared denylist. Documented in the API spec;
  clients must discard them.
- Rate limiting is in-process, so with N API replicas the effective limit is N×.
  Rate-limit at the edge in production.
