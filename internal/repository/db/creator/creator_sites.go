package creator

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

type CreatorSiteRepository interface {
	db.BaseOperation
	// Find all site records associated with creator ID
	GetByCreator(ctx context.Context, creatorID string) ([]*model.CreatorSites, error)
	Count(ctx context.Context, creatorID string) (int64, error)
	FindCreatorBySitID(ctx context.Context, siteID string) (string, error)
}

type creatorSiteRepository struct {
	*db.Repository
}

func NewCreatorWebsiteRepository(repository *db.Repository) CreatorSiteRepository {
	return &creatorSiteRepository{
		Repository: repository,
	}
}

func (r *creatorSiteRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *creatorSiteRepository) Update(ctx context.Context, entity interface{}) error {
	// Convert to creatorsite
	creatorSite := entity.(*model.CreatorSites)
	return r.DB(ctx).Where("creator_id = ? AND site_id = ?", creatorSite.CreatorID, creatorSite.SiteID).Updates(creatorSite).Error
}

// GetSitesByCreator Find all site records associated with creatorID
func (r *creatorSiteRepository) GetByCreator(ctx context.Context, creatorID string) ([]*model.CreatorSites, error) {
	var list []*model.CreatorSites
	err := r.DB(ctx).Where("creator_id = ?", creatorID).Find(&list).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return list, nil
}

func (r *creatorSiteRepository) Count(ctx context.Context, creatorID string) (int64, error) {
	var count int64
	err := r.DB(ctx).Model(&model.CreatorSites{}).Where("creator_id = ?", creatorID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *creatorSiteRepository) FindCreatorBySitID(ctx context.Context, siteID string) (string, error) {
	var creatorSite model.CreatorSites
	err := r.DB(ctx).Where("site_id = ?", siteID).First(&creatorSite).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", err
	}
	return creatorSite.CreatorID, nil
}
