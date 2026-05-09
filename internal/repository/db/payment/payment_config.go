package payment

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"

	"gorm.io/gorm"
)

// Repository 定义支付配置存储库接口
type PaymentConfigRepository interface {
	db.BaseOperation
	GetByConfigID(ctx context.Context, configID string) (*model.PaymentConfig, error)
	GetBySiteIDAndProvider(ctx context.Context, siteID, provider string) (*model.PaymentConfig, error)
}

type paymentConfigRepository struct {
	*db.Repository
}

func NewPaymentConfgRepository(r *db.Repository) PaymentConfigRepository {
	return &paymentConfigRepository{
		Repository: r,
	}
}

func (r *paymentConfigRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *paymentConfigRepository) Update(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Save(entity).Error
}

func (r *paymentConfigRepository) GetByConfigID(ctx context.Context, configID string) (*model.PaymentConfig, error) {
	var config model.PaymentConfig
	err := r.DB(ctx).Where("config_id = ?", configID).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

func (r *paymentConfigRepository) GetBySiteIDAndProvider(ctx context.Context, siteID, provider string) (*model.PaymentConfig, error) {
	var config model.PaymentConfig
	err := r.DB(ctx).Where("site_id = ? AND provider = ?", siteID, provider).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}
