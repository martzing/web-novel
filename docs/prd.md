# Product Requirements — หมอกจันทร์ (Mokchan)

## 1. Vision

หมอกจันทร์ is a Thai-first web novel platform for translated Chinese xianxia / wuxia works. It combines a distraction-free reader with rich cultural context (glossary, footnotes, arc structure) and a fair per-chapter coin economy that pays translators.

## 2. Goals

- Best-in-class Thai typography and reading experience.
- Provably correct coin accounting (auditable ledger, zero discrepancy).
- First-class writer workflow: drafts, scheduling, stats.

## 3. Non-goals (MVP)

- Native mobile apps — responsive web only.
- Original (non-translated) works.
- Audio narration.
- Non-Thai UI localisation.

## 4. Personas

- **Reader** — Thai xianxia fan, mostly mobile, cost-sensitive.
- **Translator** — solo or small "สำนักแปล" team, expects a competent writer workspace.
- **Admin** — platform ops.

## 5. Feature scope

### Discovery

- Home: welcome, "continue reading", weekly featured banner, weekly Top-N, latest updates.
- Browse: text search, genre chip filter, popularity/latest sort.
- Novel detail: cover, synopsis, rating summary, series link, glossary count, arc-grouped ToC.

### Reader

- 3 themes (light / sepia / dark), 3 font families, 3 column widths, adjustable font size & line height.
- Immersive tap toggle for chrome.
- Inline term spans open a floating note popover.
- Side panels: ToC, Glossary, Bookmarks.
- Reading position + prefs synced per user.

### Library

- Tabs: reading / saved / done, with progress bars.

### Coins & unlock

- Balance card, "best value" pack badge, ledger history.
- Chapter unlock: locked chapters show a price; unlocking spends coins atomically.
- Bonus coins expire; spend order = bonus first.

### Comments & reviews

- Chapter-scoped comments with one-level replies, spoiler-hide toggle, likes, sort tabs.
- Per-novel star reviews (1–5).

### Writer workspace

- Left rail with drafts and published chapters (with status pill).
- Editor: title, arc pill, body, toolbar (footnote / glossary bind / scene break), price picker, save-draft / schedule-publish.
- Stats page: KPI tiles (reads, followers, coins, THB) with 14-day trend indicator.

### Onboarding

- Pick favorite genres → feeds home ranking personalization.

## 6. Success metrics

| Metric                            | Target       |
| --------------------------------- | ------------ |
| D7 reader retention               | ≥ 25%        |
| Median time-to-first-unlock       | ≤ 3 sessions |
| Chapter-unlock success rate       | ≥ 99.95%     |
| Wallet reconciliation discrepancy | 0            |
| p95 chapter GET (warm cache)      | ≤ 400 ms     |
| p95 chapter GET (cold)            | ≤ 900 ms     |

## 7. Risks & mitigations

| Risk                               | Mitigation                                                                |
| ---------------------------------- | ------------------------------------------------------------------------- |
| Thai FTS relevance                 | `pg_trgm` for MVP; Meilisearch fallback when relevance complaints appear. |
| Coin correctness                   | Single write path in Go; integration tests; nightly reconciliation job.   |
| Paid-chapter piracy                | No bulk body endpoint; per-user rate limits; server-side watermarking.    |
| Payment provider variance (future) | Idempotent webhook by `(provider, provider_ref)`.                         |

## 8. Rollout phases

1. **Phase 1** — Reader + catalog + accounts (no coins).
2. **Phase 2** — Coins + unlock + comments, using a **mock payment provider only** (no external gateway this phase).
3. **Phase 3** — Writer workspace + stats + payouts.
4. **Phase 4** — Reviews, follows, notifications, ranking.
5. **Phase 5 (post-MVP)** — Wire real payment providers (Omise / TrueMoney / Stripe) behind the existing `purchases` table.

## 9. Locked design decisions

- **Multi-translator per chapter**: NO. `chapters.translator_id` stays a single FK.
- **Chapter body storage**: pre-rendered `body_html` with baked `<span data-k>` spans; keep `body_source` for the editor. Track `glossary_rev` on `chapter_bodies` and re-render when the parent novel's `glossary_rev` bumps.
- **Payment gateway**: none in this phase. Endpoints are `POST /purchases` (creates a `pending` mock purchase) and `POST /purchases/{id}/mock-complete` (dev/admin gated, credits wallet via the same ledger helper a real webhook would use). Feature flag: `PAYMENTS_MOCK_ENABLED`.
- **Bonus expiry cadence**: nightly cron 03:00 Asia/Bangkok writes `coin_ledger(kind='bonus_expire')` and zeroes `bonus_balance`. Spend-time guard treats expired bonus as 0 defensively.
