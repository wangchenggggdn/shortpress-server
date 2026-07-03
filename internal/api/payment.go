package api

import (
	"encoding/json"
	"shortpress-server/internal/types"
)

// 支付账号信息
type PaymentAccountInfo struct {
	// 账户ID
	AccountID string `json:"accountId" example:"1234567890"`
	// 账号邮箱
	Email string `json:"email" example:"johndoe@example.com"`
	// 国家/地区
	Country string `json:"country" example:"US"`
}

// PaymentProviderConfig represents payment provider configuration
type PaymentProviderConfig struct {
	// Payment provider name
	Provider string `json:"provider" example:"stripe"`
	// Stripe configuration
	StripeConf StripeConfig `json:"stripeConf,omitempty"`
	// PayPal configuration
	PaypalConf PaypalConfig `json:"paypalConf,omitempty"`
	// Site ID (internal use)
	SiteID string `json:"siteId" example:"site-123"`
}

// StripeConfig represents Stripe-specific configuration
type StripeConfig struct {
	// Stripe secret key
	SecretKey string `json:"sk" example:"sk_test_..."`
	// Stripe public key
	PublicKey string `json:"pk" example:"pk_test_..."`
}

// PaypalConfig represents PayPal-specific configuration
type PaypalConfig struct {
	// PayPal client ID
	ClientID string `json:"clientId" example:"AXxx..."`
	// PayPal client secret
	ClientSecret string `json:"clientSecret" example:"EKxx..."`
	// IsSandbox indicates whether to use PayPal sandbox mode
	IsSandbox bool `json:"isSandbox" example:"false"`
}

// PaymentProviderInfo represents information about a payment provider
type PaymentProviderInfo struct {
	Provider    string `json:"provider"`    // Payment provider code (e.g., "stripe", "paypal")
	DisplayName string `json:"displayName"` // Display name for the provider
	Enabled     bool   `json:"enabled"`     // Whether the provider is enabled
}

// PaymentConfigInfoResponse represents payment configuration information returned to client
type PaymentConfigInfoResponse struct {
	Stripe *StripeConfigInfo `json:"stripe,omitempty"` // Stripe configuration
	PayPal *PayPalConfigInfo `json:"paypal,omitempty"` // PayPal configuration
}

// StripeConfigInfo represents Stripe configuration information that's safe to return to client
type StripeConfigInfo struct {
	PublicKey    string `json:"pk,omitempty"`    // Stripe public key
	SecretKey    string `json:"sk,omitempty"`    // Stripe secret key
	AccountEmail string `json:"email,omitempty"` // Account email
	IsSandbox    bool   `json:"isSandbox"`       // Stripe sandbox mode (always return)
}

// PayPalConfigInfo represents PayPal configuration information that's safe to return to client
type PayPalConfigInfo struct {
	ClientID     string `json:"clientId,omitempty"`     // PayPal client ID
	ClientSecret string `json:"clientSecret,omitempty"` // PayPal client secret
	IsSandbox    bool   `json:"isSandbox"`              // PayPal sandbox mode (always return)
}

// CoinPackageCreateRequest defines the request structure for creating a coin package
type CoinPackageCreateRequest struct {
	SiteID             string          `json:"siteId" binding:"required"`
	Name               string          `json:"name" binding:"required"`
	Description        string          `json:"description"`
	Features           json.RawMessage `json:"features,omitempty"` // Array of feature highlights, e.g. ["买100送20", "限时优惠"]
	CoinAmount         int             `json:"coinAmount" binding:"required,min=1"`
	Price              types.Money     `json:"price" binding:"required"`
	OriginalPrice      types.Money     `json:"originalPrice"`
	DiscountPercentage int             `json:"discountPercentage"`
	IOSProductID       string          `json:"iosProductId"`
}

// CoinPackageCreateResponse defines the response structure for creating a coin package
type CoinPackageCreateResponse struct {
	PackageID string `json:"packageId"`
}

