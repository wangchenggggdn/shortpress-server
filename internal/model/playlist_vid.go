package model

import "time"

type PlaylistVid struct {
	ID         uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	PlaylistID string    `gorm:"column:playlist_id;not null;size:36"`
	VID        string    `gorm:"column:vid;not null;size:36"`
	SortOrder  int       `gorm:"column:order;not null;default:0"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (p *PlaylistVid) TableName() string {
	return "playlist_vid"
}
