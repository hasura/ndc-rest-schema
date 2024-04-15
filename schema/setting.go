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
	Servers []ServerConfig       `json:"servers" yaml:"servers" mapstructure:"servers"`
	Headers map[string]EnvString `json:"headers,omitempty" yaml:"headers,omitempty" mapstructure:"headers"`
	// configure the request timeout in seconds, default 30s
	Timeout         *EnvInt                   `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout"`
	Retry           *RetryPolicySetting       `json:"retry,omitempty" yaml:"retry,omitempty" mapstructure:"retry"`
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

	if rs.Retry != nil {
		if err := rs.Retry.Validate(); err != nil {
			return fmt.Errorf("retry: %s", err)
		}
	}
	return nil
}

// RetryPolicySetting represents retry policy settings
type RetryPolicySetting struct {
	// Number of retry times
	Times EnvInt `json:"times,omitempty" yaml:"times,omitempty" mapstructure:"times"`
	// Delay retry delay in milliseconds
	Delay EnvInt `json:"delay,omitempty" yaml:"delay,omitempty" mapstructure:"delay"`
	// HTTPStatus retries if the remote service returns one of these http status
	HTTPStatus EnvInts `json:"httpStatus,omitempty" yaml:"httpStatus,omitempty" mapstructure:"httpStatus"`
}

// Validate if the current instance is valid
func (rs RetryPolicySetting) Validate() error {
	times, err := rs.Times.Value()
	if err != nil {
		return err
	}
	if times != nil && *times < 0 {
		return errors.New("retry policy times must be positive")
	}

	delay, err := rs.Times.Value()
	if err != nil {
		return err
	}
	if delay != nil && *delay < 0 {
		return errors.New("retry delay must be larger than 0")
	}

	httpStatus, err := rs.HTTPStatus.Value()
	if err != nil {
		return err
	}
	for _, status := range httpStatus {
		if status < 400 || status >= 600 {
			return errors.New("retry http status must be in between 400 and 599")
		}
	}

	return nil
}

// ServerConfig contains server configurations
type ServerConfig struct {
	URL EnvString `json:"url" yaml:"url" mapstructure:"url"`
}

// Validate if the current instance is valid
func (ss ServerConfig) Validate() error {
	urlValue := ss.URL.Value()
	if urlValue == nil || *urlValue == "" {
		if ss.URL.EnvTemplate.IsEmpty() {
			return errors.New("url is required for server")
		}
		return nil
	}

	if _, err := parseHttpURL(*urlValue); err != nil {
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

func parseRelativeOrHttpURL(input string) (*url.URL, error) {
	if strings.HasPrefix(input, "/") {
		return &url.URL{Path: input}, nil
	}
	return parseHttpURL(input)
}
