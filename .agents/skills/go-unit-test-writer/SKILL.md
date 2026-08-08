---
name: go-unit-test-writer
description: "Use when: writing, updating, reviewing, or fixing Go tests in webnovel-backend; unit tests, table-driven tests, service tests, repository tests, handler tests, makeme fixtures, go test, gofmt."
---

# Go Unit Test Writer

Use this skill when the user asks to add, update, review, or fix Go tests in this repository.

## Goal

Write focused Go tests that are deterministic, maintainable, and consistent with the existing Clean Architecture boundaries.

Default to true unit tests with hand-written fakes or existing mocks. Use database-backed `makeme` fixtures only for handler and repository layer tests that must exercise real persistence behavior against PostgreSQL or SQL Server.

## Required Context

Before editing tests:

1. Read the relevant production file and nearby existing `_test.go` files.
2. Confirm the layer under test: handler, service, repository, model, utility, or job flow.
3. Confirm whether the behavior can be tested with pure fakes or existing mocks; use `makeme` only for handler or repository layer tests.
4. Read [references/makeme_test.md](references/makeme_test.md) before using `makeme`.

## Test Design Rules

- Prefer small, deterministic tests over broad fragile tests.
- Prefer table-driven tests when covering multiple input/output cases.
- Prefer hand-written fakes for small interfaces.
- Use existing mocking frameworks only if the repository already uses them in nearby tests.
- For service, model, utility, and job-flow tests, use fake/mock interfaces or fake methods instead of database-backed fixtures.
- Do not add new production or test dependencies unless the user explicitly approves them.
- Unit tests must not require real network services, cloud services, or shared mutable state.
- Use unique test data names to avoid collisions.
- Use `context.WithTimeout` for code paths that accept a context.
- Avoid fixed sleeps; use polling with a deadline when waiting is unavoidable.
- Keep tests close to the changed package using the existing package style.
- Preserve request/response contracts and existing error envelope behavior.

## When To Use Makeme

Use `makeme` only when a handler or repository test should validate real persistence behavior or full HTTP-to-repository wiring that cannot be meaningfully covered with a fake, such as:

- repository SQL/GORM behavior;
- handler integration behavior with real app wiring and persisted fixtures;
- handler or repository flows that require both PostgreSQL and SQL Server fixture data.

Do not use `makeme` for service, model, utility, or pure business logic tests. Use fake/mock interfaces, fake methods, or in-memory values instead.

When using `makeme`:

- Start with `m := makeme.New(t)` for PostgreSQL.
- Use `makeme.New(t, makeme.SQLServer)` for SQL Server fixtures.
- Use both instances only when a flow genuinely crosses both databases.
- Use `With`, `WithEach`, `Many`, `From`, or `FromPointers` to keep fixture intent explicit.
- Use `m.Reset()` between subtests that share one `MakeMe` instance.
- Prefer one `makeme.New(t)` per test for clean isolated state.
- Add a local `ptr[T any]` helper only when pointer fields are needed and no helper already exists.

The detailed fixture API and examples are in [references/makeme_test.md](references/makeme_test.md). Treat that file as the source of truth for `makeme` patterns.

## Workflow

1. Locate the narrow behavior to test and inspect existing conventions.
2. Add or update the smallest useful tests.
3. Run `gofmt` on changed Go files.
4. Run the narrowest relevant `go test` command first.
5. Run a wider `go test` command if the change can affect more packages.
6. Report tests added or changed, commands run, and any failures.

## Command Guidance

Use the narrowest command that covers the changed package first:

```bash
go test -count=1 ./service/...
go test -count=1 ./repository/...
go test -count=1 ./handler/...
go test -count=1 -run TestName ./service/...
```

If the change affects shared models, response contracts, routing, config, or cross-package behavior, follow with:

```bash
go test -count=1 ./...
```

Always run `gofmt` on changed Go files before tests.

## Output Expectations

When finished, summarize:

- which tests were added or changed;
- which files were touched;
- which commands were run;
- whether each command passed or failed;
- any remaining test gap or known limitation.

Keep the final report concise and specific.
