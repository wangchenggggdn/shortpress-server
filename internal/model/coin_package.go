package model

import (
	"encoding/json"
	"time"
)

// CoinPackage represents a coin package in the system
type CoinPackage struct {
	ID                 uint            `gorm:"primaryKey;column:id"`
	PackageID          string          `gorm:"column:package_id;uniqueIndex"`
	SiteID             string          `gorm:"column:site_id;index"`
	Name               string          `gorm:"column:name"`
	Description        string          `gorm:"column:description"`
	Features           json.RawMessage `gorm:"column:features;type:json"` // Array of feature highlights, e.g. ["买100送20", "限时优惠"]
	CoinAmount         int             `gorm:"column:coin_amount"`
	Price              int64           `gorm:"column:price"`
	OriginalPrice      int64           `gorm:"column:original_price"`
	Currency           string          `gorm:"column:currency;default:USD"`
	DiscountPercentage int             `gorm:"column:discount_percentage"`
	// StripeProductID    string          `gorm:"column:stripe_product_id"`
	StripePriceID   string    `gorm:"column:stripe_price_id"`
	PaypalProductID string    `gorm:"column:paypal_product_id"`
	IOSProductID    string    `gorm:"column:ios_product_id"`   // ios_product_id
	Status          int       `gorm:"column:status;default:1"` // 1:active, 2:disabled
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (CoinPackage) TableName() string {
	return "coin_packages"
}
