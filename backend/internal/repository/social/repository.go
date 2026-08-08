// Package social is the GORM adapter for comments, likes and reviews.
package social

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mokchan/webnovel-backend/internal/domain/page"
	"github.com/mokchan/webnovel-backend/internal/domain/roles"
	domain "github.com/mokchan/webnovel-backend/internal/domain/social"
	"github.com/mokchan/webnovel-backend/internal/entities"
	"github.com/mokchan/webnovel-backend/internal/httpx"
	"github.com/mokchan/webnovel-backend/internal/repository/dbctx"
)

// GormRepository implements domain.Repository.
type GormRepository struct {
	db *gorm.DB
}

// New builds a repository around the shared GORM connection.
func New(db *gorm.DB) *GormRepository { return &GormRepository{db: db} }

var _ domain.Repository = (*GormRepository)(nil)

// commentRow is a comment joined with its author.
type commentRow struct {
	ID              int64
	ChapterID       int64
	UserID          int64
	ParentID        *int64
	Body            string
	IsSpoilerHidden bool
	LikesCount      int
	IsTranslator    bool
	CreatedAt       time.Time
	DisplayName     string
	AvatarURL       *string
	Roles           string
	Liked           bool
}

func (r *GormRepository) ListComments(ctx context.Context, chapterID int64, sort string, viewerID int64, p page.Page) ([]domain.Comment, string, error) {
	db := dbctx.From(ctx, r.db)

	base := func() *gorm.DB {
		return db.Table("comments c").
			Select(`c.*, u.display_name, u.avatar_url, array_to_string(u.roles, ',') AS roles,
			        EXISTS (SELECT 1 FROM comment_likes cl
			                 WHERE cl.comment_id = c.id AND cl.user_id = ?) AS liked`, viewerID).
			Joins("JOIN users u ON u.id = c.user_id").
			Where("c.chapter_id = ? AND c.deleted_at IS NULL", chapterID)
	}

	// Only top-level comments are paginated; replies are attached below.
	query := base().Where("c.parent_id IS NULL")

	if sort == domain.SortWithReplies {
		query = query.Where("EXISTS (SELECT 1 FROM comments r WHERE r.parent_id = c.id AND r.deleted_at IS NULL)")
	}

	cursor, err := httpx.DecodeCursorFor(p.Cursor, sort)
	if err != nil {
		return nil, "", err
	}

	switch sort {
	case domain.SortLatest, domain.SortWithReplies:
		if len(cursor.Keys) == 1 {
			id, parseErr := strconv.ParseInt(cursor.Keys[0], 10, 64)
			if parseErr != nil {
				return nil, "", httpx.ErrBadCursor
			}
			query = query.Where("c.id < ?", id)
		}
		query = query.Order("c.id DESC")
	default: // popular
		if len(cursor.Keys) == 2 {
			likes, likesErr := strconv.Atoi(cursor.Keys[0])
			id, idErr := strconv.ParseInt(cursor.Keys[1], 10, 64)
			if likesErr != nil || idErr != nil {
				return nil, "", httpx.ErrBadCursor
			}
			query = query.Where("(c.likes_count, c.id) < (?, ?)", likes, id)
		}
		query = query.Order("c.likes_count DESC").Order("c.id DESC")
	}

	var rows []commentRow
	if err := query.Limit(p.Limit + 1).Scan(&rows).Error; err != nil {
		return nil, "", err
	}

	next := ""
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		last := rows[len(rows)-1]
		keys := []string{strconv.FormatInt(last.ID, 10)}
		if sort == domain.SortPopular {
			keys = []string{strconv.Itoa(last.LikesCount), strconv.FormatInt(last.ID, 10)}
		}
		next = httpx.EncodeCursor(httpx.Cursor{Sort: sort, Keys: keys})
	}

	out := make([]domain.Comment, 0, len(rows))
	index := make(map[int64]int, len(rows))
	parentIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		index[row.ID] = len(out)
		parentIDs = append(parentIDs, row.ID)
		out = append(out, toDomainComment(row))
	}

	if len(parentIDs) == 0 {
		return out, next, nil
	}

	// One extra query attaches every reply, rather than one query per comment.
	var replies []commentRow
	err = base().
		Where("c.parent_id IN ?", parentIDs).
		Order("c.id").
		Scan(&replies).Error
	if err != nil {
		return nil, "", err
	}
	for _, row := range replies {
		if row.ParentID == nil {
			continue
		}
		if idx, ok := index[*row.ParentID]; ok {
			out[idx].Replies = append(out[idx].Replies, toDomainComment(row))
		}
	}
	return out, next, nil
}

