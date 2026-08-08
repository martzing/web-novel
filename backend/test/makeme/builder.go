package makeme

import (
	"reflect"
	"strings"

	"gorm.io/gorm"
)

// Locator returns the column-to-value map used to locate a persisted row for Update/Delete.
type Locator[T any] func(*T) map[string]any

// QueryCriteria captures a subset of GORM's query surface for fixture reads.
type QueryCriteria struct {
	Where  any
	Args   []any
	Order  string
	Offset int
	Limit  int
}

// Builder holds one in-memory model plus the plumbing to persist and locate it.
type Builder[T any] struct {
	makeMe           *MakeMe
	model            *T
	locator          Locator[T]
	persistedLocator map[string]any
	customizers      []func(*T)
}

// SliceBuilder wraps a set of Builders created together via Many/From/FromPointers.
type SliceBuilder[T any] struct {
	makeMe   *MakeMe
	builders []*Builder[T]
}

func newBuilder[T any](makeMe *MakeMe, model *T, locator Locator[T]) *Builder[T] {
	makeMe.t.Helper()
	return &Builder[T]{makeMe: makeMe, model: model, locator: locator}
}

// With mutates the in-memory model. Chain before Please/Model/Many.
func (b *Builder[T]) With(customize func(*T)) *Builder[T] {
	b.makeMe.t.Helper()
	if customize != nil {
		b.customizers = append(b.customizers, customize)
		customize(b.model)
	}
	return b
}

// Many produces `count` sibling builders. The first sibling reuses the receiver's model.
func (b *Builder[T]) Many(count int) *SliceBuilder[T] {
	b.makeMe.t.Helper()
	if count < 0 {
		b.makeMe.t.Fatalf("build many %T: count must not be negative", b.model)
	}

	builders := make([]*Builder[T], 0, count)
	if count > 0 {
		builders = append(builders, b)
	}
	for len(builders) < count {
		builders = append(builders, b.newSibling())
	}
	return &SliceBuilder[T]{makeMe: b.makeMe, builders: builders}
}

// From returns builders for a caller-supplied slice of models.
func (b *Builder[T]) From(models []T) *SliceBuilder[T] {
	b.makeMe.t.Helper()
	builders := make([]*Builder[T], 0, len(models))
	for i := range models {
		builders = append(builders, b.forModel(&models[i]))
	}
	return &SliceBuilder[T]{makeMe: b.makeMe, builders: builders}
}

// FromPointers returns builders for a caller-supplied pointer slice; fails on nil entries.
func (b *Builder[T]) FromPointers(models []*T) *SliceBuilder[T] {
	b.makeMe.t.Helper()
	builders := make([]*Builder[T], 0, len(models))
	for i, model := range models {
		if model == nil {
			b.makeMe.t.Fatalf("build many %T: model at index %d must not be nil", b.model, i)
		}
		builders = append(builders, b.forModel(model))
	}
	return &SliceBuilder[T]{makeMe: b.makeMe, builders: builders}
}

// Model returns the in-memory row without persisting.
func (b *Builder[T]) Model() *T {
	b.makeMe.t.Helper()
	return b.model
}

// Find loads exactly one row that matches criteria, or fails the test.
func (b *Builder[T]) Find(criteria QueryCriteria) *T {
	b.makeMe.t.Helper()
	var result T
	query := b.query(criteria).First(&result)
	if query.Error != nil {
		b.makeMe.t.Fatalf("find %T: %v", b.model, query.Error)
	}
	return &result
}

// List loads zero or more rows.
func (b *Builder[T]) List(criteria QueryCriteria) []T {
	b.makeMe.t.Helper()
	results := make([]T, 0)
	if err := b.query(criteria).Find(&results).Error; err != nil {
		b.makeMe.t.Fatalf("list %T: %v", b.model, err)
	}
	return results
}

// Create persists the in-memory model and returns it.
func (b *Builder[T]) Create() *T {
	b.makeMe.t.Helper()
	if err := b.makeMe.DB.Create(b.model).Error; err != nil {
		b.makeMe.t.Fatalf("create %T: %v", b.model, err)
	}
	b.persistedLocator = cloneLocator(b.locator(b.model))
	return b.model
}

// Please is a friendlier alias for Create.
func (b *Builder[T]) Please() *T {
	b.makeMe.t.Helper()
	return b.Create()
}

// Update writes back the in-memory model using the row's locator.
func (b *Builder[T]) Update() *T {
	b.makeMe.t.Helper()
	where := b.currentLocator()
	result := b.makeMe.DB.Model(new(T)).Where(where).Select("*").Updates(b.model)
	if result.Error != nil {
		b.makeMe.t.Fatalf("update %T using %v: %v", b.model, where, result.Error)
	}
	if result.RowsAffected == 0 {
		b.makeMe.t.Fatalf("update %T using %v: row not found", b.model, where)
	}
	b.persistedLocator = cloneLocator(b.locator(b.model))
	return b.model
}

