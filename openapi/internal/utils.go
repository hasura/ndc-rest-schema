package internal

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	rest "github.com/hasura/ndc-rest-schema/schema"
	"github.com/hasura/ndc-rest-schema/utils"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"gopkg.in/yaml.v3"
)

func applyConvertOptions(opts ConvertOptions) *ConvertOptions {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	opts.MethodAlias = getMethodAlias(opts.MethodAlias)
	return &opts
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
			scalarName = string(rest.ScalarBinary)
			scalarType = defaultScalarTypes[rest.ScalarBinary]
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
				scalarName = string(rest.ScalarTimestampTZ)
				scalarType = defaultScalarTypes[rest.ScalarTimestampTZ]
			case "byte", "base64":
				scalarName = string(rest.ScalarBytes)
				scalarType = defaultScalarTypes[rest.ScalarBytes]
			case "binary":
				scalarName = string(rest.ScalarBinary)
				scalarType = defaultScalarTypes[rest.ScalarBinary]
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
	ps.ReadOnly = input.ReadOnly != nil && *input.ReadOnly
	ps.WriteOnly = input.WriteOnly != nil && *input.WriteOnly

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

// check if the OAS type is a scalar
func isPrimitiveScalar(name string) bool {
	return slices.Contains([]string{"boolean", "integer", "number", "string", "file", "long"}, name)
}

func isOASType(name string) bool {
	return slices.Contains([]string{"boolean", "integer", "number", "string", "file", "long", "array", "object"}, name)
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

// get the inner named type of the type encoder
func getNamedType(typeSchema schema.TypeEncoder, recursive bool, defaultValue string) string {
	switch ty := typeSchema.(type) {
	case *schema.NullableType:
		return getNamedType(ty.UnderlyingType.Interface(), recursive, defaultValue)
	case *schema.ArrayType:
		if !recursive {
			return defaultValue
		}
		return getNamedType(ty.ElementType.Interface(), recursive, defaultValue)
	case *schema.NamedType:
		return ty.Name
	default:
		return defaultValue
	}
}

func isNullableType(input schema.TypeEncoder) bool {
	_, ok := input.(*schema.NullableType)
	return ok
}

// encodeHeaderArgumentName encodes header key to NDC schema field name
func encodeHeaderArgumentName(name string) string {
	return fmt.Sprintf("header%s", utils.ToPascalCase(name))
}

func setDefaultSettings(settings *rest.NDCRestSettings, opts *ConvertOptions) {
	settings.Timeout = rest.NewEnvIntTemplate(rest.EnvTemplate{
		Name: utils.StringSliceToConstantCase([]string{opts.EnvPrefix, "TIMEOUT"}),
	})
	settings.Retry = &rest.RetryPolicySetting{
		Times: *rest.NewEnvIntTemplate(rest.EnvTemplate{
			Name: utils.StringSliceToConstantCase([]string{opts.EnvPrefix, "RETRY_TIMES"}),
		}),
		Delay: *rest.NewEnvIntTemplate(rest.EnvTemplate{
			Name: utils.StringSliceToConstantCase([]string{opts.EnvPrefix, "RETRY_DELAY"}),
		}),
		HTTPStatus: *rest.NewEnvIntsTemplate(rest.EnvTemplate{
			Name: utils.StringSliceToConstantCase([]string{opts.EnvPrefix, "RETRY_HTTP_STATUS"}),
		}),
	}
}

// evaluate and filter invalid types in allOf, anyOf or oneOf schemas
func evalSchemaProxiesSlice(schemaProxies []*base.SchemaProxy, location rest.ParameterLocation) ([]*base.SchemaProxy, bool) {
	var results []*base.SchemaProxy
	nullable := false
	for _, proxy := range schemaProxies {
		if proxy == nil {
			continue
		}
		sc := proxy.Schema()
		if sc == nil || (len(sc.Type) == 0 && len(sc.AllOf) == 0 && len(sc.AnyOf) == 0 && len(sc.OneOf) == 0) {
			continue
		}

		switch location {
		case rest.InQuery:
			// empty string enum is considered as nullable, e.g. key1=&key2=
			// however, it's redundant and prevents the tool converting correct types
			if sc.Type[0] == "string" && len(sc.Enum) == 1 && (sc.Enum[0] == nil || sc.Enum[0].Value == "") {
				nullable = true
				continue
			}
		}
		results = append(results, proxy)
	}
	return results, nullable
}

func cleanUnusedSchemaTypes(schema *rest.NDCRestSchema, usageCounter *TypeUsageCounter) {
	for key := range schema.ObjectTypes {
		cleanUnusedObjectType(schema, usageCounter, key)
	}
	for key := range schema.ScalarTypes {
		if usageCounter.Get(key) == 0 {
			delete(schema.ScalarTypes, key)
		}
	}
}

// recursively clean unused objects as well as their inner properties
func cleanUnusedObjectType(schema *rest.NDCRestSchema, usageCounter *TypeUsageCounter, key string) {
	object, ok := schema.ObjectTypes[key]
	if !ok {
		return
	}
	if usageCounter.Get(key) > 0 {
		return
	}
	delete(schema.ObjectTypes, key)
	for _, elem := range object.Fields {
		elemName := getNamedType(elem.Type.Interface(), true, "")
		if elemName == "" {
			continue
		}
		usageCounter.Decrease(elemName)
		if usageCounter.Get(elemName) <= 0 {
			cleanUnusedObjectType(schema, usageCounter, elemName)
		}
	}
}

func formatWriteObjectName(name string) string {
	return fmt.Sprintf("%sInput", name)
}

func errParameterSchemaEmpty(fieldPaths []string) error {
	return fmt.Errorf("parameter schema of $.%s is empty", strings.Join(fieldPaths, "."))
}

func getObjectTypeFromSchemaType(objectTypes schema.SchemaResponseObjectTypes, schemaType schema.Type) (*schema.ObjectType, string, error) {
	iSchemaType, err := schemaType.InterfaceT()

	switch st := iSchemaType.(type) {
	case *schema.NullableType:
		return getObjectTypeFromSchemaType(objectTypes, st.UnderlyingType)
	case *schema.NamedType:
		objectType, ok := objectTypes[st.Name]
		if !ok {
			return nil, "", fmt.Errorf("expect object type body, got %s", st.Name)
		}

		return &objectType, st.Name, nil
	case *schema.ArrayType:
		return nil, "", fmt.Errorf("expect named type body, got %s", schemaType)
	default:
		return nil, "", err
	}
}
