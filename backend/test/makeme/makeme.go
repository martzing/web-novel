package makeme

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/mokchan/webnovel-backend/migrations"
)

const (
	defaultPostgresImage = "postgres:16-alpine"
	testDatabaseName     = "mokchan_test"
	testDatabaseUser     = "mokchan_test"
	testDatabasePassword = "mokchan_test"
)

var runningNumber atomic.Int64

var sharedDatabases = struct {
	sync.Mutex
	resources map[sharedDatabaseKey]*sharedDatabase
}{
	resources: make(map[sharedDatabaseKey]*sharedDatabase),
}

type DatabaseEngine string

const Postgres DatabaseEngine = "postgres"

type Option interface {
	apply(*newConfig)
}

type optionFunc func(*newConfig)

func (fn optionFunc) apply(config *newConfig) { fn(config) }

func (engine DatabaseEngine) apply(config *newConfig) { config.engine = engine }

type newConfig struct {
	engine       DatabaseEngine
	databaseName string
}

type sharedDatabaseKey struct {
	engine       DatabaseEngine
	databaseName string
	image        string
}

type sharedDatabase struct {
	dsn          string
	container    *tcpostgres.PostgresContainer
	snapshotName string
	used         bool
}

// MakeMe is one test's handle to a migrated database and the fixture builders.
type MakeMe struct {
	DB     *gorm.DB
	Engine DatabaseEngine
	DSN    string
	t      testing.TB
}

// New returns a MakeMe ready to build fixtures. The underlying container is reused
// across calls in the same test process; each New call restores the container from
// a post-migration snapshot so tests see a clean database.
func New(t testing.TB, opts ...Option) *MakeMe {
	t.Helper()

	config := defaultNewConfig()
	for _, opt := range opts {
		if opt != nil {
			opt.apply(&config)
		}
	}
	if strings.TrimSpace(config.databaseName) == "" {
		t.Fatalf("database name must not be empty")
	}
	if config.engine != Postgres {
		t.Fatalf("unsupported makeme database engine: %s", config.engine)
	}

	shared := getSharedPostgres(t, config)
	preparePostgres(t, shared)
	db := connectPostgres(t, shared.dsn)

	m := &MakeMe{
		DB:     db,
		Engine: Postgres,
		DSN:    shared.dsn,
		t:      t,
	}
	t.Cleanup(m.close)
	return m
}

func WithDatabase(engine DatabaseEngine) Option {
	return optionFunc(func(config *newConfig) { config.engine = engine })
}

func WithDatabaseName(name string) Option {
	return optionFunc(func(config *newConfig) { config.databaseName = name })
}

func defaultNewConfig() newConfig {
	return newConfig{engine: Postgres, databaseName: testDatabaseName}
}

func getSharedPostgres(t testing.TB, config newConfig) *sharedDatabase {
	t.Helper()
	skipIfProviderUnavailable(t)

	image := postgresImage()
	key := sharedDatabaseKey{engine: Postgres, databaseName: config.databaseName, image: image}

	sharedDatabases.Lock()
	defer sharedDatabases.Unlock()
	if resource := sharedDatabases.resources[key]; resource != nil {
		return resource
	}

	ctx := context.Background()
	container, err := tcpostgres.Run(ctx,
		image,
		tcpostgres.WithDatabase(config.databaseName),
		tcpostgres.WithUsername(testDatabaseUser),
		tcpostgres.WithPassword(testDatabasePassword),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres test container: %v", err)
	}

	dsn, err := containerDSN(ctx, container, config.databaseName)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		t.Fatalf("read postgres dsn: %v", err)
	}

	if err := runMigrations(dsn); err != nil {
		_ = testcontainers.TerminateContainer(container)
		t.Fatalf("run migrations: %v", err)
	}

	snapshotName := snapshotName(config.databaseName)
	if err := container.Snapshot(ctx, tcpostgres.WithSnapshotName(snapshotName)); err != nil {
		_ = testcontainers.TerminateContainer(container)
		t.Fatalf("snapshot postgres test container: %v", err)
	}

	resource := &sharedDatabase{dsn: dsn, container: container, snapshotName: snapshotName}
	sharedDatabases.resources[key] = resource
	return resource
}

func preparePostgres(t testing.TB, resource *sharedDatabase) {
	t.Helper()
	sharedDatabases.Lock()
	defer sharedDatabases.Unlock()
	if !resource.used {
		resource.used = true
		return
	}
	if err := resource.container.Restore(context.Background(), tcpostgres.WithSnapshotName(resource.snapshotName)); err != nil {
		t.Fatalf("restore postgres test container: %v", err)
	}
}

func runMigrations(dsn string) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open sql: %w", err)
	}
	defer sqlDB.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.Up(sqlDB, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}

func containerDSN(ctx context.Context, container *tcpostgres.PostgresContainer, databaseName string) (string, error) {
	_ = databaseName
	return container.ConnectionString(ctx, "sslmode=disable")
}

func snapshotName(databaseName string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(databaseName))
	return fmt.Sprintf("makeme_snapshot_%08x", hash.Sum32())
}

func postgresImage() string {
	if image := os.Getenv("POSTGRES_IMAGE"); image != "" {
		return image
	}
	return defaultPostgresImage
}

func connectPostgres(t testing.TB, dsn string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 logger.Default.LogMode(logger.Silent),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(5)
	return db
}

// Reset truncates every fixture table. Use it between subtests that share one MakeMe.
func (m *MakeMe) Reset() {
	m.t.Helper()

	var names []string
	err := m.DB.Raw(`
		SELECT quote_ident(schemaname) || '.' || quote_ident(tablename) AS name
		  FROM pg_catalog.pg_tables
		 WHERE schemaname = 'public'
		   AND tablename NOT IN ('goose_db_version','goose_db_version_id_seq')
		 ORDER BY schemaname, tablename
	`).Scan(&names).Error
	if err != nil {
		m.t.Fatalf("list fixture tables: %v", err)
	}
	if len(names) == 0 {
		return
	}
	sort.Strings(names)
	if err := m.DB.Exec("TRUNCATE TABLE " + strings.Join(names, ", ") + " RESTART IDENTITY CASCADE").Error; err != nil {
		m.t.Fatalf("truncate fixture tables: %v", err)
	}
}

func (m *MakeMe) close() {
	if m.DB == nil {
		return
	}
	if sqlDB, err := m.DB.DB(); err == nil {
		_ = sqlDB.Close()
	}
	m.DB = nil
}

func (m *MakeMe) next() int64 {
	m.t.Helper()
	return runningNumber.Add(1)
}

func skipIfProviderUnavailable(t testing.TB) {
	t.Helper()
	tt, ok := t.(*testing.T)
	if !ok {
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Skipf("skip database integration test: testcontainers provider unavailable: %v", recovered)
		}
	}()
	testcontainers.SkipIfProviderIsNotHealthy(tt)
}
