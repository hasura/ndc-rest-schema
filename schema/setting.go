package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// NDCRestSettings represent global settings of the REST API, including base URL, headers, etc...
type NDCRestSettings struct {
	Servers []ServerConfig    `json:"servers" yaml:"servers" mapstructure:"servers"`
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty" mapstructure:"headers"`
	// configure the request timeout in seconds, default 30s
	Timeout         uint                      `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty" mapstructure:"securitySchemes"`
	Security        AuthSecurities            `json:"security,omitempty" yaml:"security,omitempty" mapstructure:"security"`
	Version         string                    `json:"version,omitempty" yaml:"version,omitempty" mapstructure:"version"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *NDCRestSettings) UnmarshalJSON(b []byte) error {
	type Plain NDCRestSettings

	var raw Plain
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	result := NDCRestSettings(raw)

	if err := result.Validate(); err != nil {
		return err
	}
	*j = result
	return nil
}

// Validate if the current instance is valid
func (rs NDCRestSettings) Validate() error {
	for _, server := range rs.Servers {
		if err := server.Validate(); err != nil {
			return err
		}
	}

	for key, scheme := range rs.SecuritySchemes {
		if err := scheme.Validate(); err != nil {
			return fmt.Errorf("securityScheme %s: %s", key, err)
		}
	}
	return nil
}

// ServerConfig contains server configurations
type ServerConfig struct {
	URL string `json:"url" yaml:"url" mapstructure:"url"`
}

// Validate if the current instance is valid
func (ss ServerConfig) Validate() error {
	if ss.URL == "" {
		return errors.New("url is required for server")
	}

	if _, err := parseHttpURL(ss.URL); err != nil {
		return fmt.Errorf("server url: %s", err)
	}
	return nil
}

// parseHttpURL parses and validate if the URL has HTTP scheme
func parseHttpURL(input string) (*url.URL, error) {
	if !strings.HasPrefix(input, "https://") && !strings.HasPrefix(input, "http://") {
		return nil, fmt.Errorf("invalid HTTP URL %s", input)
	}

	return url.Parse(input)
}
