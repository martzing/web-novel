# Docs — หมอกจันทร์ (Mokchan)

| File                                     | Purpose                                                                                          |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------ |
| [user-stories.md](user-stories.md)       | Reader / Writer / Admin stories with acceptance criteria.                                        |
| [prd.md](prd.md)                         | Vision, scope, non-goals, metrics, risks, phased rollout, locked decisions.                      |
| [architecture.md](architecture.md)       | Backend hexagonal architecture, bounded contexts, the single coin write path, background jobs.   |
| [api-spec.md](api-spec.md)               | REST endpoint catalogue under `/api/v1`, plus implementation status and deviations.              |
| [test-cases.md](test-cases.md)           | Unit / integration / e2e / load cases, and where each implemented case lives.                    |
| [database-schema.md](database-schema.md) | Human-readable schema; authoritative DDL lives in [backend/migrations/](../backend/migrations).  |

Phases 1–6 of the PRD are implemented: the catalog and reader, accounts, the
library, the coin economy, comments and reviews, the writer workspace, follows
and ranking, and now works management, series and related works, arc bundles,
tips, and auto-unlock with 24-hour early access.

Phase 7 — wiring real payment providers — is deliberately last and not built.
See the "Not implemented" section of the root [README](../README.md).
