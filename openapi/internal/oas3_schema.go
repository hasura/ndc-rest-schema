package internal

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	rest "github.com/hasura/ndc-rest-schema/schema"
	"github.com/hasura/ndc-rest-schema/utils"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/pb33f/libopenapi/datamodel/high/base"
)

type oas3SchemaBuilder struct {
	builder   *OAS3Builder
	apiPath   string
	location  rest.ParameterLocation
	writeMode bool
}

func newOAS3SchemaBuilder(builder *OAS3Builder, apiPath string, location rest.ParameterLocation, writeMode bool) *oas3SchemaBuilder {
	return &oas3SchemaBuilder{
		builder:   builder,
		apiPath:   apiPath,
		writeMode: writeMode,
		location:  location,
	}
}

// get and convert an OpenAPI data type to a NDC type
func (oc *oas3SchemaBuilder) getSchemaTypeFromProxy(schemaProxy *base.SchemaProxy, nullable bool, fieldPaths []string) (schema.TypeEncoder, *rest.TypeSchema, bool, error) {
	if schemaProxy == nil {
		return nil, nil, false, errParameterSchemaEmpty(fieldPaths)
	}
	innerSchema := schemaProxy.Schema()
	if innerSchema == nil {
		return nil, nil, false, fmt.Errorf("cannot get schema of $.%s from proxy: %s", strings.Join(fieldPaths, "."), schemaProxy.GetReference())
	}

	var ndcType schema.TypeEncoder
	var typeSchema *rest.TypeSchema
	var isRef bool
	var err error

	rawRefName := schemaProxy.GetReference()

	if rawRefName == "" {
		ndcType, typeSchema, isRef, err = oc.getSchemaType(innerSchema, fieldPaths)
		if err != nil {
			return nil, nil, false, err
		}
	} else if objectName, ok := oc.builder.evaluatingTypes[rawRefName]; ok {
		isRef = true
		ndcType = schema.NewNamedType(objectName)
		typeSchema = &rest.TypeSchema{
			Type:        objectName,
			Description: innerSchema.Description,
		}
	} else {
		// return early object from ref
		refName := getSchemaRefTypeNameV3(rawRefName)
		objectName := utils.ToPascalCase(refName)
		isRef = true
		oc.builder.evaluatingTypes[rawRefName] = objectName

		_, ok := oc.builder.schema.ObjectTypes[objectName]
		if !ok {
			ndcType, typeSchema, _, err = oc.getSchemaType(innerSchema, []string{refName})
			if err != nil {
				return nil, nil, false, err
			}
			typeSchema.Description = innerSchema.Description
		} else {
			ndcType = schema.NewNamedType(objectName)
			typeSchema = &rest.TypeSchema{
				Type:        objectName,
				Description: innerSchema.Description,
			}
		}
	}

	if ndcType == nil {
		return nil, nil, false, nil
	}

	if nullable {
		typeSchema.Nullable = true
		if !isNullableType(ndcType) {
			ndcType = schema.NewNullableType(ndcType)
		}
	}
	return ndcType, typeSchema, isRef, nil
}

