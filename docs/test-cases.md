# Test Cases — หมอกจันทร์ (Mokchan)

Tiers:

- **U** — unit, no external deps.
- **I** — integration, real Postgres/Redis.
- **E** — end-to-end, browser + API.
- **L** — load / non-functional.

## Status

Every **U** and **I** case below is implemented; the mapping table at the end of
this document names the file and function for each. **E** and **L** remain out
of scope — see "Out of scope" at the end.

The suite is 575 passing tests with 0 skips. A skip is not a pass here: without
a reachable Docker socket testcontainers skips silently, so always confirm with
`go test -count=1 -v ./... | grep -c -- '--- SKIP'`.

Run them with:

```bash
cd backend
go test ./...
```

The I tier runs against a real PostgreSQL through testcontainers. **Without a
reachable Docker socket testcontainers skips rather than fails**, so a green run
can mean nothing ran. On Rancher Desktop, export
`DOCKER_HOST=unix://$HOME/.rd/docker.sock` and confirm with
`go test -v ./... | grep SKIP`.

## Auth

| ID        | Tier | Scenario                   | Expected                                         |
| --------- | ---- | -------------------------- | ------------------------------------------------ |
| U-AUTH-01 | U    | Password hash              | Uses argon2id with configured params.            |
| I-AUTH-01 | I    | Register duplicate email   | 409, no user created.                            |
| I-AUTH-02 | I    | Login wrong password       | 401; message does not leak whether email exists. |
| I-AUTH-03 | I    | Refresh with revoked token | 401.                                             |

## Catalog

| ID       | Tier | Scenario          | Expected                                          |
| -------- | ---- | ----------------- | ------------------------------------------------- |
| I-CAT-01 | I    | Search "เซียนดาบ" | Matching novel ranked in top 3.                   |
| I-CAT-02 | I    | Genre filter      | Only novels linked in `novel_genres` returned.    |
| I-CAT-03 | I    | Novel detail      | Response has `arcs[]` and `rating_avg` populated. |

## Reader

| ID      | Tier | Scenario                          | Expected                                                           |
| ------- | ---- | --------------------------------- | ------------------------------------------------------------------ |
| I-RD-01 | I    | GET free chapter as anon          | 200 with `body_html`.                                              |
| I-RD-02 | I    | GET locked chapter without unlock | 200 with `locked=true, body_html=null`.                            |
| I-RD-03 | I    | GET locked chapter after unlock   | 200 with `body_html`.                                              |
| I-RD-04 | I    | PUT `progress` `para_anchor=42`   | Subsequent GET returns 42.                                         |
| E-RD-01 | E    | Cross-device resume               | Login on device B; "อ่านต่อ" opens the same chapter and paragraph. |
| E-RD-02 | E    | Theme persistence                 | Switch to sepia, reload — still sepia.                             |
| E-RD-03 | E    | Glossary popover                  | Tap `data-k="qi"` span — popover shows Thai + Chinese + body.      |

## Bookmarks / Library

| ID       | Tier | Scenario                       | Expected                       |
| -------- | ---- | ------------------------------ | ------------------------------ |
| I-BM-01  | I    | Bookmark isolation             | Only owner sees own bookmarks. |
| I-BM-02  | I    | Delete another user's bookmark | 403.                           |
| I-LIB-01 | I    | Move novel reading → done      | Sidebar count updates.         |

## Coins (highest-priority tier)

Invariant, per user: `sum(coin_ledger.delta) == wallet_balances.balance` and `sum(bonus_delta) == wallet_balances.bonus_balance` after bonus expiry has been applied.

