package model

import (
	"shortpress-server/internal/types"
	"time"
)

// UserPlayRecord 用户播放记录
type UserPlayRecord struct {
	ID            int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"` // 自增ID
	UserID        string          `gorm:"column:user_id" json:"userId"`                 // 用户ID
	SiteID        string          `gorm:"column:site_id" json:"siteId"`                 // 站点ID
	VID           string          `gorm:"column:vid" json:"vid"`                        // 视频ID
	PlaylistID    string          `gorm:"column:playlist_id" json:"playlistId"`         // 播放列表ID
	EpisodeNumber *int            `gorm:"column:episode_number" json:"episodeNumber"`   // 视频在播放列表中的集数
	Progress      int             `gorm:"column:progress" json:"progress"`              // 播放进度（百分比）
	Duration      *int            `gorm:"column:duration" json:"duration"`              // 视频总时长（秒）
	Cover         *types.ImageURL `gorm:"column:cover" json:"cover"`                    // 视频封面图
	VideoTitle    string          `gorm:"column:video_title" json:"videoTitle"`         // 视频标题
	PlaylistTitle string          `gorm:"column:playlist_title" json:"playlistTitle"`   // 播放列表标题
	LastPlayedAt  time.Time       `gorm:"column:last_played_at" json:"lastPlayedAt"`    // 最近一次播放时间
	CreatedAt     time.Time       `gorm:"column:created_at" json:"createdAt"`           // 创建时间
	UpdatedAt     time.Time       `gorm:"column:updated_at" json:"updatedAt"`           // 更新时间
}

// TableName specifies the table name for UserPlayRecord model
func (UserPlayRecord) TableName() string {
	return "user_play_records"
}
