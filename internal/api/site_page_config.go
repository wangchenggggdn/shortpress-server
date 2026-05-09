package api

import "encoding/json"

// SitePageConfigCreateRequest defines the request structure for creating a site page config
type SitePageConfigCreateRequest struct {
	Type   string          `json:"type" binding:"required"`   // Page type
	Config json.RawMessage `json:"config" binding:"required"` // Configuration data
}

// SitePageConfigCreateResponse defines the response structure for creating a site page config
type SitePageConfigCreateResponse struct {
	ID uint `json:"id"` // Created config ID
}

// SitePageConfigUpdateRequest defines the request structure for updating a site page config
type SitePageConfigUpdateRequest struct {
	Type   string          `json:"type" binding:"required"`   // Page type
	Config json.RawMessage `json:"config" binding:"required"` // Configuration data
}

// SitePageConfigUpdateResponse defines the response structure for updating a site page config
type SitePageConfigUpdateResponse struct {
	ID uint `json:"id"` // Updated config ID
}

// SitePageConfigListResponse defines the response structure for listing site page configs
type SitePageConfigListResponse struct {
	Items []*SitePageConfigItem `json:"items"`
	Total int64                 `json:"total"`
}

// SitePageConfigItem represents a site page config item in the response
type SitePageConfigItem struct {
	ID     uint            `json:"id"`     // Config ID
	SiteID string          `json:"siteId"` // Site ID
	Type   string          `json:"type"`   // Page type
	Config json.RawMessage `json:"config"` // Configuration data
}
