package api

import (
	"encoding/json"

	"shortpress-server/internal/types"
)

type PlaylistList struct {
	Items []*PlaylistInfo `json:"items"` // Playlist list
}

type PlaylistCreateResponse struct {
	Response
	Data string `json:"data"`
}

type PlaylistSeo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Keywords    string `json:"keywords,omitempty"`
}

// PlaylistResponse Playlist information response
type PlaylistResponse struct {
	Response
	Data PlaylistInfo `json:"data"`
}

// PlaylistDeleteRequest Request parameters for deleting playlist
type PlaylistDeleteRequest struct {
	// SiteID	 string   `json:"siteId" binding:"required" example:"site-123"` // Site ID
	PlaylistIDs []string `json:"playlistIds" binding:"required" example:"[\"playlist-123\",\"playlist-456\"]"` // List of playlist IDs to delete
}

// PlaylistVideosResponseData Playlist video list data
type PlaylistVideosResponseData struct {
	Total    int64       `json:"total" example:"100"`   // Total number of videos
	Page     int         `json:"page" example:"1"`      // Current page number
	PageSize int         `json:"pageSize" example:"20"` // Items per page
	Items    []VideoInfo `json:"items"`                 // Video list information
}

// PlaylistVideosResponse Playlist videos list response
type PlaylistVideosResponse struct {
	Response
	Data PlaylistVideosResponseData `json:"data"`
}

// PlaylistAddVideosRequest Request parameters for adding videos to playlist
type PlaylistAddVideosRequest struct {
	PlaylistID string   `json:"playlistId" binding:"required" example:"playlist-123"`    // Playlist ID
	VIDs       []string `json:"vids" binding:"required" example:"[\"vid-1\",\"vid-2\"]"` // List of video IDs to add
}

// PlaylistDelVideosRequest Request parameters for removing videos from playlist
type PlaylistDelVideosRequest struct {
	PlaylistID string   `json:"playlistId" binding:"required" example:"playlist-123"`    // Playlist ID
	VIDs       []string `json:"vids" binding:"required" example:"[\"vid-1\",\"vid-2\"]"` // List of video IDs to remove
}

type PlaylistVideData struct {
	VID    string `json:"vid"`    // Video ID
	Status int    `json:"status"` // Video status
}

// PlaylistInfo Basic playlist information
type PlaylistInfo struct {
	PlaylistID       string          `json:"playlistId"`
	Title            string          `json:"title"`
	Slug             string          `json:"slug"` // Slug for SEO-friendly URL
	Description      string          `json:"description"`
	Tags             string          `json:"tags"`
	Cover            *types.ImageURL `json:"cover"`
	Status           int             `json:"status"`
	VideoCount       int             `json:"videoCount"`       // Number of videos
	Version          int             `json:"version"`          // Version number
	AccessType       int             `json:"accessType"`       // Access type 1:free 2:paid
	SingleVideoPrice int             `json:"singleVideoPrice"` // Cost in coins to unlock a single video
	FreeVideos       int             `json:"freeVideos"`       // Number of free videos before requiring payment
	UtmSource        string          `json:"utmSource"`        // UTM 来源标识
	Seo              *PlaylistSeo    `json:"seo"`
	Videos           []*VideoItem    `json:"videos"`                         // Video list
	CreatedAt        int64           `json:"createdAt" example:"1609459200"` // Creation time
	UpdatedAt        int64           `json:"updatedAt" example:"1609459200"` // Update time
}

type VideoSortData struct {
	VIDs []string `json:"vids" binding:"required" example:"[\"vid-1\",\"vid-2\"]"` // List of video IDs
}

type PlaylistVideosOrder struct {
	PlaylistID string         `json:"playlistId" binding:"required" example:"playlist-123"` // Playlist ID
	Version    int            `json:"version" `                                             // Version number
	SortData   *VideoSortData `json:"sortData" binding:"required"`                          // Video sorting data
}

// PlaylistAccessChangeRequest represents the request to change playlist access settings
type PlaylistAccessChangeRequest struct {
	PlaylistID       string `json:"playlistId" binding:"required"`
	AccessType       int    `json:"accessType" binding:"required,min=1,max=3"` // 1:free 2:paid 3:member-only
	SingleVideoPrice int    `json:"singleVideoPrice" binding:"min=0"`          // Cost in coins to unlock a single video
	FreeVideos       *int   `json:"freeVideos"`                                // Number of free videos before requiring payment
}

type VideoItem struct {
	VID          string          `json:"vid"`                    // Video ID
	Status       int             `json:"status"`                 // Video status
	UnLockStatus int             `json:"unlockStatus"`           // Unlock status 1 - unlocked, 2 - locked  0 - not applicable
	Config       json.RawMessage `json:"config,omitempty"`       // Video config from videos table
	Cover        *types.ImageURL `json:"cover,omitempty"`        // Video cover
	LocalPath    *types.ImageURL `json:"local_path,omitempty"`   // Playback path derived from cover (extension replaced with .mp4)
	// Title       string `json:"title"`  // Video title
	// UploadDate  int64  `json:"uploadDate"` // Video upload date
}

// NewReleasePlaylistItem represents a new release playlist item
type NewReleasePlaylistItem struct {
	PlaylistID string          `json:"playlistId"` // Playlist ID
	Title      string          `json:"title"`      // Playlist name
	Slug       string          `json:"slug"`       // Slug for SEO-friendly URL
	Cover      *types.ImageURL `json:"cover"`      // Playlist cover
	CreatedAt  int64           `json:"createdAt"`  // Creation time
}

// NewReleasePlaylistsResponse represents the response for new release playlists
type NewReleasePlaylistsResponse struct {
	Response
	Data []*NewReleasePlaylistItem `json:"data"`
}

// GrantCoinsRequest 创作者手动增加金币请求
type GrantCoinsRequest struct {
	SiteID     string `json:"siteId" binding:"required"`
	UserEmail  string `json:"userEmail" binding:"required"`
	CoinAmount int    `json:"coinAmount" binding:"required,min=1"`
	Reason     string `json:"reason" binding:"max=255"`
}

// GrantCoinsResponse 创作者手动增加金币响应
type GrantCoinsResponse struct {
	UserEmail      string `json:"userEmail"`
	AmountAdded    int    `json:"amountAdded"`
	CurrentBalance int    `json:"currentBalance"`
	TransactionID  string `json:"transactionId"`
}

type PlaylistTranslateRequest struct {
	PlaylistID string `json:"playlistId" binding:"required"`
}

type PlaylistTranslateResponse struct {
	Response
	Data []*PlaylistI18nItem `json:"data"`
}

type PlaylistI18nItem struct {
	PlaylistID     string `json:"playlistId"`
	Language       string `json:"language"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	Tags           string `json:"tags"`
	SeoTitle       string `json:"seoTitle"`
	SeoDescription string `json:"seoDescription"`
	SeoKeywords    string `json:"seoKeywords"`
}

type PlaylistI18nResponse struct {
	Response
	Data []*PlaylistI18nItem `json:"data"`
}

type PlaylistI18nModifyRequest struct {
	Data []*PlaylistI18nItem `json:"data" binding:"required"`
}
