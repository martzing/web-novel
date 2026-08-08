# CLAUDE.md

Claude-specific entrypoint for this repository. Also read `AGENT.md`; it is the
shared project playbook for stack details, architecture rules, commands, and
editing guardrails.

## Quick Context

Mokchan is a xianxia web novel platform. The current implementation covers a
Phase 1 read-only experience: catalog browse, novel detail, and chapter reader.
Later phases include auth, reader preferences, wallet/coins, purchases,
unlocks, comments, writer workspace, reviews, follows, notifications, and
ranking.

## Before Editing

- Inspect the current implementation before changing behavior.
- Check `README.md` for setup and layout, then open the relevant file under
  `docs/` for product, API, architecture, schema, or test intent.
- Treat `backend/migrations/` as the source of truth for database DDL and seed
  data.
- Prefer the current code over stale prose if documentation and implementation
  disagree, then update docs when the task calls for it.

## Coding Rules

- Backend layers follow domain, service, repository, handler, and server wiring.
- Keep framework and persistence concerns out of domain packages.
- Put HTTP DTOs and status mapping in handlers.
- Put database mapping and queries in repositories.
- Register new backend routes and dependencies from `backend/internal/server/`.
- Keep React route/page work under `frontend/src/routes/`, shared API calls in
  `frontend/src/lib/api.ts`, and shell/navigation work under
  `frontend/src/layout/`.

## Common Commands

Backend:

```bash
cd backend
go test ./...
go run ./cmd/migrate -cmd up
go run ./cmd/api
```

Frontend:

```bash
cd frontend
npm run typecheck
npm run build
npm run dev
```

Docker:

```bash
docker compose up --build
docker compose exec api /app/migrate -cmd up
```

## Do Not Touch By Default

- `frontend/node_modules/`
- `frontend/dist/`
- `frontend/tsconfig.tsbuildinfo`
- `.thumbnail`

Keep documentation, tests, and validation proportional to the size and risk of
the change.
