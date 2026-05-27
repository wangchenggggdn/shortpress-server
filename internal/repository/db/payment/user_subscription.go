package payment

import (
	"context"
	"errors"
	"time"

	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

// UserSubscriptionRepository defines repository operations for user subscriptions
type UserSubscriptionRepository interface {
	db.BaseOperation
	GetBySubscriptionID(ctx context.Context, subscriptionID string) (*model.UserSubscription, error)
	GetByUserID(ctx context.Context, userID string, siteID string) ([]*model.UserSubscription, error)
	GetActiveByUserAndSite(ctx context.Context, userID, siteID string) (*model.UserSubscription, error)
	GetByProviderSubscriptionID(ctx context.Context, provider, providerSubscriptionID string) (*model.UserSubscription, error)
	ListByUserID(ctx context.Context, userID string, status int, cancelAtPeriodEnd int) ([]*model.UserSubscription, error)
	UpdateStatusByProviderSubID(ctx context.Context, subscriptionID string, status int) error
	UpdatePeriodByProviderSubID(ctx context.Context, subscriptionID string, period *model.UserSubscription) error
	CountActiveByUserAndSite(ctx context.Context, userID string, siteID string) (int64, error)
	// GetActiveLegacyStripeRecurringSubscription 当前仍有效的 Stripe 自动续订（sub_ 且未过期）
	GetActiveLegacyStripeRecurringSubscription(ctx context.Context, userID, siteID string) (*model.UserSubscription, error)
	GetExpiringSoon(ctx context.Context, endBeforeTime string, limit int) ([]*model.UserSubscription, error)
}

type userSubscriptionRepository struct {
	*db.Repository
}

// NewUserSubscriptionRepository creates a new user subscription repository
func NewUserSubscriptionRepository(r *db.Repository) UserSubscriptionRepository {
	return &userSubscriptionRepository{
		Repository: r,
	}
}

func (r *userSubscriptionRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *userSubscriptionRepository) Update(ctx context.Context, entity interface{}) error {
	sub, ok := entity.(*model.UserSubscription)
	if !ok {
		return r.DB(ctx).Save(entity).Error
	}
	if sub.SubscriptionID == "" {
		return errors.New("subscription_id is required for update")
	}
	// 表无 GORM 主键字段，不能用 Save，须按 subscription_id 更新
	return r.DB(ctx).Model(&model.UserSubscription{}).
		Where("subscription_id = ?", sub.SubscriptionID).
		Updates(map[string]interface{}{
			"user_id":                  sub.UserID,
			"site_id":                  sub.SiteID,
			"package_id":               sub.PackageID,
			"provider":                 sub.Provider,
			"provider_subscription_id": sub.ProviderSubscriptionID,
			"provider_customer_id":     sub.ProviderCustomerID,
			"status":                   sub.Status,
			"current_period_start":     sub.CurrentPeriodStart,
			"current_period_end":       sub.CurrentPeriodEnd,
			"cancel_at_period_end":     sub.CancelAtPeriodEnd,
		}).Error
}

// GetBySubscriptionID retrieves a subscription by its ID
func (r *userSubscriptionRepository) GetBySubscriptionID(ctx context.Context, subscriptionID string) (*model.UserSubscription, error) {
	var subscription model.UserSubscription
	err := r.DB(ctx).Where("subscription_id = ?", subscriptionID).First(&subscription).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &subscription, nil
}

// GetByUserID retrieves all subscriptions for a user
func (r *userSubscriptionRepository) GetByUserID(ctx context.Context, userID string, siteID string) ([]*model.UserSubscription, error) {
	var subscriptions []*model.UserSubscription
	err := r.DB(ctx).Where("user_id = ? AND site_id = ?", userID, siteID).Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}
	return subscriptions, nil
}

