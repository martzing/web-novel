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
