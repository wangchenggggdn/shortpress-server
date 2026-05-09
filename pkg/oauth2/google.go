package oauth2

import (
	"context"
	"encoding/json"
	"golang.org/x/oauth2"
)

const (
	GoogleAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	GoogleTokenURL = "https://oauth2.googleapis.com/token"
	GoogleUserURL  = "https://www.googleapis.com/oauth2/v3/userinfo"
	GoogleKeyURL   = "https://www.googleapis.com/oauth2/v3/certs"
)

const (
	GoogleScopeProfile = "https://www.googleapis.com/auth/userinfo.profile"
	GoogleScopeEmail   = "https://www.googleapis.com/auth/userinfo.email"
)

type googleAuth struct {
	cfg     *oauth2.Config
	decoder *Decoder
}

func NewGoogle() Provider {
	cfg := &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL:   GoogleAuthURL,
			TokenURL:  GoogleTokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
		Scopes: []string{GoogleScopeProfile, GoogleScopeEmail},
	}

	return &googleAuth{
		cfg:     cfg,
		decoder: NewDecoder(GoogleKeyURL),
	}
}

func (g *googleAuth) Authorize(ctx context.Context, args *AuthArgs) (*User, error) {
	if args.Credential != "" {
		return g.authorizeCredential(args.Credential)
	}

	res, err := g.cfg.Client(ctx, &oauth2.Token{
		AccessToken: args.Token,
		TokenType:   "Bearer",
	}).Get(GoogleUserURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var u struct {
		Sub     string `json:"sub"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}
	if err = json.NewDecoder(res.Body).Decode(&u); err != nil {
		return nil, err
	}
	return &User{
		ID:       u.Sub,
		Username: u.Name,
		Email:    u.Email,
		Avatar:   u.Picture,
	}, err
}

func (g *googleAuth) authorizeCredential(credential string) (*User, error) {
	claims, err := g.decoder.Decode(credential)
	if err != nil {
		return nil, err
	}
	user := &User{
		ID:       claims.Subject,
		Username: claims.Name,
		Avatar:   claims.Picture,
		Email:    claims.Email,
	}
	return user, nil
}
