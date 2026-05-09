package user

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"
)

// UserAuthRepository defines user authentication database operations
type UserAuthRepository interface {
	Create(ctx context.Context, userAuth *model.UserAuth) error
	Update(ctx context.Context, userAuth *model.UserAuth) error
	GetByUserIDAndType(ctx context.Context, userID string, authType int8) (*model.UserAuth, error)
	GetAllByUserID(ctx context.Context, userID string) ([]*model.UserAuth, error)
	UpdatePassword(ctx context.Context, userID string, passwordHash string) error
	ValidateEmailCredentials(ctx context.Context, passwordHash string) (*model.UserAuth, error)
	DeleteByUserID(ctx context.Context, userID string) error
}

type userAuthRepository struct {
	*db.Repository
}

// NewUserAuthRepository creates a new user authentication repository instance
func NewUserAuthRepository(repository *db.Repository) UserAuthRepository {
	return &userAuthRepository{
		Repository: repository,
	}
}

// Create creates a new user authentication record
func (r *userAuthRepository) Create(ctx context.Context, userAuth *model.UserAuth) error {
	return r.DB(ctx).Create(userAuth).Error
}

// Update updates an existing user authentication record
func (r *userAuthRepository) Update(ctx context.Context, userAuth *model.UserAuth) error {
	return r.DB(ctx).Where("user_id = ? AND type = ?", userAuth.UserID, userAuth.Type).Updates(userAuth).Error
}

// GetByUserIDAndType retrieves a user auth record by user_id and type
func (r *userAuthRepository) GetByUserIDAndType(ctx context.Context, userID string, authType int8) (*model.UserAuth, error) {
	var userAuth model.UserAuth
	err := r.DB(ctx).Where("user_id = ? AND type = ?", userID, authType).First(&userAuth).Error
	if err != nil {
		return nil, err
	}
	return &userAuth, nil
}

// GetAllByUserID retrieves all authentication methods for a user
func (r *userAuthRepository) GetAllByUserID(ctx context.Context, userID string) ([]*model.UserAuth, error) {
	var userAuths []*model.UserAuth
	err := r.DB(ctx).Where("user_id = ?", userID).Find(&userAuths).Error
	if err != nil {
		return nil, err
	}
	return userAuths, nil
}

// UpdatePassword updates the password hash for email authentication
func (r *userAuthRepository) UpdatePassword(ctx context.Context, userID string, passwordHash string) error {
	return r.DB(ctx).Model(&model.UserAuth{}).
		Where("user_id = ? AND type = ?", userID, model.AuthTypeEmail).
		Update("password_hash", passwordHash).Error
}

// ValidateCredentials validates user credentials for email login
func (r *userAuthRepository) ValidateEmailCredentials(ctx context.Context, passwordHash string) (*model.UserAuth, error) {
	var userAuth model.UserAuth
	err := r.DB(ctx).Where("type = ? AND password_hash = ?",
		model.AuthTypeEmail, passwordHash).First(&userAuth).Error
	if err != nil {
		return nil, err
	}
	return &userAuth, nil
}

// DeleteByUserID deletes a user authentication record by user_id
func (r *userAuthRepository) DeleteByUserID(ctx context.Context, userID string) error {
	return r.DB(ctx).Where("user_id = ?", userID).Delete(&model.UserAuth{}).Error
}
