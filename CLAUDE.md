# CLAUDE.md

Claude-specific entrypoint for this repository. Also read `AGENT.md`; it is the
shared project playbook for stack details, architecture rules, commands, and
editing guardrails.

## Quick Context

Mokchan (หมอกจันทร์) is a Thai-first web novel platform for translated Chinese
xianxia and wuxia works. PRD phases 1–6 are implemented: catalog and reader,
accounts and synced preferences, library and bookmarks, a coin economy with
mock purchases and chapter unlock, comments and reviews, the writer workspace
with stats, follows, notifications and weekly ranking, the `จัดการผลงาน` works
workspace, series and related works, and advanced monetisation — arc bundles,
tips, and auto-unlock with 24-hour early access.

Phase 7 (real payment providers) and most `/admin/*` endpoints are deliberately
not implemented. Phase 7 is intentionally the **last** phase.

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
- The frontend is feature-sliced by the same bounded contexts as the backend.
  Put a screen in `frontend/src/features/<context>/pages/`, its endpoints in
  that context's `api.ts`, and its React Query keys and hooks in its
  `queries.ts`. Cross-context imports go through `@/features/<context>`, never
  a deeper path. Truly shared code lives in `frontend/src/shared/`
  (`api/`, `ui/`, `lib/`, `styles/`), and routing/providers only in
  `frontend/src/app/`.
- Never hand-write a React Query key; use the feature's key factory.

`AGENT.md` has a "Rules With Teeth" section listing the traps that have already
caused defects here — gin wildcard names, the single coin write path, counting
runes for Thai, and Thai search ranking. Read it before touching those areas.

## Common Commands

Backend:

```bash
cd backend
go test ./...
go run ./cmd/migrate -cmd up
go run ./cmd/api
go run ./cmd/worker
```

Backend integration tests need a reachable Docker socket, or testcontainers
**skips** them silently:

```bash
export DOCKER_HOST=unix://$HOME/.rd/docker.sock    # Rancher Desktop
export TESTCONTAINERS_RYUK_DISABLED=true           # Ryuk cannot mount it
cd backend && go test -count=1 ./...
docker rm -f $(docker ps -aq)                      # nothing reaps them otherwise
```

Frontend:

```bash
cd frontend
nvm use            # 24.9.0, per .nvmrc
npm run typecheck
npm test
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