// Delete removes the persisted row using the locator.
func (b *Builder[T]) Delete() {
	b.makeMe.t.Helper()
	where := b.currentLocator()
	result := b.makeMe.DB.Unscoped().Where(where).Delete(new(T))
	if result.Error != nil {
		b.makeMe.t.Fatalf("delete %T using %v: %v", b.model, where, result.Error)
	}
	if result.RowsAffected == 0 {
		b.makeMe.t.Fatalf("delete %T using %v: row not found", b.model, where)
	}
	b.persistedLocator = nil
}

func (b *Builder[T]) query(criteria QueryCriteria) *gorm.DB {
	b.makeMe.t.Helper()
	if criteria.Offset < 0 {
		b.makeMe.t.Fatalf("query %T: offset must not be negative", b.model)
	}
	if criteria.Limit < 0 {
		b.makeMe.t.Fatalf("query %T: limit must not be negative", b.model)
	}

	query := b.makeMe.DB.Model(new(T))
	if criteria.Where != nil {
		query = query.Where(criteria.Where, criteria.Args...)
	}
	if order := strings.TrimSpace(criteria.Order); order != "" {
		query = query.Order(order)
	}
	if criteria.Offset > 0 {
		query = query.Offset(criteria.Offset)
	}
	if criteria.Limit > 0 {
		query = query.Limit(criteria.Limit)
	}
	return query
}

func (b *Builder[T]) currentLocator() map[string]any {
	b.makeMe.t.Helper()
	if len(b.persistedLocator) > 0 {
		return cloneLocator(b.persistedLocator)
	}
	where := b.locator(b.model)
	if len(where) == 0 {
		b.makeMe.t.Fatalf("locate %T: empty row locator", b.model)
	}
	return where
}

func (b *Builder[T]) newSibling() *Builder[T] {
	b.makeMe.t.Helper()
	methodName := "ANew" + modelTypeName[T]()
	method := reflect.ValueOf(b.makeMe).MethodByName(methodName)
	if !method.IsValid() {
		b.makeMe.t.Fatalf("build many %T: builder method %s not found", b.model, methodName)
	}

	results := method.Call(nil)
	if len(results) != 1 {
		b.makeMe.t.Fatalf("build many %T: builder method %s returned %d values", b.model, methodName, len(results))
	}
	sibling, ok := results[0].Interface().(*Builder[T])
	if !ok || sibling == nil {
		b.makeMe.t.Fatalf("build many %T: builder method %s returned %T", b.model, methodName, results[0].Interface())
	}
	for _, customize := range b.customizers {
		sibling.With(customize)
	}
	return sibling
}

func (b *Builder[T]) forModel(model *T) *Builder[T] {
	b.makeMe.t.Helper()
	builder := &Builder[T]{makeMe: b.makeMe, model: model, locator: b.locator}
	for _, customize := range b.customizers {
		builder.With(customize)
	}
	return builder
}

// With applies the same customization to every builder.
func (b *SliceBuilder[T]) With(customize func(*T)) *SliceBuilder[T] {
	b.makeMe.t.Helper()
	for _, builder := range b.builders {
		builder.With(customize)
	}
	return b
}

// WithEach applies a per-index customization.
func (b *SliceBuilder[T]) WithEach(customize func(*T, int)) *SliceBuilder[T] {
	b.makeMe.t.Helper()
	if customize == nil {
		return b
	}
	for i, builder := range b.builders {
		customize(builder.model, i)
	}
	return b
}

// Models returns all in-memory rows without persisting.
func (b *SliceBuilder[T]) Models() []*T {
	b.makeMe.t.Helper()
	models := make([]*T, 0, len(b.builders))
	for _, builder := range b.builders {
		models = append(models, builder.Model())
	}
	return models
}

// Create persists every builder in order.
func (b *SliceBuilder[T]) Create() []*T {
	b.makeMe.t.Helper()
	models := make([]*T, 0, len(b.builders))
	for _, builder := range b.builders {
		models = append(models, builder.Create())
	}
	return models
}

// Please is a friendlier alias for Create.
func (b *SliceBuilder[T]) Please() []*T {
	b.makeMe.t.Helper()
	return b.Create()
}

func cloneLocator(locator map[string]any) map[string]any {
	cloned := make(map[string]any, len(locator))
	for key, value := range locator {
		cloned[key] = value
	}
	return cloned
}

func modelTypeName[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		return ""
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}