func (r *GormRepository) GetComment(ctx context.Context, id int64) (*domain.Comment, error) {
	var row commentRow
	err := dbctx.From(ctx, r.db).
		Table("comments c").
		Select("c.*, u.display_name, u.avatar_url, array_to_string(u.roles, ',') AS roles, false AS liked").
		Joins("JOIN users u ON u.id = c.user_id").
		Where("c.id = ? AND c.deleted_at IS NULL", id).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	out := toDomainComment(row)
	return &out, nil
}

// CreateComment stamps is_translator from the chapter's translator rather than
// trusting the client, so the red badge cannot be spoofed.
func (r *GormRepository) CreateComment(ctx context.Context, c domain.Comment) (*domain.Comment, error) {
	db := dbctx.From(ctx, r.db)

	translatorID, err := r.ChapterTranslatorID(ctx, c.ChapterID)
	if err != nil {
		return nil, err
	}

	row := entities.Comment{
		ChapterID:       c.ChapterID,
		UserID:          c.UserID,
		ParentID:        c.ParentID,
		Body:            c.Body,
		IsSpoilerHidden: c.IsSpoilerHidden,
		IsTranslator:    translatorID != nil && *translatorID == c.UserID,
		CreatedAt:       c.CreatedAt,
	}
	if err := db.Create(&row).Error; err != nil {
		return nil, err
	}
	return r.GetComment(ctx, row.ID)
}

func (r *GormRepository) SoftDeleteComment(ctx context.Context, id int64) error {
	res := dbctx.From(ctx, r.db).Model(&entities.Comment{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", time.Now())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *GormRepository) ChapterTranslatorID(ctx context.Context, chapterID int64) (*int64, error) {
	var row struct{ TranslatorID *int64 }
	err := dbctx.From(ctx, r.db).
		Table("chapters c").
		Select("COALESCE(c.translator_id, n.primary_translator_id) AS translator_id").
		Joins("JOIN novels n ON n.id = c.novel_id").
		Where("c.id = ?", chapterID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return row.TranslatorID, nil
}

// Like inserts the like and bumps the denormalised counter only when the insert
// actually happened, so liking twice leaves the count at one (I-CM-02).
func (r *GormRepository) Like(ctx context.Context, userID, commentID int64) (int, error) {
	var likes int
	err := dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&entities.CommentLike{UserID: userID, CommentID: commentID})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 1 {
			err := tx.Model(&entities.Comment{}).
				Where("id = ?", commentID).
				UpdateColumn("likes_count", gorm.Expr("likes_count + 1")).Error
			if err != nil {
				return err
			}
		}
		// Read the count back inside the same transaction so the response is
		// consistent with what was just written.
		return tx.Model(&entities.Comment{}).
			Where("id = ?", commentID).
			Pluck("likes_count", &likes).Error
	})
	return likes, err
}

func (r *GormRepository) Unlike(ctx context.Context, userID, commentID int64) (int, error) {
	var likes int
	err := dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("user_id = ? AND comment_id = ?", userID, commentID).
			Delete(&entities.CommentLike{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 1 {
			err := tx.Model(&entities.Comment{}).
				Where("id = ?", commentID).
				UpdateColumn("likes_count", gorm.Expr("GREATEST(likes_count - 1, 0)")).Error
			if err != nil {
				return err
			}
		}
		return tx.Model(&entities.Comment{}).
			Where("id = ?", commentID).
			Pluck("likes_count", &likes).Error
	})
	return likes, err
}

// UpsertReview writes the review and recomputes the novel's rating aggregate in
// one transaction, so rating_avg can never drift from the reviews table.
func (r *GormRepository) UpsertReview(ctx context.Context, review domain.Review) (*domain.Review, error) {
	var id int64
	err := dbctx.From(ctx, r.db).Transaction(func(tx *gorm.DB) error {
		row := entities.Review{
			NovelID:   review.NovelID,
			UserID:    review.UserID,
			Rating:    int16(review.Rating),
			CreatedAt: review.CreatedAt,
		}
		if review.Body != "" {
			body := review.Body
			row.Body = &body
		}

		err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "novel_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"rating", "body"}),
		}).Create(&row).Error
		if err != nil {
			return err
		}
		id = row.ID

		return tx.Exec(`
			UPDATE novels n
			   SET rating_avg = COALESCE(agg.avg_rating, 0),
			       rating_count = COALESCE(agg.total, 0)
			  FROM (SELECT AVG(rating)::numeric(3,2) AS avg_rating, COUNT(*) AS total
			          FROM reviews WHERE novel_id = ?) agg
			 WHERE n.id = ?`, review.NovelID, review.NovelID).Error
	})
	if err != nil {
		return nil, err
	}

	review.ID = id

	// Attach the author so the response matches the shape the list endpoint
	// returns; without this the caller gets an empty author block.
	var author entities.User
	if err := dbctx.From(ctx, r.db).Where("id = ?", review.UserID).Take(&author).Error; err == nil {
		review.Author = domain.Author{
			ID:          author.ID,
			DisplayName: author.DisplayName,
			Role:        roleFromCSV(strings.Join(author.Roles, ","), false),
		}
		if author.AvatarURL != nil {
			review.Author.AvatarURL = *author.AvatarURL
		}
	}
	return &review, nil
}

