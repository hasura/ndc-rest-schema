package schema

import (
	"encoding/json"

	"github.com/hasura/ndc-sdk-go/schema"
)

// NDCRestSchema extends the [NDC SchemaResponse] with OpenAPI REST information
//
// [NDC schema]: https://github.com/hasura/ndc-sdk-go/blob/1d3339db29e13a170aa8be5ff7fae8394cba0e49/schema/schema.generated.go#L887
type NDCRestSchema struct {
	Settings *NDCRestSettings `json:"settings,omitempty" yaml:"settings,omitempty" mapstructure:"settings"`

	// Collections which are available for queries
	Collections []schema.CollectionInfo `json:"collections" yaml:"collections" mapstructure:"collections"`

	// Functions (i.e. collections which return a single column and row)
	Functions []*RESTFunctionInfo `json:"functions" yaml:"functions" mapstructure:"functions"`

	// A list of object types which can be used as the types of arguments, or return
	// types of procedures. Names should not overlap with scalar type names.
	ObjectTypes schema.SchemaResponseObjectTypes `json:"object_types" yaml:"object_types" mapstructure:"object_types"`

	// Procedures which are available for execution as part of mutations
	Procedures []*RESTProcedureInfo `json:"procedures" yaml:"procedures" mapstructure:"procedures"`

	// A list of scalar types which will be used as the types of collection columns
	ScalarTypes schema.SchemaResponseScalarTypes `json:"scalar_types" yaml:"scalar_types" mapstructure:"scalar_types"`
}

// NewNDCRestSchema creates a NDCRestSchema instance
func NewNDCRestSchema() *NDCRestSchema {
	return &NDCRestSchema{
		Settings:    &NDCRestSettings{},
		Collections: []schema.CollectionInfo{},
		Functions:   []*RESTFunctionInfo{},
		Procedures:  []*RESTProcedureInfo{},
		ObjectTypes: make(schema.SchemaResponseObjectTypes),
		ScalarTypes: make(schema.SchemaResponseScalarTypes),
	}
}

// ToSchemaResponse converts the instance to NDC schema.SchemaResponse
func (ndc NDCRestSchema) ToSchemaResponse() *schema.SchemaResponse {
	functions := make([]schema.FunctionInfo, len(ndc.Functions))
	for i, fn := range ndc.Functions {
		functions[i] = fn.FunctionInfo
	}
	procedures := make([]schema.ProcedureInfo, len(ndc.Procedures))
	for i, proc := range ndc.Procedures {
		procedures[i] = proc.ProcedureInfo
	}

	return &schema.SchemaResponse{
		Collections: ndc.Collections,
		ObjectTypes: ndc.ObjectTypes,
		ScalarTypes: ndc.ScalarTypes,
		Functions:   functions,
		Procedures:  procedures,
	}
}

// Request represents the HTTP request information of the webhook
type Request struct {
	URL        string               `json:"url,omitempty" yaml:"url,omitempty" mapstructure:"url"`
	Method     string               `json:"method,omitempty" yaml:"method,omitempty" mapstructure:"method"`
	Type       RequestType          `json:"type,omitempty" yaml:"type,omitempty" mapstructure:"type"`
	Headers    map[string]EnvString `json:"headers,omitempty" yaml:"headers,omitempty" mapstructure:"headers"`
	Parameters []RequestParameter   `json:"parameters,omitempty" yaml:"parameters,omitempty" mapstructure:"parameters"`
	Security   AuthSecurities       `json:"security,omitempty" yaml:"security,omitempty" mapstructure:"security"`
	// configure the request timeout in seconds, default 30s
	Timeout     uint           `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout"`
	Servers     []ServerConfig `json:"servers,omitempty" yaml:"servers,omitempty" mapstructure:"servers"`
	RequestBody *RequestBody   `json:"requestBody,omitempty" yaml:"requestBody,omitempty" mapstructure:"requestBody"`
	Retry       *RetryPolicy   `json:"retry,omitempty" yaml:"retry,omitempty" mapstructure:"retry"`
}

// Clone copies this instance to a new one
func (r Request) Clone() *Request {
	return &Request{
		URL:         r.URL,
		Method:      r.Method,
		Type:        r.Type,
		Headers:     r.Headers,
		Parameters:  r.Parameters,
		Timeout:     r.Timeout,
		Retry:       r.Retry,
		Security:    r.Security,
		Servers:     r.Servers,
		RequestBody: r.RequestBody,
	}
}

// RequestParameter represents an HTTP request parameter
type RequestParameter struct {
	EncodingObject `yaml:",inline"`

	Name         string            `json:"name,omitempty" yaml:"name,omitempty" mapstructure:"name"`
	ArgumentName string            `json:"argumentName,omitempty" yaml:"argumentName,omitempty" mapstructure:"argumentName,omitempty"`
	In           ParameterLocation `json:"in,omitempty" yaml:"in,omitempty" mapstructure:"in"`
	Schema       *TypeSchema       `json:"schema,omitempty" yaml:"schema,omitempty" mapstructure:"schema"`
}

