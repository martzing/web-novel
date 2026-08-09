# AGENT.md

Project guidance for AI coding agents working on Mokchan, a xianxia web novel
platform with a Go backend, React frontend, and PostgreSQL.

## Project Snapshot

- Backend: Go 1.25 module in `backend/`, using Gin, GORM, goose migrations,
  PostgreSQL, argon2id password hashing, and `golang-jwt` access tokens.
- Frontend: Vite, React 18, TypeScript, React Router, and TanStack Query in
  `frontend/`.
- Runtime: `docker-compose.yml` starts Postgres, API, and web services.
- Current product phase: PRD phases 1–6 are implemented — catalog and reader,
  accounts and preferences, library and bookmarks, coins with mock purchases
  and chapter unlock, comments and reviews, writer workspace with stats,
  follows, notifications and weekly ranking, the `จัดการผลงาน` works workspace,
  series and related works, and advanced monetisation (arc bundles, tips, and
  auto-unlock with 24-hour early access).
- Not implemented: most `/admin/*` endpoints (only `POST /admin/wallet-adjust`
  exists), and real payment providers — **Phase 7, deliberately the last phase**.

## Source Of Truth

- Start with `README.md` for setup and current project layout.
- Use `docs/README.md` as the index for product and engineering docs.
- API contracts live in `docs/api-spec.md`.
- Database intent lives in `docs/database-schema.md`; authoritative DDL and seed
  data live in `backend/migrations/`.
- Static design references live in `design/`.

## Repository Map

Backend:

- `backend/cmd/api/`: API entrypoint.
- `backend/cmd/migrate/`: migration runner.
- `backend/cmd/worker/`: background job runner.
- `backend/internal/domain/<context>/`: business types, errors, and ports.
- `backend/internal/service/<context>/`: application use cases.
- `backend/internal/repository/<context>/`: database adapters.
- `backend/internal/handler/<context>/`: HTTP handlers and response DTOs.
- `backend/internal/entities/`: persistence models.
- `backend/internal/server/`: composition root and route wiring.
- `backend/internal/auth/`: JWT issue and parse.
- `backend/internal/crypto/argon2id/`: password hashing.
- `backend/internal/middleware/`: auth, roles, rate limiting.
- `backend/internal/ratelimit/`: in-process token bucket and distinct counter.
- `backend/internal/glossaryrender/`: pure `{{term}}` → `<span data-k>` renderer.
- `backend/internal/jobs/`: scheduler and background jobs.
- `backend/internal/storage/`: cover-upload adapters.
- `backend/internal/httpx/`: error envelope, cursors, paging, idempotency keys.
- `backend/internal/domain/page`, `domain/roles`: stdlib-only shared vocabulary.
- `backend/internal/repository/dbctx/`: ambient-transaction plumbing.
- `backend/test/makeme/`: test data builder helpers.
- `backend/test/apitest/`: shared handler integration-test harness.

Frontend:

- `frontend/src/styles/`: design tokens and stylesheets.
- `frontend/src/lib/`: API client, auth context, reader prefs, formatting.
- `frontend/src/components/`: shared UI pieces.
- `frontend/src/layout/`: application shell, sidebar, bottom tab bar.
- `frontend/src/routes/`: page-level route components.

## Bounded Contexts

`identity`, `catalog`, `reading`, `library`, `wallet`, `social`, `writer`,
`notification`. Contexts never import one another; where one needs another's
data it declares a narrow port (for example `reading.Entitlements`, satisfied by
the wallet repository) that the composition root wires up.

## Architecture Rules

- Keep domain packages framework-free. They should not import handlers,
  repositories, persistence entities, Gin, GORM, or other adapters. The only
  exceptions are `domain/page` and `domain/roles`, which are stdlib-only shared
  vocabulary.
- Services depend on domain ports and own use-case orchestration.
- Repositories implement domain ports and contain database-specific code. Each
  method begins `db := dbctx.From(ctx, r.db)` so it joins an ambient
  transaction when one is present.
- Handlers own HTTP parsing, status codes, JSON DTOs, and error mapping. Never
  return a raw error string to the client; use `httpx.Internal`.
- Wire new dependencies in `backend/internal/server/server.go`.
- Add schema changes through new migration files; do not edit applied migration
  semantics casually.
- Keep persistence structs in `internal/entities`; keep JSON tags out of domain
  models unless the architecture changes deliberately.

## Rules With Teeth

These encode defects that have already bitten this codebase.

- **Gin wildcard names must match across a path segment.** `/novels/:slug` and
  `/novels/:id/chapters` panic at startup. Every `/novels/...` route uses `:id`,
  and the service resolves id-or-slug.
- **One coin write path.** Every coin movement goes through
  `wallet.Repository.Apply`, which locks `wallet_balances` FOR UPDATE first.
  Never write `coin_ledger`, `wallet_balances`, `chapter_unlocks` or
  `writer_earnings` directly.
- **`coin_ledger.idempotency_key` and `ref_id` are pointers.** Postgres treats
  NULLs as distinct; a non-pointer string writes `''` and the second key-less
  row per user violates the unique index.
- **Count runes, not bytes, for Thai.** Comment length, bookmark excerpts and
  passwords all use `utf8.RuneCountInString`. Thai is three bytes per character
  and the database CHECKs count characters.
- **Thai search cannot rely on tsvector alone.** `เซียนดาบ` is a substring of
  the single lexeme `เซียนดาบเก้าสายธาร`. The ranking blends trigram
  similarity and ILIKE with `ts_rank`; do not "simplify" it.
