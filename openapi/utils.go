package openapi

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"

	rest "github.com/hasura/ndc-rest-schema/schema"
	"github.com/hasura/ndc-rest-schema/utils"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"gopkg.in/yaml.v3"
)

var (
	bracketRegexp         = regexp.MustCompile(`[\{\}]`)
	schemaRefNameV2Regexp = regexp.MustCompile(`^#/definitions/([a-zA-Z0-9\.\-_]+)$`)
	schemaRefNameV3Regexp = regexp.MustCompile(`^#/components/schemas/([a-zA-Z0-9\.\-_]+)$`)

	errParameterSchemaEmpty = errors.New("parameter schema is empty")
)

var defaultScalarTypes = map[rest.ScalarType]*schema.ScalarType{
	rest.ScalarTypeString: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	rest.ScalarTypeInt: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationInteger().Encode(),
	},
	// big integer can be encoded as string
	rest.ScalarTypeBigInt: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
	},
	rest.ScalarTypeFloat: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationNumber().Encode(),
	},
	rest.ScalarTypeBoolean: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationBoolean().Encode(),
	},
	rest.ScalarTypeJSON: schema.NewScalarType(),
	// string format variants https://swagger.io/docs/specification/data-models/data-types/#string
	// string with date format
	rest.ScalarTypeDate: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	// string with date-time format
	rest.ScalarTypeDateTime: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	// string with byte format
	rest.ScalarTypeBase64: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	rest.ScalarTypeEmail: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	rest.ScalarTypeURI: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	rest.ScalarTypeUUID: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	rest.ScalarTypeIPV4: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	rest.ScalarTypeIPV6: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	// unix-time the timestamp integer which is measured in seconds since the Unix epoch
	rest.ScalarTypeUnixTime: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationInteger().Encode(),
	},
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

func getScalarFromType(sm *rest.NDCRestSchema, names []string, format string, enumNodes []*yaml.Node, apiPath string, fieldPaths []string) string {
	var scalarName string
	var scalarType *schema.ScalarType

	if len(names) != 1 {
		scalarName = "JSON"
		scalarType = defaultScalarTypes[rest.ScalarTypeJSON]
	} else {
		switch names[0] {
		case "boolean":
			scalarName = string(rest.ScalarTypeBoolean)
			scalarType = defaultScalarTypes[rest.ScalarTypeBoolean]
		case "integer":
			switch format {
			case "unix-time":
				scalarName = string(rest.ScalarTypeUnixTime)
				scalarType = defaultScalarTypes[rest.ScalarTypeUnixTime]
			case "int64":
				scalarName = string(rest.ScalarTypeBigInt)
				scalarType = defaultScalarTypes[rest.ScalarTypeBigInt]
			default:
				scalarName = "Int"
				scalarType = defaultScalarTypes[rest.ScalarTypeInt]
			}
		case "long":
			scalarName = string(rest.ScalarTypeBigInt)
			scalarType = defaultScalarTypes[rest.ScalarTypeBigInt]
		case "number":
			scalarName = "Float"
			scalarType = defaultScalarTypes[rest.ScalarTypeFloat]
		case "string":
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
				if _, ok := sm.ScalarTypes[scalarName]; !ok {
					sm.ScalarTypes[scalarName] = *scalarType
				}
				return scalarName
			}

			switch format {
			case "date":
				scalarName = string(rest.ScalarTypeDate)
				scalarType = defaultScalarTypes[rest.ScalarTypeDate]
			case "date-time":
				scalarName = string(rest.ScalarTypeDateTime)
				scalarType = defaultScalarTypes[rest.ScalarTypeDateTime]
			case "byte", "binary", "file":
				scalarName = string(rest.ScalarTypeBase64)
				scalarType = defaultScalarTypes[rest.ScalarTypeBase64]
			case "uuid":
				scalarName = string(rest.ScalarTypeUUID)
				scalarType = defaultScalarTypes[rest.ScalarTypeUUID]
			case "uri":
				scalarName = string(rest.ScalarTypeURI)
				scalarType = defaultScalarTypes[rest.ScalarTypeURI]
			case "ipv4":
				scalarName = string(rest.ScalarTypeIPV4)
				scalarType = defaultScalarTypes[rest.ScalarTypeIPV4]
			case "ipv6":
				scalarName = string(rest.ScalarTypeIPV6)
				scalarType = defaultScalarTypes[rest.ScalarTypeIPV6]
			default:
				scalarName = string(rest.ScalarTypeString)
				scalarType = defaultScalarTypes[rest.ScalarTypeString]
			}
		default:
			scalarName = string(rest.ScalarTypeJSON)
			scalarType = defaultScalarTypes[rest.ScalarTypeJSON]
		}
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

func isPrimitiveScalar(name string) bool {
	return slices.Contains([]string{"boolean", "integer", "number", "string", "file", "long"}, name)
}
