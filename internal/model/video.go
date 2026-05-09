package model

import (
	"encoding/json"
	"shortpress-server/internal/types"
	"time"
)

// Video defines video information
type Video struct {
	ID          uint64                `gorm:"column:id;primaryKey;autoIncrement"`
	VID         string                `gorm:"column:vid;unique;not null"`
	Tags        string                `gorm:"column:tags;size:1024"`
	Title       string                `gorm:"column:title;size:1024;not null"`
	Description string                `gorm:"column:description;type:text"`
	Cover       *types.ImageURL       `gorm:"column:cover;size:255"`
	Subtitles   *types.SubtitleTracks `gorm:"column:subtitles;type:text"` // Subtitle tracks for the video
	Config      json.RawMessage       `gorm:"column:config;type:json"`    // Custom configuration (arbitrary JSON)
	Status      int8                  `gorm:"column:status;not null"`     // 1:UnPublishded 2:Published 3:Unlisted 4:Private 5:Deleted

	// The following fields are no longer stored in videos table after schema change,
	// but we keep them here for downstream compatibility. They will be populated
	// by joining primary source from video_sources.
	// VideoPath    *types.VideoUrl    `gorm:"-"`
	// Duration     int64     `gorm:"-"`
	// Width        int32       `gorm:"-"`          // Video width
	// Height       int32       `gorm:"-"`         // Video height
	// FileSize     int64     `gorm:"-"`
	CreatorID string    `gorm:"column:creator_id;size:36;not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (v *Video) TableName() string {
	return "videos"
}

// VideoCore maps to the columns that really exist in the videos table after schema change
type VideoCore struct {
	ID          uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	VID         string          `gorm:"column:vid;unique;not null"`
	Tags        string          `gorm:"column:tags;size:1024"`
	Title       string          `gorm:"column:title;size:1024;not null"`
	Description string          `gorm:"column:description;type:text"`
	Cover       *types.ImageURL `gorm:"column:cover;size:255"`
	Config      json.RawMessage `gorm:"column:config;type:json"` // Custom configuration (arbitrary JSON)
	Status      int8            `gorm:"column:status;not null"`
	CreatorID   string          `gorm:"column:creator_id;size:36;not null"`
	CreatedAt   time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time       `gorm:"column:updated_at;autoUpdateTime"`
}

func (v *VideoCore) TableName() string { return "videos" }

// VideoSource maps to video_sources table
type VideoSource struct {
	ID           uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	SourceID     string          `gorm:"column:source_id;unique;not null"`
	VID          string          `gorm:"column:vid;not null"`
	Provider     string          `gorm:"column:provider;size:50"`
	SourceType   int8            `gorm:"column:source_type;not null"`
	URL          *string         `gorm:"column:url;size:2048"`
	UploadStatus int8            `gorm:"column:upload_status;not null"`
	LocalPath    *types.VideoUrl `gorm:"column:local_path;size:1024"` //-- 本地文件使用
	Duration     int64           `gorm:"column:duration"`
	Width        int32           `gorm:"column:width"`
	Height       int32           `gorm:"column:height"`
	Priority     int             `gorm:"column:priority;not null"` //'选择优先级（数字越小越优先）',
	Status       int8            `gorm:"column:status;not null"`   //'1:启用 2:禁用 ',
	CreatedAt    time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time       `gorm:"column:updated_at;autoUpdateTime"`
}

func (vs *VideoSource) TableName() string { return "video_sources" }

type VideoSView struct {
	ID          uint64          `gorm:"column:id;primaryKey;autoIncrement"`
	VID         string          `gorm:"column:vid;unique;not null"`
	Title       string          `gorm:"column:title;size:1024;not null"`
	Description string          `gorm:"column:description;type:text"`
	Cover       *types.ImageURL `gorm:"column:cover;size:255"`
	Status      int8            `gorm:"column:status;not null"` // 0:Unpublished 1:Published 2:Offline 3:Deleted
	VideoPath   *types.VideoUrl `gorm:"column:video_path;size:1024"`
	SortOrder   int             `gorm:"column:sort_order;not null"`
	CreatorID   string          `gorm:"column:creator_id;size:36;not null"`
	CreatedAt   time.Time       `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time       `gorm:"column:updated_at;autoUpdateTime"`
}

// VideoQuery defines query parameters structure
type VideoQuery struct {
	SiteID            string
	CreatorID         string   // Creator ID
	PlaylistID        string   // Playlist ID
	ExcludePlaylistID string   // Exclude Playlist ID
	ExcludeVids       []string // Exclude Playlist ID
	Status            *int     // Video status, -1 means no restriction
	UploadStatus      *int     // Upload status, -1 means no restriction
	KeyWord           string   // Title fuzzy search
}
