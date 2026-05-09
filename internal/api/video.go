package api

import (
	"encoding/json"
	"shortpress-server/internal/types"
)

// VideoUploadResponseData defines video upload response data
type VideoUploadResponseData struct {
	VIDs []string `json:"vids" example:"[\"vid-123\",\"vid-456\"]"` // List of successfully uploaded video IDs
}

// VideoUploadResponse defines video upload response
type VideoUploadResponse struct {
	Response
	Data VideoUploadResponseData `json:"data"`
}

type VideoUploadData struct {
	VID      string `json:"vid"`      // Unique video identifier
	Filename string `json:"filename"` // File name
	Path     string `json:"path"`     // File path
}

// VideoSourceInfo represents a single playback source for a video.
// For non-local providers, URL is an arbitrary string. For local provider, URL will be the stored path.
type VideoSourceInfo struct {
	SourceID     string `json:"sourceId" example:"source-123456"`                        // Unique source identifier
	Provider     string `json:"provider" example:"local"`                                // Source provider (local,youtube,vimeo, unknown)
	SourceType   int    `json:"sourceType" example:"1"`                                  // 1:local 2:http(s) 3:embed
	URL          string `json:"url" example:"https://wwww.xxx.com/videolib/abc/vid.mp4"` // Playback URL or embed URL/ID; local paths will auto-prepend host
	UploadStatus int8   `json:"uploadStatus" example:"5"`                                // Upload status for this source
	Priority     int    `json:"priority" example:"1"`                                    // Lower values are preferred
	Duration     int64  `json:"duration,omitempty" example:"180"`                        // Optional duration (seconds)
	Width        int32  `json:"width,omitempty" example:"1920"`                          // Optional width
	Height       int32  `json:"height,omitempty" example:"1080"`                         // Optional height
}

// VideoInfo Video information structure
type VideoInfo struct {
	VID         string                `json:"vid" example:"vid-123456"`                       // Unique video identifier
	Title       string                `json:"title" example:"Example video"`                  // Video title
	Description string                `json:"description" example:"This is an example video"` // Video description
	Tags        string                `json:"tags" example:"example,tutorial"`                // Video tags
	Cover       *types.ImageURL       `json:"cover" example:"http://example.com/cover.jpg"`   // Video cover
	Duration    int64                 `json:"duration" example:"180"`                         // Video duration (seconds)
	Subtitles   *types.SubtitleTracks `json:"subtitles,omitempty"`                            // Optional subtitle tracks specific to this source
	Config      json.RawMessage       `json:"config,omitempty"`                               // Custom configuration (arbitrary JSON)
	Status      int8                  `json:"status" example:"1"`                             // Video status: 1-Unpublished, 2-Published, 3-Offline, 127-Deleted
	CreatedAt   int64                 `json:"createdAt" example:"1609459200"`                 // Creation time
	UpdatedAt   int64                 `json:"updatedAt" example:"1609459200"`                 // Update time
	// New: multiple playback sources
	Sources []*VideoSourceInfo `json:"sources"` // All available playback sources for this video
	Seo     *VideoSeo          `json:"seo"`     // Video SEO information
}

// VideoListRequest defines video list query request
type VideoListRequest struct {
	Status    int `json:"status" example:"1"`                       // Video status filter
	OrderType int `json:"orderType" example:"1"`                    // Sorting method: 0-Creation time descending, 1-Text name sorting
	Page      int `json:"page" binding:"required" example:"1"`      // Page number
	PageSize  int `json:"pageSize" binding:"required" example:"20"` // Items per page
}

// VideoListResponseData defines video list response data
type VideoListResponseData struct {
	Total    int64        `json:"total" example:"100"`   // Total count
	Page     int          `json:"page" example:"1"`      // Current page number
	PageSize int          `json:"pageSize" example:"20"` // Items per page
	Items    []*VideoInfo `json:"items"`                 // Video list
}

// VideoListResponse defines video list response
type VideoListResponse struct {
	Response
	Data VideoListResponseData `json:"data"`
}

// VideoSeoInfo Video SEO information structure
type VideoSeo struct {
	Title       string `json:"title" example:"Example video"`                  // Video title
	Description string `json:"description" example:"This is an example video"` // Video description
	Keywords    string `json:"Keywords" example:"example,tutorial"`            // Keywords
}

// VideoDeleteRequest defines video delete request
type VideoDeleteRequest struct {
	VIDs []string `json:"vids" binding:"required" example:"[\"vid-123\",\"vid-456\"]"` // List of video IDs to delete
}

// VideoSearchRequest defines video search request
type VideoSearchRequest struct {
	Keyword  string `json:"keyword" binding:"required" example:"tutorial"` // Search keyword
	Page     int    `json:"page" binding:"required" example:"1"`           // Page number
	PageSize int    `json:"pageSize" binding:"required" example:"20"`      // Items per page
}

type VideoReplaceData struct {
	VID            string          `json:"vid"`            // Unique video identifier
	Cover          *types.ImageURL `json:"cover"`          // Video cover
	VideoSourceUrl *types.VideoUrl `json:"videoSourceUrl"` // Video source play URL
}

type VideoReplaceResonse struct {
	Response
	Data VideoReplaceData `json:"data"`
}

// VideoSourceCreateItem represents one network source to add
// VideoSourceCreateItem represents a single video source entry used when creating or updating a video.
// It describes where the video comes from, how it is served, and optional metadata such as subtitles and priority.
//
// Provider: Source provider identifier (e.g. "local", "youtube", "vimeo", "anime").
// SourceType: Source kind; 1 = local file, 2 = http(s) direct link, 3 = embed (iframe).
// URL: The video URL or embed URL/identifier. Use a typed URL field for validation/formatting.
// Priority: Integer priority where lower values are preferred. Default is 0 if omitted.
type VideoSourceCreateItem struct {
	Provider   string `json:"provider" example:"anime"`                            // Source provider (local,youtube,vimeo, ...)
	SourceType int    `json:"sourceType" example:"3"`                              // 1:local 2:http 3:Embed(iframe)
	URL        string `json:"url" example:"https://example.com/frame/xxx/animeid"` // Video URL or embed URL/identifier
	Priority   int    `json:"priority" example:"0"`                                // Lower is preferred; default 0
	// Subtitles  *types.SubtitleTracks `json:"subtitles"`                          // Optional subtitle tracks
}

// VideoAddSourcesRequest request to add multiple sources for a video
type VideoAddSourcesRequest struct {
	VID     string                   `json:"vid" binding:"required"`     // Target video id
	Sources []*VideoSourceCreateItem `json:"sources" binding:"required"` // Sources to add
}

type VideoAddSourcesResponse struct {
	Response
	Data struct {
		VID string `json:"vid"`
	} `json:"data"`
}

// Internal API for service-to-service communication

// InternalGetVideoConfigResponse defines response for getting video config
type InternalGetVideoConfigResponse struct {
	VID    string          `json:"vid" example:"video-123"` // Video ID
	Config json.RawMessage `json:"config"`                  // Config JSON data
}
