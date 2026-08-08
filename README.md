# หมอกจันทร์ (Mokchan) · Xianxia web novel platform

React (Vite + TS) + Go (chi + pgx) + PostgreSQL 16 + Redis 7. Phase-1 scaffold: catalog browse, novel detail, and chapter reading are wired end-to-end from Postgres to the browser. Coins, unlocks, auth, and writer workspace land in later phases.

## Layout

```
web-novel/
├── backend/                      Go API (chi, pgx, goose)
│   ├── cmd/
│   │   ├── api/                  HTTP server entrypoint
│   │   └── migrate/              goose migration runner
│   ├── internal/
│   │   ├── config/               environment/config loader
│   │   ├── db/                   pgx pool setup
│   │   ├── domain/catalog/       catalog business types and ports
│   │   ├── entities/             shared persistence/domain entities
│   │   ├── handler/catalog/      HTTP handlers and DTO mapping
│   │   ├── httpx/                JSON/error response helpers
│   │   ├── repository/catalog/   Postgres catalog repository
│   │   ├── server/               chi router and middleware wiring
│   │   └── service/catalog/      catalog application service + tests
│   ├── migrations/               embedded SQL migrations and seed data
│   ├── test/makeme/              test data builder helpers
│   ├── .env.example              sample backend environment
│   ├── Dockerfile
│   ├── Makefile
│   ├── go.mod
│   └── go.sum
├── frontend/                     Vite + React + TypeScript
│   ├── src/
│   │   ├── layout/               app shell/navigation
│   │   ├── lib/api.ts            typed API client
│   │   ├── routes/               Home, Browse, Novel, Reader, Stub
│   │   ├── App.tsx               route declarations
│   │   ├── main.tsx              React bootstrap
│   │   └── styles.css            global UI styles
│   ├── .env.example              sample frontend environment
│   ├── Dockerfile
│   ├── index.html
│   ├── nginx.conf
│   ├── package.json
│   ├── package-lock.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── docs/                         product and engineering docs
│   ├── api-spec.md
│   ├── architecture.md
│   ├── database-schema.md
│   ├── prd.md
│   ├── README.md
│   ├── test-cases.md
│   └── user-stories.md
├── design/                       static design mocks and support assets
│   ├── Xianxia Platform.dc.html
│   ├── Xianxia Reader.dc.html
│   └── support.js
├── .claude/skills/               local agent skill references
├── .gitignore
├── docker-compose.yml            Postgres, Redis, API, and web services
└── README.md
```

Generated/local artifacts are intentionally left out of the map, including
`frontend/node_modules/`, `frontend/dist/`, `frontend/tsconfig.tsbuildinfo`, and
`.thumbnail`.

## Requirements

- Go 1.22+
- Node 18+ (Node 16 works with the pinned Vite 4 but is not recommended)
- Docker (for Postgres/Redis; or install them locally)
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

## Local dev without docker

```bash
# 1. start Postgres + Redis
docker compose up -d postgres redis

# 2. backend
cd backend
cp .env.example .env
go mod tidy
go run ./cmd/migrate -cmd up      # applies 0001_init.sql and 0002_seed.sql
go run ./cmd/api                  # listens on :8080

# 3. frontend
cd ../frontend
cp .env.example .env
npm install
npm run dev                       # http://localhost:5173, proxies /api → :8080
```

## Verifying end-to-end

```bash
curl -s http://localhost:8080/health
curl -s http://localhost:8080/api/v1/genres              | jq .
curl -s http://localhost:8080/api/v1/novels?sort=popular | jq .
curl -s http://localhost:8080/api/v1/novels/nine-streams-sword-immortal | jq .
```

Then browse http://localhost:5173 — the home page fetches from `/api/v1/novels`, the browse page filters by genre, novel detail lists arcs + chapters, and clicking chapter 87 opens the reader with the seeded chapter body.

## What's implemented (Phase 1)

- Full DDL for all Phase 1–4 tables (including coin ledger, glossary trigger, partitioned read events).
- Seed data matching the design: novel `เซียนดาบเก้าสายธาร`, 4 arcs, chapters 86–88, glossary, coin packs.
- Read-only API: `GET /genres`, `GET /novels`, `GET /novels/{slug}`, `GET /novels/{id}/chapters`, `GET /chapters/{id}` (locked chapters return `body_html: null`).
- React shell (sidebar), home, browse with genre chips, novel detail, chapter reader honouring `data-k` glossary spans.

## What's next

- **Phase 1b**: auth (register/login/JWT), reader prefs sync, bookmarks, reading progress.
- **Phase 2**: coin wallet + mock payment (`POST /purchases`, `POST /purchases/{id}/mock-complete`), chapter unlock, comments.
- **Phase 3**: writer workspace (drafts, publish, glossary editor, stats).
- **Phase 4**: reviews, follows, notifications, ranking.

Planning artefacts (user stories, PRD, API spec, tests, schema deltas) live in the session memory file `/memories/session/plan.md`.
