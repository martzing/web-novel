# makeme — Test Fixture Patterns

`makeme` starts (or reuses) a package-scoped PostgreSQL test container, runs the goose migrations, exposes a `*gorm.DB`, and provides typed builders for common fixture rows. Snapshot/restore keeps subsequent tests fast.

Use it when a test should exercise real repository / service / handler behavior against PostgreSQL.

## Requirements

- Docker Desktop or a compatible container runtime running locally.
- Go 1.22+.

## Imports

```go
import (
    "testing"

    "github.com/mokchan/webnovel-backend/internal/entities"
    "github.com/mokchan/webnovel-backend/test/makeme"
)
```

If the package needs a pointer helper, add:

```go
func ptr[T any](value T) *T {
    return &value
}
```

## Initialize

### One PostgreSQL database

```go
func TestSomething(t *testing.T) {
    m := makeme.New(t)

    db := m.DB
    _ = db
}
```

`makeme.New(t)` registers `t.Cleanup` for the returned database connection. The underlying container is reused by later `makeme.New(t)` calls in the same process; each `New` restores the container from a post-migration snapshot so tests see a clean database.

### Custom database options

```go
m := makeme.New(t,
    makeme.WithDatabase(makeme.Postgres),
    makeme.WithDatabaseName("mokchan_case_1"),
)
```

`makeme.Postgres` also satisfies the option interface, so `makeme.New(t, makeme.Postgres)` works.

## Mock data

### Create one default row

```go
row := m.ANewGenre().Please()
```

`Please()` is an alias for `Create()`. It persists the row and returns `*T`.

### Customize one row

```go
row := m.ANewGenre().
    With(func(row *entities.Genre) {
        row.Slug   = "xianxia"
        row.NameTH = "เซียน"
    }).
    Please()

_ = row
```

Use `With` before `Please` to override only the fields that matter for the test.

### Build without persisting

```go
row := m.ANewNovel().
    With(func(row *entities.Novel) {
        row.Slug = "nine-streams-sword-immortal"
    }).
    Model()

// row is not in the database yet.
_ = row
```

### Create many generated rows

```go
rows := m.ANewGenre().Many(3).Please()
```

### Create many rows with shared customization

```go
rows := m.ANewNovel().
    With(func(row *entities.Novel) {
        row.Status = "ongoing"
    }).
    Many(3).
    Please()

_ = rows
```

### Create many rows with per-row customization

```go
titles := []string{"เซียนดาบเก้าสายธาร", "คืนกลับสู่ปีที่สิบเก้า", "หุบเขาเงามฤตยู"}

rows := m.ANewNovel().
    Many(len(titles)).
    WithEach(func(row *entities.Novel, index int) {
        row.TitleTH = titles[index]
    }).
    Please()

_ = rows
```

### Create rows from a specific value slice

Use `From` when the test should control the full fixture objects.

```go
rows := m.ANewGenre().From([]entities.Genre{
    {Slug: "xianxia", NameTH: "เซียน"},
    {Slug: "wuxia",   NameTH: "กำลังภายใน"},
}).Please()

_ = rows
```

`From([]T)` returns pointers to the supplied slice elements after they are persisted.

### Create rows from a specific pointer slice

Use `FromPointers` when another helper already returns `[]*T`.

```go
rows := []*entities.Genre{
    {Slug: "xianxia", NameTH: "เซียน"},
    {Slug: "wuxia",   NameTH: "กำลังภายใน"},
}

created := m.ANewGenre().FromPointers(rows).Please()
_ = created
```

`FromPointers` fails the test if any item is nil.

### Link fixtures across tables

