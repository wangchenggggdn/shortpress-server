package site

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

type SitePageConfigRepository interface {
	db.BaseOperation
	GetByID(ctx context.Context, id uint) (*model.SitePageConfig, error)
	GetBySiteIDAndType(ctx context.Context, siteID string, pageType string) (*model.SitePageConfig, error)
	ListBySiteID(ctx context.Context, siteID string) ([]*model.SitePageConfig, error)
	Delete(ctx context.Context, id uint) error
}

type sitePageConfigRepository struct {
	*db.Repository
}

func NewSitePageConfigRepository(repository *db.Repository) SitePageConfigRepository {
	return &sitePageConfigRepository{
		Repository: repository,
	}
}

func (r *sitePageConfigRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *sitePageConfigRepository) Update(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Save(entity).Error
}

func (r *sitePageConfigRepository) GetByID(ctx context.Context, id uint) (*model.SitePageConfig, error) {
	var config model.SitePageConfig
	err := r.DB(ctx).Where("id = ?", id).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

func (r *sitePageConfigRepository) GetBySiteIDAndType(ctx context.Context, siteID string, pageType string) (*model.SitePageConfig, error) {
	var config model.SitePageConfig
	err := r.DB(ctx).Where("site_id = ? AND type = ?", siteID, pageType).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

func (r *sitePageConfigRepository) ListBySiteID(ctx context.Context, siteID string) ([]*model.SitePageConfig, error) {
	var configs []*model.SitePageConfig
	err := r.DB(ctx).Where("site_id = ?", siteID).Find(&configs).Error
	if err != nil {
		return nil, err
	}
	return configs, nil
}

func (r *sitePageConfigRepository) Delete(ctx context.Context, id uint) error {
	return r.DB(ctx).Delete(&model.SitePageConfig{}, id).Error
}