// CoinPackageModifyRequest defines the request structure for modifying a coin package
type CoinPackageModifyRequest struct {
	PackageID   string          `json:"packageId" binding:"required"`
	SiteID      string          `json:"siteId" binding:"required"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Features     json.RawMessage `json:"features,omitempty"` // Array of feature highlights, e.g. ["买100送20", "限时优惠"]
	Status       int             `json:"status"`
	IOSProductID string          `json:"iosProductId"`
}

// CoinPackageModifyResponse defines the response structure for modifying a coin package
type CoinPackageModifyResponse struct {
	PackageID string `json:"packageId"`
}

// SubscriptionPackageResponse defines the response structure for subscription package data
type SubscriptionData struct {
	PackageID          string          `json:"packageId"`                    // 订阅套餐ID
	SiteID             string          `json:"siteId"`                       // 站点ID
	Name               string          `json:"name"`                         // 套餐名称
	Description        string          `json:"description"`                  // 套餐描述
	Interval           string          `json:"interval"`                     // 计费周期(weekly,monthly,yearly)
	Price              types.Money     `json:"price"`                        // 价格
	OriginalPrice      types.Money     `json:"originalPrice,omitempty"`      // 原价
	Currency           string          `json:"currency"`                     // 货币单位
	DiscountPercentage int             `json:"discountPercentage,omitempty"` // 折扣百分比
	Coins              int             `json:"coins"`                        // 赠送金币数量
	Rights             json.RawMessage `json:"rights"`                       // 订阅权益
	Status             int             `json:"status"`                       // 状态：1启用 2禁用
	IOSProductID       string          `json:"iosProductId"`                 // App Store 商品 ID
	CreatedAt          int64           `json:"createdAt"`                    // 创建时间
}

// // SubscriptionCreateRequest defines the request structure for creating a subscription package
// type SubscriptionCreateRequest struct {
// 	SiteID            string  `json:"siteId" binding:"required"` // 站点ID
// 	Name              string  `json:"name" binding:"required"`   // 套餐名称
// 	Description       string  `json:"description"`               // 套餐描述
// 	Interval          string  `json:"interval" binding:"required,oneof=week month year"` // 计费周期
// 	Price             float64 `json:"price" binding:"required,min=0.01"` // 价格
// 	OriginalPrice     float64 `json:"originalPrice,omitempty"`   // 原价
// 	DiscountPercentage int    `json:"discountPercentage,omitempty"` // 折扣百分比
// 	Currency          string  `json:"currency,omitempty"`        // 货币单位，默认USD
// }

// // SubscriptionCreateResponse defines the response structure for creating a subscription package
// type SubscriptionCreateResponse struct {
// 	PackageID string `json:"packageId"` // 创建的订阅套餐ID
// }

// // SubscriptionModifyRequest defines the request structure for modifying a subscription package
// type SubscriptionModifyRequest struct {
// 	PackageID         string  `json:"packageId" binding:"required"` // 订阅套餐ID
// 	SiteID            string  `json:"siteId" binding:"required"`    // 站点ID
// 	Name              string  `json:"name,omitempty"`               // 套餐名称
// 	Description       string  `json:"description,omitempty"`        // 套餐描述
// 	Status            int     `json:"status,omitempty"`             // 状态：1启用 2禁用
// }

// // SubscriptionModifyResponse defines the response structure for modifying a subscription package
// type SubscriptionModifyResponse struct {
// 	PackageID string `json:"packageId"` // 修改的订阅套餐ID
// }

// CoinPackageResponse represents a coin package in the response
type CoinPackageResponseData struct {
	PackageID          string          `json:"packageId"`
	Name               string          `json:"name"`
	Description        string          `json:"description"`
	Features           json.RawMessage `json:"features,omitempty"` // Array of feature highlights, e.g. ["买100送20", "限时优惠"]
	CoinAmount         int             `json:"coinAmount"`
	Price              types.Money     `json:"price"`
	OriginalPrice      types.Money     `json:"originalPrice,omitempty"`
	Currency           string          `json:"currency"`
	DiscountPercentage int             `json:"discountPercentage,omitempty"`
	Status             int             `json:"status"`
	IOSProductID       string          `json:"iosProductId"`
}

// UserCoinsResponse represents a user's coin account information in API responses
type UserCoinsResponse struct {
	Balance             int         `json:"balance"`
	TotalEarned         int         `json:"totalEarned"`
	TotalSpent          int         `json:"totalSpent"`
	TotalRealMoneySpent types.Money `json:"totalRealMoneySpent"`
}

// CoinTransactionResponse represents a coin transaction in API responses
type CoinTransactionResponse struct {
	TransactionID string `json:"transactionId"`
	Amount        int    `json:"amount"`
	BeforeBalance int    `json:"beforeBalance"`
	Balance       int    `json:"balance"`
	Source        string `json:"source"`
	// RelatedID      string                 `json:"relatedId,omitempty"`
	RelatedType int    `json:"relatedType,omitempty"`
	Description string `json:"description,omitempty"`
	// Snapshot       map[string]interface{} `json:"snapshot,omitempty"`
	CreatedAt int64 `json:"createdAt"`
}

// TransactionHistoryResponse represents the paginated transaction history response
type CoinTransactionHistoryResponse struct {
	Items    []*CoinTransactionResponse `json:"items"`
	Total    int64                      `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"pageSize"`
}

// VideoUnlockResponse represents a video unlock record in API responses
type VideoUnlockResponse struct {
	ContentID     string `json:"contentId"`               // 内容ID (视频或播放列表ID)
	ContentType   string `json:"contentType"`             // 内容类型 "video" or "playlist"
	PlaylistID    string `json:"playlistId,omitempty"`    // 播放列表ID (当contentType为视频时)
	CoinCost      int    `json:"coinCost"`                // 消费的金币数量
	TransactionID string `json:"transactionId"`           // 对应的交易ID
	ContentTitle  string `json:"contentTitle"`            // 内容标题
	PlaylistTitle string `json:"playlistTitle,omitempty"` // 播放列表标题
	EpisodeNumber int    `json:"episodeNumber,omitempty"` // 集数（如果是视频）
	UnlockedAt    int64  `json:"unlockedAt"`              // 解锁时间
	ExpiredAt     int64  `json:"expiredAt,omitempty"`     // 过期时间（如果有）
}

// VideoUnlockHistoryResponse represents the paginated video unlock history response
type VideoUnlockHistoryResponse struct {
	Items    []*VideoUnlockResponse `json:"items"`    // 解锁记录列表
	Total    int64                  `json:"total"`    // 总记录数
	Page     int                    `json:"page"`     // 当前页码
	PageSize int                    `json:"pageSize"` // 每页大小
}

// OrderType defines the type of order being created
type OrderType string

const (
	OrderTypeCoinPackage  OrderType = "coin"
	OrderTypeSubscription OrderType = "sub"
)

// MetaClickPayload carries Facebook click/browser ids for CAPI attribution.
type MetaClickPayload struct {
	Fbc            string `json:"fbc,omitempty"`
	Fbp            string `json:"fbp,omitempty"`
	Fbclid         string `json:"fbclid,omitempty"`
	EventSourceURL string `json:"eventSourceUrl,omitempty"`
}

// OrderCreateRequest defines the request structure for creating an order
type OrderCreateRequest struct {
	SiteID          string                 `json:"siteId" binding:"required"`
	OrderType       OrderType              `json:"orderType" binding:"required,oneof=coin sub"`
	PackageID       string                 `json:"packageId" binding:"required"`                         // ID of the coin package or subscription package
	PaymentMethod   string                 `json:"paymentMethod" binding:"required,oneof=stripe paypal"` // Payment method: stripe or paypal
	Currency        string                 `json:"currency,omitempty"`                                   // Optional override for currency
	ReturnURL       string                 `json:"returnUrl,omitempty"`                                  // URL to redirect after payment
	CancelURL       string                 `json:"cancelUrl,omitempty"`                                  // URL to redirect if canceled
	TrackingContext map[string]interface{} `json:"trackingContext,omitempty"`                            // Attribution and frontend context for backend analytics
	Meta            *MetaClickPayload      `json:"meta,omitempty"`
}

// OrderCreateResponse defines the response structure for creating an order
type OrderCreateResponse struct {
	OrderID       string `json:"orderId"`
	ClientSecret  string `json:"clientSecret,omitempty"` // For Stripe payment intents
	CheckoutURL   string `json:"checkoutUrl,omitempty"`  // URL for hosted checkout pages
	SuccessURL    string `json:"successUrl,omitempty"`   // URL to redirect after successful payment
	CancelURL     string `json:"cancelUrl,omitempty"`    // URL to redirect if payment is canceled
	PaymentStatus string `json:"paymentStatus"`
}

// BuyContentWithCoinsRequest defines the request to purchase content with coins
type BuyContentWithCoinsRequest struct {
	ContentID   string `json:"contentId" binding:"required"`                        // ID of content to purchase (video or playlist)
	ContentType string `json:"contentType" binding:"required,oneof=video playlist"` // Type of content (video or playlist)
	PlaylistID  string `json:"playlistId,omitempty"`                                // If content is a video in a playlist
}

// BuyContentWithCoinsResponse defines the response after purchasing content
type BuyContentWithCoinsResponse struct {
	TransactionID string `json:"transactionId"` // ID of the coin transaction
	CoinCost      int    `json:"coinCost"`      // Amount of coins spent
	Balance       int    `json:"balance"`       // New balance after purchase
}

// UserSubscriptionResponse represents the response for user subscription information
type UserSubscriptionResponse struct {
	SubscriptionID         string `json:"subscriptionId"`
	UserID                 string `json:"userId"`
	SiteID                 string `json:"siteId"`
	PackageID              string `json:"packageId"`
	Provider               string `json:"provider"`
	ProviderSubscriptionID string `json:"providerSubscriptionId"`
	ProviderCustomerID     string `json:"providerCustomerId"`
	Status                 int    `json:"status"`
	CurrentPeriodStart     int64  `json:"currentPeriodStart"`
	CurrentPeriodEnd       int64  `json:"currentPeriodEnd"`
	CancelAtPeriodEnd      bool   `json:"cancelAtPeriodEnd"`
	CreatedAt              int64  `json:"createdAt"`

	// Package details
	PackageName        string      `json:"packageName,omitempty"`
	PackageDescription string      `json:"packageDescription,omitempty"`
	Interval           string      `json:"interval,omitempty"`
	Price              types.Money `json:"price,omitempty"`
	Currency           string      `json:"currency,omitempty"`
}

// SubscriptionConfirmRequest confirms a subscription order after Stripe checkout
type SubscriptionConfirmRequest struct {
	OrderID string `json:"orderId" binding:"required"` // payment_transactions.transaction_id
}

// SubscriptionCancelRequest represents the request for cancelling a subscription
type SubscriptionCancelRequest struct {
	SubscriptionID    string `json:"subscriptionId" binding:"required"` // ID of the subscription to cancel
	CancelAtPeriodEnd bool   `json:"cancelAtPeriodEnd"`                 // If true, cancels at the end of the current period; if false, cancels immediately //todo
}

// CancelCustomerSubscriptionRequest represents the request for a creator to cancel a user's subscription
type CancelCustomerSubscriptionRequest struct {
	SiteID         string `json:"siteId" binding:"required"`
	SubscriptionID string `json:"subscriptionId" binding:"required"`
}

// Internal API for service-to-service communication

// InternalAddCoinsRequest defines request for adding coins (treated as coin package purchase)
type InternalAddCoinsRequest struct {
	CoinAmount    int    `json:"coinAmount" binding:"required,min=1"` // Number of coins to add
	Amount        int64  `json:"amount" binding:"required,min=0"`     // Actual money amount in cents
	Currency      string `json:"currency,omitempty"`                  // Currency code, e.g. "USD"
	Description   string `json:"description,omitempty"`               // Transaction description
	TransactionID string `json:"transactionId,omitempty"`             // External transaction ID
	Provider      string `json:"provider,omitempty"`                  // Payment provider name
}

// InternalAddCoinsResponse defines response for adding coins
type InternalAddCoinsResponse struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transactionId"` // Internal transaction ID
	Balance       int    `json:"balance"`       // New balance after adding coins
	Message       string `json:"message,omitempty"`
}

