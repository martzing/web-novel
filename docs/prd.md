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
- Every surface that shows a chapter count shows **two**: `บทที่แปลแล้ว`
  (translated) and `บทในต้นฉบับ` (source), so a reader can see how far a work
  still has to run. The source figure is translator-entered.
- Novel detail rebuilt around a resume-reading card, chapter filter pills
  (`ทั้งหมด` / `อ่านฟรี` / `ปลดล็อกแล้ว` / `ยังไม่ปลดล็อก`), and an arc-grouped
  expandable ToC whose rows carry per-chapter tags. Chapters beyond the
  translated count are listed dimmed as `ยังไม่แปล`.

### Reader

- 3 themes (light / sepia / dark), 3 font families, 3 column widths, adjustable font size & line height.
- Immersive tap toggle for chrome.
- Inline term spans open a floating note popover.
- Side panels: ToC, Glossary, Bookmarks.
- Reading position + prefs synced per user.

### Library

- Tabs: reading / saved / done, with progress bars.
- Each shelf row is two navigation targets, not one: the cover and title open
  the novel's detail page, and a `รายละเอียดและสารบัญ` button sits beside the
  accent-outlined continue CTA. Progress reads in translated chapters
  (`บทที่ 87 จาก 88 บทที่แปลแล้ว · มีบทใหม่ 1 บท`) rather than a bare percentage.

### Coins & unlock

- Balance card, "best value" pack badge, ledger history.
- Chapter unlock: locked chapters show a price; unlocking spends coins atomically.
- Bonus coins expire; spend order = bonus first.
- **Arc bundles** — where a translator enables it, a reader buys a whole arc at
  a platform-wide 15% discount. One ledger row, N `chapter_unlocks` rows, and
  one `writer_earnings` row per chapter (an arc's chapters can have different
  translators).
- **Tips** — a reader can tip the translator at the end of a chapter, 1–1000
  coins. Tips are a distinct ledger kind, draw on **purchased coins only**, and
  pay the same platform fee as an unlock.
- **Auto-unlock with 24-hour early access** — a per-novel opt-in with a
  per-chapter coin cap. Subscribers are debited by a scanning job as chapters
  publish, and read new chapters 24 hours before everyone else. Non-subscribers
  see the chapter listed as a teaser during the window.

### Works management

`จัดการผลงาน` is the translator's setup workspace, and the first entry in the
writer navigation. A two-pane master/detail layout: a work tree grouped by
series on the left, and five tabs on the right — `ข้อมูลเรื่อง`, `หน้าปก`,
`ภาคและบท`, `ชุดและเรื่องเกี่ยวเนื่อง`, `ราคาและการเผยแพร่`.

- Novel metadata, genres (max 3), and the source chapter count.
- Cover: an uploaded image **or** a generated template (style + colour + text),
  rendered in CSS so it stays crisp at every size and costs no request.
- Arcs, and a deep link from any chapter into the editor with both the work and
  the chapter preselected.
- Series membership, drag-ordered reading list, and related works.
- Pricing and publishing: price per chapter, free-until-chapter, arc sales,
  early access, tips, publication status, and `รอบปล่อยบทใหม่`.

`ซ่อนจากหน้าร้าน` is a real publication status: a hidden novel is absent from
browse, search, ranking and its own detail page for everyone except its
translator, who can still open and edit it.

### Series & related works

- A **series** (`ชุดหนังสือ`) is a translator-owned collection with a
  drag-ordered reading list. Each book carries a note — "อ่านเล่มนี้ก่อนได้
  ไม่สปอยล์" — which is the reason the public series page exists rather than a
  genre filter. A novel belongs to at most one series.
- **Related works** (`เรื่องเกี่ยวเนื่อง`) link two novels under one of five
  kinds: `sequel` (ภาคต่อโดยตรง), `prequel` (ปฐมบท), `spinoff` (ภาคแยก),
  `side_story` (ภาคพิเศษ), `same_world` (เกิดในโลกเดียวกัน). Links are stored
  once and shown from both novels, the far side carrying the inverse kind.

### Comments & reviews

