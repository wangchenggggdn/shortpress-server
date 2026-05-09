package model

import (
	"time"
)

// UserSubscription represents a user's subscription
type UserSubscription struct {
	SubscriptionID         string    `gorm:"column:subscription_id;uniqueIndex:uk_subscription_id"`
	UserID                 string    `gorm:"column:user_id;index:idx_user_id"`
	SiteID                 string    `gorm:"column:site_id;index:idx_site_id"`
	PackageID              string    `gorm:"column:package_id"`
	Provider               string    `gorm:"column:provider"`
	ProviderSubscriptionID string    `gorm:"column:provider_subscription_id"`
	ProviderCustomerID     string    `gorm:"column:provider_customer_id"`
	Status                 int       `gorm:"column:status;default:1;index:idx_status"` // 1:Active 2:Paused 3:Cancelled 4:Expired
	CurrentPeriodStart     time.Time `gorm:"column:current_period_start"`
	CurrentPeriodEnd       time.Time `gorm:"column:current_period_end;index:idx_current_period_end"`
	CancelAtPeriodEnd      bool      `gorm:"column:cancel_at_period_end;default:0"`
	CreatedAt              time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt              time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (UserSubscription) TableName() string {
	return "user_subscriptions"
}

// Subscription status constants
const (
	SubscriptionStatusNone      = 0
	SubscriptionStatusActive    = 1
	SubscriptionStatusPaused    = 2
	SubscriptionStatusCancelled = 3
	SubscriptionStatusExpired   = 4
)
