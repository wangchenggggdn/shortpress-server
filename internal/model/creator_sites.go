package model

import "time"

// CreatorWebsite defines user site association information (corresponding to creator_websites table)
type CreatorSites struct {
	ID        uint      `gorm:"primaryKey;column:id"`
	CreatorID string    `gorm:"column:creator_id"`
	SiteID    string    `gorm:"column:site_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (c *CreatorSites) TableName() string {
	return "creator_sites"
}
