package model

import (
	"encoding/json"
	"time"
)

// SitePageTemplate maps to table site_page_templates
type SitePageTemplate struct {
	ID           uint            `gorm:"primaryKey;column:id"`
	TemplateID   string          `gorm:"column:template_id"`
	Name         string          `gorm:"column:name"`
	Description  *string         `gorm:"column:description"`
	Cover        *string         `gorm:"column:cover"`
	Status       int8            `gorm:"column:status"`
	TemplateData json.RawMessage `gorm:"column:template_data;type:json"`
	CreatedAt    time.Time       `gorm:"column:created_at"`
	UpdatedAt    time.Time       `gorm:"column:updated_at"`
}

func (s *SitePageTemplate) TableName() string {
	return "site_page_templates"
}