| ID         | Tier | Scenario                                                      | Expected                                                                                                            |
| ---------- | ---- | ------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| U-COIN-01  | U    | Spend order                                                   | Bonus balance used before paid balance.                                                                             |
| U-COIN-02  | U    | Expired bonus at spend time                                   | `bonus_expire` row created before spend calculation.                                                                |
| I-COIN-01M | I    | `POST /purchases` twice with same `Idempotency-Key`           | One `pending` purchase row.                                                                                         |
| I-COIN-07  | I    | `mock-complete` twice                                         | Wallet credited once (unique on `coin_ledger.idempotency_key`).                                                     |
| I-COIN-08  | I    | `mock-complete` in prod build (`PAYMENTS_MOCK_ENABLED=false`) | 404.                                                                                                                |
| I-COIN-02  | I    | Concurrent unlock double-click                                | Exactly one 200; other `CHAPTER_ALREADY_UNLOCKED`; one debit.                                                       |
| I-COIN-03  | I    | Insufficient coins                                            | 402 `INSUFFICIENT_COINS`; no ledger row; balance unchanged.                                                         |
| I-COIN-04  | I    | Unlock of `price_coins=0`                                     | 400 `CHAPTER_NOT_FOR_SALE`.                                                                                         |
| I-COIN-05  | I    | Successful unlock                                             | `chapter_unlocks` row references the created `coin_ledger.id`.                                                      |
| I-COIN-06  | I    | Admin adjust                                                  | `coin_ledger.kind='adjust'` with `actor_user_id` and `reason`.                                                      |
| I-COIN-09  | I    | Nightly bonus expiry                                          | User with `bonus_balance=50, bonus_expires_at=yesterday` → `bonus_expire` row `bonus_delta=-50`; `bonus_balance=0`. |
| I-COIN-10  | I    | Spend after bonus expired but before cron ran                 | Bonus treated as 0; unlock uses `balance`; expiry job enqueued.                                                     |
| E-COIN-01  | E    | Buy 240-coin pack via mock                                    | Coins page shows 240 after `mock-complete`.                                                                         |

## Glossary re-render

| ID       | Tier | Scenario                           | Expected                                                                                                            |
| -------- | ---- | ---------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| I-GLO-01 | I    | Publish chapter with `{{ye}}`      | `chapter_bodies.body_html` contains `<span data-k="ye">`; `glossary_rev = novels.glossary_rev`.                     |
| I-GLO-02 | I    | Edit glossary entry `ye`           | `novels.glossary_rev` bumps; re-render worker updates `body_html` and lifts `chapter_bodies.glossary_rev` to match. |
| I-GLO-03 | I    | Reader GET on stale `glossary_rev` | Still returns valid old HTML; after worker completes, returns new HTML.                                             |

## Comments

| ID      | Tier | Scenario                | Expected                           |
| ------- | ---- | ----------------------- | ---------------------------------- |
| I-CM-01 | I    | Comment > 5000 chars    | 400.                               |
| I-CM-02 | I    | Like same comment twice | Count stays +1.                    |
| I-CM-03 | I    | Translator reply        | Serialized with `role=translator`. |

## Writer

| ID      | Tier | Scenario                           | Expected                                       |
| ------- | ---- | ---------------------------------- | ---------------------------------------------- |
| I-WR-01 | I    | Autosave revisions                 | Last 20 kept.                                  |
| I-WR-02 | I    | Publish with future `scheduled_at` | Hidden from readers until time.                |
| I-WR-03 | I    | Cross-writer edit                  | Writer A cannot edit writer B's chapter → 403. |
| I-WR-04 | I    | Stats aggregation                  | Totals match `chapter_daily_stats` fixture.    |

## Security & abuse

| ID       | Tier | Scenario                     | Expected                                                     |
| -------- | ---- | ---------------------------- | ------------------------------------------------------------ |
| I-SEC-01 | I    | Draft chapter body as reader | 404 or 403.                                                  |
| I-SEC-02 | I    | SQL injection in `q=`        | Parameterized; no error, no leak.                            |
| I-SEC-03 | I    | `/auth/*` rate limit         | 60/min per IP.                                               |
| I-SEC-04 | I    | Foreign progress/bookmarks   | Cannot list any other user's.                                |
| I-SEC-05 | I    | Body-fetch abuse             | No more than 20 distinct chapter bodies per user per minute. |

## Non-functional

| ID   | Tier | Scenario                                           | Expected                           |
| ---- | ---- | -------------------------------------------------- | ---------------------------------- |
| L-01 | L    | 500 concurrent chapter GETs (warm)                 | p95 ≤ 400 ms.                      |
| L-02 | L    | 100 concurrent unlocks on same chapter (100 users) | All succeed; no deadlocks in 60 s. |

---

## Where each case lives

Paths are relative to `backend/`.

### Unit tier

| ID        | File · Function                                                                        |
| --------- | -------------------------------------------------------------------------------------- |
| U-AUTH-01 | `internal/crypto/argon2id/hasher_test.go` · `TestHash_UsesConfiguredArgon2idParams`     |
| U-COIN-01 | `internal/domain/wallet/spend_test.go` · `TestPlanSpend_UsesBonusBeforePaid`            |
| U-COIN-02 | `internal/domain/wallet/spend_test.go` · `TestPlanSpend_ExpiredBonusEmitsExpiryEntryBeforeSpend` |

