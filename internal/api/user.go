package api

// UserRegisterRequest represents the request for user registration
type UserRegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname" `
}

// UserLoginRequest represents the request for user login
type UserLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// UserLoginResponse represents the response for user login
type UserLoginResponse struct {
	Response
	Data UserLoginData `json:"data"`
}

// UserLoginData contains the login response data
type UserLoginData struct {
	AccessToken string `json:"accessToken"`
	Ver         string `json:"ver"`
}

// UserLoginByAuthRequest represents the request for user login by auth
type UserLoginByAuthRequest struct {
	SrcType    string `json:"SrcType,omitempty"`
	Token      string `json:"Token,omitempty"`
	Credential string `json:"Credential,omitempty"`
	Code       string `json:"Code,omitempty"`
}

// UserProfileResponse represents the response for user profile
type UserProfileResponse struct {
	Response
	Data UserProfileData `json:"data"`
}

// UserProfileData contains the user profile data
type UserProfileData struct {
	UserID           string `json:"id"`
	Email            string `json:"email"`
	Nickname         string `json:"nickname"`
	AvatarURL        string `json:"avatarUrl"`
	PremiumType      int8   `json:"premiumType"` // 0: free, 1: premium ...
	OnetimeSub       int8   `json:"onetimeSub"`  // 1: 一次性订阅（非 Stripe 自动续订）
	PremiumExpiresAt int64  `json:"premiumExpiresAt"`
	AutoUnlock       bool   `json:"autoUnlock"` // Whether auto-unlock is enabled
	Status           int8   `json:"status"`
	Referer          string `json:"referer"`
	LoginType        int8   `json:"loginType"` // 0: email, 1: google, 2: facebook, 3: twitter, 4: tiktok
	HasPurchased     bool   `json:"hasPurchased"` // Whether the user has made any purchases
	Ver              string `json:"ver"` // App version from header
}

// UserInfo represents user information for API responses
type UserInfo struct {
	UserID           string `json:"userId" example:"user-123456"`
	Email            string `json:"email" example:"user@example.com"`
	Referer          string `json:"referer" example:""`
	Nickname         string `json:"nickname" example:"John Doe"`
	Status           int8   `json:"status" example:"1"`
	PremiumType      int8   `json:"premiumType" example:"0"`               // 0: free, 1: premium
	OnetimeSub       int8   `json:"onetimeSub" example:"0"`                // 1: 一次性订阅
	PremiumExpiresAt int64  `json:"premiumExpiresAt" example:"1609459200"` // Unix timestamp
	LastLoginAt      int64  `json:"lastLoginAt" example:"1609459200"`
	CreatedAt        int64  `json:"createdAt" example:"1609459200"`
	UpdatedAt        int64  `json:"updatedAt" example:"1609459200"`
}

// UserListResponseData defines the response structure for user listings
type UserListResponseData struct {
	Total    int64       `json:"total" example:"100"`   // Total number of users
	Page     int         `json:"page" example:"1"`      // Current page number
	PageSize int         `json:"pageSize" example:"20"` // Page size
	Items    []*UserInfo `json:"items"`                 // User list
}

// MetaClickSyncRequest syncs Meta fbc/fbp from the visitor client into the user row.
type MetaClickSyncRequest struct {
	Fbc            string `json:"fbc,omitempty"`
	Fbp            string `json:"fbp,omitempty"`
	Fbclid         string `json:"fbclid,omitempty"`
	EventSourceURL string `json:"eventSourceUrl,omitempty"`
}

// UserProfileModifyRequest represents the request for user profile modification
type UserProfileModifyRequest struct {
	Nickname   string `json:"nickname,omitempty"`   // Use pointer to allow optional field
	AutoUnlock *bool  `json:"autoUnlock,omitempty"` // Use pointer to allow optional field
}
