package model

import "time"

// ContentUnlock represents a content unlock record in the system
type ContentUnlock struct {
	ID            uint       `gorm:"primaryKey;column:id"`
	UserID        string     `gorm:"column:user_id;index"`
	ContentType   string     `gorm:"column:content_type"` // "video" or "playlist"
	ContentID     string     `gorm:"column:content_id"`
	PlaylistID    string     `gorm:"column:playlist_id"`
	CoinCost      int        `gorm:"column:coin_cost"`
	TransactionID string     `gorm:"column:transaction_id"`
	ContentTitle  string     `gorm:"column:content_title"`  // Title of the content
	PlaylistTitle string     `gorm:"column:playlist_title"` // Title of the playlist
	EpisodeNumber int        `gorm:"column:episode_number"` // Episode number for video content
	ExpiredAt     *time.Time `gorm:"column:expired_at"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (ContentUnlock) TableName() string {
	return "content_unlocks"
}

// ContentUnlockParams encapsulates parameters for creating a content unlock record
type ContentUnlockParams struct {
	UserID        string
	ContentType   string // "video" or "playlist"
	ContentID     string
	PlaylistID    string
	TransactionID string
	CoinCost      int
	ContentTitle  string // Title of the content
	PlaylistTitle string // Title of the playlist
	EpisodeNumber int    // Episode number for video content
	ExpiredAt     *time.Time
}
