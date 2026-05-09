package model

import (
	"encoding/json"
	"time"
)

// SiteBuilderData represents the site builder data table (draft/editing state)
type SiteBuilderData struct {
	ID                   uint            `gorm:"primaryKey;column:id"`
	SiteID               string          `gorm:"column:site_id"`
	SiteData             json.RawMessage `gorm:"column:site_data;type:json"`
	VersionNumber        int             `gorm:"column:version_number"`
	LastPublishedVersion *int            `gorm:"column:last_published_version"`
	CreatedBy            *string         `gorm:"column:created_by"`
	CreatedAt            time.Time       `gorm:"column:created_at"`
	UpdatedAt            time.Time       `gorm:"column:updated_at"`
}

func (s *SiteBuilderData) TableName() string {
	return "site_builder_data"
}
