package jobs

import (
	"log/slog"
	"time"

	"gorm.io/gorm"

	walletsvc "github.com/mokchan/webnovel-backend/internal/service/wallet"
)

// BuildScheduler registers every background job on its schedule.
//
// The cadences come from docs/prd.md: bonus expiry runs nightly at 03:00
// Asia/Bangkok, and the rollups follow it so the stats page sees fresh numbers
// each morning.
func BuildScheduler(db *gorm.DB, wallet *walletsvc.Service, now func() time.Time, logger *slog.Logger) *Scheduler {
	s := NewScheduler(now, logger)
	bkk := Bangkok()

	s.Add(&BonusExpiryJob{DB: db, Wallet: wallet},
		func(after time.Time) time.Time { return NextDailyAt(after, bkk, 3, 0) })

	s.Add(&GlossaryRerenderJob{DB: db}, Every(30*time.Second))
	s.Add(&PublishScheduledJob{DB: db}, Every(time.Minute))

	s.Add(&StatsRollupJob{DB: db},
		func(after time.Time) time.Time { return NextDailyAt(after, bkk, 3, 30) })

	s.Add(&RankingJob{DB: db},
		func(after time.Time) time.Time { return NextWeeklyAt(after, bkk, time.Monday, 4, 0) })

	s.Add(&SessionSweepJob{DB: db},
		func(after time.Time) time.Time { return NextDailyAt(after, bkk, 4, 0) })

	s.Add(&ReconcileJob{DB: db},
		func(after time.Time) time.Time { return NextDailyAt(after, bkk, 4, 30) })

	s.Add(&PartitionJob{DB: db},
		func(after time.Time) time.Time { return NextMonthlyAt(after, bkk, 25, 2, 0) })

	return s
}
