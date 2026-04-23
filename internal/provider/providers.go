package provider

import (
	"context"
	"time"
	// "net/url"

	"golang.org/x/oauth2"
)

// providerHTTPTimeout caps per-request HTTP calls to identity providers
// (token exchange, userinfo, ID-token verify including JWKS refresh, OIDC
// discovery). A slow or unresponsive provider would otherwise hang the
// auth path — or, at Setup time, startup itself.
const providerHTTPTimeout = 10 * time.Second

// Providers contains all the implemented providers
type Providers struct {
	Google       Google       `group:"Google Provider" namespace:"google" env-namespace:"GOOGLE"`
	OIDC         OIDC         `group:"OIDC Provider" namespace:"oidc" env-namespace:"OIDC"`
	GenericOAuth GenericOAuth `group:"Generic OAuth2 Provider" namespace:"generic-oauth" env-namespace:"GENERIC_OAUTH"`
}

// Provider is used to authenticate users
type Provider interface {
	Name() string
	GetLoginURL(redirectURI, state string) string
	ExchangeCode(redirectURI, code string) (string, error)
	GetUser(token string) (User, error)
	Setup() error
}

type token struct {
	Token string `json:"access_token"`
}

// User is the authenticated user
type User struct {
	Email string `json:"email"`
}

// OAuthProvider is a provider using the oauth2 library
type OAuthProvider struct {
	Resource string `long:"resource" env:"RESOURCE" description:"Optional resource indicator"`

	Config *oauth2.Config
}

// ConfigCopy returns a copy of the oauth2 config with the given redirectURI
// which ensures the underlying config is not modified
func (p *OAuthProvider) ConfigCopy(redirectURI string) oauth2.Config {
	config := *p.Config
	config.RedirectURL = redirectURI
	return config
}

// OAuthGetLoginURL provides a base "GetLoginURL" for proiders using OAauth2
func (p *OAuthProvider) OAuthGetLoginURL(redirectURI, state string) string {
	config := p.ConfigCopy(redirectURI)

	if p.Resource != "" {
		return config.AuthCodeURL(state, oauth2.SetAuthURLParam("resource", p.Resource))
	}

	return config.AuthCodeURL(state)
}

// OAuthExchangeCode provides a base "ExchangeCode" for proiders using OAauth2
func (p *OAuthProvider) OAuthExchangeCode(redirectURI, code string) (*oauth2.Token, error) {
	config := p.ConfigCopy(redirectURI)
	ctx, cancel := context.WithTimeout(context.Background(), providerHTTPTimeout)
	defer cancel()
	return config.Exchange(ctx, code)
}
