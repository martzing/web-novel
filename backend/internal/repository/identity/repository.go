// Package identity is the GORM adapter for the identity domain ports.
package identity

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	domain "github.com/mokchan/webnovel-backend/internal/domain/identity"
	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/repository/dbctx"
)

// GormRepository implements domain.Repository.
type GormRepository struct {
	db *gorm.DB
}

// New builds a repository around the shared GORM connection.
func New(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

var _ domain.Repository = (*GormRepository)(nil)

func (r *GormRepository) CreateUser(ctx context.Context, u domain.User, passwordHash string) (*domain.User, error) {
	row := entities.User{
		Username:     u.Username,
		Email:        u.Email,
		PasswordHash: passwordHash,
		DisplayName:  u.DisplayName,
		Roles:        pq.StringArray(u.Roles),
		Status:       u.Status,
	}
	if len(row.Roles) == 0 {
		row.Roles = pq.StringArray(domain.DefaultRoles())
	}
	if u.AvatarURL != "" {
		row.AvatarURL = &u.AvatarURL
	}

	if err := dbctx.From(ctx, r.db).Create(&row).Error; err != nil {
		// Rely on the citext unique indexes rather than a pre-flight SELECT,
		// which would be racy under concurrent signups.
		if code, constraint, ok := uniqueViolation(err); ok && code == "23505" {
			switch {
			case strings.Contains(constraint, "email"):
				return nil, domain.ErrEmailTaken
			case strings.Contains(constraint, "username"):
				return nil, domain.ErrUsernameTaken
			}
		}
		return nil, err
	}
	out := toDomainUser(row)
	return &out, nil
}

func (r *GormRepository) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	var row entities.User
	if err := dbctx.From(ctx, r.db).Where("id = ?", id).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	out := toDomainUser(row)
	return &out, nil
}

func (r *GormRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, string, error) {
	var row entities.User
	// email is citext, so the comparison is already case-insensitive.
	if err := dbctx.From(ctx, r.db).Where("email = ?", email).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", domain.ErrNotFound
		}
		return nil, "", err
	}
	out := toDomainUser(row)
	return &out, row.PasswordHash, nil
}

func (r *GormRepository) UpdateUser(ctx context.Context, id int64, patch domain.UserPatch) (*domain.User, error) {
	updates := map[string]any{}
	if patch.DisplayName != nil {
		updates["display_name"] = *patch.DisplayName
	}
	if patch.AvatarURL != nil {
		updates["avatar_url"] = *patch.AvatarURL
	}

	if len(updates) > 0 {
		res := dbctx.From(ctx, r.db).Model(&entities.User{}).Where("id = ?", id).Updates(updates)
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 0 {
			return nil, domain.ErrNotFound
		}
	}
	return r.GetUserByID(ctx, id)
}

func (r *GormRepository) GetPrefs(ctx context.Context, userID int64) (*domain.Prefs, error) {
	var row entities.UserPrefs
	if err := dbctx.From(ctx, r.db).Where("user_id = ?", userID).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	out := toDomainPrefs(row)
	return &out, nil
}

func (r *GormRepository) UpsertPrefs(ctx context.Context, p domain.Prefs) (*domain.Prefs, error) {
	row := entities.UserPrefs{
		UserID:      p.UserID,
		Theme:       p.Theme,
		Font:        p.Font,
		FontSize:    int16(p.FontSize),
		LineHeight:  p.LineHeight,
		ColumnWidth: p.ColumnWidth,
	}
	err := dbctx.From(ctx, r.db).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"theme", "font", "font_size", "line_height", "column_width", "updated_at",
		}),
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}
	out := toDomainPrefs(row)
	return &out, nil
}

func (r *GormRepository) ListGenrePrefs(ctx context.Context, userID int64) ([]domain.GenrePref, error) {
	var rows []entities.UserGenrePref
	if err := dbctx.From(ctx, r.db).Where("user_id = ?", userID).Order("genre_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.GenrePref, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.GenrePref{GenreID: row.GenreID, Weight: int(row.Weight)})
	}
	return out, nil
}

// ReplaceGenrePrefs swaps the whole set in one transaction, so a reader never
// observes a partially applied taste profile.
func (r *GormRepository) ReplaceGenrePrefs(ctx context.Context, userID int64, prefs []domain.GenrePref) error {
	return dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&entities.UserGenrePref{}).Error; err != nil {
			return err
		}
		if len(prefs) == 0 {
			return nil
		}
		rows := make([]entities.UserGenrePref, 0, len(prefs))
		for _, p := range prefs {
			rows = append(rows, entities.UserGenrePref{
				UserID:  userID,
				GenreID: p.GenreID,
				Weight:  int16(p.Weight),
			})
		}
		return tx.Create(&rows).Error
	})
}

func toDomainUser(row entities.User) domain.User {
	out := domain.User{
		ID:          row.ID,
		Username:    row.Username,
		Email:       row.Email,
		DisplayName: row.DisplayName,
		Roles:       []string(row.Roles),
		Status:      row.Status,
		CreatedAt:   row.CreatedAt,
	}
	if row.AvatarURL != nil {
		out.AvatarURL = *row.AvatarURL
	}
	if len(out.Roles) == 0 {
		out.Roles = domain.DefaultRoles()
	}
	return out
}

func toDomainPrefs(row entities.UserPrefs) domain.Prefs {
	return domain.Prefs{
		UserID:      row.UserID,
		Theme:       row.Theme,
		Font:        row.Font,
		FontSize:    int(row.FontSize),
		LineHeight:  row.LineHeight,
		ColumnWidth: row.ColumnWidth,
		UpdatedAt:   row.UpdatedAt,
	}
}

// uniqueViolation extracts the SQLSTATE code and constraint name from a pg
// error, so callers can map a specific index to a domain error.
func uniqueViolation(err error) (code, constraint string, ok bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code, pgErr.ConstraintName, true
	}
	return "", "", false
}
