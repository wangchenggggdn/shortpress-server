package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	// TikTok OAuth2 端点配置
	TikTokAuthURL  = "https://www.tiktok.com/v2/auth/authorize/"
	TikTokTokenURL = "https://open.tiktokapis.com/v2/oauth/token/"
	TikTokUserURL  = "https://open.tiktokapis.com/v2/user/info/"

	// TikTok OAuth2 Scopes
	TikTokScopeBasic   = "user.info.basic"
	TikTokScopeProfile = "user.info.profile"
	TikTokScopeStats   = "user.info.stats"
)

// tikTokAuth TikTok OAuth2 认证实现
type tikTokAuth struct {
	cfg *oauth2.Config
}

// TikTokTokenResponse TikTok Token 响应结构
type TikTokTokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int64  `json:"expires_in"`
	RefreshToken     string `json:"refresh_token"`
	RefreshExpiresIn int64  `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	Scope            string `json:"scope"`
}

// TikTokUserInfoResponse TikTok 用户信息响应结构
type TikTokUserInfoResponse struct {
	Data struct {
		User struct {
			OpenID          string `json:"open_id"`
			UnionID         string `json:"union_id"`
			AvatarURL       string `json:"avatar_url"`
			AvatarLargerURL string `json:"avatar_larger_url"`
			AvatarMediumURL string `json:"avatar_medium_url"`
			DisplayName     string `json:"display_name"`
			BioDescription  string `json:"bio_description"`
			ProfileDeepLink string `json:"profile_deep_link"`
			IsVerified      bool   `json:"is_verified"`
			FollowersCount  int64  `json:"followers_count"`
			FollowingCount  int64  `json:"following_count"`
			LikesCount      int64  `json:"likes_count"`
			VideoCount      int64  `json:"video_count"`
		} `json:"user"`
	} `json:"data"`
	Error struct {
		Code        string `json:"code"`
		Message     string `json:"message"`
		LogID       string `json:"log_id"`
		Description string `json:"description"`
	} `json:"error"`
}

// NewTikTok 创建 TikTok OAuth2 Provider
func NewTikTok() Provider {
	cfg := &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL:   TikTokAuthURL,
			TokenURL:  TikTokTokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
		Scopes: []string{TikTokScopeBasic, TikTokScopeProfile},
	}

	return &tikTokAuth{
		cfg: cfg,
	}
}

// Authorize 执行 TikTok OAuth2 认证
func (t *tikTokAuth) Authorize(ctx context.Context, args *AuthArgs) (*User, error) {
	if args.Credential != "" {
		// TikTok 不支持 JWT Credential 认证，需要使用 Token 方式
		return nil, fmt.Errorf("TikTok does not support credential-based authentication")
	}

	if args.Code != "" {
		// 使用授权码换取访问令牌
		return t.authorizeWithCode(ctx, args)
	} else if args.Token != "" {
		// 直接使用访问令牌获取用户信息
		return t.authorizeWithToken(ctx, args.Token)
	}

	return nil, fmt.Errorf("either authorization code or access token is required")
}

// authorizeWithCode 使用授权码进行认证
func (t *tikTokAuth) authorizeWithCode(ctx context.Context, args *AuthArgs) (*User, error) {
	// 准备 Token 交换请求参数
	data := url.Values{}
	data.Set("client_key", t.cfg.ClientID)
	data.Set("client_secret", t.cfg.ClientSecret)
	data.Set("code", args.Code)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", t.cfg.RedirectURL)

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", TikTokTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	// 解析 Token 响应
	var tokenResp TikTokTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("no access token received from TikTok")
	}

	// 使用获取到的访问令牌获取用户信息
	return t.authorizeWithToken(ctx, tokenResp.AccessToken)
}

// authorizeWithToken 使用访问令牌获取用户信息
func (t *tikTokAuth) authorizeWithToken(ctx context.Context, accessToken string) (*User, error) {
	// 创建获取用户信息的请求
	req, err := http.NewRequestWithContext(ctx, "GET", TikTokUserURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user info request: %w", err)
	}

	// 设置访问令牌和请求头
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// 添加查询参数
	q := req.URL.Query()
	q.Add("fields", "open_id,union_id,avatar_url,display_name")
	req.URL.RawQuery = q.Encode()

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read user info response: %w", err)
	}

	// 解析用户信息响应
	var userResp TikTokUserInfoResponse
	if err := json.Unmarshal(body, &userResp); err != nil {
		return nil, fmt.Errorf("failed to parse user info response: %w", err)
	}

	// 检查是否有错误（"ok" 表示成功）
	if userResp.Error.Code != "" && userResp.Error.Code != "ok" {
		return nil, fmt.Errorf("TikTok API error: %s - %s", userResp.Error.Code, userResp.Error.Message)
	}

	// 转换为统一的 User 结构
	user := &User{
		ID:       userResp.Data.User.OpenID,
		Username: userResp.Data.User.DisplayName,
		Avatar:   userResp.Data.User.AvatarURL,
		Email:    "", // TikTok API 不提供邮箱信息
	}

	// 如果没有 open_id，使用 union_id
	if user.ID == "" && userResp.Data.User.UnionID != "" {
		user.ID = userResp.Data.User.UnionID
	}

	return user, nil
}
