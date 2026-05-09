package site

import (
	"context"
	"encoding/json"

	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

type SiteBuilderDataRepository interface {
	db.BaseOperation
	GetBySiteID(ctx context.Context, siteID string) (*model.SiteBuilderData, error)
	UpdateBySiteID(ctx context.Context, siteID string, createdBy string, siteData interface{}) (*model.SiteBuilderData, error)
}

func NewSiteBuilderDataRepository(
	repository *db.Repository,
) SiteBuilderDataRepository {
	return &siteBuilderDataRepository{
		Repository: repository,
	}
}

type siteBuilderDataRepository struct {
	*db.Repository
}

func (r *siteBuilderDataRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *siteBuilderDataRepository) Update(ctx context.Context, entity interface{}) error {
	// Convert to SiteBuilderData
	data := entity.(*model.SiteBuilderData)
	return r.DB(ctx).Where("site_id = ?", data.SiteID).Updates(data).Error
}

// GetBySiteID gets site builder data by site ID
func (r *siteBuilderDataRepository) GetBySiteID(ctx context.Context, siteID string) (*model.SiteBuilderData, error) {
	var data model.SiteBuilderData
	if err := r.DB(ctx).Where("site_id = ?", siteID).First(&data).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &data, nil
}

// UpdateBySiteID updates or creates site builder data by site ID
func (r *siteBuilderDataRepository) UpdateBySiteID(ctx context.Context, siteID string, createdBy string, siteData interface{}) (*model.SiteBuilderData, error) {
	// Marshal the site data to JSON
	siteDataBytes, err := json.Marshal(siteData)
	if err != nil {
		return nil, err
	}

	// Try to find existing record
	var existingData model.SiteBuilderData
	err = r.DB(ctx).Where("site_id = ?", siteID).First(&existingData).Error

	if err == gorm.ErrRecordNotFound {
		// Create new record
		newData := &model.SiteBuilderData{
			SiteID:        siteID,
			SiteData:      siteDataBytes,
			VersionNumber: 1,
			CreatedBy:     &createdBy,
		}
		if err := r.DB(ctx).Create(newData).Error; err != nil {
			return nil, err
		}
		return newData, nil
	} else if err != nil {
		return nil, err
	}

	// Update existing record
	existingData.SiteData = siteDataBytes
	existingData.VersionNumber++

	if err := r.DB(ctx).Save(&existingData).Error; err != nil {
		return nil, err
	}

	return &existingData, nil
}
