package writer

import (
	"context"
	"strings"

	domain "github.com/mokchan/webnovel-backend/internal/domain/writer"
)

// ListSeries returns the caller's series with their book counts.
func (s *Service) ListSeries(ctx context.Context, ownerID int64) ([]domain.Series, error) {
	return s.repo.ListSeries(ctx, ownerID)
}

// CreateSeries opens a new collection owned by the caller.
func (s *Service) CreateSeries(ctx context.Context, ownerID int64, in domain.Series) (*domain.Series, error) {
	in.Title = strings.TrimSpace(in.Title)
	if err := domain.ValidateSeries(in); err != nil {
		return nil, err
	}

	seq, err := s.repo.NextSlugSeq(ctx)
	if err != nil {
		return nil, err
	}
	in.Slug = domain.SlugFromTitle(in.Title, seq)
	in.OwnerUserID = &ownerID

	return s.repo.CreateSeries(ctx, in)
}

// UpdateSeries patches a series the caller owns.
func (s *Service) UpdateSeries(ctx context.Context, userID, seriesID int64, in domain.Series) (*domain.Series, error) {
	if err := s.assertOwnsSeries(ctx, userID, seriesID); err != nil {
		return nil, err
	}
	in.Title = strings.TrimSpace(in.Title)
	// An empty title here means "leave it alone", so only validate a supplied
	// one — unlike create, where a title is mandatory.
	if in.Title != "" {
		if err := domain.ValidateSeries(in); err != nil {
			return nil, err
		}
	}
	return s.repo.UpdateSeries(ctx, seriesID, in)
}

// DeleteSeries removes a collection, leaving its novels in place.
func (s *Service) DeleteSeries(ctx context.Context, userID, seriesID int64) error {
	if err := s.assertOwnsSeries(ctx, userID, seriesID); err != nil {
		return err
	}
	return s.repo.DeleteSeries(ctx, seriesID)
}

// SeriesBooks lists a series' novels in reading order.
func (s *Service) SeriesBooks(ctx context.Context, userID, seriesID int64) ([]domain.SeriesBook, error) {
	if err := s.assertOwnsSeries(ctx, userID, seriesID); err != nil {
		return nil, err
	}
	return s.repo.SeriesBooks(ctx, seriesID)
}

// ReorderSeries rewrites the reading order from a client-supplied id list.
//
// Ids the caller does not own — or that belong to another series — are dropped
// rather than rejected: the list comes from a drag interaction, and a stale tab
// should reorder what it legitimately can instead of failing wholesale.
func (s *Service) ReorderSeries(ctx context.Context, userID, seriesID int64, novelIDs []int64) ([]domain.SeriesBook, error) {
	if err := s.assertOwnsSeries(ctx, userID, seriesID); err != nil {
		return nil, err
	}

	books, err := s.repo.SeriesBooks(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	member := make(map[int64]bool, len(books))
	for _, b := range books {
		member[b.NovelID] = true
	}

	ordered := make([]int64, 0, len(novelIDs))
	for _, id := range novelIDs {
		if member[id] {
			ordered = append(ordered, id)
		}
	}
	// Anything the client omitted keeps its relative order behind the rest, so
	// a novel added in another tab is never silently unpositioned.
	seen := make(map[int64]bool, len(ordered))
	for _, id := range ordered {
		seen[id] = true
	}
	for _, b := range books {
		if !seen[b.NovelID] {
			ordered = append(ordered, b.NovelID)
		}
	}

	if err := s.repo.SetSeriesOrder(ctx, seriesID, domain.ReorderPositions(ordered)); err != nil {
		return nil, err
	}
	return s.repo.SeriesBooks(ctx, seriesID)
}

// SetSeriesNote updates one book's note in the reading order.
func (s *Service) SetSeriesNote(ctx context.Context, userID, novelID int64, note string) error {
	if err := s.assertOwnsNovel(ctx, userID, novelID); err != nil {
		return err
	}
	if len([]rune(note)) > 500 {
		return domain.ErrInvalidInput
	}
	return s.repo.SetSeriesNote(ctx, novelID, note)
}

// ListRelations returns both the relations declared on this novel and those
// declared on others pointing at it, the latter mirrored with the inverse kind.
func (s *Service) ListRelations(ctx context.Context, userID, novelID int64) ([]domain.Relation, error) {
	if err := s.assertOwnsNovel(ctx, userID, novelID); err != nil {
		return nil, err
	}
	return s.repo.ListRelations(ctx, novelID)
}

// LinkNovels declares a relation between two novels.
//
// Only the near novel's ownership is checked. Requiring both would stop a
// translator pointing at another translator's work, which is exactly what
// "เกิดในโลกเดียวกัน" is for; the link is only ever shown as an outbound
// reference, so it grants no rights over the far novel.
func (s *Service) LinkNovels(ctx context.Context, userID int64, rel domain.Relation) (*domain.Relation, error) {
	if err := s.assertOwnsNovel(ctx, userID, rel.NovelID); err != nil {
		return nil, err
	}
	if err := domain.ValidateRelation(rel); err != nil {
		return nil, err
	}
	// The far novel must exist, or the editor would list a dangling card.
	if _, err := s.repo.GetNovel(ctx, rel.RelatedNovelID); err != nil {
		return nil, err
	}
	return s.repo.UpsertRelation(ctx, rel)
}

// UnlinkNovels removes a relation in either direction.
func (s *Service) UnlinkNovels(ctx context.Context, userID, novelID, relatedNovelID int64) error {
	if err := s.assertOwnsNovel(ctx, userID, novelID); err != nil {
		return err
	}
	return s.repo.DeleteRelation(ctx, novelID, relatedNovelID)
}

func (s *Service) assertOwnsSeries(ctx context.Context, userID, seriesID int64) error {
	series, err := s.repo.GetSeries(ctx, seriesID)
	if err != nil {
		return err
	}
	if series.OwnerUserID == nil || *series.OwnerUserID != userID {
		return domain.ErrForbidden
	}
	return nil
}
