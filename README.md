# หมอกจันทร์ (Mokchan) · Xianxia web novel platform

React (Vite + TS + TanStack Query) + Go 1.25 (Gin + GORM + pgx) + PostgreSQL 16.

PRD phases 1–6 are implemented end to end: catalog and reader, accounts with
synced reader preferences, library and bookmarks, a coin economy with mock
purchases and chapter unlock, comments and reviews, the writer workspace with
stats, follows, notifications and weekly ranking — plus the `จัดการผลงาน` works
workspace, series and related works, and advanced monetisation (arc bundles at
−15%, tips, and auto-unlock with a 24-hour early-access window).

A novel's detail page and its series page show two chapter counts side by side —
`บทที่แปลแล้ว` (translated) against `บทในต้นฉบับ` (source) — so a reader can see
how far a translation still has to run. The shelf reads progress against the
translated count instead (`บทที่ 87 จาก 88 บทที่แปลแล้ว`), which is the figure
that answers "how much is left for me to read".

## Layout

```
web-novel/
├── backend/                        Go API (Gin, GORM, goose)
│   ├── cmd/
│   │   ├── api/                    HTTP server entrypoint
│   │   ├── migrate/                goose migration runner
│   │   └── worker/                 background job runner
│   ├── internal/
│   │   ├── auth/                   HS256 access tokens
│   │   ├── config/                 environment loader
│   │   ├── crypto/argon2id/        password hashing
│   │   ├── db/                     GORM connection factory
│   │   ├── domain/<context>/       business types, ports, pure policy
│   │   ├── entities/               GORM persistence models
│   │   ├── glossaryrender/         {{term}} → <span data-k> renderer
│   │   ├── handler/<context>/      Gin handlers and response DTOs
│   │   ├── httpx/                  error envelope, cursors, paging, idempotency
│   │   ├── jobs/                   scheduler and background jobs
│   │   ├── middleware/             auth, roles, rate limiting
│   │   ├── ratelimit/              in-process token bucket, distinct counter
│   │   ├── repository/<context>/   GORM adapters
│   │   ├── repository/dbctx/       ambient-transaction plumbing
│   │   ├── server/                 composition root and route wiring
│   │   ├── service/<context>/      application use cases
│   │   └── storage/                cover-upload adapters
│   ├── migrations/                 embedded SQL migrations and seed data
│   ├── test/apitest/               handler integration-test harness
│   ├── test/makeme/                test data builders (testcontainers Postgres)
│   ├── .env.example
│   ├── Dockerfile
│   ├── Makefile
│   ├── go.mod
│   └── go.sum
├── frontend/                       Vite + React + TypeScript
│   ├── src/
│   │   ├── app/                    composition root: entry, providers, router
│   │   ├── features/<context>/     one folder per bounded context
│   │   │   ├── api.ts              that context's endpoints and wire types
│   │   │   ├── queries.ts          query-key factory and React Query hooks
│   │   │   ├── components/         components only this context uses
│   │   │   ├── pages/              route-level screens
│   │   │   └── index.ts            the context's public surface
│   │   ├── layout/                 app shell, sidebar, bottom tab bar
│   │   ├── shared/api/             HTTP client and cross-context wire types
│   │   ├── shared/lib/             formatting and reorder helpers
│   │   ├── shared/styles/          design tokens and stylesheets
│   │   └── shared/ui/              the UI kit, one component per file
│   ├── .env.example
│   ├── .nvmrc                      Node 18+ (vitest needs it)
│   ├── Dockerfile
│   ├── index.html
│   ├── nginx.conf
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── docs/                           product and engineering docs
├── design/                         static design mocks and support assets
├── docker-compose.yml              Postgres, API, and web services
├── AGENT.md                        shared agent playbook
├── CLAUDE.md                       Claude-specific entrypoint
└── README.md
```

