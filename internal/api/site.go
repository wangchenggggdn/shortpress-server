package api

import (
	"encoding/json"
	"shortpress-server/internal/types"
)

// SiteSeo defines site SEO information request parameters
type SiteSeo struct {
	Title       string `json:"title" example:"Welcome to MySite Updated"`        // SEO title, VARCHAR(1024), optional
	Description string `json:"description" example:"This is my updated website"` // SEO description, TEXT, optional
	Keywords    string `json:"keywords" example:"site,example,updated"`          // SEO keywords, VARCHAR(1024), optional
}

// SiteCreateResponse defines create site response
type SiteCreateResponse struct {
	Response
	Data string `json:"data"`
}

// SiteGetResponseData defines response data for site and SEO information query
type SiteInfo struct {
	SiteID            string                `json:"siteId" example:"123e4567-e89b-12d3-a456-426614174000"`
	OfficialDomain    *types.OfficialDomain `json:"officialDomain" `
	Domain            string                `json:"domain" example:"example.com"`
	Redirect          bool                  `json:"redirect" example:"false"`
	Path              string                `json:"path" example:"/home"` // Site path, VARCHAR(255), required
	Name              string                `json:"name" example:"MySite Updated"`
	Logo              *types.ImageURL       `json:"logo" example:"http://example.com/logo_updated.png"`
	TemplateID        *string               `json:"templateId,omitempty" example:"tpl_01f3b2"`
	TemplateName      *string               `json:"templateName,omitempty" example:"Modern Template"` // Template name
	GoogleAnalyticsID *string               `json:"googleAnalyticsId" example:"UA-123456789-1"`       // Google Analytics ID, optional
	FacebookPixelID          *string `json:"facebookPixelId" example:"1234567890"`                    // Facebook Pixel ID, optional
	FacebookCapiAccessToken  *string `json:"facebookCapiAccessToken" example:"EAAxxxx..."`            // Meta CAPI access token (creator only), optional
	ThinkingDataAppId        *string `json:"thinkingdataAppId" example:"app-1234567890"`              // Thinking Data App ID, optional
	Theme             *int                  `json:"theme,omitempty" example:"0"`                        // accent palette index (visitor); omit on modify to leave unchanged
	Status            int                   `json:"status" example:"1"`
	Seo               *SiteSeo              `json:"seo"`
	SiteMultiLang     json.RawMessage       `json:"siteMultiLang"` // 多语言配置，JSON格式，不做处理，直接存储
	SeoMultiLang      json.RawMessage       `json:"seoMultiLang"`  // 多语言配置，JSON格式，不做处理，直接存储
}

// SiteGetResponse defines query site response
type SiteGetResponse struct {
	Response
	Data SiteInfo `json:"data"`
}

type SiteListResponse struct {
	Items []*SiteInfo `json:"items"`
}

// SiteAddPlaylistsRequest defines request parameters for adding playlists to a site
type SiteAddPlaylistsRequest struct {
	SiteID      string   `json:"siteId" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`               // Site ID, CHAR(36)
	PlaylistIDs []string `json:"playlistIds" binding:"required,dive,required" example:"[\"playlist-1\",\"playlist-2\"]"` // Playlist ID array
}

// SiteDelPlaylistsRequest defines request parameters for deleting playlists from a site
type SiteDelPlaylistsRequest struct {
	SiteID      string   `json:"siteId" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	PlaylistIDs []string `json:"playlistIds" binding:"required,dive,required" example:"[\"playlist-1\",\"playlist-2\"]"`
}

// SitePlaylistsResponseData defines response data for site playlists
type SitePlaylistsResponseData struct {
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
	Items    []*PlaylistInfo `json:"items"`
}

// SitePlaylistsResponse defines query site playlists response
type SitePlaylistsResponse struct {
	Response
	Data SitePlaylistsResponseData `json:"data"`
}

// SiteVideosResponseData defines response data for site videos query
type SiteVideosResponseData struct {
	Total    int64       `json:"total" example:"100"`
	Page     int         `json:"page" example:"1"`
	PageSize int         `json:"pageSize" example:"20"`
	Videos   []VideoInfo `json:"videos"`
}

// SiteVideosResponse defines query site videos response
type SiteVideosResponse struct {
	Response
	Data SiteVideosResponseData `json:"data"`
}

type ClientSiteInfo struct {
	SiteID string `json:"siteId" example:"123e4567-e89b-12d3-a456-426614174000"`
	Domain string `json:"domain" example:"example.com"`
	Path   string `json:"path" example:"/home"`
	Name   string `json:"name" example:"MySite Updated"`
	Logo   string `json:"logo" example:"http://example.com/logo_updated.png"`
}

type CompleteGuidesRequest struct {
	Guides []string `json:"guides" binding:"required,dive,required" example:"[\"guide1\",\"guide2\"]"`
}

// Define request structure for changing user status
// UserChangeStatusRequest represents a request to change a user's status in a site.
// swagger:model UserChangeStatusRequest
type UserChangeStatusRequest struct {
	// SiteID is the unique identifier of the site
	// Required: true
	SiteID string `json:"siteId" binding:"required"`

	// Email is the user's email address
	// Required: true
	// Format: email
	Email string `json:"email" binding:"required,email"`

	// Status is the new status to assign to the user
	// Required: true
	// Enum: 2 3 127
	// 2: activate - User is active
	// 3: forbidden - User is forbidden
	// 127: delete - User is deleted
	Status int8 `json:"status" binding:"required"` // Allowed: 2=activate, 3=forbidden, 127=delete
}
