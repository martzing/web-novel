# Docs — หมอกจันทร์ (Mokchan)

| File                                     | Purpose                                                                                          |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------ |
| [user-stories.md](user-stories.md)       | Reader / Writer / Admin stories with acceptance criteria.                                        |
| [prd.md](prd.md)                         | Vision, scope, non-goals, metrics, risks, phased rollout, locked decisions.                      |
| [architecture.md](architecture.md)       | Backend hexagonal architecture, bounded contexts, the single coin write path, background jobs.   |
| [api-spec.md](api-spec.md)               | REST endpoint catalogue under `/api/v1`, plus implementation status and deviations.              |
| [test-cases.md](test-cases.md)           | Unit / integration / e2e / load cases, and where each implemented case lives.                    |
| [database-schema.md](database-schema.md) | Human-readable schema; authoritative DDL lives in [backend/migrations/](../backend/migrations).  |

Phases 1–4 of the PRD are implemented. See the "Not implemented" section of the
root [README](../README.md) for what is deliberately left out.
