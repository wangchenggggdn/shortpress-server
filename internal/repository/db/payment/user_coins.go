package payment

import (
	"context"
	"shortpress-server/internal/common"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

// UserCoinsRepository defines the repository interface for user coin accounts
type UserCoinsRepository interface {
	db.BaseOperation
	GetByUserAndSite(ctx context.Context, userID, siteID string) (*model.UserCoins, error)
	UpdateBalance(ctx context.Context, userID, siteID string, coinAmount int, realMoneySpent int64) (*model.UserCoins, error)
	AddPresentCoins(ctx context.Context, userID, siteID string, coinAmount int) (*model.UserCoins, error)
	DeductCoins(ctx context.Context, userID, siteID string, coinAmount int) (*model.UserCoins, error)
}

type userCoinsRepository struct {
	*db.Repository
}

// NewUserCoinsRepository creates a new user coins repository
func NewUserCoinsRepository(r *db.Repository) UserCoinsRepository {
	return &userCoinsRepository{
		Repository: r,
	}
}

func (r *userCoinsRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *userCoinsRepository) Update(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Save(entity).Error
}

func (r *userCoinsRepository) GetByUserAndSite(ctx context.Context, userID, siteID string) (*model.UserCoins, error) {
	var userCoins model.UserCoins
	err := r.DB(ctx).Where("user_id = ? AND site_id = ?", userID, siteID).First(&userCoins).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &userCoins, nil
}

// UpdateBalance updates a user's coin balance atomically, using a transaction if provided
func (r *userCoinsRepository) UpdateBalance(ctx context.Context, userID, siteID string, coinAmount int, realMoneySpent int64) (*model.UserCoins, error) {
	db := r.DB(ctx)

	// First, try to get the user's coin account
	var userCoins model.UserCoins
	err := db.Where("user_id = ? AND site_id = ?", userID, siteID).First(&userCoins).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create a new account if it doesn't exist
			userCoins = model.UserCoins{
				UserID:  userID,
				SiteID:  siteID,
				Balance: 0,
			}

			if coinAmount < 0 || realMoneySpent < 0 {
				return nil, gorm.ErrRecordNotFound
			}
		} else {
			return nil, err
		}
	}

	// Check if balance would go negative
	if userCoins.Balance+coinAmount < 0 {
		return nil, common.ErrInsufficientCoins
	}

	// Update the balance
	if coinAmount > 0 {
		userCoins.TotalEarned += coinAmount
	} else if coinAmount < 0 {
		userCoins.TotalSpent -= coinAmount // amount is negative
	}

	if realMoneySpent > 0 {
		userCoins.TotalRealMoneySpent += realMoneySpent
	}

	userCoins.Balance += coinAmount

	// Save the updated account
	if userCoins.ID > 0 {
		err = db.Save(&userCoins).Error
	} else {
		err = db.Create(&userCoins).Error
	}

	if err != nil {
		return nil, err
	}

	return &userCoins, nil
}

// AddPresentCoins adds coins to the present field for subscription rewards
// This only updates the present field for tracking purposes, does not increase balance or total_earned
func (r *userCoinsRepository) AddPresentCoins(ctx context.Context, userID, siteID string, coinAmount int) (*model.UserCoins, error) {
	db := r.DB(ctx)

	// First, try to get the user's coin account
	var userCoins model.UserCoins
	err := db.Where("user_id = ? AND site_id = ?", userID, siteID).First(&userCoins).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create a new account if it doesn't exist
			userCoins = model.UserCoins{
				UserID:  userID,
				SiteID:  siteID,
				Balance: 0,
			}
		} else {
			return nil, err
		}
	}

	// Add coins to present field only (for tracking subscription rewards)
	userCoins.Present += coinAmount

	// Save the updated account
	if userCoins.ID > 0 {
		err = db.Save(&userCoins).Error
	} else {
		err = db.Create(&userCoins).Error
	}

	if err != nil {
		return nil, err
	}

	return &userCoins, nil
}

// DeductCoins deducts coins, prioritizing present coins first, then balance
func (r *userCoinsRepository) DeductCoins(ctx context.Context, userID, siteID string, coinAmount int) (*model.UserCoins, error) {
	db := r.DB(ctx)

	// Get the user's coin account
	var userCoins model.UserCoins
	err := db.Where("user_id = ? AND site_id = ?", userID, siteID).First(&userCoins).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}

	// Check if total coins (present + balance) is enough
	totalCoins := userCoins.Present + userCoins.Balance
	if totalCoins < coinAmount {
		return nil, common.ErrInsufficientCoins
	}

	remainingToDeduct := coinAmount

	// First, deduct from present
	if userCoins.Present >= remainingToDeduct {
		// Present is enough
		userCoins.Present -= remainingToDeduct
		remainingToDeduct = 0
	} else {
		// Deduct all present, then deduct from balance
		remainingToDeduct -= userCoins.Present
		userCoins.Present = 0
	}

	// Then deduct remaining from balance
	if remainingToDeduct > 0 {
		userCoins.Balance -= remainingToDeduct
		userCoins.TotalSpent += remainingToDeduct
	}

	// Save the updated account
	err = db.Save(&userCoins).Error
	if err != nil {
		return nil, err
	}

	return &userCoins, nil
}
