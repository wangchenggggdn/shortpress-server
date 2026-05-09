package model

import (
	"time"
)

// WebhookEvent represents a webhook event received from a payment provider
type WebhookEvent struct {
	EventID      string     `gorm:"column:event_id;uniqueIndex"`
	WebhookID    string     `gorm:"column:webhook_id;index"`
	Provider     string     `gorm:"column:provider;index"` // stripe, paypal
	EventType    string     `gorm:"column:event_type;index"`
	Payload      string     `gorm:"column:payload;type:json"`
	Result       int        `gorm:"column:result"` // 1:success 2:failed
	ErrorMessage string     `gorm:"column:error_message"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime"`
	ProcessedAt  *time.Time `gorm:"column:processed_at"`
}

func (WebhookEvent) TableName() string {
	return "webhook_events"
}