// get and convert an OpenAPI data type to a NDC type
func (oc *oas3SchemaBuilder) getSchemaType(typeSchema *base.Schema, fieldPaths []string) (schema.TypeEncoder, *rest.TypeSchema, bool, error) {

	if typeSchema == nil {
		return nil, nil, false, errParameterSchemaEmpty(fieldPaths)
	}

	nullable := typeSchema.Nullable != nil && *typeSchema.Nullable
	if len(typeSchema.AllOf) > 0 {
		enc, ty, isRef, err := oc.buildAllOfAnyOfSchemaType(typeSchema.AllOf, nullable, fieldPaths)
		if err != nil {
			return nil, nil, false, err
		}
		if ty != nil {
			ty.Description = typeSchema.Description
		}
		return enc, ty, isRef, nil
	}

	if len(typeSchema.AnyOf) > 0 {
		enc, ty, isRef, err := oc.buildAllOfAnyOfSchemaType(typeSchema.AnyOf, true, fieldPaths)
		if err != nil {
			return nil, nil, false, err
		}
		if ty != nil {
			ty.Description = typeSchema.Description
		}
		return enc, ty, isRef, nil
	}

	oneOfLength := len(typeSchema.OneOf)
	if oneOfLength == 1 {
		enc, ty, isRef, err := oc.getSchemaTypeFromProxy(typeSchema.OneOf[0], *typeSchema.Nullable, fieldPaths)
		if err != nil {
			return nil, nil, false, err
		}
		if ty != nil {
			ty.Description = typeSchema.Description
		}
		return enc, ty, isRef, nil
	}

	var typeResult *rest.TypeSchema
	var isRef bool
	if oneOfLength > 0 || (typeSchema.AdditionalProperties != nil && (typeSchema.AdditionalProperties.B || typeSchema.AdditionalProperties.A != nil)) {
		typeResult = createSchemaFromOpenAPISchema(typeSchema, string(rest.ScalarJSON))
		return oc.builder.buildScalarJSON(), typeResult, false, nil
	}

	if len(typeSchema.Type) == 0 {
		return nil, nil, false, errParameterSchemaEmpty(fieldPaths)
	}

	var result schema.TypeEncoder
	if len(typeSchema.Type) > 1 || isPrimitiveScalar(typeSchema.Type[0]) {
		scalarName := getScalarFromType(oc.builder.schema, typeSchema.Type, typeSchema.Format, typeSchema.Enum, oc.builder.trimPathPrefix(oc.apiPath), fieldPaths)
		result = schema.NewNamedType(scalarName)
		typeResult = createSchemaFromOpenAPISchema(typeSchema, scalarName)
	} else {
		typeName := typeSchema.Type[0]
		typeResult = createSchemaFromOpenAPISchema(typeSchema, typeName)
		switch typeName {
		case "object":
			refName := utils.StringSliceToPascalCase(fieldPaths)

			if typeSchema.Properties == nil || typeSchema.Properties.IsZero() {
				if typeSchema.AdditionalProperties != nil && (typeSchema.AdditionalProperties.A == nil || !typeSchema.AdditionalProperties.B) {
					return nil, nil, false, nil
				}
				// treat no-property objects as a JSON scalar
				return oc.builder.buildScalarJSON(), &rest.TypeSchema{Type: string(rest.ScalarJSON)}, false, nil
			}

			object := schema.ObjectType{
				Fields: make(schema.ObjectTypeFields),
			}
			readObject := schema.ObjectType{
				Fields: make(schema.ObjectTypeFields),
			}
			writeObject := schema.ObjectType{
				Fields: make(schema.ObjectTypeFields),
			}
			if typeSchema.Description != "" {
				object.Description = &typeSchema.Description
				readObject.Description = &typeSchema.Description
				writeObject.Description = &typeSchema.Description
			}

			typeResult.Properties = make(map[string]rest.TypeSchema)
			for prop := typeSchema.Properties.First(); prop != nil; prop = prop.Next() {
				propName := prop.Key()
				oc.builder.Logger.Debug(
					"property",
					slog.String("name", propName),
					slog.Any("field", fieldPaths))
				nullable := !slices.Contains(typeSchema.Required, propName)
				propType, propApiSchema, _, err := oc.getSchemaTypeFromProxy(prop.Value(), nullable, append(fieldPaths, propName))
				if err != nil {
					return nil, nil, false, err
				}
				if propType == nil {
					continue
				}
				objField := schema.ObjectField{
					Type: propType.Encode(),
				}
				if propApiSchema.Description != "" {
					objField.Description = &propApiSchema.Description
				}

				if (!propApiSchema.ReadOnly && !propApiSchema.WriteOnly) || (!oc.writeMode && propApiSchema.ReadOnly) || (oc.writeMode || propApiSchema.WriteOnly) {
					propApiSchema.Nullable = nullable
					typeResult.Properties[propName] = *propApiSchema
				}
				if !propApiSchema.ReadOnly && !propApiSchema.WriteOnly {
					object.Fields[propName] = objField
				} else if !oc.writeMode && propApiSchema.ReadOnly {
					readObject.Fields[propName] = objField
				} else {
					writeObject.Fields[propName] = objField
				}
			}
			if len(readObject.Fields) == 0 && len(writeObject.Fields) == 0 {
				oc.builder.schema.ObjectTypes[refName] = object
				result = schema.NewNamedType(refName)
			} else {
				for key, field := range object.Fields {
					readObject.Fields[key] = field
					writeObject.Fields[key] = field
				}
				writeRefName := formatWriteObjectName(refName)
				oc.builder.schema.ObjectTypes[refName] = readObject
				oc.builder.schema.ObjectTypes[writeRefName] = writeObject
				if oc.writeMode {
					result = schema.NewNamedType(writeRefName)
				} else {
					result = schema.NewNamedType(refName)
				}
			}
		case "array":
			if typeSchema.Items == nil || typeSchema.Items.A == nil {
				return nil, nil, false, errors.New("array item is empty")
			}

			itemName := getSchemaRefTypeNameV3(typeSchema.Items.A.GetReference())
			if itemName != "" {
				result = schema.NewArrayType(schema.NewNamedType(utils.ToPascalCase(itemName)))
			} else {
				itemSchemaA := typeSchema.Items.A.Schema()
				if itemSchemaA != nil {
					itemSchema, propType, _isRef, err := oc.getSchemaType(itemSchemaA, fieldPaths)
					if err != nil {
						return nil, nil, isRef, err
					}
					if itemSchema != nil {
						result = schema.NewArrayType(itemSchema)
					} else {
						result = schema.NewArrayType(oc.builder.buildScalarJSON())
					}

					typeResult.Items = propType
					isRef = _isRef
				}
			}

			if result == nil {
				return nil, nil, false, fmt.Errorf("cannot parse type reference name: %s", typeSchema.Items.A.GetReference())
			}
		default:
			return nil, nil, false, fmt.Errorf("unsupported schema type %s", typeName)
		}
	}

	return result, typeResult, isRef, nil
}

