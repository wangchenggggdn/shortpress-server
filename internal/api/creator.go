package api

import "time"

// RegisterCreatorRequest Defines email registration request parameters, all fields are required
type RegisterCreatorRequest struct {
	Email    string `json:"email" binding:"required" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"secret"`
	NickName string `json:"nickname"  example:"shortpress"`
}

// ResetCreatorPwdRequest Defines request parameters for resetting creator password
type ResetCreatorPwdRequest struct {
	// email corresponds to the email field in MySQL table
	Email string `json:"email" binding:"required" example:"user@example.com"`
	// newPassword is the new password
	NewPassword string `json:"newPassword" binding:"required" example:"newSecret"`
}

// CreatorLoginRequest Defines creator login request parameters, all fields are required
type CreatorLoginRequest struct {
	// Email corresponds to the email field in MySQL table, type VARCHAR(255), cannot be null
	Email string `json:"email" binding:"required" example:"user@example.com"`
	// Password corresponds to the plaintext password used for password verification in MySQL table, type VARCHAR(255), cannot be null
	Password string `json:"password" binding:"required" example:"secret"`
}

// CreatorLoginResponse Defines creator login response parameters
type CreatorLoginData struct {
	// AccessToken is the token returned after successful login, type VARCHAR(255)
	AccessToken string `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6xxxxxc"`
}

type CreatorGuides struct {
	Name   string `json:"name"`
	Status int    `json:"status"` // 1 Completed, other values Incomplete
}

// CreatorProfileResponse Defines creator profile information response parameters
type CreatorProfileData struct {
	// Email
	Email string `json:"email" example:"user@example.com"`
	// Nickname
	Nickname string `json:"nickname" example:"JohnDoe"`
	// AvatarURL
	AvatarURL string `json:"avatarUrl" example:"http://example.com/avatar.jpg"`
	// DefaultDomain
	DefultSiteDomain string `json:"defultSiteDomain" example:"https://myshortpress.ai"`
	// New task guides
	Guides []*CreatorGuides `json:"guides"`
	// CreatedAt Creation time, Unix timestamp
	CreatedAt time.Time `json:"createdAt" example:"1609459200"`
	// UpdatedAt Update time, Unix timestamp
	UpdatedAt time.Time `json:"updatedAt" example:"1609545600"`
}

type CreatorProfileResponse struct {
	Response
	Data CreatorProfileData `json:"data"`
}

// CreatorSitesResponse Defines response for returning a list of site IDs under the user
type CreatorSitesResponse struct {
	Response
	Data []*SiteInfo `json:"data"`
}

type CreatorStatsResponseData struct {
	VideoCount    int64 `json:"videoCount" example:"100"`
	PlaylistCount int64 `json:"playlistCount" example:"10"`
	SiteCount     int64 `json:"siteCount" example:"1"`
}

type CreatorStatsResponse struct {
	Response
	Data CreatorLoginData `json:"data"`
}
