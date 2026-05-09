package model

import (
	"encoding/json"
	"time"
)

// SitePublishedHistory represents the site published history table
type SitePublishedHistory struct {
	ID            uint            `gorm:"primaryKey;column:id"`
	SiteID        string          `gorm:"column:site_id"`
	VersionNumber int             `gorm:"column:version_number"`
	PublishedData json.RawMessage `gorm:"column:published_data;type:json"`
	PublishedBy   *string         `gorm:"column:published_by"`
	PublishedAt   time.Time       `gorm:"column:published_at"`
	CreatedAt     time.Time       `gorm:"column:created_at"`
}

func (s *SitePublishedHistory) TableName() string {
	return "site_published_history"
}
