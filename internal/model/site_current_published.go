package model

import (
	"encoding/json"
	"time"
)

// SiteCurrentPublished represents the current published site data table
type SiteCurrentPublished struct {
	ID            uint            `gorm:"primaryKey;column:id"`
	SiteID        string          `gorm:"column:site_id"`
	VersionNumber int             `gorm:"column:version_number"`
	PublishedData json.RawMessage `gorm:"column:published_data;type:json"`
	PublishedBy   *string         `gorm:"column:published_by"`
	PublishedAt   time.Time       `gorm:"column:published_at"`
	CreatedAt     time.Time       `gorm:"column:created_at"`
}

func (s *SiteCurrentPublished) TableName() string {
	return "site_current_published"
}
