package playlist

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PlaylistI18nRepository defines the interface for playlist i18n operations
type PlaylistI18nRepository interface {
	db.BaseOperation
	Get(ctx context.Context, playlistID string, language string) (*model.PlaylistI18n, error)
	List(ctx context.Context, playlistID string) ([]*model.PlaylistI18n, error)
	ListByPlaylistIDs(ctx context.Context, playlistIDs []string) ([]*model.PlaylistI18n, error)
	Delete(ctx context.Context, playlistID string, language string) error
	DeleteByPlaylistID(ctx context.Context, playlistID string) error
	BatchCreate(ctx context.Context, entities []*model.PlaylistI18n) error
	BatchSaveOrUpdate(ctx context.Context, entities []*model.PlaylistI18n) error
}

type playlistI18nRepository struct {
	*db.Repository
}

// NewPlaylistI18nRepository creates a new playlist i18n repository instance
func NewPlaylistI18nRepository(repository *db.Repository) PlaylistI18nRepository {
	return &playlistI18nRepository{
		Repository: repository,
	}
}

// Create inserts a new playlist i18n record into the database
func (r *playlistI18nRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

// Update modifies an existing playlist i18n record
// It updates the record based on playlist_id and language combination
func (r *playlistI18nRepository) Update(ctx context.Context, entity interface{}) error {
	playlistI18n := entity.(*model.PlaylistI18n)
	return r.DB(ctx).
		Where("playlist_id = ? AND language = ?", playlistI18n.PlaylistID, playlistI18n.Language).
		Updates(playlistI18n).Error
}

// Get retrieves a playlist i18n record by playlistID and language
// Returns nil if the record is not found (no error)
func (r *playlistI18nRepository) Get(ctx context.Context, playlistID string, language string) (*model.PlaylistI18n, error) {
	var i18n model.PlaylistI18n
	err := r.DB(ctx).
		Where("playlist_id = ? AND language = ?", playlistID, language).
		First(&i18n).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &i18n, nil
}

// List retrieves all i18n records for a specific playlist
// Returns an empty slice if no records are found
func (r *playlistI18nRepository) List(ctx context.Context, playlistID string) ([]*model.PlaylistI18n, error) {
	var i18ns []*model.PlaylistI18n
	err := r.DB(ctx).
		Where("playlist_id = ?", playlistID).
		Find(&i18ns).Error
	if err != nil {
		return nil, err
	}
	return i18ns, nil
}

// Delete removes a playlist i18n record by playlistID and language
func (r *playlistI18nRepository) Delete(ctx context.Context, playlistID string, language string) error {
	return r.DB(ctx).
		Where("playlist_id = ? AND language = ?", playlistID, language).
		Delete(&model.PlaylistI18n{}).Error
}

// DeleteByPlaylistID removes all i18n records for a specific playlist
// Useful when deleting a playlist and need to clean up all its i18n data
func (r *playlistI18nRepository) DeleteByPlaylistID(ctx context.Context, playlistID string) error {
	return r.DB(ctx).
		Where("playlist_id = ?", playlistID).
		Delete(&model.PlaylistI18n{}).Error
}

// BatchCreate creates multiple playlist i18n records in a single transaction
// More efficient than creating records one by one
func (r *playlistI18nRepository) BatchCreate(ctx context.Context, entities []*model.PlaylistI18n) error {
	if len(entities) == 0 {
		return nil
	}
	return r.DB(ctx).CreateInBatches(entities, 100).Error
}

// BatchSaveOrUpdate creates or updates multiple playlist i18n records in a single transaction
func (r *playlistI18nRepository) BatchSaveOrUpdate(ctx context.Context, entities []*model.PlaylistI18n) error {
	if len(entities) == 0 {
		return nil
	}
	return r.DB(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "playlist_id"}, {Name: "language"}},
			DoUpdates: clause.AssignmentColumns([]string{"title", "slug", "description", "tags", "seo_title", "seo_description", "seo_keywords"}),
		}).
		CreateInBatches(entities, 100).Error
}

// ListByPlaylistIDs retrieves all i18n records for multiple playlists in a single query
// Returns an empty slice if no records are found or if playlistIDs is empty
func (r *playlistI18nRepository) ListByPlaylistIDs(ctx context.Context, playlistIDs []string) ([]*model.PlaylistI18n, error) {
	if len(playlistIDs) == 0 {
		return []*model.PlaylistI18n{}, nil
	}

	var i18ns []*model.PlaylistI18n
	err := r.DB(ctx).
		Where("playlist_id IN ?", playlistIDs).
		Find(&i18ns).Error
	if err != nil {
		return nil, err
	}
	return i18ns, nil
}
