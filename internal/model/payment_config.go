package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// JSONMap is used to store JSON format map data
type JSONMap map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONMap)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, j)
}

// PaymentConfig represents payment configuration information
type PaymentConfig struct {
	ID                 uint64     `json:"id" gorm:"primaryKey;column:id"`
	ConfigID           string     `json:"configId" gorm:"column:config_id"`
	SiteID             string     `json:"siteId" gorm:"column:site_id"`
	Provider           string     `json:"provider" gorm:"column:provider"`
	IsActive           bool       `json:"isActive" gorm:"column:is_active"`
	IsSandbox          bool       `json:"isSandbox" gorm:"column:is_sandbox"`
	StripePublicKey    string     `json:"stripePublicKey" gorm:"column:stripe_public_key"`
	StripeSecretKey    string     `json:"stripeSecretKey" gorm:"column:stripe_secret_key"`
	StripeAccountID    string     `json:"stripeAccountId" gorm:"column:stripe_account_id"`
	PaypalClientID     string     `json:"paypalClientId" gorm:"column:paypal_client_id"`
	PaypalClientSecret string     `json:"paypalClientSecret" gorm:"column:paypal_client_secret"`
	AccountInfo        JSONMap    `json:"accountInfo" gorm:"column:account_info;type:json"`
	LastVerifiedAt     *time.Time `json:"lastVerifiedAt" gorm:"column:last_verified_at"`
	ProviderWebhookID  string     `json:"providerWebhookId" gorm:"column:provider_webhook_id"`
	ProviderWebhookSK  string     `json:"providerWebhookSk" gorm:"column:provider_webhook_sk"`
	EndPointUrl        string     `json:"endpointUrl" gorm:"column:endpoint_url"`
	VerificationStatus int        `json:"verificationStatus" gorm:"column:verification_status"`
	ErrorMessage       string     `json:"errorMessage" gorm:"column:error_message"`
	CreatedAt          time.Time  `json:"createdAt" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt          time.Time  `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

// TableName specifies the table name
func (PaymentConfig) TableName() string {
	return "payment_configs"
}
