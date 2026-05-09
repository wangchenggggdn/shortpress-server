package model

import (
	"time"
)

// PlaylistI18n defines playlist i18n information
type PlaylistI18n struct {
	ID             uint      `gorm:"primaryKey;column:id"`
	PlaylistID     string    `gorm:"column:playlist_id"`
	Language       string    `gorm:"column:language"`
	Title          string    `gorm:"column:title"`
	Slug           string    `gorm:"column:slug"` // Slug for SEO-friendly URLs
	Description    string    `gorm:"column:description"`
	Tags           string    `gorm:"column:tags"`
	SeoTitle       string    `gorm:"column:seo_title"`
	SeoDescription string    `gorm:"column:seo_description"`
	SeoKeywords    string    `gorm:"column:seo_keywords"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (p *PlaylistI18n) TableName() string {
	return "playlist_i18n"
}
