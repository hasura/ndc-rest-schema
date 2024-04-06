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

var defaultScalarTypes = map[rest.ScalarName]*schema.ScalarType{
	rest.ScalarBoolean: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationBoolean().Encode(),
	},
	rest.ScalarString: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	rest.ScalarInt32: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationInt32().Encode(),
	},
	rest.ScalarInt64: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationInt64().Encode(),
	},
	rest.ScalarFloat32: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationFloat32().Encode(),
	},
	rest.ScalarFloat64: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationFloat64().Encode(),
	},
	rest.ScalarJSON: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationJSON().Encode(),
	},
	// string format variants https://swagger.io/docs/specification/data-models/data-types/#string
	// string with date format
	rest.ScalarDate: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	// string with date-time format
	rest.ScalarTimestamp: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	rest.ScalarTimestampTZ: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	// string with byte format
	rest.ScalarBytes: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	rest.ScalarEmail: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	rest.ScalarURI: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	rest.ScalarUUID: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	rest.ScalarIPV4: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	rest.ScalarIPV6: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationString().Encode(),
	},
	// unix-time the timestamp integer which is measured in seconds since the Unix epoch
	rest.ScalarUnixTime: {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationInt32().Encode(),
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
		scalarType = defaultScalarTypes[rest.ScalarJSON]
	} else {
		switch names[0] {
		case "boolean":
			scalarName = string(rest.ScalarBoolean)
			scalarType = defaultScalarTypes[rest.ScalarBoolean]
		case "integer":
			switch format {
			case "unix-time":
				scalarName = string(rest.ScalarUnixTime)
				scalarType = defaultScalarTypes[rest.ScalarUnixTime]
			case "int64":
				scalarName = string(rest.ScalarInt64)
				scalarType = defaultScalarTypes[rest.ScalarInt64]
			default:
				scalarName = "Int32"
				scalarType = defaultScalarTypes[rest.ScalarInt32]
			}
		case "long":
			scalarName = string(rest.ScalarInt64)
			scalarType = defaultScalarTypes[rest.ScalarInt64]
		case "number":
			switch format {
			case "float":
				scalarName = string(rest.ScalarFloat32)
				scalarType = defaultScalarTypes[rest.ScalarFloat32]
			default:
				scalarName = string(rest.ScalarFloat64)
				scalarType = defaultScalarTypes[rest.ScalarFloat64]
			}
		case "file":
			scalarName = string(rest.ScalarBytes)
			scalarType = defaultScalarTypes[rest.ScalarBytes]
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
				scalarName = string(rest.ScalarDate)
				scalarType = defaultScalarTypes[rest.ScalarDate]
			case "date-time":
				scalarName = string(rest.ScalarTimestamp)
				scalarType = defaultScalarTypes[rest.ScalarTimestamp]
			case "byte", "binary":
				scalarName = string(rest.ScalarBytes)
				scalarType = defaultScalarTypes[rest.ScalarBytes]
			case "uuid":
				scalarName = string(rest.ScalarUUID)
				scalarType = defaultScalarTypes[rest.ScalarUUID]
			case "uri":
				scalarName = string(rest.ScalarURI)
				scalarType = defaultScalarTypes[rest.ScalarURI]
			case "ipv4":
				scalarName = string(rest.ScalarIPV4)
				scalarType = defaultScalarTypes[rest.ScalarIPV4]
			case "ipv6":
				scalarName = string(rest.ScalarIPV6)
				scalarType = defaultScalarTypes[rest.ScalarIPV6]
			default:
				scalarName = string(rest.ScalarString)
				scalarType = defaultScalarTypes[rest.ScalarString]
			}
		default:
			scalarName = string(rest.ScalarJSON)
			scalarType = defaultScalarTypes[rest.ScalarJSON]
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

func createSchemaFromOpenAPISchema(input *base.Schema, typeName string) *rest.TypeSchema {
	ps := &rest.TypeSchema{}
	if input == nil {
		return ps
	}
	if typeName != "" {
		ps.Type = typeName
	} else {
		if len(input.Type) > 1 {
			ps.Type = string(rest.ScalarJSON)
		} else if len(input.Type) > 0 {
			ps.Type = input.Type[0]
		}
		ps.Format = input.Format
	}
	ps.Pattern = input.Pattern
	ps.Maximum = input.Maximum
	ps.Minimum = input.Minimum
	ps.MaxLength = input.MaxLength
	ps.MinLength = input.MinLength
	ps.Description = input.Description

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

// sort request parameters in order: in -> name
func sortRequestParameters(input []rest.RequestParameter) []rest.RequestParameter {
	slices.SortFunc(input, func(a rest.RequestParameter, b rest.RequestParameter) int {
		if result := strings.Compare(string(a.In), string(b.In)); result != 0 {
			return result
		}
		return strings.Compare(a.Name, b.Name)
	})

	return input
}

func getNamedType(typeSchema schema.TypeEncoder, defaultValue string) string {
	switch ty := typeSchema.(type) {
	case *schema.NullableType:
		return getNamedType(ty.UnderlyingType.Interface(), defaultValue)
	case *schema.NamedType:
		return ty.Name
	default:
		return defaultValue
	}
}
