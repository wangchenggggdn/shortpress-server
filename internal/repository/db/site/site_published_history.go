package site

import (
	"context"
	"encoding/json"
	"time"

	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"
)

type SitePublishedHistoryRepository interface {
	db.BaseOperation
	CreateHistoryRecord(ctx context.Context, siteID string, publishedBy string, versionNumber int, publishedData interface{}) (*model.SitePublishedHistory, error)
	GetPublishHistory(ctx context.Context, siteID string, limit int, offset int) ([]model.SitePublishedHistory, int64, error)
}

func NewSitePublishedHistoryRepository(
	repository *db.Repository,
) SitePublishedHistoryRepository {
	return &sitePublishedHistoryRepository{
		Repository: repository,
	}
}

type sitePublishedHistoryRepository struct {
	*db.Repository
}

func (r *sitePublishedHistoryRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *sitePublishedHistoryRepository) Update(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Save(entity).Error
}

// CreateHistoryRecord creates a history record
func (r *sitePublishedHistoryRepository) CreateHistoryRecord(ctx context.Context, siteID string, publishedBy string, versionNumber int, publishedData interface{}) (*model.SitePublishedHistory, error) {
	// Marshal the published data to JSON
	publishedDataBytes, err := json.Marshal(publishedData)
	if err != nil {
		return nil, err
	}

	// Create new history record
	historyData := &model.SitePublishedHistory{
		SiteID:        siteID,
		VersionNumber: versionNumber,
		PublishedData: publishedDataBytes,
		PublishedBy:   &publishedBy,
		PublishedAt:   time.Now(),
	}

	if err := r.DB(ctx).Create(historyData).Error; err != nil {
		return nil, err
	}

	return historyData, nil
}

// GetPublishHistory gets publish history for a site
func (r *sitePublishedHistoryRepository) GetPublishHistory(ctx context.Context, siteID string, limit int, offset int) ([]model.SitePublishedHistory, int64, error) {
	var history []model.SitePublishedHistory
	var total int64

	// Get total count
	if err := r.DB(ctx).Model(&model.SitePublishedHistory{}).Where("site_id = ?", siteID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated history
	if err := r.DB(ctx).Where("site_id = ?", siteID).
		Order("published_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&history).Error; err != nil {
		return nil, 0, err
	}

	return history, total, nil
}
