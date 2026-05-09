package model

import "time"

type CreatorProfile struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement;column:id"`
	CreatorID string    `gorm:"column:creator_id"`
	Nickname  string    `gorm:"column:nickname"`
	AvatarURL string    `gorm:"column:avatar_url"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (c *CreatorProfile) TableName() string {
	return "creator_profiles"
}
