package user

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"
	"time"

	"gorm.io/gorm"
)

// UserRepository defines user database operations
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	GetByUserID(ctx context.Context, userID string) (*model.User, error)
	GetByEmailAndSiteID(ctx context.Context, email string, siteID string) (*model.User, error)
	GetByIdentifierAndSiteID(ctx context.Context, identifier string, siteID string) (*model.User, error)
	UpdateLoginTime(ctx context.Context, userID string) error
	UpdatePremiumType(ctx context.Context, userID string, premiumType int, expiresAt *time.Time, onetimeSub *int8) error
	GetSiteUsers(ctx context.Context, query *model.UserQuery, page, pageSize, sortType int) ([]*model.UserInfoView, int64, error)
	Delete(ctx context.Context, userID string) error
	UpdateIdentifierAndEmail(ctx context.Context, user *model.User) error
	UpdateMetaClickIds(ctx context.Context, userID, fbc, fbp, fbclid string) error
}

type userRepository struct {
	*db.Repository
}

// NewUserRepository creates a new user repository instance
func NewUserRepository(repository *db.Repository) UserRepository {
	return &userRepository{
		Repository: repository,
	}
}

// Create creates a new user
func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	return r.DB(ctx).Create(user).Error
}

// Update updates an existing user
func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	return r.DB(ctx).Where("user_id = ?", user.UserID).Updates(user).Error
}

// Update updates an existing user
func (r *userRepository) UpdateIdentifierAndEmail(ctx context.Context, user *model.User) error {
	return r.DB(ctx).Where("user_id = ?", user.UserID).Select("identifier", "email").Updates(user).Error
}

func (r *userRepository) UpdateMetaClickIds(ctx context.Context, userID, fbc, fbp, fbclid string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"meta_click_captured_at": now,
		"updated_at":             now,
	}
	if fbc != "" {
		updates["meta_fbc"] = fbc
	}
	if fbp != "" {
		updates["meta_fbp"] = fbp
	}
	if fbclid != "" {
		updates["meta_fbclid"] = fbclid
	}
	return r.DB(ctx).Model(&model.User{}).Where("user_id = ?", userID).Updates(updates).Error
}

// GetByUserID retrieves a user by user_id
func (r *userRepository) GetByUserID(ctx context.Context, userID string) (*model.User, error) {
	var user model.User
	err := r.DB(ctx).Where("user_id = ?", userID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.DB(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmailAndSiteID retrieves a user by email and site ID
func (r *userRepository) GetByEmailAndSiteID(ctx context.Context, email string, siteID string) (*model.User, error) {
	var user model.User
	err := r.DB(ctx).Where("email = ? AND site_id = ?", email, siteID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByIdentifierAndSiteID retrieves a user by identifier and site ID
func (r *userRepository) GetByIdentifierAndSiteID(ctx context.Context, identifier string, siteID string) (*model.User, error) {
	var user model.User
	err := r.DB(ctx).Where("identifier = ? AND site_id = ?", identifier, siteID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// UpdateLoginTime updates the last login time for a user
func (r *userRepository) UpdateLoginTime(ctx context.Context, userID string) error {
	return r.DB(ctx).Model(&model.User{}).
		Where("user_id = ?", userID).
		UpdateColumn("last_login_at", gorm.Expr("NOW()")).Error
}

// GetSiteUsers retrieves users associated with a specific site ID, with optional search, pagination, and sorting.
func (r *userRepository) GetSiteUsers(ctx context.Context, query *model.UserQuery, page, pageSize, sortType int) ([]*model.UserInfoView, int64, error) {
	db := r.DB(ctx)
	var users []*model.UserInfoView
	var total int64

	// Base query for users with the specified site ID, join with user_profiles to get nickname
	baseQuery := db.Table("users").
		Select("users.*, user_profiles.nickname").
		Joins("LEFT JOIN user_profiles ON users.user_id = user_profiles.user_id").
		Where("users.site_id = ?", query.SiteID)

	// Add specific user ID filter if provided
	if query.UserID != "" {
		baseQuery = baseQuery.Where("users.user_id = ?", query.UserID)
	}

	// Add status filter if not requesting all statuses
	if query.Status != -1 {
		baseQuery = baseQuery.Where("users.status = ?", query.Status)
	}

	// Add search condition if query parameter is provided
	if query.SearchTerm != "" {
		searchTerm := "%" + query.SearchTerm + "%"
		baseQuery = baseQuery.Where("users.email LIKE ? OR user_profiles.nickname LIKE ?", searchTerm, searchTerm)
	}

	// Count total matching records
	err := baseQuery.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// Apply sorting based on sortType
	var orderSQL string
	switch sortType {
	case 0: // Login time desc
		orderSQL = "users.last_login_at DESC"
	case 1: // Created time desc (default)
		orderSQL = "users.created_at DESC"
	case 2: // Login time asc
		orderSQL = "users.last_login_at ASC"
	case 3: // Created time asc
		orderSQL = "users.created_at ASC"
	default:
		orderSQL = "users.created_at DESC"
	}

	// Execute the query with pagination and sorting
	err = baseQuery.
		Order(orderSQL).
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&users).Error

	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Delete completely removes a user and all related records from the database
func (r *userRepository) Delete(ctx context.Context, userID string) error {
	return r.DB(ctx).Where("user_id = ?", userID).Delete(&model.User{}).Error

}

// UpdatePremiumType updates the premium type for a user. onetimeSub 非 nil 时同时更新 onetime_sub。
func (r *userRepository) UpdatePremiumType(ctx context.Context, userID string, premiumType int, expiresAt *time.Time, onetimeSub *int8) error {
	updates := map[string]interface{}{
		"premium_type":       premiumType,
		"premium_expires_at": expiresAt,
	}
	if onetimeSub != nil {
		updates["onetime_sub"] = *onetimeSub
	}
	return r.DB(ctx).Model(&model.User{}).
		Where("user_id = ?", userID).
		Updates(updates).Error
}
