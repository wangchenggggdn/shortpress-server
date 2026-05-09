package ads

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"
)

// AdLocationRepository defines ad location database operations
type AdLocationRepository interface {
	Create(ctx context.Context, adLocation *model.AdLocation) error
	Update(ctx context.Context, adLocation *model.AdLocation) error
	GetByAdIDAndLocation(ctx context.Context, adID string, location string) (*model.AdLocation, error)
	GetByAdID(ctx context.Context, adID string) ([]*model.AdLocation, error)
	GetByLocation(ctx context.Context, location string, siteID string, status int8) ([]*model.AdLocation, error)
	List(ctx context.Context, siteID string) ([]*model.AdLocationView, error)
	UpdateStatus(ctx context.Context, adID string, location string, status int8) error
	Delete(ctx context.Context, adID string, location string) error
}

type adLocationRepository struct {
	*db.Repository
}

// NewAdLocationRepository creates a new ad location repository instance
func NewAdLocationRepository(repository *db.Repository) AdLocationRepository {
	return &adLocationRepository{
		Repository: repository,
	}
}

// Create creates a new ad location record
func (r *adLocationRepository) Create(ctx context.Context, adLocation *model.AdLocation) error {
	return r.DB(ctx).Create(adLocation).Error
}

// Update updates an existing ad location record
func (r *adLocationRepository) Update(ctx context.Context, adLocation *model.AdLocation) error {
	return r.DB(ctx).Where("ad_id = ? AND location = ?", adLocation.AdID, adLocation.Location).
		Updates(adLocation).Error
}

// GetByAdIDAndLocation retrieves an ad location by ad_id and location
func (r *adLocationRepository) GetByAdIDAndLocation(ctx context.Context, adID string, location string) (*model.AdLocation, error) {
	var adLocation model.AdLocation
	err := r.DB(ctx).Where("ad_id = ? AND location = ?", adID, location).First(&adLocation).Error
	if err != nil {
		return nil, err
	}
	return &adLocation, nil
}

// GetByAdID retrieves all locations for a specific ad
func (r *adLocationRepository) GetByAdID(ctx context.Context, adID string) ([]*model.AdLocation, error) {
	var adLocations []*model.AdLocation
	err := r.DB(ctx).Where("ad_id = ?", adID).Find(&adLocations).Error
	if err != nil {
		return nil, err
	}
	return adLocations, nil
}

// GetByLocation retrieves all ads for a specific location and site ID
func (r *adLocationRepository) GetByLocation(ctx context.Context, location string, siteID string, status int8) ([]*model.AdLocation, error) {
	var adLocations []*model.AdLocation
	query := r.DB(ctx).Where("location = ? AND site_id = ?", location, siteID)

	if status > 0 {
		query = query.Where("status = ?", status)
	}

	err := query.Find(&adLocations).Error
	if err != nil {
		return nil, err
	}
	return adLocations, nil
}

// UpdateStatus updates the status of an ad location
func (r *adLocationRepository) UpdateStatus(ctx context.Context, adID string, location string, status int8) error {
	return r.DB(ctx).Model(&model.AdLocation{}).
		Where("ad_id = ? AND location = ?", adID, location).
		Update("status", status).Error
}

// Delete deletes an ad location
func (r *adLocationRepository) Delete(ctx context.Context, adID string, location string) error {
	return r.DB(ctx).Where("ad_id = ? AND location = ?", adID, location).
		Delete(&model.AdLocation{}).Error
}

func (r *adLocationRepository) List(ctx context.Context, siteID string) ([]*model.AdLocationView, error) {
	var locations []*model.AdLocationView
	query := r.DB(ctx).
		Table("ad_locations").
		Select("ad_locations.*, ads.ad_id, ads.format, ads.ad_network, ads.conf").
		Joins("LEFT JOIN ads ON ad_locations.ad_id = ads.ad_id").
		Where("ad_locations.site_id = ?", siteID).
		Order("ad_locations.created_at DESC")

	if err := query.Find(&locations).Error; err != nil {
		return nil, err
	}

	return locations, nil
}
