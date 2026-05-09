package plugins

import (
	"context"
	"shortpress-server/internal/model"
	"shortpress-server/internal/repository/db"
	"time"

	"gorm.io/gorm"
)

// PluginOrderRepository 插件订单仓库接口
type PluginOrderRepository interface {
	db.BaseOperation
	// GetByOrderID 根据 order_id 获取订单
	GetByOrderID(ctx context.Context, orderID string) (*model.PluginOrder, error)
	// UpdateStatus 更新订单状态
	UpdateStatus(ctx context.Context, orderID string, status string) error
	// MarkAsProcessed 标记订单为已处理
	MarkAsProcessed(ctx context.Context, orderID string) error
	// GetPendingByOrderID 获取待处理的订单
	GetPendingByOrderID(ctx context.Context, orderID string) (*model.PluginOrder, error)
	// CheckRecentAddOrder 检查是否存在近期的相同充值订单（防重放）
	// 只针对充值（add）类型，消费（deduct）不需要防重放
	CheckRecentAddOrder(ctx context.Context, userID, siteID string, amount int, timeWindow time.Duration) (*model.PluginOrder, error)
}

type pluginOrderRepository struct {
	*db.Repository
}

// NewPluginOrderRepository 创建插件订单仓库
func NewPluginOrderRepository(r *db.Repository) PluginOrderRepository {
	return &pluginOrderRepository{
		Repository: r,
	}
}

func (r *pluginOrderRepository) Create(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Create(entity).Error
}

func (r *pluginOrderRepository) Update(ctx context.Context, entity interface{}) error {
	return r.DB(ctx).Save(entity).Error
}

func (r *pluginOrderRepository) GetByOrderID(ctx context.Context, orderID string) (*model.PluginOrder, error) {
	var order model.PluginOrder
	err := r.DB(ctx).Where("order_id = ?", orderID).First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

func (r *pluginOrderRepository) UpdateStatus(ctx context.Context, orderID string, status string) error {
	return r.DB(ctx).Model(&model.PluginOrder{}).
		Where("order_id = ?", orderID).
		Update("status", status).Error
}

func (r *pluginOrderRepository) MarkAsProcessed(ctx context.Context, orderID string) error {
	return r.DB(ctx).Model(&model.PluginOrder{}).
		Where("order_id = ?", orderID).
		Updates(map[string]interface{}{
			"status":       model.PluginOrderStatusCompleted,
			"processed_at": time.Now(),
		}).Error
}

func (r *pluginOrderRepository) GetPendingByOrderID(ctx context.Context, orderID string) (*model.PluginOrder, error) {
	var order model.PluginOrder
	err := r.DB(ctx).Where("order_id = ? AND status = ?", orderID, model.PluginOrderStatusPending).First(&order).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// CheckRecentAddOrder 检查是否存在近期的相同充值订单（防重放）
// 只针对充值（add）类型，消费（deduct）不需要防重放
func (r *pluginOrderRepository) CheckRecentAddOrder(ctx context.Context, userID, siteID string, amount int, timeWindow time.Duration) (*model.PluginOrder, error) {
	var order model.PluginOrder

	// 查询在时间窗口内，同一用户、同一站点、相同金额的充值订单（type='add'）
	err := r.DB(ctx).
		Where("user_id = ? AND site_id = ? AND amount = ? AND type = ? AND status IN (?)", userID, siteID, amount, "add", []string{
			model.PluginOrderStatusPending,
			model.PluginOrderStatusCompleted,
		}).
		Where("created_at >= ?", time.Now().Add(-timeWindow)).
		Order("created_at DESC").
		First(&order).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &order, nil
}
