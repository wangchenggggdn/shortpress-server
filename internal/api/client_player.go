package api

import "shortpress-server/internal/types"

type FeedResponse struct {
	Response
	Data FeedResponseData `json:"data"`
}

type FeedResponseData struct {
	Page    int         `json:"page"`
	HasMore bool        `json:"hasMore"`
	Items   []*FeedItem `json:"items"`
}

type FeedItem struct {
	VID        string `json:"vid"`        // Unique video identifier
	PlaylistID string `json:"playlistId"` // Playlist ID to which the video belongs
	Episode    int    `json:"episode"`
}

type AnonRegisterRequest struct {
	SitePath string `json:"sitePath" example:"abc"`     // Site path
	Host     string `json:"host" example:"example.com"` // Host
}

type AnonRegisterResponse struct {
	Response
	Data RegisterResponseData `json:"data"`
}

type RegisterResponseData struct {
	Token  string `json:"accessToken"`
	SiteID string `json:"siteId"`
	Ver    string `json:"ver"`
}

// PlaybackReportRequest represents the request parameters for reporting video playback records
type PlaybackReportRequest struct {
	VID           string          `json:"vid" binding:"required"`      // Video ID
	PlaylistID    string          `json:"playlistId"`                  // Playlist ID
	EpisodeNumber *int            `json:"episodeNumber"`               // Episode number of the video in the playlist
	Progress      int             `json:"progress" binding:"required"` // Playback progress (seconds)
	Duration      *int            `json:"duration"`                    // Total video duration (seconds)
	Cover         *types.ImageURL `json:"cover"`                       // Video cover image URL
	VideoTitle    string          `json:"videoTitle"`                  // Video title
	PlaylistTitle string          `json:"playlistTitle"`               // Playlist title
}

// VideoHistoryItem represents a single video history entry
type VideoHistoryItem struct {
	VID           string          `json:"vid"`                     // 视频ID
	Title         string          `json:"title"`                   // 视频标题
	PlaylistID    string          `json:"playlistId,omitempty"`    // 播放列表ID
	PlaylistTitle string          `json:"playlistTitle,omitempty"` // 播放列表标题
	EpisodeNumber *int            `json:"episodeNumber,omitempty"` // 视频在播放列表中的集数
	Progress      int             `json:"progress"`                // 播放进度（秒）
	Duration      *int            `json:"duration,omitempty"`      // 视频总时长（秒）
	Cover         *types.ImageURL `json:"cover"`                   // 视频封面图URL
	LastPlayedAt  int64           `json:"lastPlayedAt"`            // 最后播放时间
}

// VideoHistoryResponse represents the paginated video history response
type VideoHistoryResponse struct {
	Total    int64               `json:"total"`    // 总记录数
	Page     int                 `json:"page"`     // 当前页码
	PageSize int                 `json:"pageSize"` // 每页大小
	Items    []*VideoHistoryItem `json:"items"`    // 播放记录列表
}

type PlaylistSlugItem struct {
	PlaylistID string          `json:"playlistId"` // 播放列表ID
	Slug       string          `json:"slug"`       // 播放列表Slug
	Title      string          `json:"title"`
	Cover      *types.ImageURL `json:"cover"`
	Slugs      []SlugI18n      `json:"slugs"`
	Vids       []string        `json:"vids"`
}

type SlugI18n struct {
	Lang string `json:"lang"`
	Slug string `json:"slug"`
}
