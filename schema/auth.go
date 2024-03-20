package schema

import (
	"encoding/json"
	"fmt"
	"slices"
)

// SecuritySchemeType represents the authentication scheme enum
type SecuritySchemeType string

const (
	APIKeyScheme        SecuritySchemeType = "apiKey"
	HTTPScheme          SecuritySchemeType = "http"
	OAuth2Scheme        SecuritySchemeType = "oauth2"
	OpenIDConnectScheme SecuritySchemeType = "openIdConnect"
)

var securityScheme_enums = []SecuritySchemeType{
	APIKeyScheme,
	HTTPScheme,
	OAuth2Scheme,
	OpenIDConnectScheme,
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SecuritySchemeType) UnmarshalJSON(b []byte) error {
	var rawResult string
	if err := json.Unmarshal(b, &rawResult); err != nil {
		return err
	}

	result, err := ParseSecuritySchemeType(rawResult)
	if err != nil {
		return err
	}

	*j = result
	return nil
}

// ParseSecuritySchemeType parses SecurityScheme from string
func ParseSecuritySchemeType(value string) (SecuritySchemeType, error) {
	result := SecuritySchemeType(value)
	if !slices.Contains(securityScheme_enums, result) {
		return result, fmt.Errorf("invalid SecuritySchemeType. Expected %+v, got <%s>", securityScheme_enums, value)
	}
	return result, nil
}

// ApiKeyLocation represents the location enum for apiKey auth
type APIKeyLocation string

const (
	APIKeyInHeader APIKeyLocation = "header"
	APIKeyInQuery  APIKeyLocation = "query"
	APIKeyInCookie APIKeyLocation = "cookie"
)

var apiKeyLocation_enums = []APIKeyLocation{APIKeyInHeader, APIKeyInQuery, APIKeyInCookie}

// UnmarshalJSON implements json.Unmarshaler.
func (j *APIKeyLocation) UnmarshalJSON(b []byte) error {
	var rawResult string
	if err := json.Unmarshal(b, &rawResult); err != nil {
		return err
	}

	result, err := ParseAPIKeyLocation(rawResult)
	if err != nil {
		return err
	}

	*j = result
	return nil
}

// ParseAPIKeyLocation parses APIKeyLocation from string
func ParseAPIKeyLocation(value string) (APIKeyLocation, error) {
	result := APIKeyLocation(value)
	if !slices.Contains(apiKeyLocation_enums, result) {
		return result, fmt.Errorf("invalid APIKeyLocation. Expected %+v, got <%s>", apiKeyLocation_enums, value)
	}
	return result, nil
}

// SecurityScheme contains authentication configurations.
// The schema follows [OpenAPI 3] specification
//
// [OpenAPI 3]: https://swagger.io/docs/specification/authentication
type SecurityScheme struct {
	Type              SecuritySchemeType `json:"type" yaml:"type" mapstructure:"type"`
	Value             string             `json:"value,omitempty" yaml:"value,omitempty" mapstructure:"value"`
	*APIKeyAuthConfig `yaml:",inline"`
	*HTTPAuthConfig   `yaml:",inline"`
	*OAuthConfig      `yaml:",inline"`
	*OpenIDConfig     `yaml:",inline"`
}

// APIKeyAuthConfig contains configurations for [apiKey authentication]
//
// [apiKey authentication]: https://swagger.io/docs/specification/authentication/api-keys/
type APIKeyAuthConfig struct {
	In   APIKeyLocation `json:"in" yaml:"in" mapstructure:"in"`
	Name string         `json:"name" yaml:"name" mapstructure:"name"`
}

// HTTPAuthConfig contains configurations for http authentication
// If the scheme is [basic] or [bearer], the authenticator follows OpenAPI 3 specification.
//
// [basic]: https://swagger.io/docs/specification/authentication/basic-authentication
// [bearer]: https://swagger.io/docs/specification/authentication/bearer-authentication
type HTTPAuthConfig struct {
	Header string `json:"header" yaml:"header" mapstructure:"header"`
	Scheme string `json:"scheme" yaml:"scheme" mapstructure:"scheme"`
}

// OAuthFlowType represents the OAuth flow type enum
type OAuthFlowType string

