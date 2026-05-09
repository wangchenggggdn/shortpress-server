package model

import "time"

// SitePlugin 网站插件安装表（插件在站点中的安装）
type SitePlugin struct {
	ID          uint      `gorm:"column:id;primaryKey;autoIncrement"`
	SiteID      string    `gorm:"column:site_id;index;not null;comment:站点ID"`
	PluginID    string    `gorm:"column:plugin_id;index;not null;comment:插件ID"`
	Secret      string    `gorm:"column:secret;not null;comment:插件密钥（安装时生成）"`
	Status      string    `gorm:"column:status;not null;default:active;comment:状态:active, disabled"`
	Config      string    `gorm:"column:config;type:text;comment:插件配置（JSON格式）"`
	InstalledAt time.Time `gorm:"column:installed_at;autoCreateTime;comment:安装时间"`
	LastUsedAt  time.Time `gorm:"column:last_used_at;comment:最后使用时间"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;comment:创建时间"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime;comment:更新时间"`
}

// TableName 指定表名
func (SitePlugin) TableName() string {
	return "site_plugins"
}
