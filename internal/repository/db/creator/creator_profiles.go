package creator

import (
	"context"

	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

type CreatorProfileRepository interface {
	db.BaseOperation
	GetProfileByCreatorID(ctx context.Context, creatorID string) (*model.CreatorProfile, error)
	UpdateProfile(ctx context.Context, profile *model.CreatorProfile) error
}

func NewCreatorProfileRepository(repository *db.Repository) CreatorProfileRepository {
	return &creatorProfileRepository{
		Repository: repository,
	}
}

type creatorProfileRepository struct {
	*db.Repository
}

func (r *creatorProfileRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *creatorProfileRepository) Update(ctx context.Context, entity interface{}) error {
	//转creatorprofile
	creatorProfile := entity.(*model.CreatorProfile)
	return r.DB(ctx).Where("creator_id = ?", creatorProfile.CreatorID).Updates(creatorProfile).Error
}

func (r *creatorProfileRepository) GetProfileByCreatorID(ctx context.Context, creatorID string) (*model.CreatorProfile, error) {
	var profile model.CreatorProfile
	if err := r.DB(ctx).Where("creator_id = ?", creatorID).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

// UpdateProfile updates a creator profile in the database.
// It uses GORM's Save method which will update the record if the primary key exists,
// otherwise it will create a new record.
//
// Parameters:
//   - ctx: Context for database operation
//   - profile: The creator profile to be updated
//
// Returns:
//   - error: Any error that occurred during the update operation
func (r *creatorProfileRepository) UpdateProfile(ctx context.Context, profile *model.CreatorProfile) error {
	return r.DB(ctx).Save(profile).Error
}
