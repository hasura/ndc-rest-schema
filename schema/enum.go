package schema

import (
	"encoding/json"
	"fmt"
	"slices"
)

// SchemaSpecType represents the spec enum of schema
type SchemaSpecType string

const (
	OpenAPIv3Spec SchemaSpecType = "openapi3"
	OpenAPIv2Spec SchemaSpecType = "openapi2"
	NDCSpec       SchemaSpecType = "ndc"
)

var schemaSpecType_enums = []SchemaSpecType{OpenAPIv3Spec, OpenAPIv2Spec, NDCSpec}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SchemaSpecType) UnmarshalJSON(b []byte) error {
	var rawResult string
	if err := json.Unmarshal(b, &rawResult); err != nil {
		return err
	}

	result, err := ParseSchemaSpecType(rawResult)
	if err != nil {
		return err
	}

	*j = result
	return nil
}

// ParseSchemaSpecType parses SchemaSpecType from string
func ParseSchemaSpecType(value string) (SchemaSpecType, error) {
	result := SchemaSpecType(value)
	if !slices.Contains(schemaSpecType_enums, result) {
		return result, fmt.Errorf("invalid SchemaSpecType. Expected %+v, got <%s>", schemaSpecType_enums, value)
	}
	return result, nil
}

// RequestType represents the request type enum
type RequestType string

const (
	RequestTypeREST         RequestType = "rest"
	RequestTypeHasuraAction RequestType = "hasura_action"
)

var requestType_enums = []RequestType{RequestTypeREST, RequestTypeHasuraAction}

// UnmarshalJSON implements json.Unmarshaler.
func (j *RequestType) UnmarshalJSON(b []byte) error {
	var rawResult string
	if err := json.Unmarshal(b, &rawResult); err != nil {
		return err
	}

	result, err := ParseRequestType(rawResult)
	if err != nil {
		return err
	}

	*j = result
	return nil
}

// ParseRequestType parses RequestType from string
func ParseRequestType(value string) (RequestType, error) {
	result := RequestType(value)
	if !slices.Contains(requestType_enums, result) {
		return result, fmt.Errorf("invalid RequestType. Expected %+v, got <%s>", schemaSpecType_enums, value)
	}
	return result, nil
}

// SchemaFileFormat represents the file format enum for NDC REST schema file
type SchemaFileFormat string

const (
	SchemaFileJSON SchemaFileFormat = "json"
	SchemaFileYAML SchemaFileFormat = "yaml"
)

var schemaFileFormat_enums = []SchemaFileFormat{SchemaFileYAML, SchemaFileJSON}

// UnmarshalJSON implements json.Unmarshaler.
func (j *SchemaFileFormat) UnmarshalJSON(b []byte) error {
	var rawResult string
	if err := json.Unmarshal(b, &rawResult); err != nil {
		return err
	}

	result, err := ParseSchemaFileFormat(rawResult)
	if err != nil {
		return err
	}

	*j = result
	return nil
}

// ParseSchemaFileFormat parse SchemaFileFormat from file extension
func ParseSchemaFileFormat(extension string) (SchemaFileFormat, error) {
	result := SchemaFileFormat(extension)
	if !slices.Contains(schemaFileFormat_enums, result) {
		return result, fmt.Errorf("invalid SchemaFileFormat. Expected %+v, got <%s>", schemaFileFormat_enums, extension)
	}
	return result, nil
}

// ParameterLocation is [the location] of the parameter.
// Possible values are "query", "header", "path" or "cookie".
//
// [the location]: https://swagger.io/specification/#parameter-object
type ParameterLocation string

const (
	InQuery  ParameterLocation = "query"
	InHeader ParameterLocation = "header"
	InPath   ParameterLocation = "path"
	InCookie ParameterLocation = "cookie"
	InBody   ParameterLocation = "body"
)

var parameterLocation_enums = []ParameterLocation{InQuery, InHeader, InPath, InCookie, InBody}

// UnmarshalJSON implements json.Unmarshaler.
func (j *ParameterLocation) UnmarshalJSON(b []byte) error {
	var rawResult string
	if err := json.Unmarshal(b, &rawResult); err != nil {
		return err
	}

	result, err := ParseParameterLocation(rawResult)
	if err != nil {
		return err
	}

	*j = result
	return nil
}

// ParseParameterLocation parse ParameterLocation from string
func ParseParameterLocation(input string) (ParameterLocation, error) {
	result := ParameterLocation(input)
	if !slices.Contains(parameterLocation_enums, result) {
		return result, fmt.Errorf("invalid ParameterLocation. Expected %+v, got <%s>", parameterLocation_enums, input)
	}
	return result, nil
}