- Chapter-scoped comments with one-level replies, spoiler-hide toggle, likes, sort tabs.
- Per-novel star reviews (1–5).

### Writer workspace

- `เขียนบท` is a three-step wizard: pick the work, pick or create the chapter,
  then edit. The steps derive from the URL, so a deep link from `จัดการผลงาน`
  lands straight in the editor.
- Left rail with drafts and published chapters (with status pill).
- Editor: title, arc pill, body, toolbar (footnote / glossary bind / scene break), price picker, save-draft / schedule-publish.
- A new chapter inherits `price_per_chapter` from its novel, and is forced free
  below `free_until_chapter`.
- Stats page: KPI tiles (reads, followers, coins, `อ่านจบต่อบท`) with a 14-day
  trend indicator. Coin earnings include tips as well as unlocks.
  - `อ่านจบต่อบท` is completions ÷ reads over the window, from a `completed`
    flag on the read event. It is a ratio of the window's own totals rather
    than an average of per-chapter rates, so a chapter with a single read
    cannot swing it.
  - The tiles count **coins, not baht**. A coin has no single baht value: the
    rate depends on which pack the reader bought, so any THB figure beside a
    coin total would be a guess presented as a fact. Fiat appears only where it
    is real — the `รายได้` tab, where `available_satang` is what a translator
    can actually withdraw.
- `รายได้` is a second tab beside the stats, not a fourth item in the writer
  navigation: the design fixes that rail at three entries, and money is
  something a translator checks while looking at performance anyway. It shows
  net earnings, the withdrawable balance, and the payout request form.
- Glossary management (`อภิธานศัพท์`) is a sixth tab in `จัดการผลงาน`. Deleting
  a term bumps the novel's `glossary_rev`, so every chapter that bound it is
  re-rendered and the marker degrades to plain text; deleting a group that
  still holds terms is refused rather than cascading.

### Onboarding

- Two steps. First pick favourite genres — **at least three**, because one pick
  produces a home page that looks like a genre filter rather than a
  personalised feed. Then pick a preferred length (สั้น / กลาง / ยาว).
- Both feed the home ranking through one column: the length becomes a weight on
  the chosen genres in `user_genre_prefs` (สั้น=1, กลาง=2, ยาว=3) rather than a
  field of its own. It is therefore not separately recoverable — if the length
  ever needs to drive anything by itself, it needs its own column first.

### Reader-facing follow

- `ติดตามทั้งชุด` follows every visible book in a series in one action. It is a
  fan-out over the existing per-novel follows rather than a subscription row of
  its own, so the button reports `none` / `partial` / `all`, and a book that
  joins the series later leaves it `partial` rather than following itself.

## 6. Success metrics

| Metric                            | Target       |
| --------------------------------- | ------------ |
| D7 reader retention               | ≥ 25%        |
| Median time-to-first-unlock       | ≤ 3 sessions |
| Chapter-unlock success rate       | ≥ 99.95%     |
| Arc-bundle purchase success rate  | ≥ 99.95%     |
| Tip success rate                  | ≥ 99.9%      |
| Auto-unlock fan-out lag (p95)     | ≤ 15 min     |
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

Wiring real payment providers is deliberately the **last** phase: every phase
before it delivers reader- or translator-facing value on the mock provider, and
none of them is blocked on a gateway integration.

| Phase | Content | Status |
| --- | --- | --- |
| 1 | Reader + catalog + accounts (no coins) | done |
| 2 | Coins + unlock + comments, **mock payment provider only** | done |
| 3 | Writer workspace + stats + payouts | done |
| 4 | Reviews, follows, notifications, ranking | done |
| 5 | Works management, series & related works, translated/source chapter split | done |
| 6 | Advanced monetisation — arc bundles, tips, auto-unlock + 24h early access | done |
| 7 (last, post-MVP) | Wire real payment providers (Omise / TrueMoney / Stripe) behind the existing `purchases` table | **not implemented** |

Phase 7 keeps its original shape: the `purchases` table, the idempotent
`(provider, provider_ref)` webhook key, and the `PAYMENTS_MOCK_ENABLED` flag are
already in place, so the work is a provider adapter rather than a schema change.

