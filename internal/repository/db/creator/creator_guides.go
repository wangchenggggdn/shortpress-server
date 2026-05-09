package creator

import (
	"context"

	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

type CreatorGuidesRepository interface {
	db.BaseOperation
	GetByCreatorID(ctx context.Context, creatorID string) ([]*model.CreatorGuides, error)
}

func NewCreatorGuidesRepository(repository *db.Repository) CreatorGuidesRepository {
	return &creatorGuidesRepository{
		Repository: repository,
	}
}

type creatorGuidesRepository struct {
	*db.Repository
}

func (r *creatorGuidesRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *creatorGuidesRepository) Update(ctx context.Context, entity interface{}) error {
	creatorGuides := entity.(*model.CreatorGuides)
	return r.DB(ctx).Where("creator_id = ?", creatorGuides.CreatorID).Updates(creatorGuides).Error
}

// GetByCreatorID
func (r *creatorGuidesRepository) GetByCreatorID(ctx context.Context, creatorID string) ([]*model.CreatorGuides, error) {
	var guides []*model.CreatorGuides
	err := r.DB(ctx).Where("creator_id = ?", creatorID).Find(&guides).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return guides, nil
		}
		return nil, err
	}
	return guides, err
}
