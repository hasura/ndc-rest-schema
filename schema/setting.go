package schema

// NDCRestSettings represent global settings of the REST API, including base URL, headers, etc...
type NDCRestSettings struct {
	Servers []ServerConfig    `json:"servers" yaml:"servers" mapstructure:"servers"`
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty" mapstructure:"headers"`
	// configure the request timeout in seconds, default 30s
	Timeout         uint                      `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty" mapstructure:"securitySchemes"`
	Security        map[string][]string       `json:"security,omitempty" yaml:"security,omitempty" mapstructure:"security"`
	Version         string                    `json:"version,omitempty" yaml:"version,omitempty" mapstructure:"version"`
}

// ServerConfig contains server configurations
type ServerConfig struct {
	URL string `json:"url" yaml:"url" mapstructure:"url"`
}
