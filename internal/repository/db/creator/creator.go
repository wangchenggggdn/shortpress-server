package creator

import (
	"context"

	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

type CreatorRepository interface {
	db.BaseOperation
	GetByEmail(ctx context.Context, email string) (*model.Creator, error)
	GetByCreatorID(ctx context.Context, creatorID string) (*model.Creator, error)
}

func NewCreatorRepository(
	repository *db.Repository,
) CreatorRepository {
	return &creatorRepository{
		Repository: repository,
	}
}

type creatorRepository struct {
	*db.Repository
}

func (r *creatorRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *creatorRepository) Update(ctx context.Context, entity interface{}) error {
	// Convert to creator
	creator := entity.(*model.Creator)
	return r.DB(ctx).Where("creator_id = ?", creator.CreatorID).Updates(creator).Error
}

// Get creator by email
func (r *creatorRepository) GetByEmail(ctx context.Context, email string) (*model.Creator, error) {
	var creator model.Creator
	if err := r.DB(ctx).Where("email = ?", email).First(&creator).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &creator, nil
}

// Get creator by email or creator name
func (r *creatorRepository) GetByEmailOrName(ctx context.Context, email string) (*model.Creator, error) {
	var creator model.Creator
	err := r.DB(ctx).Where("email = ? ", email).First(&creator).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &creator, nil
}

// Get creator by creator ID
func (r *creatorRepository) GetByCreatorID(ctx context.Context, creatorID string) (*model.Creator, error) {
	var creator model.Creator
	if err := r.DB(ctx).Where("creator_id = ?", creatorID).First(&creator).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &creator, nil
}
