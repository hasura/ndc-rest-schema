package openapi

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	rest "github.com/hasura/ndc-rest-schema/schema"
	"github.com/hasura/ndc-rest-schema/utils"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"gopkg.in/yaml.v3"
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

var defaultScalarTypes = map[string]*schema.ScalarType{
	"String": {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	"Int": {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationInteger().Encode(),
	},
	"Float": {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationNumber().Encode(),
	},
	"Boolean": {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationBoolean().Encode(),
	},
	"JSON": schema.NewScalarType(),
}

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

func getScalarFromType(sm *rest.NDCRestSchema, name string, enumNodes []*yaml.Node, apiPath string, fieldPaths []string) string {
	var scalarName string
	var scalarType *schema.ScalarType

	switch name {
	case "boolean":
		scalarName = "Boolean"
		scalarType = defaultScalarTypes[scalarName]
	case "integer":
		scalarName = "Int"
		scalarType = defaultScalarTypes[scalarName]
	case "number":
		scalarName = "Float"
		scalarType = defaultScalarTypes[scalarName]
	case "string":
		scalarName = "String"
		scalarType = defaultScalarTypes[scalarName]

		schemaEnumLength := len(enumNodes)
		if schemaEnumLength > 0 {
			enums := make([]string, schemaEnumLength)
			for i, enum := range enumNodes {
				enums[i] = enum.Value
			}
			scalarType = schema.NewScalarType()
			scalarType.Representation = schema.NewTypeRepresentationEnum(enums).Encode()

			// build scalar name strategies
			// 1. combine resource name and field name
			apiPath = strings.TrimPrefix(apiPath, "/")
			if apiPath != "" {
				apiPaths := strings.Split(apiPath, "/")
				resourceName := fieldPaths[0]
				if len(apiPaths) > 0 {
					resourceName = apiPaths[0]
				}
				enumName := "Enum"
				if len(fieldPaths) > 1 {
					enumName = fieldPaths[len(fieldPaths)-1]
				}

				scalarName = utils.StringSliceToPascalCase([]string{resourceName, enumName})
				if canSetEnumToSchema(sm, scalarName, enums) {
					sm.ScalarTypes[scalarName] = *scalarType
					return scalarName
				}
			}

			// 2. if the scalar type exists, fallback to field paths
			scalarName = utils.StringSliceToPascalCase(fieldPaths)
			if canSetEnumToSchema(sm, scalarName, enums) {
				sm.ScalarTypes[scalarName] = *scalarType
				return scalarName
			}

			// 3. Reuse above name with Enum suffix
			scalarName = fmt.Sprintf("%sEnum", scalarName)
		}
	default:
		scalarName = "JSON"
		scalarType = defaultScalarTypes[scalarName]
	}

	if _, ok := sm.ScalarTypes[scalarName]; !ok {
		sm.ScalarTypes[scalarName] = *scalarType
	}
	return scalarName
}

func canSetEnumToSchema(sm *rest.NDCRestSchema, scalarName string, enums []string) bool {
	existedScalar, ok := sm.ScalarTypes[scalarName]
	if !ok {
		return true
	}

	existedEnum, err := existedScalar.Representation.AsEnum()
	if err == nil && utils.SliceUnorderedEqual(enums, existedEnum.OneOf) {
		return true
	}

	return false
}

// ParseTypeSchemaFromOpenAPISchema creates a TypeSchema from OpenAPI schema object
func ParseTypeSchemaFromOpenAPISchema(input *base.Schema, typeName string) *rest.TypeSchema {
	if input == nil {
		return nil
	}
	ps := &rest.TypeSchema{}
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

func convertSecurities(securities []*base.SecurityRequirement) rest.AuthSecurities {
	var results rest.AuthSecurities
	for _, security := range securities {
		s := convertSecurity(security)
		if s != nil {
			results = append(results, s)
		}
	}
	return results
}

func convertSecurity(security *base.SecurityRequirement) rest.AuthSecurity {
	if security == nil {
		return nil
	}
	results := make(map[string][]string)
	for s := security.Requirements.First(); s != nil; s = s.Next() {
		v := s.Value()
		if v == nil {
			v = []string{}
		}
		results[s.Key()] = v
	}
	return results
}