```go
translator := m.ANewUser().Please()

novel := m.ANewNovel().
    With(func(row *entities.Novel) {
        row.Slug                = "nine-streams-sword-immortal"
        row.TitleTH             = "เซียนดาบเก้าสายธาร"
        row.PrimaryTranslatorID = &translator.ID
    }).
    Please()

genre := m.ANewGenre().
    With(func(row *entities.Genre) {
        row.Slug   = "xianxia"
        row.NameTH = "เซียน"
    }).
    Please()

m.ANewNovelGenre().
    With(func(row *entities.NovelGenre) {
        row.NovelID = novel.ID
        row.GenreID = genre.ID
    }).
    Please()

arc := m.ANewArc().
    With(func(row *entities.Arc) {
        row.NovelID = novel.ID
        row.ArcNo   = 2
        row.Name    = "สำนักเมฆาวสันต์"
    }).
    Please()

chapter := m.ANewChapter().
    With(func(row *entities.Chapter) {
        row.NovelID   = novel.ID
        row.ArcID     = &arc.ID
        row.ChapterNo = 87
        row.Title     = "ดาบแรกใต้ฟ้าหมอก"
    }).
    Please()

m.ANewChapterBody().
    With(func(row *entities.ChapterBody) {
        row.ChapterID = chapter.ID
        row.BodyHTML  = "<p>หิมะบางโปรยลงเหนือหุบเขาเมฆาวสันต์...</p>"
        row.BodySource = "หิมะบางโปรยลงเหนือหุบเขาเมฆาวสันต์..."
    }).
    Please()
```

### Query fixtures

```go
found := m.ANewNovel().Find(makeme.QueryCriteria{
    Where: map[string]any{"slug": "nine-streams-sword-immortal"},
    Order: "id DESC",
})

rows := m.ANewChapter().List(makeme.QueryCriteria{
    Where:  "novel_id = ? AND status = ?",
    Args:   []any{found.ID, "published"},
    Order:  "chapter_no ASC",
    Offset: 0,
    Limit:  20,
})

_ = rows
```

### Update or delete a fixture

```go
builder := m.ANewGenre()
row := builder.Please()

row.NameTH = "เซียน (แก้ไข)"
builder.Update()

builder.Delete()
```

`Update` and `Delete` locate the row using the builder's locator, including generated IDs and composite keys.

## Clean data

### Clean database per test

The simplest pattern is one `makeme.New(t)` per test.

```go
func TestOneThing(t *testing.T) {
    m := makeme.New(t)
    m.ANewGenre().Please()
}
```

`makeme.New(t)` restores from the post-migration snapshot before returning, so data from a previous test in the same package does not leak into the next one.

### Reset before each subtest

Use this when subtests share one database container.

```go
func TestManyCases(t *testing.T) {
    m := makeme.New(t)

    t.Run("case one", func(t *testing.T) {
        m.Reset()
        m.ANewGenre().Please()
    })

    t.Run("case two", func(t *testing.T) {
        m.Reset()
        m.ANewNovel().Please()
    })
}
```

`Reset` truncates every application table (leaving `goose_db_version` intact) and restarts identity sequences.

### Reset after each subtest

```go
func TestCleanupAfterSubtest(t *testing.T) {
    m := makeme.New(t)

    t.Run("creates data", func(t *testing.T) {
        t.Cleanup(m.Reset)

        m.ANewNovel().Please()
    })
}
```

## Recommended test shape

### Repository / service test

```go
func TestListGenres(t *testing.T) {
    m := makeme.New(t)
    store := catalog.NewStore(m.DB)

    m.ANewGenre().From([]entities.Genre{
        {Slug: "xianxia", NameTH: "เซียน"},
        {Slug: "wuxia",   NameTH: "กำลังภายใน"},
    }).Please()

    got, err := store.ListGenres(context.Background())
    if err != nil {
        t.Fatal(err)
    }
    if len(got) < 2 {
        t.Fatalf("want at least 2 genres, got %d", len(got))
    }
}
```

### Handler test

```go
func TestGenresEndpoint(t *testing.T) {
    m := makeme.New(t)
    m.ANewGenre().Many(3).Please()

    engine := server.New(&config.Config{Env: "test", CORSOrigins: []string{"*"}}, m.DB)

    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/api/v1/genres", nil)
    engine.ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
    }
}
```

## Notes

- The container image defaults to `postgres:16-alpine`. Override with the `POSTGRES_IMAGE` environment variable.
- Snapshot/restore is a native feature of the `testcontainers-go/postgres` module and requires the Postgres 16 image.
- Tests that use makeme skip automatically if the container runtime is not available on the machine.
