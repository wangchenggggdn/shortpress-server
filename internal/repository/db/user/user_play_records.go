package user

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserPlayRecordsRepository defines the interface for interacting with user play records.
// It provides methods for creating, updating, and retrieving user play records.
type UserPlayRecordsRepository interface {
	// Create creates a new play record.
	Create(ctx context.Context, record *model.UserPlayRecord) error

	// It returns an error if the operation fails.
	CreateOrUpdate(ctx context.Context, record *model.UserPlayRecord) error

	// It returns the UserPlayRecord model and an error if the record is not found or an error occurs.
	GetByUserIDAndPlaylistID(ctx context.Context, userID, playlist_id string) (*model.UserPlayRecord, error)

	// It returns a slice of UserPlayRecord models, the total count of records, and an error if the retrieval fails.
	GetByUserIDAndSiteID(ctx context.Context, userID, siteID string, page, pageSize int) ([]*model.UserPlayRecord, int64, error)
}

type userPlayRecordsRepository struct {
	*db.Repository
}

func NewUserPlayRecordsRepository(repository *db.Repository) UserPlayRecordsRepository {
	return &userPlayRecordsRepository{
		Repository: repository,
	}
}

func (r *userPlayRecordsRepository) Create(ctx context.Context, record *model.UserPlayRecord) error {
	now := time.Now()
	record.CreatedAt = now
	record.UpdatedAt = now
	record.LastPlayedAt = now
	return r.DB(ctx).Create(record).Error
}

func (r *userPlayRecordsRepository) CreateOrUpdate(ctx context.Context, record *model.UserPlayRecord) error {
	now := time.Now()
	record.LastPlayedAt = now

	return r.DB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "playlist_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"site_id", "playlist_id", "vid", "episode_number", "progress", "duration", "video_title", "playlist_title", "last_played_at"}),
	}).Create(record).Error
}

func (r *userPlayRecordsRepository) GetByUserIDAndPlaylistID(ctx context.Context, userID, playlist_id string) (*model.UserPlayRecord, error) {
	var record model.UserPlayRecord
	err := r.DB(ctx).Where("user_id = ? AND playlist_id = ?", userID, playlist_id).First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

func (r *userPlayRecordsRepository) GetByUserIDAndSiteID(ctx context.Context, userID, siteID string, page, pageSize int) ([]*model.UserPlayRecord, int64, error) {
	var records []*model.UserPlayRecord
	var total int64

	err := r.DB(ctx).Model(&model.UserPlayRecord{}).
		Where("user_id = ? AND site_id = ?", userID, siteID).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = r.DB(ctx).
		Where("user_id = ? AND site_id = ?", userID, siteID).
		Order("last_played_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&records).Error
	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}
