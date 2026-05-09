package model

import "time"

type CreatorGuides struct {
	CreatorID string    `gorm:"column:creator_id"`
	Guides    string    `gorm:"column:guides"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}
