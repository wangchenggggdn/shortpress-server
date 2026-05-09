package payment

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

// WebhookEventRepository defines the repository interface for webhook events
type WebhookEventRepository interface {
	db.BaseOperation
	GetByEventID(ctx context.Context, eventID string) (*model.WebhookEvent, error)
}

type webhookEventRepository struct {
	*db.Repository
}

// NewWebhookEventRepository creates a new webhook event repository
func NewWebhookEventRepository(r *db.Repository) WebhookEventRepository {
	return &webhookEventRepository{
		Repository: r,
	}
}

func (r *webhookEventRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *webhookEventRepository) Update(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Save(entity).Error
}

func (r *webhookEventRepository) GetByEventID(ctx context.Context, eventID string) (*model.WebhookEvent, error) {
	var event model.WebhookEvent
	err := r.DB(ctx).Where("event_id = ?", eventID).First(&event).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &event, nil
}