### Integration tier

| ID         | File · Function                                                                                       |
| ---------- | ----------------------------------------------------------------------------------------------------- |
| I-AUTH-01  | `internal/handler/identity/handler_integration_test.go` · `TestRegister_DuplicateEmailReturns409AndCreatesNoUser` |
| I-AUTH-02  | same · `TestLogin_WrongPasswordReturns401WithoutLeakingEmailExistence`                                |
| I-AUTH-03  | same · `TestRefresh_RevokedTokenReturns401AndRevokesFamily`                                           |
| I-CAT-01   | `internal/handler/catalog/handler_integration_test.go` · `TestNovelsEndpoint_SearchGenreAndSort`       |
| I-CAT-02   | same · `TestNovelsEndpoint_SearchGenreAndSort/I-CAT-02_genre_filter`                                  |
| I-CAT-03   | same · `TestNovelDetailEndpoint`                                                                      |
| I-RD-01    | `internal/handler/reading/handler_integration_test.go` · `TestGetChapter_FreeChapterAsAnonymousReturnsBody` |
| I-RD-02    | same · `TestGetChapter_LockedWithoutUnlockReturnsLockedTrueNullBody`                                  |
| I-RD-03    | same · `TestGetChapter_LockedAfterUnlockReturnsBody`                                                  |
| I-RD-04    | same · `TestProgress_PutThenGetReturnsParaAnchor42`                                                   |
| I-BM-01    | `internal/handler/library/handler_integration_test.go` · `TestBookmarks_OnlyOwnerSeesOwnBookmarks`     |
| I-BM-02    | same · `TestBookmarks_DeleteAnotherUsersBookmarkReturns403`                                           |
| I-LIB-01   | same · `TestLibrary_MoveReadingToDoneUpdatesCounts`                                                   |
| I-COIN-01M | `internal/handler/wallet/handler_integration_test.go` · `TestCreatePurchase_SameIdempotencyKeyCreatesOnePendingRow` |
| I-COIN-02  | same · `TestUnlockChapter_ConcurrentDoubleClickYieldsOneDebit`                                        |
| I-COIN-03  | same · `TestUnlockChapter_InsufficientCoinsReturns402AndWritesNoLedgerRow`                            |
| I-COIN-04  | same · `TestUnlockChapter_ZeroPriceReturns400ChapterNotForSale`                                       |
| I-COIN-05  | same · `TestUnlockChapter_ChapterUnlockReferencesCreatedLedgerID`                                     |
| I-COIN-06  | same · `TestAdminWalletAdjust_WritesAdjustLedgerWithActorAndReason`                                   |
| I-COIN-07  | same · `TestMockComplete_TwiceCreditsWalletOnce`                                                      |
| I-COIN-08  | same · `TestMockComplete_Returns404WhenPaymentsMockDisabled`                                          |
| I-COIN-09  | `internal/jobs/jobs_integration_test.go` · `TestBonusExpiryJob_WritesBonusExpireLedgerAndZeroesBalance` |
| I-COIN-10  | `internal/handler/wallet/handler_integration_test.go` · `TestUnlockChapter_ExpiredBonusBeforeCronTreatsBonusAsZero` |
| invariant  | same · `TestLedger_ReconcilesWithTheWalletBalance`; `internal/jobs/jobs_integration_test.go` · `TestReconcileJob_ReportsZeroDiscrepancyOnAConsistentLedger` |
| I-GLO-01   | `internal/handler/writer/handler_integration_test.go` · `TestPublishChapter_RendersGlossarySpansAndStampsGlossaryRev` |
| I-GLO-02   | `internal/jobs/jobs_integration_test.go` · `TestRerenderJob_UpdatesBodyHTMLAndLiftsGlossaryRev`        |
| I-GLO-03   | same · `TestRerenderJob_.../a_stale_body_still_serves_the_old_HTML`                                   |
| I-CM-01    | `internal/handler/social/handler_integration_test.go` · `TestCreateComment_Over5000CharsReturns400`    |
| I-CM-02    | same · `TestLikeComment_TwiceKeepsCountAtOne`                                                          |
| I-CM-03    | same · `TestTranslatorReply_SerializedWithRoleTranslator`                                             |
| I-WR-01    | `internal/handler/writer/handler_integration_test.go` · `TestAutosave_KeepsLast20Revisions`            |
| I-WR-02    | same · `TestPublish_FutureScheduledAtHiddenFromReadersUntilTime`                                      |
| I-WR-03    | same · `TestWriterA_CannotEditWriterBsChapter`                                                        |
| I-WR-04    | same · `TestWriterStats_TotalsMatchChapterDailyStatsFixture`                                          |
| I-SEC-01   | `internal/handler/reading/handler_integration_test.go` · `TestGetChapter_DraftChapterAsReaderReturns404` |
| I-SEC-02   | `internal/handler/catalog/handler_integration_test.go` · `TestSearchNovels_SQLInjectionInQueryIsParameterized` |
| I-SEC-03   | `internal/handler/identity/handler_integration_test.go` · `TestAuthRoutes_RateLimitedPerIP`            |
| I-SEC-04   | `internal/handler/library/handler_integration_test.go` · `TestLibrary_CannotSeeAnotherUsersProgressOrShelf` |
| I-SEC-05   | `internal/handler/reading/handler_integration_test.go` · `TestGetChapter_LimitsDistinctBodiesPerUserPerMinute` |