// InternalGetBalanceResponse defines response for getting user coin balance
type InternalGetBalanceResponse struct {
	Balance int `json:"balance" example:"1500"` // Current coin balance
}

// InternalDeductCoinsRequest defines request for deducting coins
type InternalDeductCoinsRequest struct {
	CoinAmount  int    `json:"coinAmount" binding:"required,min=1"` // Number of coins to deduct
	RelatedID   string `json:"relatedId,omitempty"`                 // ID of related content (e.g., video ID)
	Description string `json:"description,omitempty"`               // Transaction description
	Source      string `json:"source,omitempty"`                    // Transaction source, e.g., "plugin_deduct"
}

// InternalDeductCoinsResponse defines response for deducting coins
type InternalDeductCoinsResponse struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transactionId"` // Internal transaction ID
	Balance       int    `json:"balance"`       // New balance after deducting coins
	Message       string `json:"message,omitempty"`
}

// ClaimTaskRewardRequest defines request for claiming task reward
type ClaimTaskRewardRequest struct {
	TaskName string `json:"taskName" binding:"required"` // 任务名称，如 "daily_login", "watch_video" 等
}

// ClaimTaskRewardResponse defines response for claiming task reward
type ClaimTaskRewardResponse struct {
	Success bool   `json:"success"`           // 是否成功领取（true=首次完成，false=已领取过）
	Balance int    `json:"balance"`           // 当前金币余额
	Message string `json:"message,omitempty"` // 提示信息
}

