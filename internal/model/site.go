package model

import (
	"encoding/json"
	"shortpress-server/internal/types"
)

// Site defines site basic information (corresponding to sites table), and accompanying SEO information (corresponding to site_seo table)
type Site struct {
	ID                uint            `gorm:"primaryKey;column:id"`
	SiteID            string          `gorm:"column:site_id"`
	Domain            string          `gorm:"column:domain"`
	Redirect          bool            `gorm:"column:redirect"`
	Path              string          `gorm:"column:path"`
	Name              string          `gorm:"column:name"`
	Logo              *types.ImageURL `gorm:"column:logo"`
	Status            int             `gorm:"column:status"`
	TemplateID        *string         `gorm:"column:template_id"`
	I18n              json.RawMessage `gorm:"column:i18n"` // 多语言配置，JSON格式，不做处理，直接存储
	GoogleAnalyticsID *string         `gorm:"column:google_analytics_id"`
	FacebookPixelID   *string         `gorm:"column:facebook_pixel_id"`
	ThinkingDataAppId *string         `gorm:"column:thinking_data_id"`
	Theme             int             `gorm:"column:theme;default:0"` // accent palette index for visitor UI
	SeoTitle          string          `gorm:"-"`
	SeoDescription    string          `gorm:"-"`
	SeoKeywords       string          `gorm:"-"`
	SeoI18n           json.RawMessage `gorm:"-"`
	TemplateName      *string         `gorm:"-"` // Virtual field: template name (from site_page_templates)
}

func (s *Site) TableName() string {
	return "sites"
}