Generated/local artifacts are intentionally left out of the map, including
`frontend/node_modules/`, `frontend/dist/`, `frontend/tsconfig.tsbuildinfo`, and
`.thumbnail`.

## Requirements

- Go 1.25+
- Node 18+
- Docker (for Postgres, and for the backend integration tests)
- `psql` if you want to poke the database

## Quick start (docker-compose)

```bash
docker compose up --build
# in another shell, run migrations & seed
docker compose exec api /app/migrate -cmd up
```

Open:

- Web app → http://localhost:8081
- API → http://localhost:8080/health
- Postgres → localhost:5432 (user/pass/db: `mokchan`)

The seeded translator account is `mokchan@example.com` / `mokchan-dev`.

## Local dev without docker

```bash
# 1. start Postgres
docker compose up -d postgres

# 2. backend
cd backend
cp .env.example .env
go mod tidy
go run ./cmd/migrate -cmd up      # applies 0001 … 0009
go run ./cmd/api                  # listens on :8080

# 3. frontend
cd ../frontend
cp .env.example .env
npm install
npm run dev                       # http://localhost:5173, proxies /api → :8080
```

`JWT_SECRET` must be at least 16 characters or the API refuses to start.

Nine background jobs (bonus expiry, glossary re-render, scheduled publishing,
auto-unlock fan-out, stats rollups, weekly ranking, session sweep, wallet
reconciliation, and read-event partition creation) run inside the API when
`RUN_JOBS_IN_API=true`, the default outside production. To run them separately:

```bash
cd backend
go run ./cmd/worker          # scheduled
go run ./cmd/worker -once    # run every job once and exit
```

## Verifying end-to-end

```bash
curl -s http://localhost:8080/health
curl -s http://localhost:8080/api/v1/genres | jq .
curl -s 'http://localhost:8080/api/v1/novels?q=เซียนดาบ' | jq .

TOKEN=$(curl -sX POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"smoke","email":"smoke@example.com","password":"hunter2hunter2"}' \
  | jq -r .token)

curl -s http://localhost:8080/api/v1/auth/me -H "Authorization: Bearer $TOKEN" | jq .
curl -sX POST http://localhost:8080/api/v1/purchases \
  -H "Authorization: Bearer $TOKEN" -H 'Idempotency-Key: smoke-1' \
  -H 'Content-Type: application/json' -d '{"pack_id":"3"}' | jq .
```

Then browse http://localhost:5173: register, pick genres, browse, open a novel,
read a free chapter, change the reader theme and reload, tap an inline glossary
term, hit a paid chapter, top up through the mock checkout, unlock, comment and
rate.

## Testing

```bash
cd backend
gofmt -l .
go vet ./...
go test ./...
```

The repository/handler suites run against a real PostgreSQL through
testcontainers. **Without a reachable Docker socket they skip rather than fail**,
so a green run can mean nothing ran. On Rancher Desktop:

```bash
export DOCKER_HOST=unix://$HOME/.rd/docker.sock
export TESTCONTAINERS_RYUK_DISABLED=true   # Ryuk cannot mount that socket
cd backend && go test -count=1 ./...
go test -count=1 -v ./... | grep SKIP      # should print nothing
docker rm -f $(docker ps -aq)              # nothing reaps them with Ryuk off
```

Frontend:

```bash
cd frontend
npm run typecheck
npm run build
```

`docs/test-cases.md` maps every unit and integration case to its test file and
function.

## Not implemented

- **Phase 7** — real payment providers (Omise / TrueMoney / Stripe), deliberately
  the last phase. Phase 2 ships a mock provider behind `PAYMENTS_MOCK_ENABLED`;
  the checkout screen is a UI shell over `POST /purchases` + `/mock-complete`
  and transmits no card data.
- Most `/admin/*` endpoints. Only `POST /admin/wallet-adjust` is implemented,
  because the coin test matrix requires it.
- Browser end-to-end and load tests (tiers E and L in `docs/test-cases.md`).
