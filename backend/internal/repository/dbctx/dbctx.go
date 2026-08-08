// Package dbctx carries an ambient GORM transaction on the request context.
//
// Every repository method starts with `db := dbctx.From(ctx, r.db)`. That makes
// each repository transparently transaction-aware without a transaction type
// ever appearing in a domain port signature, which is what lets a cross-context
// use case (publish chapter -> fan out notifications) commit atomically.
package dbctx

import (
	"context"

	"gorm.io/gorm"
)

type key struct{}

// With returns a context carrying tx as the ambient transaction.
func With(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, key{}, tx)
}

// From returns the ambient transaction when one is present, else fallback.
// The returned handle already has the context applied.
func From(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(key{}).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return fallback.WithContext(ctx)
}
