package model

import (
	"shortpress-server/internal/types"
	"time"
)

// Playlist defines playlist information
type Playlist struct {
	ID               uint            `gorm:"primaryKey;column:id"`
	PlaylistID       string          `gorm:"column:playlist_id"`
	CreatorID        string          `gorm:"column:creator_id"`
	Title            string          `gorm:"column:title"`
	Slug             string          `gorm:"column:slug"` // Slug for SEO-friendly URLs
	Description      string          `gorm:"column:description"`
	Tags             string          `gorm:"column:tags"`
	Cover            *types.ImageURL `gorm:"column:cover"`
	Status           int             `gorm:"column:status"`
	VideoCount       int             `gorm:"column:video_count"`
	OrderVids        string          `gorm:"column:order_vids"`
	Version          int             `gorm:"column:version"`
	SingleVideoPrice int             `gorm:"column:single_video_price;default:0"` // 单个视频解锁所需金币
	FreeVideos       *int            `gorm:"column:free_videos"`                  // 免费视频数量(从开始计算)
	AccessType       int             `gorm:"column:access_type;default:1"`        // 访问类型 1:免费 2:付费 3:会员专属
	UtmSource        string          `gorm:"column:utm_source"`                   // UTM 来源标识
	CreatedAt        time.Time       `gorm:"column:created_at"`
	UpdatedAt        time.Time       `gorm:"column:updated_at"`
}

func (p *Playlist) TableName() string {
	return "playlists"
}

type PlaylistView struct {
	ID          uint            `gorm:"primaryKey;column:id"`
	PlaylistID  string          `gorm:"column:playlist_id"`
	CreatorID   string          `gorm:"column:creator_id"`
	Title       string          `gorm:"column:title"`
	Description string          `gorm:"column:description"`
	Tags        string          `gorm:"column:tags"`
	Cover       *types.ImageURL `gorm:"column:cover"`
	Status      int             `gorm:"column:status"`
	VideoCount  int             `gorm:"column:video_count"`
	CreatedAt   time.Time       `gorm:"column:created_at"`
	UpdatedAt   time.Time       `gorm:"column:updated_at"`
}

type PlaylistQuery struct {
	CreatorID     string
	SiteID        string
	ExcludeSiteId string
	Status        *int
	Keyword       string
	UtmSource     string // UTM 来源过滤
	FilterUtm     bool   // 是否需要强制进行 UTM 过滤
}

// Define video list sorting method constants (0:Create time descending 1:Name sorting)
const (
	PlaylistSortByCreatedAtDesc = 0 // Sort by creation time descending
	PlaylistSortByTitleAsc      = 1 // Sort by creation time ascending
	PlaylistSortByCreatedAtAsc  = 2 // Sort by creation time ascending
	PlaylistSortByTitleDesc     = 3 // Sort by creation time descending
)
