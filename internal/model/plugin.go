package model

import "time"

// Plugin 插件注册表（全局插件库）
type Plugin struct {
	ID          uint      `gorm:"column:id;primaryKey;autoIncrement"`
	PluginID    string    `gorm:"column:plugin_id;uniqueIndex;not null;comment:插件ID（唯一标识）"`
	Name        string    `gorm:"column:name;not null;comment:插件名称"`
	Description string    `gorm:"column:description;comment:插件描述"`
	Type        string    `gorm:"column:type;not null;comment:插件类型：payment, content, analytics 等"`
	Version     string    `gorm:"column:version;comment:插件版本"`
	Author      string    `gorm:"column:author;comment:插件作者"`
	PageURL     string    `gorm:"column:page_url;comment:插件页面URL（用于 iframe 展示）"`
	Status      string    `gorm:"column:status;not null;default:active;comment:状态:active, disabled, deprecated"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;comment:创建时间"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime;comment:更新时间"`
}

// TableName 指定表名
func (Plugin) TableName() string {
	return "plugins"
}

// 插件状态常量
const (
	PluginStatusActive     = "active"     // 活跃
	PluginStatusDisabled   = "disabled"   // 已禁用
	PluginStatusDeprecated = "deprecated" // 已弃用
)

// PluginWithInfo 插件信息（包含插件本身和站点安装信息）
type PluginWithInfo struct {
	Plugin     *Plugin
	SitePlugin *SitePlugin
}
