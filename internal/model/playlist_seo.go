package model

type PlaylistSeo struct {
	ID          uint   `gorm:"primaryKey;column:id"`
	PlaylistID  string `gorm:"column:playlist_id"`
	Title       string `gorm:"column:title"`
	Description string `gorm:"column:description"`
	Keywords    string `gorm:"column:keywords"`
}

func (p *PlaylistSeo) TableName() string {
	return "playlist_seo"
}
