package site

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

type SiteRepository interface {
	db.BaseOperation
	ExistsCreatorAndSiteID(ctx context.Context, creatorID string, siteId string) (bool, error)
	GetBySiteID(ctx context.Context, siteId string) (*model.Site, error)
	GetBySiteIDs(ctx context.Context, siteIds []string) ([]*model.Site, error)
	GetByPath(ctx context.Context, sitePath string) (*model.Site, error)
	GetByDomain(ctx context.Context, host string) (*model.Site, error)
	UpdateTheme(ctx context.Context, siteID string, theme int) error
}

func NewSiteRepository(
	repository *db.Repository,
) SiteRepository {
	return &siteRepository{
		Repository: repository,
	}
}

type siteRepository struct {
	*db.Repository
}

func (r *siteRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

// update
func (r *siteRepository) Update(ctx context.Context, entity interface{}) error {
	site := entity.(*model.Site)
	return r.DB(ctx).Where("site_id = ?", site.SiteID).Updates(site).Error
}

func (r *siteRepository) GetBySiteID(ctx context.Context, siteId string) (*model.Site, error) {
	var site model.Site
	err := r.DB(ctx).Where("site_id = ?", siteId).First(&site).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &site, nil
}

// BatchGetSites batch query site information
func (r *siteRepository) GetBySiteIDs(ctx context.Context, siteIds []string) ([]*model.Site, error) {
	if len(siteIds) == 0 {
		return nil, nil
	}
	var sites []*model.Site
	err := r.DB(ctx).Where("site_id IN ?", siteIds).Find(&sites).Error
	if err != nil {
		return nil, err
	}
	return sites, nil
}

// FindSiteByPath find site information by site path
func (r *siteRepository) GetByPath(ctx context.Context, sitePath string) (*model.Site, error) {
	var site model.Site // TODO: After customization is introduced, consider there might be multiple PATHs
	err := r.DB(ctx).Where("path = ?", sitePath).First(&site).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &site, nil
}

func (r *siteRepository) GetByDomain(ctx context.Context, domain string) (*model.Site, error) {
	var site model.Site
	err := r.DB(ctx).Where("domain = ?", domain).First(&site).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &site, nil
}

func (r *siteRepository) UpdateTheme(ctx context.Context, siteID string, theme int) error {
	return r.DB(ctx).Model(&model.Site{}).Where("site_id = ?", siteID).Update("theme", theme).Error
}

// ListBySiteID retrieves all sites for a given site ID

func (r *siteRepository) ExistsCreatorAndSiteID(ctx context.Context, creatorID string, siteId string) (bool, error) {
	var count int64
	err := r.DB(ctx).
		Table("sites").
		Joins("JOIN creator_sites ON sites.site_id = creator_sites.site_id").
		Where("creator_sites.creator_id = ? AND sites.site_id = ?", creatorID, siteId).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
