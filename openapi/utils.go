package openapi

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"unicode"

	"github.com/hasura/ndc-rest-schema/schema"
	"github.com/hasura/ndc-rest-schema/utils"
	"github.com/pb33f/libopenapi/datamodel/high/base"
)

var (
	bracketRegexp         = regexp.MustCompile(`[\{\}]`)
	schemaRefNameV2Regexp = regexp.MustCompile(`#/definitions/([\w]+)`)
	schemaRefNameV3Regexp = regexp.MustCompile(`#/components/schemas/([\w]+)`)

	errParameterSchemaEmpty = errors.New("parameter schema is empty")
)

const (
	ContentTypeHeader = "Content-Type"
	ContentTypeJSON   = "application/json"
)

// ConvertOptions represent the common convert options for both OpenAPI v2 and v3
type ConvertOptions struct {
	MethodAlias map[string]string
	TrimPrefix  string
	EnvPrefix   string
	Logger      *slog.Logger
}

func validateConvertOptions(opts *ConvertOptions) (*ConvertOptions, error) {
	logger := slog.Default()
	if opts == nil {
		return &ConvertOptions{
			MethodAlias: getMethodAlias(),
			Logger:      logger,
		}, nil
	}
	if opts.Logger != nil {
		logger = opts.Logger
	}
	return &ConvertOptions{
		MethodAlias: getMethodAlias(opts.MethodAlias),
		TrimPrefix:  opts.TrimPrefix,
		EnvPrefix:   opts.EnvPrefix,
		Logger:      logger,
	}, nil
}

func buildPathMethodName(apiPath string, method string, options *ConvertOptions) string {
	if options.TrimPrefix != "" {
		apiPath = strings.TrimPrefix(apiPath, options.TrimPrefix)
	}
	encodedPath := utils.ToPascalCase(bracketRegexp.ReplaceAllString(strings.TrimLeft(apiPath, "/"), ""))
	if alias, ok := options.MethodAlias[method]; ok {
		method = alias
	}
	return utils.ToCamelCase(method + encodedPath)
}

func getSchemaRefTypeNameV2(name string) string {
	result := schemaRefNameV2Regexp.FindStringSubmatch(name)
	if len(result) < 2 {
		return ""
	}
	return result[1]
}

func getSchemaRefTypeNameV3(name string) string {
	result := schemaRefNameV3Regexp.FindStringSubmatch(name)
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

// getMethodAlias merge method alias map with default value
func getMethodAlias(inputs ...map[string]string) map[string]string {
	methodAlias := map[string]string{
		"get":    "get",
		"post":   "post",
		"put":    "put",
		"patch":  "patch",
		"delete": "delete",
	}
	for _, input := range inputs {
		for k, alias := range input {
			methodAlias[k] = alias
		}
	}
	return methodAlias
}

func buildEnvVariableName(prefix string, names ...string) string {
	if prefix == "" {
		return fmt.Sprintf("{{%s}}", strings.Join(names, "_"))
	}
	return fmt.Sprintf("{{%s_%s}}", prefix, strings.Join(names, "_"))
}

func toSnakeCase(input string) string {
	var sb strings.Builder
	inputLen := len(input)
	for i := 0; i < inputLen; i++ {
		char := rune(input[i])
		if char == '_' || char == '-' {
			sb.WriteRune('_')
			continue
		}
		if unicode.IsDigit(char) || unicode.IsLower(char) {
			sb.WriteRune(char)
			continue
		}

		if unicode.IsUpper(char) {
			if i == 0 {
				sb.WriteRune(unicode.ToLower(char))
				continue
			}
			lastChar := rune(input[i-1])
			if unicode.IsDigit(lastChar) || unicode.IsLower(lastChar) {
				sb.WriteRune('_')
				sb.WriteRune(unicode.ToLower(char))
				continue
			}
			if i < inputLen-1 {
				nextChar := rune(input[i+1])
				if unicode.IsUpper(lastChar) && !unicode.IsUpper(nextChar) {
					sb.WriteRune('_')
					sb.WriteRune(unicode.ToLower(char))
					continue
				}
			}

			sb.WriteRune(unicode.ToLower(char))
		}
	}
	return sb.String()
}

func toConstantCase(input string) string {
	return strings.ToUpper(toSnakeCase(input))
}

func convertSecurities(securities []*base.SecurityRequirement) []map[string][]string {
	var results []map[string][]string
	for _, security := range securities {
		s := convertSecurity(security)
		if s != nil {
			results = append(results, s)
		}
	}
	return results
}

func convertSecurity(security *base.SecurityRequirement) map[string][]string {
	if security == nil {
		return nil
	}
	results := make(map[string][]string)
	for s := security.Requirements.First(); s != nil; s = s.Next() {
		results[s.Key()] = s.Value()
	}
	return results
}
