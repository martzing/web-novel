# TASK.md — Design update: works management, monetisation & series

Tracks the increment that brings the codebase in line with the updated
mockups in `design/`. Phases 1–4 are already shipped; this is Phases 5–6.

Legend: `[ ]` todo · `[~]` in progress · `[x]` done

---

## Scope decisions

| Decision | Choice |
| --- | --- |
| Arc bundle purchase (−15%) | Full flow — atomic multi-chapter unlock |
| Tips at chapter end | Full flow — new ledger kind, reader control |
| Auto-unlock + 24h early access | Full flow — opt-in, fan-out job, viewer-dependent visibility |
| Release schedule (`รอบปล่อยบทใหม่`) | Display-only metadata; does not drive the scheduler |
| Series | Full — writer management *and* the public `ชุดหนังสือ` page |
| Covers | Upload *or* template (style + colour + text), rendered in CSS |
| Reordering | Native HTML5 drag, no new dependency, with up/down fallback |
| Real payment providers | Deferred — moved to the last phase in `prd.md`, not built |

---

## Part A — Backend

- [ ] **A1 · Migration `0007_monetization.sql`, entities, fixtures**
  - [ ] `novels`: `source_chapters_count`, `price_per_chapter`,
        `free_until_chapter`, `sell_by_arc`, `tips_enabled`,
        `early_access_hours`, `release_schedule`, `cover_style`,
        `cover_color`, `cover_text`, `series_position`, `series_note`
  - [ ] `novels.status` CHECK gains `'hidden'`
  - [ ] `chapters.public_at TIMESTAMPTZ` + backfill + index
  - [ ] `coin_ledger.kind` CHECK gains `'tip'` (note the production caveat)
  - [ ] `writer_earnings.kind ('unlock'|'tip')` + indexes
  - [ ] `series`: `owner_user_id`, `slug` (add → backfill → NOT NULL → unique)
  - [ ] New tables: `auto_unlock_subscriptions`, `auto_unlock_attempts`,
        `novel_relations`
  - [ ] Entities + `test/makeme` builders (`ANew<EntityStructName>`, every
        composite-PK column tagged `primaryKey`)

- [ ] **A2 · Fix latent idempotency bug — do this first**
  - [ ] `sameTarget` in `internal/repository/wallet/apply.go` must compare
        `ref_type` as well as `ref_id`, or an arc bundle and a chapter unlock
        sharing an id would replay each other's receipts

- [ ] **A3 · Tips**
  - [ ] `wallet.PlanSpendPaidOnly` — paid coins only, distinct
        `ErrInsufficientPaidCoins`
  - [ ] `KindTip`, `ChildTip`, platform fee via `NetCoins`
  - [ ] `POST /chapters/:id/tip` (1–1000, idempotency-key mandatory)
  - [ ] Extend `StatsRollupJob` so tips reach the writer's revenue tile

- [ ] **A4 · Arc bundles**
  - [ ] `QuoteBundle` + `AllocateProportional` (largest remainder; parts sum
        exactly to the debit)
  - [ ] `ChildArcBundle` with `Items` — N unlocks + N earnings, one ledger row
  - [ ] Arc membership by chapter-number range, never `chapters.arc_id`
  - [ ] `GET /arcs/:id/bundle`, `POST /arcs/:id/unlock`

- [ ] **A5 · Early access (read side)**
  - [ ] `public_at` written by `PublishChapter`, `UnpublishChapter`,
        `PublishScheduledJob`
  - [ ] `reading.See(availability, access, now)` — hidden / teaser / full,
        beside `Decide`, which stays untouched
  - [ ] `ChapterForSale` port; `403 EARLY_ACCESS_ONLY` on the sale paths
  - [ ] `reading.Subscriptions` narrow port satisfied by the wallet repo

- [ ] **A6 · Auto-unlock (write side)**
  - [ ] Subscription CRUD (`/me/auto-unlock*`) with a per-chapter cap
  - [ ] `AutoUnlockChapter` with a derived idempotency key
  - [ ] `AutoUnlockJob` — advisory lock claims the batch, each debit in its
        own transaction so one broke subscriber cannot roll back the rest
  - [ ] Attempt outcomes, backoff, one notification, no auto-pause

- [ ] **A7 · Series and relations**
  - [ ] Series CRUD owned by a translator; membership, position, note
  - [ ] `novel_relations` with five kinds; `same_world` mirrored for display
  - [ ] Public `GET /series/:id`

- [ ] **A8 · Novel settings, counts, `hidden`**
  - [ ] `NovelDraft`/`UpdateNovel` carry every new field
  - [ ] `CreateChapter` defaults price from the novel, free below the threshold
  - [ ] Hidden novels excluded from list, detail, search and ranking

## Part B — Frontend

- [ ] **B1 · Works screen** (`/works`) — work tree, 5 tabs, 4 sheets, cover
      template renderer, drag reorder, deep link into the editor
- [ ] **B2 · Write wizard** — pick work → pick/create chapter → edit
- [ ] **B3 · Detail rebuild** — resume card, two-stat split, filter pills,
      arc-grouped expandable ToC, buy-arc and auto-unlock controls
- [ ] **B4 · Public series page** (`/series/:id`)
- [ ] **B5 · Library shelf** — cover and title link to detail, new
      `รายละเอียดและสารบัญ` button beside the continue CTA, translated/source
      progress. `ShelfRow`'s outer `<Link>` must go or the nested buttons are
      invalid markup
- [ ] **B6 · Reader and home** — ToC tags and dimming, tip control, teaser
      state, `บทที่แปลแล้ว` counts

## Part C — Documentation

- [ ] **C1 · `docs/prd.md`** — new scope sections; §8 renumbered so real
      payment providers become the **last** phase (7) and stay unimplemented;
      §9 locked decisions; §6 metrics
- [ ] **C2 · rest of `docs/`** — `user-stories.md`, `api-spec.md`,
      `database-schema.md`, `test-cases.md`, `architecture.md`, `README.md`
- [ ] **C3 · root docs** — `AGENT.md` (three new rules with teeth),
      `README.md`, `CLAUDE.md`

## Verification

- [ ] `gofmt -l .` and `go vet ./...` clean
- [ ] `go test -count=1 ./...` green with **0 skips**
      (`export DOCKER_HOST=unix://$HOME/.rd/docker.sock` first)
- [ ] `npm run typecheck && npm run build`
- [ ] Smoke: buy an arc, tip a chapter, auto-unlock fan-out, early-access teaser
- [ ] Release gate: `sum(coin_ledger.delta) == wallet_balances.balance`
      for every user, via `ReconcileJob`
- [ ] Responsive check at 375 / 768 / 1024 / 1440px