The checkout screen the design mocks up in full — accordion payment methods, a
saved-card checkbox, a PromptPay QR countdown, a discount code and VAT lines —
lands with Phase 7 rather than before it. Building those against a mock would
put a convincing payment form in front of readers that cannot take a payment;
the current screen says so plainly instead.

## 8a. Deferred

Drawn in the design mocks, deliberately not built, and not lost:

| Item | Why it is deferred |
| --- | --- |
| `อ่านสะสมสัปดาห์นี้ N ชั่วโมง` on the home page | Needs per-session read duration — a schema column, a client heartbeat and a rollup job — for one line of copy. Revisit if retention work needs a time-spent metric anyway. |
| `ค่าเฉลี่ยหมวด 71%` beside `อ่านจบต่อบท` | The per-novel rate ships; the genre benchmark needs an aggregate across every novel in a genre, which is a different query shape and a different cache story. |
| `ดูรายได้ที่คาดการณ์` in the pricing tab | A forecast multiplies price × subscribers × an assumed conversion rate; each assumption compounds, and the `รายได้` tab already shows what actually happened. |
| Pinch-to-zoom in the reader | Collides with browser zoom on the platforms that matter, and the settings panel already has A− / A+ bound to the same preference. |
| Chinese characters on the onboarding genre chips | Would need `genres` to carry a Chinese name for decoration alone. |
| The design's four `รอบปล่อยบทใหม่` options | The implementation offers five, including `เดือนละครั้ง`, and drops the day names the design bakes into the label — a cadence and a schedule are different promises. |

## 9. Locked design decisions

- **Multi-translator per chapter**: NO. `chapters.translator_id` stays a single FK.
- **Chapter body storage**: pre-rendered `body_html` with baked `<span data-k>` spans; keep `body_source` for the editor. Track `glossary_rev` on `chapter_bodies` and re-render when the parent novel's `glossary_rev` bumps.
- **Payment gateway**: none in this phase. Endpoints are `POST /purchases` (creates a `pending` mock purchase) and `POST /purchases/{id}/mock-complete` (dev/admin gated, credits wallet via the same ledger helper a real webhook would use). Feature flag: `PAYMENTS_MOCK_ENABLED`.
- **Bonus expiry cadence**: nightly cron 03:00 Asia/Bangkok writes `coin_ledger(kind='bonus_expire')` and zeroes `bonus_balance`. Spend-time guard treats expired bonus as 0 defensively.
- **Tips are paid-coin-only**: bonus coins are promotional, and letting them fund
  a tip turns them into an unbounded, reader-chosen payout obligation with
  nothing exchanged — sign up, collect the bonus, tip a controlled translator,
  withdraw. A distinct `INSUFFICIENT_PAID_COINS` (402) is returned rather than a
  generic shortfall, because "เหรียญไม่พอ" beside a visible bonus balance is a
  guaranteed support ticket.
- **The arc discount is a platform constant, not per-novel data**: the settings
  panel offers a checkbox, not a percentage field, and a column today would let a
  novel set 100%.
- **`chapters.public_at` is snapshotted at publish, not derived at read time**:
  deriving it would force a join to `novels` into the hottest queries in the app,
  and would let a translator flipping the setting retroactively un-publish the
  last day of chapters.
- **An early-access chapter is a teaser, not a hidden one**: it stays in the ToC
  for everyone with its metadata and no body. It is the conversion surface for
  auto-unlock, and `novels.chapters_count` is a stored column that a
  viewer-dependent list would contradict for 24 hours.
- **A novel belongs to at most one series**: the design's series picker is
  single-select, so the reading-order slot and note live on `novels` rather than
  in a join table that would carry a uniqueness constraint and no extra data.
- **Auto-unlock fan-out is a scanning job keyed on the missing-unlock
  invariant**: debiting inside the publish request would make publish
  O(subscribers) and hold its transaction across many wallet locks. Scanning for
  *absent* `chapter_unlocks` rows is idempotent by construction, self-heals after
  an incident, and never charges a reader who unlocked manually.
- **`รอบปล่อยบทใหม่` is display-only metadata**: it tells readers when to come
  back and deliberately does not drive the publishing scheduler.