// WheelPrizeItem defines a wheel segment for display
type WheelPrizeItem struct {
	ID    string `json:"id"`
	Coins int    `json:"coins"`
}

// WheelStatusResponse defines wheel availability and config
type WheelStatusResponse struct {
	Balance           int              `json:"balance"`
	FreeSpinAvailable bool             `json:"freeSpinAvailable"`
	NextFreeSpinAt    int64            `json:"nextFreeSpinAt"`
	PaidCostPerSpin   int              `json:"paidCostPerSpin"`
	PaidSpinsToday    int              `json:"paidSpinsToday"`
	PaidDailyCap      int              `json:"paidDailyCap"`
	FreePrizes        []WheelPrizeItem `json:"freePrizes"`
	PaidPrizes        []WheelPrizeItem `json:"paidPrizes"`
}

// WheelSpinRequest defines a wheel spin request
type WheelSpinRequest struct {
	Mode string `json:"mode" binding:"required,oneof=free paid"` // free | paid
}

// WheelSpinResultItem defines a single spin outcome
type WheelSpinResultItem struct {
	PrizeID string `json:"prizeId"`
	Coins   int    `json:"coins"`
	Index   int    `json:"index"`
}

// WheelSpinResponse defines wheel spin result
type WheelSpinResponse struct {
	Success  bool                  `json:"success"`
	Mode     string                `json:"mode"`
	Cost     int                   `json:"cost"`
	Results  []WheelSpinResultItem `json:"results"`
	Balance  int                   `json:"balance"`
	Message  string                `json:"message,omitempty"`
}