// Support converting allOf and anyOf to object types with merge strategy
func (oc *oas3SchemaBuilder) buildAllOfAnyOfSchemaType(schemaProxies []*base.SchemaProxy, nullable bool, fieldPaths []string) (schema.TypeEncoder, *rest.TypeSchema, bool, error) {
	proxies, isNullable := evalSchemaProxiesSlice(schemaProxies, oc.location)
	nullable = nullable || isNullable

	if len(proxies) == 1 {
		return oc.getSchemaTypeFromProxy(proxies[0], nullable, fieldPaths)
	}
	readObject := schema.ObjectType{
		Fields: schema.ObjectTypeFields{},
	}
	writeObject := schema.ObjectType{
		Fields: schema.ObjectTypeFields{},
	}
	typeSchema := &rest.TypeSchema{
		Type:       "object",
		Properties: map[string]rest.TypeSchema{},
	}

	for i, item := range proxies {
		itemFieldPaths := append(fieldPaths, fmt.Sprint(i))
		enc, ty, isRef, err := oc.getSchemaTypeFromProxy(item, nullable, itemFieldPaths)
		if err != nil {
			return nil, nil, false, err
		}

		name := getNamedType(enc, true, "")
		writeName := formatWriteObjectName(name)
		isObject := !isPrimitiveScalar(ty.Type) && ty.Type != "array"
		if isObject {
			if _, ok := oc.builder.schema.ScalarTypes[name]; ok {
				isObject = false
			}
		}
		if !isObject {
			if !isRef {
				delete(oc.builder.schema.ObjectTypes, name)
				delete(oc.builder.schema.ObjectTypes, writeName)
				delete(oc.builder.schema.ScalarTypes, name)
			}
			// TODO: should we keep the original anyOf or allOf type schema
			ty = &rest.TypeSchema{
				Type:        string(rest.ScalarJSON),
				Nullable:    ty.Nullable,
				Description: ty.Description,
			}
			return oc.builder.buildScalarJSON(), ty, false, nil
		}

		readObj, ok := oc.builder.schema.ObjectTypes[name]
		if ok {
			if readObject.Description == nil && readObj.Description != nil {
				readObject.Description = readObj.Description
				if ty.Description == "" {
					ty.Description = *readObj.Description
				}
			}
			for k, v := range readObj.Fields {
				if _, ok := readObject.Fields[k]; !ok {
					readObject.Fields[k] = v
				}
			}
		}
		writeObj, ok := oc.builder.schema.ObjectTypes[writeName]
		if ok {
			if writeObject.Description == nil && writeObj.Description != nil {
				writeObject.Description = writeObj.Description
			}
			for k, v := range writeObj.Fields {
				if _, ok := writeObject.Fields[k]; !ok {
					writeObject.Fields[k] = v
				}
			}
		}
		if !isRef {
			delete(oc.builder.schema.ObjectTypes, name)
			delete(oc.builder.schema.ObjectTypes, writeName)
		}
	}

	refName := utils.ToPascalCase(strings.Join(fieldPaths, " "))
	writeRefName := formatWriteObjectName(refName)
	if len(readObject.Fields) > 0 {
		oc.builder.schema.ObjectTypes[refName] = readObject
	}
	if len(writeObject.Fields) > 0 {
		oc.builder.schema.ObjectTypes[writeRefName] = writeObject
	}

	if oc.writeMode && len(writeObject.Fields) > 0 {
		refName = writeRefName
	}
	if len(typeSchema.Properties) == 0 {
		typeSchema = &rest.TypeSchema{
			Type: refName,
		}
	}
	return schema.NewNamedType(refName), typeSchema, false, nil
}
