package model

// SitePlaylists represents the base site_playlists table
type SitePlaylists struct {
	ID               uint   `gorm:"primaryKey;column:id"`
	SiteID           string `gorm:"column:site_id"`
	PlaylistID       string `gorm:"column:playlist_id"`
	Status           int    `gorm:"column:status"`
	AccessType       int    `gorm:"-"` // Virtual fieldUsing the alias name
	SingleVideoPrice int    `gorm:"-"` // Virtual fieldhe alias name
	FreeVideos       int    `gorm:"-"` // Virtual fieldumn:FreeVideos"` // Using the alias name
}

// SitePlaylistsView represents the joined view used in queries
type SitePlaylistsView struct {
	ID               uint   `gorm:"primaryKey;column:id"`
	SiteID           string `gorm:"column:site_id"`
	PlaylistID       string `gorm:"column:playlist_id"`
	Title            string `gorm:"column:title"`      // Assuming this is a field in the playlists table
	Slug             string `gorm:"column:slug"`       // Assuming this is a field in the playlists table
	OrderVids        string `gorm:"column:order_vids"` // Assuming this is a field in the playlists table
	Cover            string `gorm:"column:cover"`
	Status           int    `gorm:"column:status"`
	AccessType       int    `gorm:"column:access_type"`
	SingleVideoPrice int    `gorm:"column:single_video_price"`
	FreeVideos       int    `gorm:"column:free_videos"`
}

func (s *SitePlaylists) TableName() string {
	return "site_playlists"
}