func (r *GormRepository) ListReviews(ctx context.Context, novelID int64, p page.Page) ([]domain.Review, string, error) {
	query := dbctx.From(ctx, r.db).
		Table("reviews rv").
		Select("rv.*, u.display_name, u.avatar_url, array_to_string(u.roles, ',') AS roles").
		Joins("JOIN users u ON u.id = rv.user_id").
		Where("rv.novel_id = ?", novelID)

	cursor, err := httpx.DecodeCursorFor(p.Cursor, "reviews")
	if err != nil {
		return nil, "", err
	}
	if len(cursor.Keys) == 1 {
		id, parseErr := strconv.ParseInt(cursor.Keys[0], 10, 64)
		if parseErr != nil {
			return nil, "", httpx.ErrBadCursor
		}
		query = query.Where("rv.id < ?", id)
	}

	type reviewRow struct {
		ID          int64
		NovelID     int64
		UserID      int64
		Rating      int16
		Body        *string
		CreatedAt   time.Time
		DisplayName string
		AvatarURL   *string
		Roles       string
	}
	var rows []reviewRow
	if err := query.Order("rv.id DESC").Limit(p.Limit + 1).Scan(&rows).Error; err != nil {
		return nil, "", err
	}

	next := ""
	if len(rows) > p.Limit {
		rows = rows[:p.Limit]
		next = httpx.EncodeCursor(httpx.Cursor{
			Sort: "reviews",
			Keys: []string{strconv.FormatInt(rows[len(rows)-1].ID, 10)},
		})
	}

	out := make([]domain.Review, 0, len(rows))
	for _, row := range rows {
		item := domain.Review{
			ID:        row.ID,
			NovelID:   row.NovelID,
			UserID:    row.UserID,
			Rating:    int(row.Rating),
			CreatedAt: row.CreatedAt,
			Author: domain.Author{
				ID:          row.UserID,
				DisplayName: row.DisplayName,
				Role:        roleFromCSV(row.Roles, false),
			},
		}
		if row.Body != nil {
			item.Body = *row.Body
		}
		if row.AvatarURL != nil {
			item.Author.AvatarURL = *row.AvatarURL
		}
		out = append(out, item)
	}
	return out, next, nil
}

func (r *GormRepository) GetUserReview(ctx context.Context, novelID, userID int64) (*domain.Review, error) {
	var row entities.Review
	err := dbctx.From(ctx, r.db).
		Where("novel_id = ? AND user_id = ?", novelID, userID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	out := domain.Review{
		ID:        row.ID,
		NovelID:   row.NovelID,
		UserID:    row.UserID,
		Rating:    int(row.Rating),
		CreatedAt: row.CreatedAt,
	}
	if row.Body != nil {
		out.Body = *row.Body
	}
	return &out, nil
}

func toDomainComment(row commentRow) domain.Comment {
	out := domain.Comment{
		ID:              row.ID,
		ChapterID:       row.ChapterID,
		UserID:          row.UserID,
		ParentID:        row.ParentID,
		Body:            row.Body,
		IsSpoilerHidden: row.IsSpoilerHidden,
		LikesCount:      row.LikesCount,
		IsTranslator:    row.IsTranslator,
		CreatedAt:       row.CreatedAt,
		Liked:           row.Liked,
		Replies:         []domain.Comment{},
		Author: domain.Author{
			ID:          row.UserID,
			DisplayName: row.DisplayName,
			Role:        roleFromCSV(row.Roles, row.IsTranslator),
		},
	}
	if row.AvatarURL != nil {
		out.Author.AvatarURL = *row.AvatarURL
	}
	return out
}

// roleFromCSV picks the single role the UI badges on. isTranslator wins because
// it is scoped to this chapter, which is what the red badge actually means.
func roleFromCSV(csv string, isTranslator bool) string {
	if isTranslator {
		return roles.Translator
	}
	if slices.Contains(strings.Split(csv, ","), roles.Admin) {
		return roles.Admin
	}
	return roles.Reader
}
