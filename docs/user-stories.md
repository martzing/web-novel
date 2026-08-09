# User Stories — หมอกจันทร์ (Mokchan)

Personas: **R** = Reader, **W** = Translator/Writer, **A** = Admin.

## Reader

| ID   | Story                                                                                                              | Acceptance                                                                                   |
| ---- | ------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------- |
| R-01 | As a reader, I see a home page with "continue reading", weekly featured, weekly Top-N ranking, and latest updates. | Cards render within 400 ms of TTFB; "continue" opens the last chapter at the last paragraph. |
| R-02 | As a reader, I can search novels by Thai title, Chinese title, translator, or character name.                      | Query "เซียนดาบ" returns the matching novel in the top 3.                                    |
| R-03 | As a reader, I can filter novels by genre chips.                                                                   | Selecting "เซียน" filters the list; the result count matches.                                |
| R-04 | As a reader, I can view a novel detail page with cover, synopsis, rating, arcs, and full ToC.                      | ToC groups chapters under their arc name and shows lock icons for paid chapters.             |
| R-05 | As a reader, I can add/remove a novel from my library and move it between reading / saved / done.                  | Sidebar count updates in real time.                                                          |
| R-06 | As a reader, I can adjust theme, font family, size, line height, and column width; settings persist.               | Reload preserves settings; a new device shows the same settings after login.                 |
| R-07 | As a reader, I can tap an inline glossary term to see a popover with title, Chinese, and body.                     | Popover reads from the glossary entry, not chapter-local text.                               |
| R-08 | As a reader, I can open a full glossary side panel grouped by category and search it.                              | Search "ตันเถียน" filters entries live.                                                      |
| R-09 | As a reader, I can bookmark my current paragraph, list all bookmarks, and jump back.                               | Bookmark stores `para_anchor` + excerpt; delete requires ownership.                          |
| R-10 | As a reader, my reading progress is saved automatically and syncs across devices.                                  | Login on device B resumes on the same paragraph within ±1.                                   |
| R-11 | As a reader, I can navigate prev/next chapter and via arc-grouped ToC.                                             | Next on the last chapter of a novel is disabled.                                             |
| R-12 | As a reader, I can top up coins by choosing a coin pack.                                                           | Redirect to mock checkout; on success, wallet is credited exactly once.                      |
| R-13 | As a reader, I can spend coins to unlock a locked chapter; unlocked chapters stay unlocked.                        | Idempotent; concurrent double-click yields one debit.                                        |
| R-14 | As a reader, I can see wallet balance, bonus balance, bonus expiry, and transaction history.                       | Ledger paginated; entry kinds localized.                                                     |
| R-15 | As a reader, I can post a chapter comment, hide as spoiler, and like others' comments.                             | Like is one-per-user; own like removable.                                                    |
| R-16 | As a reader, I can see translator replies distinctly.                                                              | Reply tagged `role=translator` with a red badge.                                             |
| R-17 | As a reader, I can follow a novel and get notified of new chapters.                                                | Notification row created on chapter publish.                                                 |
| R-18 | As a reader, I can set favorite genres in onboarding.                                                              | Genre chips saved to `user_genre_prefs`.                                                     |
| R-19 | As a reader, I can rate a novel 1–5 and review it.                                                                 | One review per user per novel.                                                               |

## Translator / Writer

