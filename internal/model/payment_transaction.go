package model

import (
	"time"
)

const (
	PaymentTypeSubscription = 1 //Subscription purchase
	PaymentTypeCoinPackage  = 2 //Coin package purchase
	PaymentTypeGrantByAdmin = 3 //Granted by administrator

	RelatedTypeSubscription = 1
	RelatedTypeCoinPackage  = 2
	RelatedTypeGrantByAdmin = 3

	PaymentStatusPending         = 1
	PaymentStatusSuccess         = 2
	PaymentStatusFailed          = 3
	PaymentStatusRefunded        = 4
	PaymentStatusPartialRefunded = 5
	PaymentStatusExpired         = 6
)

// PaymentTransaction represents a payment transaction in the system
type PaymentTransaction struct {
	ID                uint    `gorm:"primaryKey;column:id"`
	TransactionID     string  `gorm:"column:transaction_id;uniqueIndex"`
	OrderID           string  `gorm:"column:order_id;index"`
	UserID            string  `gorm:"column:user_id;index"`
	SiteID            string  `gorm:"column:site_id;index"`
	Amount            int64   `gorm:"column:amount"`
	Currency          string  `gorm:"column:currency;default:USD"`
	Provider          string  `gorm:"column:provider"` // stripe, paypal
	ProviderPaymentID string  `gorm:"column:provider_payment_id;uniqueIndex:uk_provider_payment"`
	PaymentType       int     `gorm:"column:payment_type"`       // 1: subscription, 2: coin package, 3: single purchase
	Status            int     `gorm:"column:status;index"`       // 1: pending, 2: success, 3: failed, 4: refunded, 5: partial refunded
	RelatedID         string  `gorm:"column:related_id"`         // ID of related package
	RelatedType       int     `gorm:"column:related_type"`       // 1: subscription package, 2: coin package, 3: single purchase
	Snapshot          JSONMap `gorm:"column:snapshot;type:json"` // Store package info at time of purchase
	RefundAmount      *int64  `gorm:"column:refund_amount"`
	PayerEmail        string  `gorm:"column:payer_email"` // Email entered on Stripe/PayPal checkout
	ErrorMessage      string  `gorm:"column:error_message"`
	// StripeCustomerID string    `gorm:"column:stripe_customer_id"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

type PaymentTransactionView struct {
	PaymentTransaction
	Email    string `gorm:"column:email"`
	PixelID  string `gorm:"column:pixel_id"`
	Platform string `gorm:"column:platform"`
}

// DailyIncomeStatistics represents daily income statistics from the database
type DailyIncomeStatistics struct {
	Date               string `gorm:"column:date"`
	TotalAmount        int64  `gorm:"column:total_amount"`
	TransactionCount   int    `gorm:"column:transaction_count"`
	IapAmount          int64  `gorm:"column:iap_amount"`
	SubscriptionAmount int64  `gorm:"column:subscription_amount"`
	RenewalAmount      int64  `gorm:"column:renewal_amount"`
}

func (PaymentTransaction) TableName() string {
	return "payment_transactions"
}