### Advanced monetisation, series and works management

| ID | File · function |
| --- | --- |
| I-COIN-11 | `internal/handler/wallet/monetization_integration_test.go` · `TestArcBundle_ChargesFifteenPercentOffAndWritesOneLedgerRowWithNUnlocks` |
| I-COIN-12 | `internal/handler/wallet/monetization_integration_test.go` · `TestArcBundle_ConcurrentDoubleBuyYieldsOneDebit` |
| I-COIN-13 | `internal/handler/wallet/monetization_integration_test.go` · `TestArcBundle_RacingASingleChapterUnlockYieldsOneDebitPerChapter` |
| I-COIN-14 | `internal/repository/wallet/apply_integration_test.go` · `TestApply_SameKeyDifferentRefTypeConflicts` |
| I-TIP-01 | `internal/handler/wallet/tip_integration_test.go` · `TestTip_WritesTipLedgerRowAndWriterEarningNetOfPlatformFee` |
| I-TIP-02 | `internal/handler/wallet/tip_integration_test.go` · `TestTip_RefusesToSpendBonusCoinsEvenWhenAmpleAndReports402` |
| I-TIP-03 | `internal/handler/wallet/tip_integration_test.go` · `TestTip_RepeatsWithNewKeysAndReplaysWithTheSameKey` |
| I-EA-01 | `internal/handler/reading/earlyaccess_integration_test.go` · `TestEarlyAccess_NonSubscriberSeesATeaserWithNoBody` |
| I-EA-02 | `internal/handler/reading/earlyaccess_integration_test.go` · `TestEarlyAccess_SubscriberReadsInsideTheWindow` |
| I-EA-03 | `internal/handler/reading/earlyaccess_integration_test.go` · `TestEarlyAccess_OwnershipSurvivesUnsubscribing` |
| I-EA-04 | `internal/handler/reading/earlyaccess_integration_test.go` · `TestEarlyAccess_TranslatorReadsTheirOwnEarlyChapter` |
| I-EA-05 | `internal/handler/reading/earlyaccess_integration_test.go` · `TestEarlyAccess_WindowExpiresAndTheChapterOpensToEveryone` |
| I-EA-06 | `internal/handler/reading/earlyaccess_integration_test.go` · `TestEarlyAccess_BuyingAnEarlyChapterIsRefused` |
| I-AU-01 | `internal/jobs/autounlock_integration_test.go` · `TestAutoUnlockJob_DebitsASubscriberAndWritesTheUnlock` |
| I-AU-02 | `internal/jobs/autounlock_integration_test.go` · `TestAutoUnlockJob_RunningTwiceChargesOnce` |
| I-AU-03 | `internal/jobs/autounlock_integration_test.go` · `TestAutoUnlockJob_OneBrokeSubscriberDoesNotRollBackTheOthers` |
| I-AU-04 | `internal/jobs/autounlock_integration_test.go` · `TestAutoUnlockJob_InsufficientCoinsNotifiesAndKeepsTheSubscriptionActive` |
| I-AU-05 | `internal/jobs/autounlock_integration_test.go` · `TestAutoUnlockJob_RetriesWithBackoffThenStopsAtMaxAttempts` |
| I-AU-06 | `internal/jobs/autounlock_integration_test.go` · `TestAutoUnlockJob_SkipsChaptersOverTheReadersCap` |
| I-AU-07 | `internal/jobs/autounlock_integration_test.go` · `TestAutoUnlockJob_NeverChargesForAManuallyUnlockedChapter` |
| I-AU-08 | `internal/jobs/autounlock_integration_test.go` · `TestAutoUnlockJob_IgnoresInactiveSubscriptionsAndPreSubscriptionChapters` |
| I-SER-01 | `internal/handler/writer/series_integration_test.go` · `TestSeries_ReorderRenumbersFromOneAndSurvivesAPartialList` |
| I-SER-02 | `internal/handler/writer/series_integration_test.go` · `TestSeries_RepeatedPermutationsDoNotTripTheUniquePositionIndex` |
| I-SER-03 | `internal/handler/writer/series_integration_test.go` · `TestSeries_DeletingLeavesTheNovelsInPlaceAndClearsTheirOrder` |
| I-SER-04 | `internal/handler/writer/series_integration_test.go` · `TestSeries_JoiningAnotherTranslatorsSeriesIsForbidden` |
| I-REL-01 | `internal/handler/writer/series_integration_test.go` · `TestRelations_AppearOnTheFarNovelWithTheInverseKind` |
| I-REL-02 | `internal/handler/catalog/series_integration_test.go` · `TestPublicRelated_GroupsBothDirectionsWithLabels` |
| I-WRK-01 | `internal/handler/writer/settings_integration_test.go` · `TestNovelSettings_ExplicitFalseAndZeroAreApplied` |
| I-WRK-02 | `internal/handler/writer/settings_integration_test.go` · `TestNovelSettings_OmittedFieldsAreLeftAlone` |
| I-WRK-03 | `internal/handler/writer/settings_integration_test.go` · `TestCreateChapter_DefaultsPriceFromTheNovelAndHonoursFreeUntil` |
| I-HID-01 | `internal/handler/catalog/series_integration_test.go` · `TestHidden_IsExcludedFromBrowseSearchAndRanking` |
| I-HID-02 | `internal/handler/catalog/series_integration_test.go` · `TestHidden_DetailIs404ForReadersButOpenToItsTranslator` |
| I-PUB-01 | `internal/handler/catalog/series_integration_test.go` · `TestPublicSeries_ReturnsBooksInReadingOrderWithSummedCounts` |

