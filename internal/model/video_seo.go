package model

import "time"

type VideoSeo struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	VID         string    `gorm:"column:vid;unique;not null"`
	Title       string    `gorm:"column:title"`
	Description string    `gorm:"column:description;type:text"`
	Keywords    string    `gorm:"column:keywords"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (v *VideoSeo) TableName() string {
	return "video_seo"
}
