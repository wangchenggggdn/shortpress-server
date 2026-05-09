package model

import (
	"encoding/json"
	"time"
)

// SubscriptionPackage represents a subscription package in the database
type SubscriptionPackage struct {
	// ID                 int64      `gorm:"column:id;primary_key;auto_increment" json:"id"`
	PackageID          string `gorm:"column:package_id;unique" json:"package_id"`
	SiteID             string `gorm:"column:site_id" json:"site_id"` // Site ID
	Name               string `gorm:"column:name" json:"name"`
	Description        string `gorm:"column:description" json:"description"`
	Interval           string `gorm:"column:interval" json:"interval"` // weekly, monthly, yearly
	Price              int64  `gorm:"column:price" json:"price"`
	OriginalPrice      int64  `gorm:"column:original_price" json:"original_price"`
	DiscountPercentage int    `gorm:"column:discount_percentage" json:"discount_percentage"`
	Currency           string `gorm:"column:currency" json:"currency"`
	// StripeProductID    string     `gorm:"column:stripe_product_id" json:"stripe_product_id"`
	// StripePriceID      string     `gorm:"column:stripe_price_id" json:"stripe_price_id"`
	// PaypalPlanID       string     `gorm:"column:paypal_plan_id" json:"paypal_plan_id"`
	IOSProductID string    `gorm:"column:ios_product_id" json:"ios_product_id"`
	Status       int       `gorm:"column:status;default:1" json:"status"` // 1:enabled 2:disabled
	Coins        int       `gorm:"column:coins;default:0" json:"coins"`   // Coins granted with subscription
	Rights       json.RawMessage `gorm:"column:rights;type:json" json:"rights"` // Subscription benefits/rights
	CreatedAt    time.Time `gorm:"column:created_at;->;<-:create" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;->;<-:update" json:"updated_at"`
}

// TableName returns the table name for the model
func (SubscriptionPackage) TableName() string {
	return "subscription_packages"
}
