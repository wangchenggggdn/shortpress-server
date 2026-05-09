package payment

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

// ContentUnlockRepository defines the repository interface for content unlocks
type ContentUnlockRepository interface {
	db.BaseOperation
	ListByUserID(ctx context.Context, userID string, limit int, offset int) ([]*model.ContentUnlock, error)
	CountByUserID(ctx context.Context, userID string) (int64, error)
	// Return all unlock records for a user within a playlist
	GetByPlaylistID(ctx context.Context, userID string, playlistID string) ([]*model.ContentUnlock, error)
	// Check if content is already unlocked by user
	CheckExistingUnlock(ctx context.Context, userID string, contentID string, contentType string, playlistID string) (*model.ContentUnlock, error)
	// CreateUnlock creates a content unlock record and returns its ID
}

type contentUnlockRepository struct {
	*db.Repository
}

// NewContentUnlockRepository creates a new content unlock repository
func NewContentUnlockRepository(r *db.Repository) ContentUnlockRepository {
	return &contentUnlockRepository{
		Repository: r,
	}
}

func (r *contentUnlockRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *contentUnlockRepository) Update(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Save(entity).Error
}

func (r *contentUnlockRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*model.ContentUnlock, error) {
	var unlocks []*model.ContentUnlock
	query := r.DB(ctx).Where("user_id = ?", userID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&unlocks).Error
	if err != nil {
		return nil, err
	}

	return unlocks, nil
}

func (r *contentUnlockRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.DB(ctx).Model(&model.ContentUnlock{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *contentUnlockRepository) GetByPlaylistID(ctx context.Context, userID string, playlistID string) ([]*model.ContentUnlock, error) {
	var unlocks []*model.ContentUnlock
	err := r.DB(ctx).Where("user_id = ? AND playlist_id = ?", userID, playlistID).Find(&unlocks).Error
	if err != nil {
		return nil, err
	}
	return unlocks, nil
}

// CheckExistingUnlock checks if the user has already unlocked the content
func (r *contentUnlockRepository) CheckExistingUnlock(ctx context.Context, userID, contentID, contentType string, playlistID string) (*model.ContentUnlock, error) {
	var unlock model.ContentUnlock

	// Query for an active unlock (not expired or permanent)
	query := r.DB(ctx).Where(
		"user_id = ? AND content_id = ? AND content_type = ? ",
		userID, contentID, contentType,
	)

	if playlistID != "" {
		query = query.Where("playlist_id = ?", playlistID)
	}

	err := query.First(&unlock).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No unlock found, but not an error
		}
		return nil, err // Database error
	}

	return &unlock, nil
}