### Supporting unit tests

These have no row above but de-risk the integration tier:

`internal/auth/jwt_test.go` (round trip, alg-none rejected, tampered signature,
foreign secret, expiry) · `internal/domain/wallet/spend_test.go` (insufficient
funds, credit expiry, adjust clamping, `NetCoins` rounding, and
`PlanSpendPaidOnly` refusing an ample bonus) ·
`internal/domain/wallet/bundle_test.go` (`QuoteBundle` rounds the discount down;
`AllocateProportional` sums exactly to the total and never exceeds list price) ·
`internal/domain/reading/visibility_test.go` (`See` across every timing ×
claim combination) ·
`internal/glossaryrender/render_test.go` (binding, unknown markers preserved,
escaping, dedupe) · `internal/domain/reading/entitlement_test.go` ·
`internal/domain/social/validate_test.go` (runes not bytes, `CanDelete`) ·
`internal/domain/identity/prefs_test.go` · `internal/domain/writer/rules_test.go`
(`PruneRevisions`, `ResolveArcID`, `PublishDecision`, `TrendPct`, `Aggregate`,
slug safety) · `internal/httpx/cursor_test.go` ·
`internal/ratelimit/limiter_test.go` (fake clock) ·
`internal/jobs/scheduler_test.go` (Bangkok rollover, month clamping) ·
`internal/service/identity/service_test.go` ·
`internal/service/catalog/service_test.go` ·
`internal/handler/catalog/handler_integration_test.go` ·
`TestServerNew_RegistersAllRoutesWithoutPanic` (guards the gin wildcard trap).

### Out of scope

| ID                              | Why                                                                                                                    |
| ------------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| E-RD-01, E-RD-02, E-RD-03, E-COIN-01 | Browser end-to-end; the repository has no browser toolchain. Each has an I-tier proxy: I-RD-04 (cross-device resume), the prefs round-trip test, I-GLO-01 (glossary payload), I-COIN-07 (wallet after top-up). |
| L-01, L-02                      | Load testing needs k6/vegeta and a provisioned environment. The design choices that serve them are in place: keyset pagination, one lock-acquisition order in `wallet.Apply`, and the `chapters_novel_pub` partial index. Note `test/makeme` caps the pool at 5 connections, so L-02 would need that raised. |
