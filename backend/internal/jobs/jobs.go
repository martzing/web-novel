package jobs

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"gorm.io/gorm"

	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/glossaryrender"
	"github.com/mokchan/webnovel-backend/internal/repository/dbctx"
	walletsvc "github.com/mokchan/webnovel-backend/internal/service/wallet"
)

// withJobLock runs fn inside one transaction holding a Postgres advisory lock
// keyed on the job name, so several API replicas can each run the scheduler
// without duplicating work. It reports false when the lock was already held.
//
// The lock is transaction-scoped rather than session-scoped: GORM hands out
// pooled connections, so a session-level lock could be taken on one connection
// and released on another, leaking it forever. A transaction lock is released
// automatically at commit or rollback.
func withJobLock(ctx context.Context, db *gorm.DB, name string, fn func(ctx context.Context, tx *gorm.DB) error) (bool, error) {
	acquired := false
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw("SELECT pg_try_advisory_xact_lock(?)", lockKey(name)).Scan(&acquired).Error; err != nil {
			return err
		}
		if !acquired {
			return nil
		}
		// Publish the transaction on the context so repositories called inside
		// fn join it rather than opening their own connection.
		return fn(dbctx.With(ctx, tx), tx)
	})
	return acquired, err
}

// lockKey hashes a job name into the 64-bit space advisory locks use.
func lockKey(name string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	return int64(h.Sum64())
}

// BonusExpiryJob writes off lapsed bonus coins (I-COIN-09).
//
// Every write goes through the wallet's single Apply path, so the job cannot
// violate the ledger invariant, and the idempotency key is stamped with the
// date so a same-day re-run is a no-op.
type BonusExpiryJob struct {
	DB     *gorm.DB
	Wallet *walletsvc.Service
	Batch  int
}

func (j *BonusExpiryJob) Name() string { return "bonus_expiry" }

func (j *BonusExpiryJob) Run(ctx context.Context, now time.Time) (Report, error) {
	batch := j.Batch
	if batch <= 0 {
		batch = 500
	}

	processed := 0
	acquired, err := withJobLock(ctx, j.DB, j.Name(), func(ctx context.Context, _ *gorm.DB) error {
		for {
			userIDs, err := j.Wallet.ExpiryCandidates(ctx, now, batch)
			if err != nil {
				return err
			}
			if len(userIDs) == 0 {
				return nil
			}
			for _, userID := range userIDs {
				if _, err := j.Wallet.ExpireBonus(ctx, userID, now); err != nil {
					return err
				}
				processed++
			}
			if len(userIDs) < batch {
				return nil
			}
		}
	})
	return Report{Name: j.Name(), Processed: processed, Skipped: !acquired}, err
}

// GlossaryRerenderJob re-renders chapter bodies whose glossary revision has
// fallen behind the novel's (I-GLO-02).
type GlossaryRerenderJob struct {
	DB    *gorm.DB
	Batch int
}

func (j *GlossaryRerenderJob) Name() string { return "glossary_rerender" }

func (j *GlossaryRerenderJob) Run(ctx context.Context, now time.Time) (Report, error) {
	batch := j.Batch
	if batch <= 0 {
		batch = 200
	}

	processed := 0
	acquired, err := withJobLock(ctx, j.DB, j.Name(), func(ctx context.Context, tx *gorm.DB) error {
		type staleRow struct {
			ChapterID   int64
			NovelID     int64
			GlossaryRev int
			BodySource  string
		}
		var stale []staleRow
		err := tx.Table("chapter_bodies cb").
			Select("cb.chapter_id, c.novel_id, n.glossary_rev, cb.body_source").
			Joins("JOIN chapters c ON c.id = cb.chapter_id").
			Joins("JOIN novels n ON n.id = c.novel_id").
			Where("cb.glossary_rev < n.glossary_rev").
			Limit(batch).
			Scan(&stale).Error
		if err != nil {
			return err
		}

		for _, row := range stale {
			terms, err := loadTerms(tx, row.NovelID)
			if err != nil {
				return err
			}
			rendered := glossaryrender.Render(row.BodySource, terms)

			// The `glossary_rev < ?` guard matters: the glossary_entries
			// trigger bumps the novel's revision once per edited row, so a bulk
			// edit landing mid-render must not let us stamp the body as fresher
			// than what we actually rendered against — that chapter would then
			// never be picked up again.
			res := tx.Model(&entities.ChapterBody{}).
				Where("chapter_id = ? AND glossary_rev < ?", row.ChapterID, row.GlossaryRev).
				Updates(map[string]any{
					"body_html":    rendered.HTML,
					"glossary_rev": row.GlossaryRev,
					"rendered_at":  now,
				})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected > 0 {
				processed++
			}
		}
		return nil
	})
	return Report{Name: j.Name(), Processed: processed, Skipped: !acquired}, err
}