// PurchaseRecord represents a single purchase record in the purchase history
type PurchaseRecord struct {
	TransactionID   string      `json:"transactionId"`   // Transaction ID
	OrderID         string      `json:"orderId"`         // Order ID
	Amount          types.Money `json:"amount"`          // Purchase amount
	Currency        string      `json:"currency"`        // Currency code
	Provider        string      `json:"provider"`        // Payment provider (stripe, paypal, etc.)
	PaymentType     int         `json:"paymentType"`     // 1: subscription, 2: coin package
	PaymentTypeName string      `json:"paymentTypeName"` // Subscription or Coin Package
	Status          int         `json:"status"`          // Transaction status
	StatusName      string      `json:"statusName"`      // Status name
	RelatedID       string      `json:"relatedId"`       // Package ID
	RelatedType     int         `json:"relatedType"`     // 1: subscription package, 2: coin package
	// Package details from snapshot
	PackageName        string `json:"packageName,omitempty"`        // Package name
	PackageDescription string `json:"packageDescription,omitempty"` // Package description
	CoinAmount         int    `json:"coinAmount,omitempty"`         // Coin amount (for coin packages)
	Interval           string `json:"interval,omitempty"`           // Billing interval (for subscriptions)
	CreatedAt          int64  `json:"createdAt"`                    // Purchase timestamp
}

// PurchaseHistoryResponse represents the paginated purchase history response
type PurchaseHistoryResponse struct {
	Items    []*PurchaseRecord `json:"items"`    // Purchase record list
	Total    int64             `json:"total"`    // Total record count
	Page     int               `json:"page"`     // Current page number
	PageSize int               `json:"pageSize"` // Items per page
}
