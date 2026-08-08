package identity

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	domain "github.com/mokchan/webnovel-backend/internal/domain/identity"
	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/repository/dbctx"
)

// SessionStore persists refresh-token sessions in Postgres.
//
// Postgres rather than Redis: rotation needs a compare-and-swap, and reuse
// detection needs the revoked row to survive so the family can be revoked and
// audited. A TTL cache would evict exactly the evidence that matters.
type SessionStore struct {
	db *gorm.DB
}

// NewSessionStore builds the store.
func NewSessionStore(db *gorm.DB) *SessionStore { return &SessionStore{db: db} }

var _ domain.SessionStore = (*SessionStore)(nil)

func (s *SessionStore) Create(ctx context.Context, session domain.Session, tokenHash []byte) (*domain.Session, error) {
	row := entities.RefreshToken{
		UserID:    session.UserID,
		FamilyID:  session.FamilyID,
		TokenHash: tokenHash,
		ExpiresAt: session.ExpiresAt,
	}
	if session.UserAgent != "" {
		row.UserAgent = &session.UserAgent
	}
	if err := dbctx.From(ctx, s.db).Create(&row).Error; err != nil {
		return nil, err
	}
	out := toDomainSession(row)
	return &out, nil
}

// Rotate consumes tokenHash and issues its successor inside one transaction.
//
// Presenting an already-revoked token is treated as theft: the entire family is
// revoked and ErrTokenReused is returned, so a stolen token cannot outlive the
// legitimate holder's next refresh (I-AUTH-03).
func (s *SessionStore) Rotate(
	ctx context.Context,
	tokenHash []byte,
	next domain.Session,
	nextHash []byte,
	now time.Time,
) (*domain.Session, error) {
	db := dbctx.From(ctx, s.db)

	var created entities.RefreshToken
	// Captured inside the transaction but acted on after it: returning an error
	// rolls the transaction back, so revoking the family in place would undo
	// the very revocation the reuse detection exists to perform.
	var reusedFamily string

	err := db.Transaction(func(tx *gorm.DB) error {
		var current entities.RefreshToken
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token_hash = ?", tokenHash).
			Take(&current).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrInvalidRefreshToken
			}
			return err
		}

		if current.RevokedAt != nil {
			reusedFamily = current.FamilyID
			return domain.ErrTokenReused
		}
		if !current.ExpiresAt.After(now) {
			return domain.ErrInvalidRefreshToken
		}

		created = entities.RefreshToken{
			UserID:    current.UserID,
			FamilyID:  current.FamilyID,
			TokenHash: nextHash,
			ExpiresAt: next.ExpiresAt,
		}
		if next.UserAgent != "" {
			created.UserAgent = &next.UserAgent
		}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}

		return tx.Model(&entities.RefreshToken{}).
			Where("id = ?", current.ID).
			Updates(map[string]any{"revoked_at": now, "replaced_by": created.ID}).Error
	})

	// Reuse means the token was almost certainly stolen: revoke the whole
	// rotation family in its own committed transaction so the attacker's
	// successor token dies alongside the replayed one (I-AUTH-03).
	if reusedFamily != "" {
		if revokeErr := revokeFamily(db, reusedFamily, now); revokeErr != nil {
			return nil, revokeErr
		}
		return nil, domain.ErrTokenReused
	}
	if err != nil {
		return nil, err
	}

	out := toDomainSession(created)
	return &out, nil
}

// RevokeFamilyByToken revokes every live token in the presented token's family.
func (s *SessionStore) RevokeFamilyByToken(ctx context.Context, tokenHash []byte, now time.Time) error {
	return dbctx.From(ctx, s.db).Transaction(func(tx *gorm.DB) error {
		var current entities.RefreshToken
		err := tx.Where("token_hash = ?", tokenHash).Take(&current).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Logging out with an unknown token is a no-op, never an error:
				// reporting it would let a caller probe which tokens exist.
				return nil
			}
			return err
		}
		return revokeFamily(tx, current.FamilyID, now)
	})
}

// RevokeAllForUser ends every session the user has open.
func (s *SessionStore) RevokeAllForUser(ctx context.Context, userID int64, now time.Time) error {
	return dbctx.From(ctx, s.db).Model(&entities.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

// DeleteExpired removes tokens that expired before the cutoff.
func (s *SessionStore) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	res := dbctx.From(ctx, s.db).
		Where("expires_at < ?", before).
		Delete(&entities.RefreshToken{})
	return res.RowsAffected, res.Error
}

func revokeFamily(tx *gorm.DB, familyID string, now time.Time) error {
	return tx.Model(&entities.RefreshToken{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", now).Error
}

func toDomainSession(row entities.RefreshToken) domain.Session {
	out := domain.Session{
		ID:        row.ID,
		UserID:    row.UserID,
		FamilyID:  row.FamilyID,
		ExpiresAt: row.ExpiresAt,
		RevokedAt: row.RevokedAt,
	}
	if row.UserAgent != nil {
		out.UserAgent = *row.UserAgent
	}
	return out
}
