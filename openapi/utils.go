package openapi

import (
	"errors"
	"regexp"
	"strings"

	"github.com/hasura/ndc-rest-schema/schema"
	"github.com/hasura/ndc-rest-schema/utils"
	"github.com/pb33f/libopenapi/datamodel/high/base"
)

var (
	bracketRegexp       = regexp.MustCompile(`[\{\}]`)
	schemaRefNameRegexp = regexp.MustCompile(`#/components/schemas/([\w]+)`)

	errParameterSchemaEmpty = errors.New("parameter schema is empty")
)

const (
	ContentTypeHeader = "Content-Type"
	ContentTypeJSON   = "application/json"
)

func buildPathMethodName(apiPath string, method string) string {
	encodedPath := utils.ToPascalCase(bracketRegexp.ReplaceAllString(strings.TrimLeft(apiPath, "/"), ""))
	if method == "get" {
		return encodedPath
	}
	return utils.StringSliceToPascalCase([]string{method, encodedPath})
}

func getSchemaRefTypeName(name string) string {
	result := schemaRefNameRegexp.FindStringSubmatch(name)
	if len(result) < 2 {
		return ""
	}
	return result[1]
}

func getScalarNameFromType(name string) string {
	switch name {
	case "boolean":
		return "Boolean"
	case "integer":
		return "Int"
	case "number":
		return "Float"
	case "string":
		return "String"
	default:
		return "JSON"
	}
}

// ParseTypeSchemaFromOpenAPISchema creates a TypeSchema from OpenAPI schema object
func ParseTypeSchemaFromOpenAPISchema(input *base.Schema, typeName string) *schema.TypeSchema {
	if input == nil {
		return nil
	}
	ps := &schema.TypeSchema{}
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
