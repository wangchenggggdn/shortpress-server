package site

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SiteSeoRepository interface {
	db.BaseOperation
	GetBySiteID(ctx context.Context, siteId string) (*model.SiteSeo, error)
	Save(ctx context.Context, seo *model.SiteSeo) error
}

type siteSeoRepository struct {
	*db.Repository
}

func NewSiteSeoRepository(repository *db.Repository) SiteSeoRepository {
	return &siteSeoRepository{
		Repository: repository,
	}
}

func (r *siteSeoRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

// update
func (r *siteSeoRepository) Update(ctx context.Context, entity interface{}) error {
	seo := entity.(*model.SiteSeo)
	return r.DB(ctx).Where("site_id = ?", seo.SiteID).Updates(seo).Error
}

func (r *siteSeoRepository) GetBySiteID(ctx context.Context, siteId string) (*model.SiteSeo, error) {
	var seo model.SiteSeo
	err := r.DB(ctx).Where("site_id = ?", siteId).First(&seo).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &seo, nil
}

func (r *siteSeoRepository) Save(ctx context.Context, seo *model.SiteSeo) error {
	return r.DB(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "site_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "description", "keywords", "i18n", "updated_at"}),
	}).Create(seo).Error
}
