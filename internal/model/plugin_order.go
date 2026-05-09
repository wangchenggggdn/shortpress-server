package model

import "time"

// PluginOrder 插件订单模型
type PluginOrder struct {
	ID             uint       `gorm:"column:id;primaryKey;autoIncrement"`
	OrderID        string     `gorm:"column:order_id;uniqueIndex;not null;comment:订单ID"`
	UserID         string     `gorm:"column:user_id;index;not null;comment:用户ID"`
	SiteID         string     `gorm:"column:site_id;index;not null;comment:站点ID"`
	PluginID       string     `gorm:"column:plugin_id;index;not null;comment:插件ID"`
	ThirdPartyID   string     `gorm:"column:third_party_id;index;comment:第三方订单ID（如充值平台订单号）"`
	ThirdPartyName string     `gorm:"column:third_party_name;comment:第三方平台名称（如微信、支付宝、Stripe等）"`
	Amount         int        `gorm:"column:amount;not null;comment:金额"`
	Type           string     `gorm:"column:type;not null;comment:类型:add 或 deduct"`
	Status         string     `gorm:"column:status;not null;default:pending;comment:状态:pending,completed,failed"`
	Reason         string     `gorm:"column:reason;comment:原因"`
	ProcessedAt    *time.Time `gorm:"column:processed_at;comment:处理时间"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime;comment:创建时间"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime;comment:更新时间"`
}

// TableName 指定表名
func (PluginOrder) TableName() string {
	return "plugin_orders"
}

// 订单状态常量
const (
	PluginOrderStatusPending   = "pending"   // 待处理
	PluginOrderStatusCompleted = "completed" // 已完成
	PluginOrderStatusFailed    = "failed"    // 失败
)
