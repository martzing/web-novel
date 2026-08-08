# AGENT.md

Project guidance for AI coding agents working on Mokchan, a xianxia web novel
platform with a Go backend, React frontend, PostgreSQL, and Redis.

## Project Snapshot

- Backend: Go module in `backend/`, currently using Gin, GORM, goose
  migrations, PostgreSQL, and Redis configuration.
- Frontend: Vite, React 18, TypeScript, and React Router in `frontend/`.
- Runtime: `docker-compose.yml` starts Postgres, Redis, API, and web services.
- Current product phase: read-only catalog, novel detail, and chapter reader.
  Auth, wallet, purchases, unlocks, comments, and writer workspace are planned
  for later phases.

## Source Of Truth

- Start with `README.md` for setup and current project layout.
- Use `docs/README.md` as the index for product and engineering docs.
- API contracts live in `docs/api-spec.md`.
- Database intent lives in `docs/database-schema.md`; authoritative DDL and seed
  data live in `backend/migrations/`.
- Static design references live in `design/`.

## Repository Map

- `backend/cmd/api/`: API entrypoint.
- `backend/cmd/migrate/`: migration runner.
- `backend/internal/domain/<context>/`: business types, errors, and ports.
- `backend/internal/service/<context>/`: application use cases.
- `backend/internal/repository/<context>/`: database adapters.
- `backend/internal/handler/<context>/`: HTTP handlers and response DTOs.
- `backend/internal/entities/`: persistence models.
- `backend/internal/server/`: composition root and route wiring.
- `backend/test/makeme/`: test data builder helpers.
- `frontend/src/layout/`: application shell/navigation.
- `frontend/src/lib/api.ts`: typed API client.
- `frontend/src/routes/`: page-level route components.

## Architecture Rules

- Keep domain packages framework-free. They should not import handlers,
  repositories, persistence entities, Gin, GORM, or other adapters.
- Services depend on domain ports and own use-case orchestration.
- Repositories implement domain ports and contain database-specific code.
- Handlers own HTTP parsing, status codes, JSON DTOs, and error mapping.
- Wire new dependencies in `backend/internal/server/server.go`.
- Add schema changes through new migration files; do not edit applied migration
  semantics casually.
- Keep persistence structs in `internal/entities`; keep JSON tags out of domain
  models unless the architecture changes deliberately.

## Development Commands

Backend:

```bash
cd backend
go mod tidy
go test ./...
go run ./cmd/migrate -cmd up
go run ./cmd/api
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
curl -s http://localhost:8080/api/v1/novels?sort=popular
curl -s http://localhost:8080/api/v1/novels/nine-streams-sword-immortal
```

## Testing Guidance

- Prefer focused Go unit tests for domain/service behavior.
- Use hand-written fakes for service tests; `backend/internal/service/catalog`
  is the current reference pattern.
- Use real Postgres integration coverage for repository behavior when database
  behavior matters.
- Run `npm run typecheck` or `npm run build` after TypeScript or routing
  changes.
- For migrations, verify both the migration runner and affected API paths when
  practical.

## Editing Guardrails

- Do not edit generated or local artifacts unless explicitly asked:
  `frontend/node_modules/`, `frontend/dist/`, `frontend/tsconfig.tsbuildinfo`,
  and `.thumbnail`.
- Preserve Thai copy and seeded novel data unless the task specifically changes
  product content.
- Keep frontend changes consistent with the existing app shell and route
  patterns.
- Update docs when changing setup, commands, API behavior, schema, or project
  structure.
- Keep changes scoped; avoid broad rewrites when a small patch solves the task.