| ID   | Story                                                                         | Acceptance                                                   |
| ---- | ----------------------------------------------------------------------------- | ------------------------------------------------------------ |
| W-01 | Create a novel with th/cn titles, author, description, cover, genres, series. | Slug auto-generated, editable before first publish.          |
| W-02 | Create arcs and assign a chapter range.                                       | Chapter's `arc_id` auto-resolves from `chapter_no`.          |
| W-03 | Write/edit a chapter with autosave.                                           | Autosave every 20 s or on blur; last 20 revisions retained.  |
| W-04 | Insert footnotes and bind glossary terms inline.                              | Selection wraps in `<span data-k="...">`; ref recorded.      |
| W-05 | Manage per-novel glossary (groups + entries).                                 | CRUD; `term_key` unique per group.                           |
| W-06 | Save draft, schedule publish, or publish now.                                 | Scheduled chapters hidden from readers until `scheduled_at`. |
| W-07 | Set chapter unlock price in coins.                                            | Price ≥ 0; 0 = free.                                         |
| W-08 | View stats: reads, followers, coins, over 14 / 30 days.                       | KPI matches sum from `chapter_daily_stats`.                  |
| W-09 | Reply to comments on my chapters.                                             | Reply flagged `is_translator=true`.                          |
| W-10 | Request payout of coins to fiat.                                              | Creates `payouts` row in `requested` state.                  |
| W-11 | Manage all my works from one screen, grouped by series.                       | จัดการผลงาน lists every owned novel under its series; selecting one loads its five tabs. |
| W-12 | Record how many chapters exist in the original.                               | `source_chapters_count` is writer-entered and feeds every `บทในต้นฉบับ` figure. |
| W-13 | Build a cover from a template when I have no artwork.                         | Style + colour + text render in CSS; an uploaded image always wins over a template. |
| W-14 | Create a series and drag its books into reading order.                        | Order is 1..n with no gaps after any permutation; readers see the same order on the series page. |
| W-15 | Add a note to each book explaining where it sits in the series.               | Note is stored per novel and shown on the public series page. |
| W-16 | Link a work to its sequel, prequel, spin-off, side story, or shared world.    | Stored once; the far novel shows the inverse kind and cannot unlink it. |
| W-17 | Set a price once for the whole novel rather than per chapter.                 | New chapters inherit `price_per_chapter`; chapters at or below `free_until_chapter` are forced free. |
| W-18 | Sell a whole arc at a discount.                                               | With `sell_by_arc` on, readers see a buy-arc control priced at 85% of the sum. |
| W-19 | Accept tips from readers at the end of a chapter.                             | With `tips_enabled` on, a tip credits `writer_earnings` net of the platform fee. |
| W-20 | Give my auto-unlock subscribers a 24-hour head start.                         | New chapters carry a `public_at` snapshot; non-subscribers see a teaser until it passes. |
| W-21 | Hide a work from the storefront without deleting it.                          | `hidden` removes it from browse, search, ranking and its detail page for everyone but me. |
| W-22 | Tell readers how often new chapters land.                                     | `รอบปล่อยบทใหม่` is shown on the detail page; it does not schedule anything. |
| W-23 | Jump from a chapter in จัดการผลงาน straight into the editor.                   | The link preselects both the work and the chapter rather than restarting the wizard. |

## Reader — series, bundles, tips and auto-unlock

| ID   | Story                                                              | Acceptance                                                                 |
| ---- | ------------------------------------------------------------------ | -------------------------------------------------------------------------- |
| R-20 | See how far a translation has to run.                              | Every novel surface shows `บทที่แปลแล้ว` beside `บทในต้นฉบับ`.               |
| R-21 | Browse a series in the order the translator recommends.            | `/series/{id}` lists the books by position with each one's note.            |
| R-22 | Find a work's sequels, prequels and side stories.                  | Related works are grouped by kind on the detail page.                       |
| R-23 | Buy a whole arc in one go and pay less than chapter by chapter.    | The quote shows gross, 15% discount and total; the purchase writes one ledger row and N unlocks. |
| R-24 | Tip a translator for a chapter I enjoyed.                          | 1–1000 coins, purchased coins only; a repeat tip needs a new idempotency key. |
| R-25 | Have new chapters unlock automatically, up to a price I set.       | Per-novel opt-in with a per-chapter cap; running out of coins notifies once and keeps the subscription on. |
| R-26 | Read new chapters before everyone else while subscribed.           | Subscribers read immediately; others see a teaser with metadata and no body for 24 hours. |
| R-27 | Reach a novel's detail page and full ToC from my shelf.            | The cover and title link to the detail page; a `รายละเอียดและสารบัญ` button sits beside the continue CTA. |
| R-28 | Filter a long table of contents.                                   | Pills for ทั้งหมด / อ่านฟรี / ปลดล็อกแล้ว / ยังไม่ปลดล็อก over an arc-grouped, expandable list. |

## Admin

| ID   | Story                                   | Acceptance                                       |
| ---- | --------------------------------------- | ------------------------------------------------ |
| A-01 | Moderate reported comments/novels.      | Soft-delete via `deleted_at`; audit logged.      |
| A-02 | Create/edit coin packs and set bonuses. | `is_best_value` at most one active per currency. |
| A-03 | Adjust a wallet balance with reason.    | Writes `coin_ledger.kind='adjust'` with actor.   |
| A-04 | Approve or reject payout requests.      | State transitions enforced.                      |
| A-05 | Manage genres.                          | Cannot delete a genre used by any novel.         |
