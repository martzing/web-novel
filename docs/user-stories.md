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

## Admin

| ID   | Story                                   | Acceptance                                       |
| ---- | --------------------------------------- | ------------------------------------------------ |
| A-01 | Moderate reported comments/novels.      | Soft-delete via `deleted_at`; audit logged.      |
| A-02 | Create/edit coin packs and set bonuses. | `is_best_value` at most one active per currency. |
| A-03 | Adjust a wallet balance with reason.    | Writes `coin_ledger.kind='adjust'` with actor.   |
| A-04 | Approve or reject payout requests.      | State transitions enforced.                      |
| A-05 | Manage genres.                          | Cannot delete a genre used by any novel.         |
