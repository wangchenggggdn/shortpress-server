package model

import (
	"encoding/json"
)

type SitePageConfig struct {
	ID     uint            `gorm:"primaryKey;column:id"`
	SiteID string          `gorm:"column:site_id"`
	Type   string          `gorm:"column:type"`
	Config json.RawMessage `gorm:"column:config;type:json"`
}

// TableName specifies the table name for GORM
func (SitePageConfig) TableName() string {
	return "site_page_config"
}
