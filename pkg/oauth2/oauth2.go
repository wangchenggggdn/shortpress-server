package oauth2

import (
	"context"
	"fmt"
)

type OauthType string

const (
	TypeGoogle OauthType = "google"
	TypeTikTok OauthType = "tiktok"
)

type User struct {
	ID       string
	Username string
	Avatar   string
	Email    string
}

type AuthArgs struct {
	// 登录类型
	Type OauthType
	// 授权码
	Code string
	// 随机值，防止XSRF攻击
	State string
	// 客户端直接传递token,省略了code换token的步骤
	Token string
	//jwt凭证
	Credential string
}

type Builder func() Provider

type Provider interface {
	Authorize(ctx context.Context, args *AuthArgs) (*User, error)
}

type Client interface {
	Authenticate(ctx context.Context, args *AuthArgs) (*User, error)
}

type client struct {
	provider map[OauthType]Provider
	builder  map[OauthType]Builder
}

func New(ots ...OauthType) Client {
	c := &client{
		provider: make(map[OauthType]Provider),
		builder: map[OauthType]Builder{
			TypeGoogle: NewGoogle,
			TypeTikTok: NewTikTok,
		},
	}
	for _, ot := range ots {
		if builder, ok := c.builder[ot]; ok {
			c.register(ot, builder())
		}
	}
	return c
}

func (c *client) register(ot OauthType, p Provider) {
	if _, ok := c.provider[ot]; ok {
		panic("duplicate provider" + ot)
	}
	c.provider[ot] = p
}

func (c *client) Authenticate(ctx context.Context, args *AuthArgs) (*User, error) {
	if p, ok := c.provider[args.Type]; ok {
		return p.Authorize(ctx, args)
	}
	return nil, fmt.Errorf("not support oauth type: %s", args.Type)
}
