package api

import "shortpress-server/internal/types"

type IncomeTransactionsRequest struct {
	SiteID    string `json:"siteId" binding:"required"`
	UserID    string `json:"userId"`    // Optional user ID for filtering
	UserEmail string `json:"userEmail"` // Optional email filter (matches account email or payment email)
	StartTime int64  `json:"startTime"` // Optional start timestamp
	EndTime   int64  `json:"endTime"`   // Optional end timestamp
	Page      int    `json:"page"`      // Optional page number
	PageSize  int    `json:"pageSize"`  // Optional page size
}

type IncomeTransactionItem struct {
	TransactionID string      `json:"transactionId"`
	Email         string      `json:"email"`
	PayerEmail    string      `json:"payerEmail"`
	PixelID       string      `json:"pixelId"`
	Platform      string      `json:"platform"`
	Name          string      `json:"name"`
	Amount        types.Money `json:"amount"`
	Provider      string      `json:"provider"`
	Description   string      `json:"description,omitempty"`
	CreatedAt     int64       `json:"createdAt"`
}

type IncomeTransactionHistoryResponse struct {
	Items    []*IncomeTransactionItem `json:"items"`
	Total    int64                    `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"pageSize"`
}

type IncomeStatisticsRequest struct {
	SiteID    string `json:"siteId" binding:"required"`
	StartTime int64  `json:"startTime"` // Optional start timestamp
	EndTime   int64  `json:"endTime"`   // Optional end timestamp
	// TimezoneOffset is minutes east of UTC used to bucket daily revenue
	// (e.g. UTC+8 => 480, UTC-7 => -420). Omit / 0 means UTC.
	TimezoneOffset *int `json:"timezoneOffset"`
}

type DailyIncomeStatistics struct {
	Date               string      `json:"date"`               // Date in YYYY-MM-DD format
	TotalAmount        types.Money `json:"totalAmount"`        // Total income for the day
	TransactionCount   int         `json:"transactionCount"`   // Number of successful transactions
	IapAmount          types.Money `json:"iapAmount"`          // Coin package / in-app purchase revenue
	SubscriptionAmount types.Money `json:"subscriptionAmount"` // New subscription revenue
	RenewalAmount      types.Money `json:"renewalAmount"`      // Subscription renewal revenue (provider_payment_id prefix in_)
}

type IncomeStatisticsResponse struct {
	Items []DailyIncomeStatistics `json:"items"` // Daily statistics records
}

// IncomeTransactionDetailResponse represents detailed information about a payment transaction
type IncomeTransactionDetailResponse struct {
	TransactionID       string      `json:"transactionId"`
	Name                string      `json:"name"`
	UserID              string      `json:"userId"`
	Email               string      `json:"email"`
	PayerEmail          string      `json:"payerEmail"`
	Platform            string      `json:"platform"`
	Amount              types.Money `json:"amount"`
	Currency            string      `json:"currency"`
	Provider            string      `json:"provider"`
	PaymentType         int         `json:"paymentType"`
	Status              int         `json:"status"`
	RelatedID           string      `json:"relatedId,omitempty"`
	RelatedType         int         `json:"relatedType,omitempty"`
	CreatedAt           int64       `json:"createdAt"`
	IsSubscriptionOrder bool        `json:"isSubscriptionOrder"`
	SubscriptionID      string      `json:"subscriptionId,omitempty"`
}

type CreationsRequest struct {
	SiteID   string `json:"siteId" binding:"required"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

type CreationVideo struct {
	URL      string `json:"url"`
	CoverURL string `json:"coverUrl,omitempty"`
}

type CreationRecordItem struct {
	TaskID          string          `json:"taskId"`
	Status          int32           `json:"status"`
	Model           string          `json:"model"`
	VideoID         string          `json:"videoId,omitempty"`
	SiteID          string          `json:"siteId,omitempty"`
	UserID          string          `json:"userId,omitempty"`
	Prompt          string          `json:"prompt,omitempty"`
	ReferenceImages []string        `json:"referenceImages,omitempty"`
	Videos          []CreationVideo `json:"videos,omitempty"`
	Images          []string        `json:"images,omitempty"`
	ErrorMsg        string          `json:"errorMsg,omitempty"`
	CreatedAt       int64           `json:"createdAt"`
	UpdatedAt       int64           `json:"updatedAt"`
}

type CreationsResponse struct {
	Items    []*CreationRecordItem `json:"items"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}
