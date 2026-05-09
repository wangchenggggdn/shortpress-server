package user

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

// UserProfileRepository defines user profile database operations
type UserProfileRepository interface {
	Create(ctx context.Context, profile *model.UserProfile) error
	Update(ctx context.Context, profile *model.UserProfile) error
	GetByUserID(ctx context.Context, userID string) (*model.UserProfile, error)
	DeleteByUserID(ctx context.Context, userID string) error
}

type userProfileRepository struct {
	*db.Repository
}

// NewUserProfileRepository creates a new user profile repository instance
func NewUserProfileRepository(repository *db.Repository) UserProfileRepository {
	return &userProfileRepository{
		Repository: repository,
	}
}

// Create creates a new user profile
func (r *userProfileRepository) Create(ctx context.Context, profile *model.UserProfile) error {
	return r.DB(ctx).Create(profile).Error
}

// Update updates an existing user profile
func (r *userProfileRepository) Update(ctx context.Context, profile *model.UserProfile) error {
	return r.DB(ctx).Where("user_id = ?", profile.UserID).Updates(profile).Error
}

// GetByUserID retrieves a user profile by user_id
func (r *userProfileRepository) GetByUserID(ctx context.Context, userID string) (*model.UserProfile, error) {
	var profile model.UserProfile
	err := r.DB(ctx).Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

// DeleteByUserID deletes a user profile by user_id
func (r *userProfileRepository) DeleteByUserID(ctx context.Context, userID string) error {
	return r.DB(ctx).Where("user_id = ?", userID).Delete(&model.UserProfile{}).Error
}