- **Rate limiters are per-engine**, constructed inside `server.New`, never
  package globals, or handler tests leak throttle state into each other.
- **`SetTrustedProxies` must stay explicit.** Gin's `ClientIP` trusts
  `X-Forwarded-For` by default, which defeats the per-IP limiter.
- **Advisory locks in jobs are transaction-scoped** (`pg_try_advisory_xact_lock`).
  A session-scoped lock can be taken and released on different pooled
  connections and leak forever.
- **`makeme` builders must be named `ANew<EntityStructName>`.** `Many(n)`
  resolves siblings by reflection on that name.
- **Composite-PK entities must tag every key column** `gorm:"primaryKey"`.
- **The idempotency replay check compares `ref_type` as well as `ref_id`.**
  `spend_unlock` with `ref_type='arc_bundle', ref_id=42` and one with
  `ref_type='chapter_unlock', ref_id=42` are different targets. Keys are also
  namespaced per operation by the service; both belt and braces stay.
- **Arc membership resolves by chapter-number range, never `chapters.arc_id`.**
  `arc_id` is NULL for chapters created before their arc existed, so keying off
  it silently drops chapters and undercharges the reader.
- **The auto-unlock job applies each debit in its own transaction.**
  `withJobLock` claims the batch and nothing more. Moving the debits inside its
  transaction makes one broke subscriber roll back everyone else's unlock;
  `TestAutoUnlockJob_OneBrokeSubscriberDoesNotRollBackTheOthers` exists to catch
  exactly that.
- **Reordering a series renumbers through negatives first.**
  `novels_series_position` is a *partial* unique index, which Postgres cannot
  defer and therefore enforces row by row. A single `UPDATE ... CASE` permuting
  1,2,3 into 2,3,1 collides halfway through, so `SetSeriesOrder` negates every
  position before writing the final order.
- **Novel settings patch on presence, not on non-zero.** `sell_by_arc: false`
  and `free_until_chapter: 0` are real edits. The settings fields are pointers
  in `NovelDraft`, and the handler reads the body twice — typed and as a map —
  because `encoding/json` cannot otherwise report which keys were present.
- **`hidden` is enforced, not decorative.** `ListNovels`, `novelDetail`,
  `WeeklyRanking` and search all exclude it for non-owners. A status the reader
  paths ignore changes nothing.

## Development Commands

Backend:

```bash
cd backend
go mod tidy
gofmt -l .
go vet ./...
go test ./...
go run ./cmd/migrate -cmd up
go run ./cmd/api
go run ./cmd/worker            # or ./cmd/worker -once
```

Frontend:

```bash
cd frontend
npm install
npm run typecheck
npm run build
npm run dev
```

Docker:

```bash
docker compose up --build
docker compose exec api /app/migrate -cmd up
```

Useful smoke checks:

```bash
curl -s http://localhost:8080/health
curl -s http://localhost:8080/api/v1/genres
curl -s 'http://localhost:8080/api/v1/novels?q=เซียนดาบ'
curl -s http://localhost:8080/api/v1/novels/nine-streams-sword-immortal
```

## Testing Guidance

- Domain and service behaviour: focused unit tests with hand-written fakes.
  `internal/service/catalog/service_test.go` is the reference pattern.
- Pure policy lives in the domain so it can be tested without a database:
  `wallet.PlanSpend`, `reading.Decide`, `social.ValidateComment`,
  `writer.PruneRevisions`, `glossaryrender.Render`, `jobs.NextDailyAt`.
- Handler and repository behaviour: integration tests through `test/apitest`,
  which builds a real engine over a testcontainers Postgres.
- **Integration tests need a reachable Docker socket.** Without it testcontainers
  *skips* rather than fails, so a green run can mean nothing ran. On Rancher
  Desktop: `export DOCKER_HOST=unix://$HOME/.rd/docker.sock` and
  `export TESTCONTAINERS_RYUK_DISABLED=true` — Ryuk's reaper cannot bind-mount
  that socket and errors out. Confirm with `go test -v ./... | grep SKIP`, and
  **prune afterwards** (`docker rm -f $(docker ps -aq)`): with Ryuk off nothing
  reaps the containers, and a few sessions can fill the VM's disk, at which
  point the suite fails with `initdb: ... No space left on device`.
- Run `npm run typecheck` or `npm run build` after TypeScript or routing changes.
- `docs/test-cases.md` maps every U and I case to its test file and function.

## Editing Guardrails

- Do not edit generated or local artifacts unless explicitly asked:
  `frontend/node_modules/`, `frontend/dist/`, `frontend/tsconfig.tsbuildinfo`,
  and `.thumbnail`.
- Preserve Thai copy and seeded novel data unless the task specifically changes
  product content.
- Keep frontend changes consistent with the existing app shell and route
  patterns. Styling belongs in `frontend/src/styles/` as tokens and classes;
  inline styles cannot express the responsive breakpoints.
- Update docs when changing setup, commands, API behaviour, schema, or project
  structure.
- Keep changes scoped; avoid broad rewrites when a small patch solves the task.

## Known Risks

- `chapter_bodies.body_source` is rendered into `body_html` without
  sanitisation, so a translator can inject HTML. Acceptable while translator
  accounts are admin-provisioned; close it with an allowlist sanitiser before
  opening translator signup.
- Access tokens stay valid until they expire (15 minutes by default) even after
  logout, because there is no shared denylist. Clients must discard them.
- Rate limiting is in-process: with N API replicas the effective limit is N×.
  Rate-limit at the edge in production.
