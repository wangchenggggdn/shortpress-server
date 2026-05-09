package model

import "encoding/json"

// SiteSeo defines site SEO information (corresponding to site_seo table)
type SiteSeo struct {
	SiteID      string          `gorm:"column:site_id"`
	Title       string          `gorm:"column:title"`
	Description string          `gorm:"column:description"`
	Keywords    string          `gorm:"column:keywords"`
	I18n        json.RawMessage `gorm:"column:i18n"`
}

func (s *SiteSeo) TableName() string {
	return "site_seo"
}
