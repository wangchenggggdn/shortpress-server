package ads

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"
)

// AdRepository defines ad database operations
type AdRepository interface {
	db.BaseOperation
	GetByAdID(ctx context.Context, adID string) (*model.Ad, error)
	GetBySiteID(ctx context.Context, siteID string, format int8, status int8) ([]*model.Ad, error)
	UpdateStatus(ctx context.Context, adID string, status int8) error
	Delete(ctx context.Context, adID string) error
	BatchGetByAdIDs(ctx context.Context, adIDs []string) ([]*model.Ad, error)
}

type adRepository struct {
	*db.Repository
}

// NewAdRepository creates a new ad repository instance
func NewAdRepository(repository *db.Repository) AdRepository {
	return &adRepository{
		Repository: repository,
	}
}

// Create creates a new ad record
func (r *adRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

// Update updates an existing ad record
func (r *adRepository) Update(ctx context.Context, entity interface{}) error {
	ad := entity.(*model.Ad)
	return r.DB(ctx).Where("ad_id = ?", ad.AdID).Updates(ad).Error
}

// GetByAdID retrieves an ad by ad_id
func (r *adRepository) GetByAdID(ctx context.Context, adID string) (*model.Ad, error) {
	var ad model.Ad
	err := r.DB(ctx).Where("ad_id = ? AND status != ?", adID, model.AdStatusDeleted).First(&ad).Error
	if err != nil {
		return nil, err
	}
	return &ad, nil
}

// GetBySiteID retrieves all ads by site_id, optionally filtered by format and status
func (r *adRepository) GetBySiteID(ctx context.Context, siteID string, format int8, status int8) ([]*model.Ad, error) {
	var ads []*model.Ad
	query := r.DB(ctx).Model(&model.Ad{}).Where("site_id = ? AND status != ?", siteID, model.AdStatusDeleted)

	if format > 0 {
		query = query.Where("format = ?", format)
	}

	if status > 0 {
		query = query.Where("status = ?", status)
	}

	err := query.Find(&ads).Error
	if err != nil {
		return nil, err
	}
	return ads, nil
}

// UpdateStatus updates an ad's status
func (r *adRepository) UpdateStatus(ctx context.Context, adID string, status int8) error {
	return r.DB(ctx).Model(&model.Ad{}).
		Where("ad_id = ?", adID).
		Update("status", status).Error
}

// Delete soft deletes an ad by setting its status to deleted
func (r *adRepository) Delete(ctx context.Context, adID string) error {
	return r.DB(ctx).Model(&model.Ad{}).
		Where("ad_id = ?", adID).
		Update("status", model.AdStatusDeleted).Error
}

// BatchGetByAdIDs retrieves multiple ads by their ad_ids
func (r *adRepository) BatchGetByAdIDs(ctx context.Context, adIDs []string) ([]*model.Ad, error) {
	if len(adIDs) == 0 {
		return []*model.Ad{}, nil
	}

	var ads []*model.Ad
	err := r.DB(ctx).Where("ad_id IN (?) AND status != ?", adIDs, model.AdStatusDeleted).
		Find(&ads).Error
	if err != nil {
		return nil, err
	}
	return ads, nil
}
