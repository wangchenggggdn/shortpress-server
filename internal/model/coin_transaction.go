package model

import "time"

const (
	// Coin transaction types
	CoinTransactionTypePurchase        = 1
	CoinTransactionTypeConsumption     = 2
	CoinTransactionTypeRefund          = 3
	CoinTransactionTypeReward          = 4
	CoinTransactionTypeAdminAdjustment = 5

	// Coin transaction sources
	CoinSourcePurchase        = "purchase"
	CoinSourceUnlock          = "unlock"
	CoinSourceRefund          = "refund"
	CoinSourceReward          = "reward"
	CoinSourceAdminAdjustment = "admin_adjustment"
	CoinSourcePluginAdd       = "plugin_add"    // 插件加点
	CoinSourcePluginDeduct    = "plugin_deduct" // 插件扣点

	// Related types for coin transactions
	CoinRelatedTypePayment  = 1
	CoinRelatedTypeVideo    = 2
	CoinRelatedTypePlaylist = 3
	CoinRelatedTypeAdminID  = 4
	CoinRelatedTypePlugin   = 5 // 插件扣点
)

// CoinTransaction represents a coin transaction record
type CoinTransaction struct {
	ID            uint      `gorm:"primaryKey;column:id"`
	TransactionID string    `gorm:"column:transaction_id;uniqueIndex"`
	UserID        string    `gorm:"column:user_id;index"`
	SiteID        string    `gorm:"column:site_id;index"`
	Amount        int       `gorm:"column:amount"`
	BeforeBalance int       `gorm:"column:before_balance"`
	Balance       int       `gorm:"column:balance"`
	Source        string    `gorm:"column:source"`
	RelatedID     string    `gorm:"column:related_id"`
	RelatedType   int       `gorm:"column:related_type"`
	AdminID       string    `gorm:"column:admin_id"`
	Description   string    `gorm:"column:description"`
	Snapshot      JSONMap   `gorm:"column:snapshot;type:json"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (CoinTransaction) TableName() string {
	return "coin_transactions"
}
