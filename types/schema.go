package types

import (
	"encoding/json"
	"errors"

	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/pb33f/libopenapi/datamodel/high/base"
)

// NDCRestSettings represent global settings of the REST API, including base URL, headers, etc...
type NDCRestSettings struct {
	Request
	Version string `json:"version,omitempty" yaml:"version,omitempty" mapstructure:"version"`
}

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
	URL        string             `json:"url,omitempty" yaml:"url,omitempty" mapstructure:"url"`
	Method     string             `json:"method,omitempty" yaml:"method,omitempty" mapstructure:"method"`
	Type       RequestType        `json:"type,omitempty" yaml:"type,omitempty" mapstructure:"type"`
	Headers    map[string]string  `json:"headers,omitempty" yaml:"headers,omitempty" mapstructure:"headers"`
	Parameters []RequestParameter `json:"parameters,omitempty" yaml:"parameters,omitempty" mapstructure:"parameters"`
	// configure the request timeout in seconds, default 30s
	Timeout uint `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout"`
}

// RequestParameter represents an HTTP request parameter
type RequestParameter struct {
	Name     string            `json:"name" yaml:"name" mapstructure:"name"`
	In       ParameterLocation `json:"in" yaml:"in" mapstructure:"in"`
	Required bool              `json:"required" yaml:"required" mapstructure:"required"`
	Schema   *TypeSchema       `json:"schema,omitempty" yaml:"schema,omitempty" mapstructure:"schema"`
}

// TypeSchema represents a serializable object of OpenAPI schema
// that is used for validation
type TypeSchema struct {
	Type       string                `json:"type" yaml:"type" mapstructure:"type"`
	Format     string                `json:"format,omitempty" yaml:"format,omitempty" mapstructure:"format"`
	Pattern    string                `json:"pattern,omitempty" yaml:"pattern,omitempty" mapstructure:"pattern"`
	Nullable   *bool                 `json:"nullable,omitempty" yaml:"nullable,omitempty" mapstructure:"nullable"`
	Maximum    *float64              `json:"maximum,omitempty" yaml:"maximum,omitempty" mapstructure:"maximum"`
	Minimum    *float64              `json:"minimum,omitempty," yaml:"minimum,omitempty" mapstructure:"minimum"`
	MaxLength  *int64                `json:"maxLength,omitempty" yaml:"maxLength,omitempty" mapstructure:"maxLength"`
	MinLength  *int64                `json:"minLength,omitempty" yaml:"minLength,omitempty" mapstructure:"minLength"`
	Enum       []string              `json:"enum,omitempty" yaml:"enum,omitempty" mapstructure:"enum"`
	Items      *TypeSchema           `json:"items,omitempty" yaml:"items,omitempty" mapstructure:"items"`
	Properties map[string]TypeSchema `json:"properties,omitempty" yaml:"properties,omitempty" mapstructure:"properties"`
}

// FromOpenAPIv3Schema applies value from OpenAPI v3 schema object
func (ps *TypeSchema) FromOpenAPIv3Schema(input *base.Schema, typeName string) *TypeSchema {
	ps.Type = typeName
	ps.Format = input.Format
	ps.Pattern = input.Pattern
	ps.Nullable = input.Nullable
	ps.Maximum = input.Maximum
	ps.Minimum = input.Minimum
	ps.MaxLength = input.MaxLength
	ps.MinLength = input.MinLength
	enumLength := len(input.Enum)
	if enumLength > 0 {
		ps.Enum = make([]string, enumLength)
		for i, enum := range input.Enum {
			ps.Enum[i] = enum.Value
		}
	}

	return ps
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
		return nil
	}

	rawReq, ok := raw["request"]
	if !ok {
		return errors.New("RESTFunctionInfo.request is required")
	}
	var request Request
	if err := json.Unmarshal(rawReq, &request); err != nil {
		return err
	}

	var function schema.FunctionInfo
	if err := function.UnmarshalJSONMap(raw); err != nil {
		return err
	}

	j.Request = &request
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
		return nil
	}

	rawReq, ok := raw["request"]
	if !ok {
		return errors.New("RESTProcedureInfo.request is required")
	}
	var request Request
	if err := json.Unmarshal(rawReq, &request); err != nil {
		return err
	}

	var procedure schema.ProcedureInfo
	if err := procedure.UnmarshalJSONMap(raw); err != nil {
		return err
	}

	j.Request = &request
	j.ProcedureInfo = procedure
	return nil
}
