package site

import (
	"context"
	"encoding/json"
	"time"

	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

type SiteCurrentPublishedRepository interface {
	db.BaseOperation
	GetBySiteID(ctx context.Context, siteID string) (*model.SiteCurrentPublished, error)
	CreateCurrentPublished(ctx context.Context, siteID string, publishedBy string, versionNumber int, publishedData interface{}) (*model.SiteCurrentPublished, error)
	UpdateCurrentPublished(ctx context.Context, siteID string, publishedBy string, versionNumber int, publishedData interface{}) (*model.SiteCurrentPublished, error)
}

func NewSiteCurrentPublishedRepository(
	repository *db.Repository,
) SiteCurrentPublishedRepository {
	return &siteCurrentPublishedRepository{
		Repository: repository,
	}
}

type siteCurrentPublishedRepository struct {
	*db.Repository
}

func (r *siteCurrentPublishedRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *siteCurrentPublishedRepository) Update(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Save(entity).Error
}

// GetBySiteID gets current published site data by site ID
func (r *siteCurrentPublishedRepository) GetBySiteID(ctx context.Context, siteID string) (*model.SiteCurrentPublished, error) {
	var data model.SiteCurrentPublished
	if err := r.DB(ctx).Where("site_id = ?", siteID).First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &data, nil
}

// CreateCurrentPublished creates current published data record
func (r *siteCurrentPublishedRepository) CreateCurrentPublished(ctx context.Context, siteID string, publishedBy string, versionNumber int, publishedData interface{}) (*model.SiteCurrentPublished, error) {
	// Marshal the published data to JSON
	publishedDataBytes, err := json.Marshal(publishedData)
	if err != nil {
		return nil, err
	}

	// Create new published data record
	newData := &model.SiteCurrentPublished{
		SiteID:        siteID,
		VersionNumber: versionNumber,
		PublishedData: publishedDataBytes,
		PublishedBy:   &publishedBy,
		PublishedAt:   time.Now(),
	}

	if err := r.DB(ctx).Create(newData).Error; err != nil {
		return nil, err
	}

	return newData, nil
}

// UpdateCurrentPublished updates existing current published data record
func (r *siteCurrentPublishedRepository) UpdateCurrentPublished(ctx context.Context, siteID string, publishedBy string, versionNumber int, publishedData interface{}) (*model.SiteCurrentPublished, error) {
	// Marshal the published data to JSON
	publishedDataBytes, err := json.Marshal(publishedData)
	if err != nil {
		return nil, err
	}

	// Update existing record
	updatedData := &model.SiteCurrentPublished{
		SiteID:        siteID,
		VersionNumber: versionNumber,
		PublishedData: publishedDataBytes,
		PublishedBy:   &publishedBy,
		PublishedAt:   time.Now(),
	}

	if err := r.DB(ctx).Where("site_id = ?", siteID).Updates(updatedData).Error; err != nil {
		return nil, err
	}

	// Return the updated record
	return r.GetBySiteID(ctx, siteID)
}
