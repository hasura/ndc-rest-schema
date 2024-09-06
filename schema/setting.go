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

// TLSConfig represents the transport layer security (LTS) configuration for the HTTP client
type TLSConfig struct {
	// Path to the TLS cert to use for TLS required connections.
	CertFile string `json:"certFile,omitempty" yaml:"certFile,omitempty" mapstructure:"certFile"`
	// Alternative to cert_file. Provide the certificate contents as a string instead of a filepath.
	CertPem string `json:"certPem,omitempty" yaml:"certPem,omitempty" mapstructure:"certPem"`
	// Path to the TLS key to use for TLS required connections.
	KeyFile string `json:"keyFile,omitempty" yaml:"keyFile,omitempty" mapstructure:"keyFile"`
	// Alternative to key_file. Provide the key contents as a string instead of a filepath.
	KeyPem string `json:"keyPem,omitempty" yaml:"keyPem,omitempty" mapstructure:"keyPem"`
	// Path to the CA cert. For a client this verifies the server certificate. For a server this verifies client certificates.
	// If empty uses system root CA.
	CAFile string `json:"caFile,omitempty" yaml:"caFile,omitempty" mapstructure:"caFile"`
	// Alternative to ca_file. Provide the CA cert contents as a string instead of a filepath.
	CAPem string `json:"caPem,omitempty" yaml:"caPem,omitempty" mapstructure:"caPem"`
	// Additionally you can configure TLS to be enabled but skip verifying the server's certificate chain.
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty" mapstructure:"insecureSkipVerify"`
	// Whether to load the system certificate authorities pool alongside the certificate authority.
	IncludeSystemCACertsPool bool `json:"includeSystemCACertsPool,omitempty" yaml:"includeSystemCACertsPool,omitempty" mapstructure:"includeSystemCACertsPool"`
	// Minimum acceptable TLS version.
	MinVersion string `json:"minVersion,omitempty" yaml:"minVersion,omitempty" mapstructure:"minVersion"`
	// Maximum acceptable TLS version.
	MaxVersion string `json:"maxVersion,omitempty" yaml:"maxVersion,omitempty" mapstructure:"maxVersion"`
	// Explicit cipher suites can be set. If left blank, a safe default list is used.
	// See https://go.dev/src/crypto/tls/cipher_suites.go for a list of supported cipher suites.
	CipherSuites []string `json:"cipherSuites,omitempty" yaml:"cipherSuites,omitempty" mapstructure:"cipherSuites"`
	// Specifies the duration after which the certificate will be reloaded. If not set, it will never be reloaded.
	// The interval unit is minute
	ReloadInterval uint `json:"reloadInterval,omitempty" yaml:"reloadInterval,omitempty" mapstructure:"reloadInterval"`
}

// ServerConfig contains server configurations
type ServerConfig struct {
	URL     EnvString            `json:"url" yaml:"url" mapstructure:"url"`
	ID      string               `json:"id,omitempty" yaml:"id,omitempty" mapstructure:"id"`
	Headers map[string]EnvString `json:"headers,omitempty" yaml:"headers,omitempty" mapstructure:"headers"`
	// configure the request timeout in seconds, default 30s
	Timeout         *EnvInt                   `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout"`
	Retry           *RetryPolicySetting       `json:"retry,omitempty" yaml:"retry,omitempty" mapstructure:"retry"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty" mapstructure:"securitySchemes"`
	Security        AuthSecurities            `json:"security,omitempty" yaml:"security,omitempty" mapstructure:"security"`
	// The transport layer security (LTS) configuration for the HTTP client to communicate the current server
	TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty" mapstructure:"tls"`
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