// GetActiveByUserAndSite retrieves active subscriptions for a user in a specific site
func (r *userSubscriptionRepository) GetActiveByUserAndSite(ctx context.Context, userID, siteID string) (*model.UserSubscription, error) {
	var subscription model.UserSubscription
	err := r.DB(ctx).Where("user_id = ? AND site_id = ? AND status = ?", userID, siteID, model.SubscriptionStatusActive).First(&subscription).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &subscription, nil
}

// GetByProviderSubscriptionID retrieves a subscription by provider subscription ID
func (r *userSubscriptionRepository) GetByProviderSubscriptionID(ctx context.Context, provider, providerSubscriptionID string) (*model.UserSubscription, error) {
	var subscription model.UserSubscription
	err := r.DB(ctx).Where("provider = ? AND provider_subscription_id = ?", provider, providerSubscriptionID).First(&subscription).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &subscription, nil
}

// ListByUserID lists subscriptions for a user with pagination
func (r *userSubscriptionRepository) ListByUserID(ctx context.Context, userID string, status int, cancelAtPeriodEnd int) ([]*model.UserSubscription, error) {
	var subscriptions []*model.UserSubscription
	query := r.DB(ctx).Where("user_id = ?", userID).Order("created_at DESC")
	if status != -1 {
		query = query.Where("status = ?", status)
	}
	if cancelAtPeriodEnd != -1 {
		query = query.Where("cancel_at_period_end = ?", cancelAtPeriodEnd)
	}

	err := query.Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}

	return subscriptions, nil
}

// UpdateStatus updates the status of a subscription
func (r *userSubscriptionRepository) UpdateStatusByProviderSubID(ctx context.Context, subscriptionID string, status int) error {
	return r.DB(ctx).Model(&model.UserSubscription{}).
		Where("provider_subscription_id = ?", subscriptionID).
		Update("status", status).Error
}

// UpdatePeriod updates the current period start and end times of a subscription
func (r *userSubscriptionRepository) UpdatePeriodByProviderSubID(ctx context.Context, subscriptionID string, period *model.UserSubscription) error {
	return r.DB(ctx).Model(&model.UserSubscription{}).
		Where("provider_subscription_id = ?", subscriptionID).
		Updates(map[string]interface{}{
			"cancel_at_period_end": period.CancelAtPeriodEnd,
			"current_period_start": period.CurrentPeriodStart,
			"current_period_end":   period.CurrentPeriodEnd,
		}).Error
}

// CountActiveByUserAndSite counts active subscriptions for a user in a site
func (r *userSubscriptionRepository) CountActiveByUserAndSite(ctx context.Context, userID string, siteID string) (int64, error) {
	var count int64
	err := r.DB(ctx).Model(&model.UserSubscription{}).
		Where("user_id = ? AND site_id = ? AND status = ?",
			userID, siteID, model.SubscriptionStatusActive).
		Count(&count).Error
	return count, err
}

// GetActiveLegacyStripeRecurringSubscription returns an active, in-period Stripe auto-renewing subscription if one exists.
func (r *userSubscriptionRepository) GetActiveLegacyStripeRecurringSubscription(ctx context.Context, userID, siteID string) (*model.UserSubscription, error) {
	var subscription model.UserSubscription
	err := r.DB(ctx).
		Where("user_id = ? AND site_id = ? AND provider = ? AND provider_subscription_id LIKE ? AND status = ? AND current_period_end > ?",
			userID, siteID, "stripe", "sub_%", model.SubscriptionStatusActive, time.Now()).
		First(&subscription).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &subscription, nil
}

// GetExpiringSoon retrieves subscriptions that expire soon
func (r *userSubscriptionRepository) GetExpiringSoon(ctx context.Context, endBeforeTime string, limit int) ([]*model.UserSubscription, error) {
	var subscriptions []*model.UserSubscription
	query := r.DB(ctx).
		Where("status = ? AND current_period_end <= ? AND cancel_at_period_end = ?",
			model.SubscriptionStatusActive, endBeforeTime, true).
		Order("current_period_end ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}

	return subscriptions, nil
}
