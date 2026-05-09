package api

// SavePagesBuilderDataRequest represents the request for saving pages builder data
type SavePagesBuilderDataRequest struct {
	SiteID   string      `json:"siteId" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	SiteData interface{} `json:"siteData" binding:"required"`
}

// SavePagesBuilderDataResponse represents the response for saving pages builder data
type SavePagesBuilderDataResponse struct {
	SiteID        string `json:"siteId" example:"550e8400-e29b-41d4-a716-446655440000"`
	VersionNumber int    `json:"versionNumber" example:"1"`
	UpdatedAt     int64  `json:"updatedAt" example:"1672531199"`
}

// PagesBuilderDataItem represents pages builder data
type PagesBuilderDataItem struct {
	SiteID               string      `json:"siteId" example:"550e8400-e29b-41d4-a716-446655440000"`
	SiteData             interface{} `json:"siteData" example:"{\"pages\":[],\"sections\":[],\"config\":{}}"`
	VersionNumber        int         `json:"versionNumber" example:"1"`
	LastPublishedVersion *int        `json:"lastPublishedVersion" example:"1"`
	CreatedBy            string      `json:"createdBy" example:"550e8400-e29b-41d4-a716-446655440003"`
	CreatedAt            int64       `json:"createdAt" example:"1672531199"`
	UpdatedAt            int64       `json:"updatedAt" example:"1672531199"`
}

// GetPagesBuilderDataRequest represents the request for getting pages builder data
type GetPagesBuilderDataRequest struct {
	SiteID string `form:"site_id" json:"site_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// GetPagesBuilderDataResponse represents the response for pages builder data
type GetPagesBuilderDataResponse struct {
	SiteID               string      `json:"site_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	SiteData             interface{} `json:"site_data"`
	VersionNumber        int         `json:"version_number" example:"1"`
	LastPublishedVersion *int        `json:"last_published_version" example:"1"`
	CreatedBy            string      `json:"created_by" example:"550e8400-e29b-41d4-a716-446655440001"`
	CreatedAt            int64       `json:"created_at" example:"1672531199"`
	UpdatedAt            int64       `json:"updated_at" example:"1672531199"`
}

// PublishPagesBuilderDataRequest represents the request for publishing pages builder data
type PublishPagesBuilderDataRequest struct {
	SiteID string `json:"siteId" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// PublishPagesBuilderDataResponse represents the response for publishing pages builder data
type PublishPagesBuilderDataResponse struct {
	SiteID        string `json:"siteId" example:"550e8400-e29b-41d4-a716-446655440000"`
	VersionNumber int    `json:"versionNumber" example:"1"`
	PublishedAt   int64  `json:"publishedAt" example:"1672531199"`
}

// GetPublishHistoryRequest represents the request for getting publish history
type GetPublishHistoryRequest struct {
	SiteID string `form:"site_id" json:"site_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Limit  int    `form:"limit" json:"limit" example:"10"`
	Offset int    `form:"offset" json:"offset" example:"0"`
}

// PublishHistoryItem represents a single publish history record
type PublishHistoryItem struct {
	VersionNumber int    `json:"version_number" example:"1"`
	PublishedBy   string `json:"published_by" example:"550e8400-e29b-41d4-a716-446655440001"`
	PublishedAt   int64  `json:"published_at" example:"1672531199"`
	IsCurrent     bool   `json:"is_current" example:"true"`
}

// GetPublishHistoryResponse represents the response for publish history
type GetPublishHistoryResponse struct {
	SiteID  string               `json:"site_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	History []PublishHistoryItem `json:"history"`
	Total   int                  `json:"total" example:"5"`
}

// GetSitePagesRequest represents the request for getting published site pages
type GetSitePagesRequest struct {
	SiteID string `form:"site_id" json:"site_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// GetSitePagesResponse represents the response for published site pages
type GetSitePagesResponse struct {
	SiteID        string      `json:"site_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	VersionNumber int         `json:"version_number" example:"1"`
	PublishedData interface{} `json:"site_data"`
	PublishedAt   int64       `json:"published_at" example:"1672531199"`
}

// TemplateItem represents a single page template item
type TemplateItem struct {
	TemplateID  string  `json:"templateId" example:"tpl_01f3b2"`
	Name        string  `json:"name" example:"Anime Template 1"`
	Description *string `json:"description" example:"Suitable for anime video sites"`
	Cover       *string `json:"cover" example:"https://cdn.example.com/covers/tpl_01.png"`
	Version     int     `json:"version" example:"1"`
}

// TemplateListResponseData is the data payload for template list
type TemplateListResponseData struct {
	Items    []TemplateItem `json:"items"`
	Total    int            `json:"total" example:"2"`
	Page     int            `json:"page" example:"1"`
	PageSize int            `json:"pageSize" example:"10"`
}

// PageTranslateRequest 页面翻译请求
type PageTranslateRequest struct {
	Items []PageTranslateItem `json:"items"`
}

type PageTranslateItem struct {
	FieldType any                `json:"fieldType"`
	Texts     PageTranslateTexts `json:"texts"`
	Context   any                `json:"context"`
}

type PageTranslateTexts struct {
	Name        string `json:"name,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Keywords    string `json:"keywords,omitempty"`
}

// PageTranslateResponse 页面翻译响应
type PageTranslateResponse struct {
	FieldType    any                `json:"fieldType"`
	Translations []PageTranslations `json:"translations"`
	Context      any                `json:"context"`
}

type PageTranslations struct {
	Lang string `json:"lang"`
	PageTranslateTexts
}