const (
	AuthorizationCodeFlow OAuthFlowType = "authorizationCode"
	ImplicitFlow          OAuthFlowType = "implicit"
	PasswordFlow          OAuthFlowType = "password"
	ClientCredentialsFlow OAuthFlowType = "clientCredentials"
)

var oauthFlow_enums = []OAuthFlowType{
	AuthorizationCodeFlow,
	ImplicitFlow,
	PasswordFlow,
	ClientCredentialsFlow,
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *OAuthFlowType) UnmarshalJSON(b []byte) error {
	var rawResult string
	if err := json.Unmarshal(b, &rawResult); err != nil {
		return err
	}

	result, err := ParseOAuthFlowType(rawResult)
	if err != nil {
		return err
	}

	*j = result
	return nil
}

// ParseOAuthFlowType parses OAuthFlowType from string
func ParseOAuthFlowType(value string) (OAuthFlowType, error) {
	result := OAuthFlowType(value)
	if !slices.Contains(oauthFlow_enums, result) {
		return result, fmt.Errorf("invalid OAuthFlowType. Expected %+v, got <%s>", oauthFlow_enums, value)
	}
	return result, nil
}

// OAuthFlow contains flow configurations for [OAuth 2.0] API specification
//
// [OAuth 2.0]: https://swagger.io/docs/specification/authentication/oauth2
type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty" yaml:"authorizationUrl,omitempty" mapstructure:"authorizationUrl"`
	TokenURL         string            `json:"tokenUrl,omitempty" yaml:"tokenUrl,omitempty" mapstructure:"tokenUrl"`
	RefreshURL       string            `json:"refreshUrl,omitempty" yaml:"refreshUrl,omitempty" mapstructure:"refreshUrl"`
	Scopes           map[string]string `json:"scopes,omitempty" yaml:"scopes,omitempty" mapstructure:"scopes"`
}

// OAuthConfig contains configurations for [OAuth 2.0] API specification
//
// [OAuth 2.0]: https://swagger.io/docs/specification/authentication/oauth2
type OAuthConfig struct {
	Flows map[OAuthFlowType]OAuthFlow `json:"flows" yaml:"flows" mapstructure:"flows"`
}

// OpenIDConfig contains configurations for [OpenID Connect] API specification
//
// [OpenID Connect]: https://swagger.io/docs/specification/authentication/openid-connect-discovery
type OpenIDConfig struct {
	OpenIDConnectURL string `json:"openIdConnectUrl" yaml:"openIdConnectUrl" mapstructure:"openIdConnectUrl"`
}

// AuthSecurity wraps the raw security requirement with helpers
type AuthSecurity map[string][]string

// NewAuthSecurity creates an AuthSecurity instance from name and scope
func NewAuthSecurity(name string, scopes []string) AuthSecurity {
	return AuthSecurity{
		name: scopes,
	}
}

// Name returns the name of security requirement
func (as AuthSecurity) Name() string {
	if len(as) > 0 {
		for k := range as {
			return k
		}
	}
	return ""
}

// Scopes returns scopes of security requirement
func (as AuthSecurity) Scopes() []string {
	if len(as) > 0 {
		for _, scopes := range as {
			return scopes
		}
	}
	return []string{}
}

// IsOptional checks if the security is optional
func (as AuthSecurity) IsOptional() bool {
	return len(as) == 0
}

// AuthSecurities wraps list of security requirements with helpers
type AuthSecurities []AuthSecurity

// IsEmpty checks if there is no security
func (ass AuthSecurities) IsEmpty() bool {
	return len(ass) == 0
}

// IsOptional checks if the security is optional
func (ass AuthSecurities) IsOptional() bool {
	if ass.IsEmpty() {
		return true
	}
	for _, as := range ass {
		if as.IsOptional() {
			return true
		}
	}
	return false
}

// Add adds a security with name and scope
func (ass *AuthSecurities) Add(item AuthSecurity) {
	*ass = append(*ass, item)
}

// Get gets a security by name
func (ass AuthSecurities) Get(name string) AuthSecurity {
	for _, as := range ass {
		if as.Name() == name {
			return as
		}
	}
	return nil
}

// First returns the first security
func (ass AuthSecurities) First() AuthSecurity {
	for _, as := range ass {
		return as
	}
	return nil
}