// TypeSchema represents a serializable object of OpenAPI schema
// that is used for validation
type TypeSchema struct {
	Type        string                `json:"type" yaml:"type" mapstructure:"type"`
	Format      string                `json:"format,omitempty" yaml:"format,omitempty" mapstructure:"format"`
	Pattern     string                `json:"pattern,omitempty" yaml:"pattern,omitempty" mapstructure:"pattern"`
	Nullable    bool                  `json:"nullable,omitempty" yaml:"nullable,omitempty" mapstructure:"nullable"`
	Maximum     *float64              `json:"maximum,omitempty" yaml:"maximum,omitempty" mapstructure:"maximum"`
	Minimum     *float64              `json:"minimum,omitempty," yaml:"minimum,omitempty" mapstructure:"minimum"`
	MaxLength   *int64                `json:"maxLength,omitempty" yaml:"maxLength,omitempty" mapstructure:"maxLength"`
	MinLength   *int64                `json:"minLength,omitempty" yaml:"minLength,omitempty" mapstructure:"minLength"`
	Enum        []string              `json:"enum,omitempty" yaml:"enum,omitempty" mapstructure:"enum"`
	Items       *TypeSchema           `json:"items,omitempty" yaml:"items,omitempty" mapstructure:"items"`
	Properties  map[string]TypeSchema `json:"properties,omitempty" yaml:"properties,omitempty" mapstructure:"properties"`
	Description string                `json:"-" yaml:"-"`
	ReadOnly    bool                  `json:"-" yaml:"-"`
	WriteOnly   bool                  `json:"-" yaml:"-"`
}

// RetryPolicy represents the retry policy of request
type RetryPolicy struct {
	// Number of retry times
	Times uint `json:"times,omitempty" yaml:"times,omitempty" mapstructure:"times"`
	// Delay retry delay in milliseconds
	Delay uint `json:"delay,omitempty" yaml:"delay,omitempty" mapstructure:"delay"`
	// HTTPStatus retries if the remote service returns one of these http status
	HTTPStatus []int `json:"httpStatus,omitempty" yaml:"httpStatus,omitempty" mapstructure:"httpStatus"`
}

// EncodingObject represents the [Encoding Object] that contains serialization strategy for application/x-www-form-urlencoded
//
// [Encoding Object]: https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.1.0.md#encoding-object
type EncodingObject struct {
	// Describes how a specific property value will be serialized depending on its type.
	// See Parameter Object for details on the style property.
	// The behavior follows the same values as query parameters, including default values.
	// This property SHALL be ignored if the request body media type is not application/x-www-form-urlencoded or multipart/form-data.
	// If a value is explicitly defined, then the value of contentType (implicit or explicit) SHALL be ignored
	Style ParameterEncodingStyle `json:"style,omitempty" yaml:"style,omitempty" mapstructure:"style"`
	// When this is true, property values of type array or object generate separate parameters for each value of the array, or key-value-pair of the map.
	// For other types of properties this property has no effect. When style is form, the default value is true. For all other styles, the default value is false.
	// This property SHALL be ignored if the request body media type is not application/x-www-form-urlencoded or multipart/form-data.
	// If a value is explicitly defined, then the value of contentType (implicit or explicit) SHALL be ignored
	Explode *bool `json:"explode,omitempty" yaml:"explode,omitempty" mapstructure:"explode"`
	// By default, reserved characters :/?#[]@!$&'()*+,;= in form field values within application/x-www-form-urlencoded bodies are percent-encoded when sent.
	// AllowReserved allows these characters to be sent as is:
	AllowReserved bool `json:"allowReserved,omitempty" yaml:"allowReserved,omitempty" mapstructure:"allowReserved"`
	// For more complex scenarios, such as nested arrays or JSON in form data, use the contentType keyword to specify the media type for encoding the value of a complex field.
	ContentType []string `json:"contentType,omitempty" yaml:"contentType,omitempty" mapstructure:"contentType"`
	// A map allowing additional information to be provided as headers, for example Content-Disposition.
	// Content-Type is described separately and SHALL be ignored in this section.
	// This property SHALL be ignored if the request body media type is not a multipart.
	Headers map[string]RequestParameter `json:"headers,omitempty" yaml:"headers,omitempty" mapstructure:"headers"`
}

// RequestBody defines flexible request body with content types
type RequestBody struct {
	ContentType string                    `json:"contentType,omitempty" yaml:"contentType,omitempty" mapstructure:"contentType"`
	Schema      *TypeSchema               `json:"schema,omitempty" yaml:"schema,omitempty" mapstructure:"schema"`
	Encoding    map[string]EncodingObject `json:"encoding,omitempty" yaml:"encoding,omitempty" mapstructure:"encoding"`
}

// RESTFunctionInfo extends NDC query function with OpenAPI REST information
type RESTFunctionInfo struct {
	Request             *Request `json:"request" yaml:"request" mapstructure:"request"`
	schema.FunctionInfo `yaml:",inline"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *RESTFunctionInfo) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	rawReq, ok := raw["request"]
	if ok {
		var request Request
		if err := json.Unmarshal(rawReq, &request); err != nil {
			return err
		}
		j.Request = &request
	}

	var function schema.FunctionInfo
	if err := function.UnmarshalJSONMap(raw); err != nil {
		return err
	}

	j.FunctionInfo = function
	return nil
}

// RESTProcedureInfo extends NDC mutation procedure with OpenAPI REST information
type RESTProcedureInfo struct {
	Request              *Request `json:"request" yaml:"request" mapstructure:"request"`
	schema.ProcedureInfo `yaml:",inline"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *RESTProcedureInfo) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	rawReq, ok := raw["request"]
	if ok {
		var request Request
		if err := json.Unmarshal(rawReq, &request); err != nil {
			return err
		}
		j.Request = &request
	}

	var procedure schema.ProcedureInfo
	if err := procedure.UnmarshalJSONMap(raw); err != nil {
		return err
	}

	j.ProcedureInfo = procedure
	return nil
}
