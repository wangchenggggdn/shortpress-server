package site

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

type SitePageTemplateRepository interface {
	db.BaseOperation
	List(ctx context.Context, page, pageSize int) ([]*model.SitePageTemplate, int64, error)
	GetByTemplateID(ctx context.Context, templateID string) (*model.SitePageTemplate, error)
}

func NewSitePageTemplateRepository(
	repository *db.Repository,
) SitePageTemplateRepository {
	return &sitePageTemplateRepository{Repository: repository}
}

type sitePageTemplateRepository struct {
	*db.Repository
}

func (r *sitePageTemplateRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *sitePageTemplateRepository) Update(ctx context.Context, entity interface{}) error {
	t := entity.(*model.SitePageTemplate)
	return r.DB(ctx).Where("template_id = ?", t.TemplateID).Updates(t).Error
}

func (r *sitePageTemplateRepository) List(ctx context.Context, page, pageSize int) ([]*model.SitePageTemplate, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var total int64
	if err := r.DB(ctx).Model(&model.SitePageTemplate{}).Where("status = ?", 1).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*model.SitePageTemplate{}, 0, nil
	}

	var items []*model.SitePageTemplate
	err := r.DB(ctx).
		Where("status = ?", 1).
		Order("id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *sitePageTemplateRepository) GetByTemplateID(ctx context.Context, templateID string) (*model.SitePageTemplate, error) {
	var t model.SitePageTemplate
	err := r.DB(ctx).Where("template_id = ? AND status = ?", templateID, 1).First(&t).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}