// PublishScheduledJob flips scheduled chapters live once their time arrives
// (W-06).
type PublishScheduledJob struct {
	DB *gorm.DB
}

func (j *PublishScheduledJob) Name() string { return "publish_scheduled" }

func (j *PublishScheduledJob) Run(ctx context.Context, now time.Time) (Report, error) {
	processed := 0
	acquired, err := withJobLock(ctx, j.DB, j.Name(), func(_ context.Context, tx *gorm.DB) error {
		due := tx.Model(&entities.Chapter{}).
			Where("status = ? AND scheduled_at IS NOT NULL AND scheduled_at <= ?",
				entities.ChapterScheduled, now)

		var novelIDs []int64
		if err := due.Session(&gorm.Session{}).Distinct().Pluck("novel_id", &novelIDs).Error; err != nil {
			return err
		}

		// public_at is stamped from the novel's early-access window, so a
		// scheduled publish behaves exactly like a manual one.
		res := tx.Exec(`
			UPDATE chapters c
			   SET status = ?,
			       published_at = ?,
			       -- The cast is required: an untyped placeholder makes
			       -- Postgres read this as interval + interval.
			       public_at = ?::timestamptz + make_interval(hours => n.early_access_hours)
			  FROM novels n
			 WHERE n.id = c.novel_id
			   AND c.status = ?
			   AND c.scheduled_at IS NOT NULL
			   AND c.scheduled_at <= ?`,
			entities.ChapterPublished, now, now, entities.ChapterScheduled, now)
		if res.Error != nil {
			return res.Error
		}
		processed = int(res.RowsAffected)

		for _, novelID := range novelIDs {
			err := tx.Exec(`
				UPDATE novels
				   SET chapters_count = (SELECT COUNT(*) FROM chapters
				                          WHERE novel_id = ? AND status = 'published')
				 WHERE id = ?`, novelID, novelID).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
	return Report{Name: j.Name(), Processed: processed, Skipped: !acquired}, err
}

// StatsRollupJob folds raw read events into the daily aggregates that power the
// writer stats page.
type StatsRollupJob struct {
	DB *gorm.DB
}

func (j *StatsRollupJob) Name() string { return "stats_rollup" }

func (j *StatsRollupJob) Run(ctx context.Context, now time.Time) (Report, error) {
	// Roll up the day that just ended.
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)

	processed := 0
	acquired, err := withJobLock(ctx, j.DB, j.Name(), func(_ context.Context, tx *gorm.DB) error {
		err := tx.Exec(`
			INSERT INTO chapter_daily_stats (chapter_id, day, reads, unique_readers, completions, coins_earned)
			SELECT e.chapter_id,
			       ?::date,
			       COUNT(*),
			       COUNT(DISTINCT e.user_id),
			       -- อ่านจบต่อบท: reads that reached the end of the chapter.
			       COUNT(*) FILTER (WHERE e.completed),
			       -- Unlocks plus tips. A tip writes no chapter_unlocks row, so
			       -- without the second term it would silently vanish from the
			       -- writer's revenue tile.
			       COALESCE((SELECT SUM(u.coins_spent) FROM chapter_unlocks u
			                  WHERE u.chapter_id = e.chapter_id
			                    AND u.unlocked_at >= ?::date
			                    AND u.unlocked_at < ?::date + 1), 0)
			     + COALESCE((SELECT SUM(w.gross_coins) FROM writer_earnings w
			                  WHERE w.chapter_id = e.chapter_id AND w.kind = 'tip'
			                    AND w.created_at >= ?::date
			                    AND w.created_at < ?::date + 1), 0)
			  FROM chapter_read_events e
			 WHERE e.occurred_at >= ?::date AND e.occurred_at < ?::date + 1
			 GROUP BY e.chapter_id
			ON CONFLICT (chapter_id, day) DO UPDATE
			   SET reads = EXCLUDED.reads,
			       unique_readers = EXCLUDED.unique_readers,
			       completions = EXCLUDED.completions,
			       coins_earned = EXCLUDED.coins_earned`,
			day, day, day, day, day, day, day).Error
		if err != nil {
			return err
		}

		res := tx.Exec(`
			INSERT INTO novel_daily_stats (novel_id, day, reads, followers_gained, completions, coins_earned)
			SELECT c.novel_id,
			       ?::date,
			       SUM(s.reads),
			       COALESCE((SELECT COUNT(*) FROM follows f
			                  WHERE f.novel_id = c.novel_id
			                    AND f.since >= ?::date AND f.since < ?::date + 1), 0),
			       SUM(s.completions),
			       SUM(s.coins_earned)
			  FROM chapter_daily_stats s
			  JOIN chapters c ON c.id = s.chapter_id
			 WHERE s.day = ?::date
			 GROUP BY c.novel_id
			ON CONFLICT (novel_id, day) DO UPDATE
			   SET reads = EXCLUDED.reads,
			       followers_gained = EXCLUDED.followers_gained,
			       completions = EXCLUDED.completions,
			       coins_earned = EXCLUDED.coins_earned`,
			day, day, day, day)
		if res.Error != nil {
			return res.Error
		}
		processed = int(res.RowsAffected)
		return nil
	})
	return Report{Name: j.Name(), Processed: processed, Skipped: !acquired}, err
}

// RankingJob snapshots the weekly leaderboard.
//
// The score blends recent reads, coins and followers so a novel with a burst of
// activity can outrank one coasting on an old following.
type RankingJob struct {
	DB    *gorm.DB
	Limit int
}

func (j *RankingJob) Name() string { return "weekly_ranking" }

func (j *RankingJob) Run(ctx context.Context, now time.Time) (Report, error) {
	limit := j.Limit
	if limit <= 0 {
		limit = 50
	}
	period := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	since := period.AddDate(0, 0, -7)

	processed := 0
	acquired, err := withJobLock(ctx, j.DB, j.Name(), func(_ context.Context, tx *gorm.DB) error {
		res := tx.Exec(`
			INSERT INTO ranking_snapshots (novel_id, period, rank, score)
			SELECT novel_id, ?::date, ROW_NUMBER() OVER (ORDER BY score DESC), score
			  FROM (
			        SELECT n.id AS novel_id,
			               COALESCE(SUM(s.reads), 0)
			             + COALESCE(SUM(s.coins_earned), 0) * 3
			             + n.followers_count * 0.1 AS score
			          FROM novels n
			          LEFT JOIN novel_daily_stats s
			                 ON s.novel_id = n.id AND s.day >= ?::date AND s.day < ?::date
			         GROUP BY n.id, n.followers_count
			         ORDER BY score DESC
			         LIMIT ?
			       ) ranked
			ON CONFLICT (novel_id, period) DO UPDATE
			   SET rank = EXCLUDED.rank, score = EXCLUDED.score`,
			period, since, period, limit)
		if res.Error != nil {
			return res.Error
		}
		processed = int(res.RowsAffected)
		return nil
	})
	return Report{Name: j.Name(), Processed: processed, Skipped: !acquired}, err
}

// PartitionJob creates next month's chapter_read_events partition, so events
// never pile up in the DEFAULT partition.
type PartitionJob struct {
	DB *gorm.DB
}

func (j *PartitionJob) Name() string { return "read_event_partitions" }

func (j *PartitionJob) Run(ctx context.Context, now time.Time) (Report, error) {
	processed := 0
	acquired, err := withJobLock(ctx, j.DB, j.Name(), func(_ context.Context, tx *gorm.DB) error {
		// This month and next, so a month boundary never finds itself without
		// a partition.
		for _, offset := range []int{0, 1} {
			name, from, to := PartitionSpecFor(now, offset)
			// DDL cannot take bind parameters, so the bounds are formatted in.
			// Every value here derives from `now` and a fixed layout, never
			// from user input.
			stmt := fmt.Sprintf(
				`CREATE TABLE IF NOT EXISTS %s PARTITION OF chapter_read_events
				 FOR VALUES FROM ('%s') TO ('%s')`,
				name,
				from.Format("2006-01-02 15:04:05-07"),
				to.Format("2006-01-02 15:04:05-07"),
			)
			if err := tx.Exec(stmt).Error; err != nil {
				return err
			}
			processed++
		}
		return nil
	})
	return Report{Name: j.Name(), Processed: processed, Skipped: !acquired}, err
}

// PartitionSpecFor names the monthly partition `offset` months after now, with
// its inclusive lower and exclusive upper bounds.
func PartitionSpecFor(now time.Time, offset int) (name string, from, to time.Time) {
	from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, offset, 0)
	to = from.AddDate(0, 1, 0)
	return "chapter_read_events_" + from.Format("2006_01"), from, to
}

// SessionSweepJob deletes refresh tokens that expired long enough ago that the
// forensic value of keeping them has passed.
type SessionSweepJob struct {
	DB     *gorm.DB
	Retain time.Duration
}

func (j *SessionSweepJob) Name() string { return "session_sweep" }

func (j *SessionSweepJob) Run(ctx context.Context, now time.Time) (Report, error) {
	retain := j.Retain
	if retain <= 0 {
		retain = 7 * 24 * time.Hour
	}

	res := j.DB.WithContext(ctx).
		Where("expires_at < ?", now.Add(-retain)).
		Delete(&entities.RefreshToken{})
	return Report{Name: j.Name(), Processed: int(res.RowsAffected)}, res.Error
}

// ReconcileJob checks the PRD's zero-discrepancy invariant: every wallet's
// cached balance must equal the sum of its ledger deltas. Processed counts the
// wallets that disagree, which should always be zero.
type ReconcileJob struct {
	DB *gorm.DB
}

func (j *ReconcileJob) Name() string { return "wallet_reconcile" }

func (j *ReconcileJob) Run(ctx context.Context, _ time.Time) (Report, error) {
	var mismatches int64
	err := j.DB.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		  FROM wallet_balances w
		  LEFT JOIN (SELECT user_id,
		                    COALESCE(SUM(delta), 0) AS delta,
		                    COALESCE(SUM(bonus_delta), 0) AS bonus_delta
		               FROM coin_ledger GROUP BY user_id) l
		         ON l.user_id = w.user_id
		 WHERE w.balance <> COALESCE(l.delta, 0)
		    OR w.bonus_balance <> COALESCE(l.bonus_delta, 0)`).
		Scan(&mismatches).Error
	return Report{Name: j.Name(), Processed: int(mismatches)}, err
}

func loadTerms(tx *gorm.DB, novelID int64) ([]glossaryrender.Term, error) {
	type termRow struct {
		ID      int64
		TermKey string
		TitleTH string
	}
	var rows []termRow
	err := tx.Table("glossary_entries e").
		Select("e.id, e.term_key, e.title_th").
		Joins("JOIN glossary_groups g ON g.id = e.group_id").
		Where("g.novel_id = ?", novelID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]glossaryrender.Term, 0, len(rows))
	for _, row := range rows {
		out = append(out, glossaryrender.Term{EntryID: row.ID, Key: row.TermKey, TitleTH: row.TitleTH})
	}
	return out, nil
}
