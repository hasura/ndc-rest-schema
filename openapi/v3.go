package openapi

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	rest "github.com/hasura/ndc-rest-schema/schema"
	"github.com/hasura/ndc-rest-schema/utils"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

type openAPIv3Builder struct {
	schema *rest.NDCRestSchema
	*ConvertOptions
}

// OpenAPIv3ToNDCSchema converts OpenAPI v3 JSON bytes to NDC REST schema
func OpenAPIv3ToNDCSchema(input []byte, options *ConvertOptions) (*rest.NDCRestSchema, []error) {
	opts, err := validateConvertOptions(options)
	if err != nil {
		return nil, []error{err}
	}

	document, err := libopenapi.NewDocument(input)
	if err != nil {
		return nil, []error{err}
	}

	docModel, errs := document.BuildV3Model()
	// The errors won’t prevent the model from building
	if docModel == nil && len(errs) > 0 {
		return nil, errs
	}

	if docModel.Model.Paths == nil || docModel.Model.Paths.PathItems == nil || docModel.Model.Paths.PathItems.IsZero() {
		return nil, append(errs, errors.New("there is no API to be converted"))
	}

	converter := &openAPIv3Builder{
		schema:         rest.NewNDCRestSchema(),
		ConvertOptions: opts,
	}
	setDefaultSettings(converter.schema.Settings, opts)

	if docModel.Model.Info != nil {
		converter.schema.Settings.Version = docModel.Model.Info.Version
	}

	converter.schema.Settings.Servers = converter.convertServers(docModel.Model.Servers)

	if docModel.Model.Components != nil && docModel.Model.Components.Schemas != nil {
		for cSchema := docModel.Model.Components.Schemas.First(); cSchema != nil; cSchema = cSchema.Next() {
			if err := converter.convertComponentSchemas(cSchema); err != nil {
				return nil, append(errs, err)
			}
		}
	}
	for iterPath := docModel.Model.Paths.PathItems.First(); iterPath != nil; iterPath = iterPath.Next() {
		if err := converter.pathToNDCOperations(iterPath); err != nil {
			return nil, append(errs, err)
		}
	}

	if docModel.Model.Components.SecuritySchemes != nil {
		converter.schema.Settings.SecuritySchemes = make(map[string]rest.SecurityScheme)
		for scheme := docModel.Model.Components.SecuritySchemes.First(); scheme != nil; scheme = scheme.Next() {
			err := converter.convertSecuritySchemes(scheme)
			if err != nil {
				return nil, append(errs, err)
			}
		}
	}
	converter.schema.Settings.Security = convertSecurities(docModel.Model.Security)
	return converter.schema, nil
}

func (oc *openAPIv3Builder) convertServers(servers []*v3.Server) []rest.ServerConfig {
	var results []rest.ServerConfig

	for i, server := range servers {
		if server.URL != "" {
			envName := utils.StringSliceToConstantCase([]string{oc.ConvertOptions.EnvPrefix, "SERVER_URL"})
			if i > 0 {
				envName = fmt.Sprintf("%s_%d", envName, i+1)
			}
			results = append(results, rest.ServerConfig{
				URL: *rest.NewEnvStringTemplate(rest.NewEnvTemplateWithDefault(envName, server.URL)),
			})
		}
	}

	return results
}

func (oc *openAPIv3Builder) convertSecuritySchemes(scheme orderedmap.Pair[string, *v3.SecurityScheme]) error {
	key := scheme.Key()
	security := scheme.Value()
	if security == nil {
		return nil
	}
	securityType, err := rest.ParseSecuritySchemeType(security.Type)
	if err != nil {
		return err
	}
	result := rest.SecurityScheme{
		Type: securityType,
	}
	switch securityType {
	case rest.APIKeyScheme:
		inLocation, err := rest.ParseAPIKeyLocation(security.In)
		if err != nil {
			return err
		}
		apiConfig := rest.APIKeyAuthConfig{
			In:   inLocation,
			Name: security.Name,
		}
		result.Value = rest.NewEnvStringTemplate(rest.NewEnvTemplate(utils.StringSliceToConstantCase([]string{oc.EnvPrefix, key})))
		result.APIKeyAuthConfig = &apiConfig
	case rest.HTTPAuthScheme:
		httpConfig := rest.HTTPAuthConfig{
			Scheme: security.Scheme,
			Header: "Authorization",
		}
		result.Value = rest.NewEnvStringTemplate(rest.NewEnvTemplate(utils.StringSliceToConstantCase([]string{oc.EnvPrefix, key, "TOKEN"})))
		result.HTTPAuthConfig = &httpConfig
	case rest.OAuth2Scheme:
		if security.Flows == nil {
			return fmt.Errorf("flows of security scheme %s is required", key)
		}
		oauthConfig := rest.OAuth2Config{
			Flows: make(map[rest.OAuthFlowType]rest.OAuthFlow),
		}
		if security.Flows.Implicit != nil {
			oauthConfig.Flows[rest.ImplicitFlow] = *convertV3OAuthFLow(security.Flows.Implicit)
		}
		if security.Flows.AuthorizationCode != nil {
			oauthConfig.Flows[rest.AuthorizationCodeFlow] = *convertV3OAuthFLow(security.Flows.AuthorizationCode)
		}
		if security.Flows.ClientCredentials != nil {
			oauthConfig.Flows[rest.ClientCredentialsFlow] = *convertV3OAuthFLow(security.Flows.ClientCredentials)
		}
		if security.Flows.Password != nil {
			oauthConfig.Flows[rest.PasswordFlow] = *convertV3OAuthFLow(security.Flows.Password)
		}
		result.OAuth2Config = &oauthConfig
	case rest.OpenIDConnectScheme:
		result.OpenIDConfig = &rest.OpenIDConfig{
			OpenIDConnectURL: security.OpenIdConnectUrl,
		}
	default:
		return fmt.Errorf("invalid security scheme: %s", security.Type)
	}

	oc.schema.Settings.SecuritySchemes[key] = result
	return nil
}

func (oc *openAPIv3Builder) pathToNDCOperations(pathItem orderedmap.Pair[string, *v3.PathItem]) error {
	pathKey := pathItem.Key()
	pathValue := pathItem.Value()

	if pathValue.Get != nil {
		funcGet, err := newOpenAPIv3OperationBuilder(oc).BuildFunction(pathKey, pathValue.Get)
		if err != nil {
			return err
		}
		if funcGet != nil {
			oc.schema.Functions = append(oc.schema.Functions, funcGet)
		}
	}

	procPost, err := newOpenAPIv3OperationBuilder(oc).BuildProcedure(pathKey, "post", pathValue.Post)
	if err != nil {
		return err
	}
	if procPost != nil {
		oc.schema.Procedures = append(oc.schema.Procedures, procPost)
	}

	procPut, err := newOpenAPIv3OperationBuilder(oc).BuildProcedure(pathKey, "put", pathValue.Put)
	if err != nil {
		return err
	}
	if procPut != nil {
		oc.schema.Procedures = append(oc.schema.Procedures, procPut)
	}

	procPatch, err := newOpenAPIv3OperationBuilder(oc).BuildProcedure(pathKey, "patch", pathValue.Patch)
	if err != nil {
		return err
	}
	if procPatch != nil {
		oc.schema.Procedures = append(oc.schema.Procedures, procPatch)
	}

	procDelete, err := newOpenAPIv3OperationBuilder(oc).BuildProcedure(pathKey, "delete", pathValue.Delete)
	if err != nil {
		return err
	}
	if procDelete != nil {
		oc.schema.Procedures = append(oc.schema.Procedures, procDelete)
	}
	return nil
}

func (oc *openAPIv3Builder) getObjectTypeFromSchemaType(schemaType schema.Type) (*schema.ObjectType, string, error) {
	iSchemaType, err := schemaType.InterfaceT()

	switch st := iSchemaType.(type) {
	case *schema.NullableType:
		return oc.getObjectTypeFromSchemaType(st.UnderlyingType)
	case *schema.NamedType:
		objectType, ok := oc.schema.ObjectTypes[st.Name]
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

// get and convert an OpenAPI data type to a NDC type
func (oc *openAPIv3Builder) getSchemaTypeFromProxy(schemaProxy *base.SchemaProxy, nullable bool, apiPath string, writeMode bool, fieldPaths []string) (schema.TypeEncoder, *rest.TypeSchema, bool, error) {
	if schemaProxy == nil {
		return nil, nil, false, errParameterSchemaEmpty(fieldPaths)
	}
	innerSchema := schemaProxy.Schema()
	if innerSchema == nil {
		return nil, nil, false, fmt.Errorf("cannot get schema of $.%s from proxy: %s", strings.Join(fieldPaths, "."), schemaProxy.GetReference())
	}
	refName := getSchemaRefTypeNameV3(schemaProxy.GetReference())

	// return early object from ref
	if refName != "" {
		objectName := utils.ToPascalCase(refName)
		_, ok := oc.schema.ObjectTypes[objectName]
		if !ok {
			ndcType, typeSchema, _, err := oc.getSchemaType(innerSchema, apiPath, writeMode, []string{refName})
			if err != nil {
				return nil, nil, false, err
			}
			typeSchema.Description = innerSchema.Description
			if nullable && ndcType != nil {
				typeSchema.Nullable = true
				if !isNullableType(ndcType) {
					ndcType = schema.NewNullableType(ndcType)
				}
			}
			return ndcType, typeSchema, true, nil
		}

		var ndcType schema.TypeEncoder = schema.NewNamedType(objectName)
		typeSchema := &rest.TypeSchema{
			Type:        objectName,
			Description: innerSchema.Description,
		}

		if writeMode {
			writeObjectName := formatWriteObjectName(objectName)
			if _, ok := oc.schema.ObjectTypes[writeObjectName]; ok {
				ndcType = schema.NewNamedType(writeObjectName)
				typeSchema.Type = writeObjectName
			}
		}
		if nullable {
			typeSchema.Nullable = true
			if !isNullableType(ndcType) {
				ndcType = schema.NewNullableType(ndcType)
			}
		}
		return ndcType, typeSchema, true, nil
	}

	ndcType, typeSchema, isRef, err := oc.getSchemaType(innerSchema, apiPath, writeMode, fieldPaths)
	if err != nil {
		return nil, nil, false, err
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
func (oc *openAPIv3Builder) getSchemaType(typeSchema *base.Schema, apiPath string, writeMode bool, fieldPaths []string) (schema.TypeEncoder, *rest.TypeSchema, bool, error) {

	if typeSchema == nil {
		return nil, nil, false, errParameterSchemaEmpty(fieldPaths)
	}

	nullable := typeSchema.Nullable != nil && *typeSchema.Nullable
	if len(typeSchema.AllOf) > 0 {
		enc, ty, isRef, err := oc.buildAllOfAnyOfSchemaType(typeSchema.AllOf, nullable, apiPath, writeMode, fieldPaths)
		if err != nil {
			return nil, nil, false, err
		}
		if ty != nil {
			ty.Description = typeSchema.Description
		}
		return enc, ty, isRef, nil
	}

	if len(typeSchema.AnyOf) > 0 {
		enc, ty, isRef, err := oc.buildAllOfAnyOfSchemaType(typeSchema.AnyOf, true, apiPath, writeMode, fieldPaths)
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
		enc, ty, isRef, err := oc.getSchemaTypeFromProxy(typeSchema.OneOf[0], *typeSchema.Nullable, apiPath, writeMode, fieldPaths)
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
		return oc.buildScalarJSON(), typeResult, false, nil
	}

	if len(typeSchema.Type) == 0 {
		return nil, nil, false, errParameterSchemaEmpty(fieldPaths)
	}

	var result schema.TypeEncoder
	if len(typeSchema.Type) > 1 || isPrimitiveScalar(typeSchema.Type[0]) {
		scalarName := getScalarFromType(oc.schema, typeSchema.Type, typeSchema.Format, typeSchema.Enum, oc.trimPathPrefix(apiPath), fieldPaths)
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
				return oc.buildScalarJSON(), &rest.TypeSchema{Type: string(rest.ScalarJSON)}, false, nil
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
				nullable := !slices.Contains(typeSchema.Required, propName)
				propType, propApiSchema, _, err := oc.getSchemaTypeFromProxy(prop.Value(), nullable, apiPath, writeMode, append(fieldPaths, propName))
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

				if (!propApiSchema.ReadOnly && !propApiSchema.WriteOnly) || (!writeMode && propApiSchema.ReadOnly) || (writeMode || propApiSchema.WriteOnly) {
					propApiSchema.Nullable = nullable
					typeResult.Properties[propName] = *propApiSchema
				}
				if !propApiSchema.ReadOnly && !propApiSchema.WriteOnly {
					object.Fields[propName] = objField
				} else if !writeMode && propApiSchema.ReadOnly {
					readObject.Fields[propName] = objField
				} else {
					writeObject.Fields[propName] = objField
				}
			}
			if len(readObject.Fields) == 0 && len(writeObject.Fields) == 0 {
				oc.schema.ObjectTypes[refName] = object
				result = schema.NewNamedType(refName)
			} else {
				for key, field := range object.Fields {
					readObject.Fields[key] = field
					writeObject.Fields[key] = field
				}
				writeRefName := formatWriteObjectName(refName)
				oc.schema.ObjectTypes[refName] = readObject
				oc.schema.ObjectTypes[writeRefName] = writeObject
				if writeMode {
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
					itemSchema, propType, _isRef, err := oc.getSchemaType(itemSchemaA, apiPath, writeMode, fieldPaths)
					if err != nil {
						return nil, nil, isRef, err
					}
					if itemSchema != nil {
						result = schema.NewArrayType(itemSchema)
					} else {
						result = schema.NewArrayType(oc.buildScalarJSON())
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
func (oc *openAPIv3Builder) buildAllOfAnyOfSchemaType(schemaProxies []*base.SchemaProxy, nullable bool, apiPath string, writeMode bool, fieldPaths []string) (schema.TypeEncoder, *rest.TypeSchema, bool, error) {
	if len(schemaProxies) == 1 {
		return oc.getSchemaTypeFromProxy(schemaProxies[0], nullable, apiPath, writeMode, fieldPaths)
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
	for i, item := range schemaProxies {
		_, ok := getSchemaFromProxy(item)
		if !ok {
			continue
		}
		itemFieldPaths := append(fieldPaths, fmt.Sprint(i))
		enc, ty, isRef, err := oc.getSchemaTypeFromProxy(item, nullable, apiPath, false, itemFieldPaths)
		if err != nil {
			return nil, nil, false, err
		}

		name := getNamedType(enc, true, "")
		writeName := formatWriteObjectName(name)
		isObject := !isPrimitiveScalar(ty.Type) && ty.Type != "array"
		if isObject {
			if _, ok := oc.schema.ScalarTypes[name]; ok {
				isObject = false
			}
		}
		if !isObject {
			if !isRef {
				delete(oc.schema.ObjectTypes, name)
				delete(oc.schema.ObjectTypes, writeName)
				delete(oc.schema.ScalarTypes, name)
			}
			if name == ty.Type {
				ty.Type = string(rest.ScalarJSON)
			}
			return oc.buildScalarJSON(), ty, false, nil
		}

		for k, v := range ty.Properties {
			ty.Properties[k] = v
		}
		readObj, ok := oc.schema.ObjectTypes[name]
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
		writeObj, ok := oc.schema.ObjectTypes[writeName]
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
			delete(oc.schema.ObjectTypes, name)
			delete(oc.schema.ObjectTypes, writeName)
		}
	}

	refName := utils.ToPascalCase(strings.Join(fieldPaths, " "))
	writeRefName := formatWriteObjectName(refName)
	if len(readObject.Fields) > 0 {
		oc.schema.ObjectTypes[refName] = readObject
	}
	if len(writeObject.Fields) > 0 {
		oc.schema.ObjectTypes[writeRefName] = writeObject
	}

	if writeMode && len(writeObject.Fields) > 0 {
		refName = writeRefName
	}
	if len(typeSchema.Properties) == 0 {
		typeSchema = &rest.TypeSchema{
			Type: refName,
		}
	}
	return schema.NewNamedType(refName), typeSchema, false, nil
}

func (oc *openAPIv3OperationBuilder) convertRequestBody(reqBody *v3.RequestBody, apiPath string, fieldPaths []string) (*rest.RequestBody, schema.TypeEncoder, error) {
	if reqBody == nil || reqBody.Content == nil {
		return nil, nil, nil
	}

	contentType := rest.ContentTypeJSON
	content, ok := reqBody.Content.Get(contentType)
	if !ok {
		contentPair := reqBody.Content.First()
		contentType = contentPair.Key()
		content = contentPair.Value()
	}

	bodyRequired := false
	if reqBody.Required != nil && *reqBody.Required {
		bodyRequired = true
	}
	schemaType, typeSchema, _, err := oc.builder.getSchemaTypeFromProxy(content.Schema, !bodyRequired, apiPath, true, fieldPaths)
	if err != nil {
		return nil, nil, err
	}

	if typeSchema == nil {
		return nil, nil, nil
	}

	bodyResult := &rest.RequestBody{
		ContentType: contentType,
		Schema:      typeSchema,
	}

	if content.Encoding != nil {
		encoding := make(map[string]rest.EncodingObject)
		for iter := content.Encoding.First(); iter != nil; iter = iter.Next() {
			encodingValue := iter.Value()
			if encodingValue == nil {
				continue
			}

			item := rest.EncodingObject{
				ContentType:   utils.SplitStringsAndTrimSpaces(encodingValue.ContentType, ","),
				AllowReserved: encodingValue.AllowReserved,
				Explode:       encodingValue.Explode,
			}

			if encodingValue.Style != "" {
				style, err := rest.ParseParameterEncodingStyle(encodingValue.Style)
				if err != nil {
					return nil, nil, err
				}
				item.Style = style
			}

			if encodingValue.Headers != nil {
				item.Headers = make(map[string]rest.RequestParameter)
				for encodingHeader := encodingValue.Headers.First(); encodingHeader != nil; encodingHeader = encodingHeader.Next() {
					key := strings.TrimSpace(encodingHeader.Key())
					header := encodingHeader.Value()
					if key == "" || header == nil {
						continue
					}

					ndcType, typeSchema, _, err := oc.builder.getSchemaTypeFromProxy(header.Schema, header.AllowEmptyValue, apiPath, true, append(fieldPaths, key))
					if err != nil {
						return nil, nil, err
					}

					headerEncoding := rest.EncodingObject{
						AllowReserved: header.AllowReserved,
						Explode:       &header.Explode,
						Headers:       map[string]rest.RequestParameter{},
					}

					if header.Style != "" {
						style, err := rest.ParseParameterEncodingStyle(header.Style)
						if err != nil {
							return nil, nil, err
						}
						headerEncoding.Style = style
					}

					argumentName := encodeHeaderArgumentName(key)
					item.Headers[key] = rest.RequestParameter{
						ArgumentName:   argumentName,
						Schema:         typeSchema,
						EncodingObject: headerEncoding,
					}

					argument := schema.ArgumentInfo{
						Type: ndcType.Encode(),
					}
					if header.Description != "" {
						argument.Description = &header.Description
					}

					oc.Arguments[argumentName] = argument
				}
			}

			encoding[iter.Key()] = item
		}
		bodyResult.Encoding = encoding
	}
	return bodyResult, schemaType, nil
}

func (oc *openAPIv3OperationBuilder) convertResponse(responses *v3.Responses, apiPath string, fieldPaths []string) (schema.TypeEncoder, error) {
	if responses == nil || responses.Codes == nil || responses.Codes.IsZero() {
		return nil, nil
	}

	var resp *v3.Response
	for _, code := range []int{200, 201, 204} {
		res := responses.FindResponseByCode(code)
		if res != nil {
			resp = res
			break
		}
	}

	// return nullable boolean type if the response content is null
	if resp == nil || resp.Content == nil {
		return schema.NewNullableNamedType("Boolean"), nil
	}
	jsonContent, ok := resp.Content.Get("application/json")
	if !ok {
		return nil, nil
	}

	schemaType, _, _, err := oc.builder.getSchemaTypeFromProxy(jsonContent.Schema, false, apiPath, false, fieldPaths)
	if err != nil {
		return nil, err
	}
	return schemaType, nil
}

func (oc *openAPIv3Builder) convertComponentSchemas(schemaItem orderedmap.Pair[string, *base.SchemaProxy]) error {
	typeValue := schemaItem.Value()
	typeSchema := typeValue.Schema()

	if typeSchema == nil {
		return nil
	}

	typeKey := schemaItem.Key()
	if _, ok := oc.schema.ObjectTypes[typeKey]; ok {
		return nil
	}
	typeEncoder, _, _, err := oc.getSchemaType(typeSchema, "", false, []string{typeKey})
	if err != nil {
		return err
	}

	// treat no-property objects as a Arbitrary JSON scalar
	if typeEncoder == nil {
		refName := utils.ToPascalCase(typeKey)
		oc.schema.ScalarTypes[refName] = *schema.NewScalarType()
	}

	return err
}

func convertV3OAuthFLow(input *v3.OAuthFlow) *rest.OAuthFlow {
	result := &rest.OAuthFlow{
		AuthorizationURL: input.AuthorizationUrl,
		TokenURL:         input.TokenUrl,
		RefreshURL:       input.RefreshUrl,
	}

	if input.Scopes != nil {
		scopes := make(map[string]string)
		for iter := input.Scopes.First(); iter != nil; iter = iter.Next() {
			key := iter.Key()
			value := iter.Value()
			if key == "" || value == "" {
				continue
			}
			scopes[key] = value
		}
		result.Scopes = scopes
	}

	return result
}

func (oc *openAPIv3Builder) trimPathPrefix(input string) string {
	if oc.ConvertOptions.TrimPrefix == "" {
		return input
	}
	return strings.TrimPrefix(input, oc.ConvertOptions.TrimPrefix)
}

type openAPIv3OperationBuilder struct {
	builder       *openAPIv3Builder
	Arguments     map[string]schema.ArgumentInfo
	RequestParams []rest.RequestParameter
}

func newOpenAPIv3OperationBuilder(builder *openAPIv3Builder) *openAPIv3OperationBuilder {
	return &openAPIv3OperationBuilder{
		builder:   builder,
		Arguments: make(map[string]schema.ArgumentInfo),
	}
}

// BuildFunction build a REST NDC function information from OpenAPI v3 operation
func (oc *openAPIv3OperationBuilder) BuildFunction(pathKey string, itemGet *v3.Operation) (*rest.RESTFunctionInfo, error) {
	funcName := itemGet.OperationId
	if funcName == "" {
		funcName = buildPathMethodName(pathKey, "get", oc.builder.ConvertOptions)
	}
	resultType, err := oc.convertResponse(itemGet.Responses, pathKey, []string{funcName, "Result"})
	if err != nil {
		return nil, fmt.Errorf("%s: %s", pathKey, err)
	}
	if resultType == nil {
		return nil, nil
	}

	err = oc.convertParameters(itemGet.Parameters, pathKey, []string{funcName})
	if err != nil {
		return nil, fmt.Errorf("%s: %s", funcName, err)
	}

	function := rest.RESTFunctionInfo{
		Request: &rest.Request{
			URL:        pathKey,
			Method:     "get",
			Parameters: sortRequestParameters(oc.RequestParams),
			Security:   convertSecurities(itemGet.Security),
			Servers:    oc.builder.convertServers(itemGet.Servers),
		},
		FunctionInfo: schema.FunctionInfo{
			Name:       funcName,
			Arguments:  oc.Arguments,
			ResultType: resultType.Encode(),
		},
	}

	if itemGet.Summary != "" {
		function.Description = &itemGet.Summary
	}

	return &function, nil
}

func (oc *openAPIv3OperationBuilder) BuildProcedure(pathKey string, method string, operation *v3.Operation) (*rest.RESTProcedureInfo, error) {
	if operation == nil {
		return nil, nil
	}

	procName := operation.OperationId
	if procName == "" {
		procName = buildPathMethodName(pathKey, method, oc.builder.ConvertOptions)
	}

	resultType, err := oc.convertResponse(operation.Responses, pathKey, []string{procName, "Result"})
	if err != nil {
		return nil, fmt.Errorf("%s: %s", pathKey, err)
	}

	if resultType == nil {
		return nil, nil
	}

	err = oc.convertParameters(operation.Parameters, pathKey, []string{procName})
	if err != nil {
		return nil, fmt.Errorf("%s: %s", pathKey, err)
	}

	reqBody, schemaType, err := oc.convertRequestBody(operation.RequestBody, pathKey, []string{procName, "Body"})
	if err != nil {
		return nil, fmt.Errorf("%s: %s", pathKey, err)
	}
	if reqBody != nil {
		if reqBody.ContentType == rest.ContentTypeFormURLEncoded {
			// convert URL encoded body to parameters
			if reqBody.Schema != nil {
				if reqBody.Schema.Type == "object" {
					objectType, objectTypeName, err := oc.builder.getObjectTypeFromSchemaType(schemaType.Encode())
					if err != nil {
						return nil, fmt.Errorf("%s: %s", pathKey, err)
					}
					// remove unused request body type
					delete(oc.builder.schema.ObjectTypes, objectTypeName)
					for key, prop := range reqBody.Schema.Properties {
						propType, ok := objectType.Fields[key]
						if !ok {
							continue
						}
						// renaming query parameter name `body` if exist to avoid conflicts
						if paramData, ok := oc.Arguments[key]; ok {
							oc.Arguments[fmt.Sprintf("param%s", key)] = paramData
						}

						desc := prop.Description
						argument := schema.ArgumentInfo{
							Type: propType.Type,
						}
						if desc != "" {
							argument.Description = &desc
						}
						oc.Arguments[key] = argument
						schemaProp := prop
						oc.RequestParams = append(oc.RequestParams, rest.RequestParameter{
							EncodingObject: reqBody.Encoding[key],
							Name:           key,
							In:             rest.InQuery,
							Schema:         &schemaProp,
						})
					}
				} else {
					description := fmt.Sprintf("Request body of %s %s", method, pathKey)
					// renaming query parameter name `body` if exist to avoid conflicts
					if paramData, ok := oc.Arguments["body"]; ok {
						oc.Arguments["paramBody"] = paramData
						for i, param := range oc.RequestParams {
							if param.Name == "body" {
								param.ArgumentName = "paramBody"
								oc.RequestParams[i] = param
								break
							}
						}
					}

					oc.Arguments["body"] = schema.ArgumentInfo{
						Description: &description,
						Type:        schemaType.Encode(),
					}
					oc.RequestParams = append(oc.RequestParams, rest.RequestParameter{
						Name:   "body",
						In:     rest.InQuery,
						Schema: reqBody.Schema,
					})
				}
			}
			reqBody = &rest.RequestBody{
				ContentType: rest.ContentTypeFormURLEncoded,
			}
		} else {
			description := fmt.Sprintf("Request body of %s %s", strings.ToUpper(method), pathKey)
			// renaming query parameter name `body` if exist to avoid conflicts
			if paramData, ok := oc.Arguments["body"]; ok {
				oc.Arguments["paramBody"] = paramData
			}

			oc.Arguments["body"] = schema.ArgumentInfo{
				Description: &description,
				Type:        schemaType.Encode(),
			}
		}
	}

	procedure := rest.RESTProcedureInfo{
		Request: &rest.Request{
			URL:         pathKey,
			Method:      method,
			Parameters:  sortRequestParameters(oc.RequestParams),
			Security:    convertSecurities(operation.Security),
			Servers:     oc.builder.convertServers(operation.Servers),
			RequestBody: reqBody,
		},
		ProcedureInfo: schema.ProcedureInfo{
			Name:       procName,
			Arguments:  oc.Arguments,
			ResultType: resultType.Encode(),
		},
	}

	if operation.Summary != "" {
		procedure.Description = &operation.Summary
	}

	return &procedure, nil
}

func (oc *openAPIv3OperationBuilder) convertParameters(params []*v3.Parameter, apiPath string, fieldPaths []string) error {

	if len(params) == 0 {
		return nil
	}

	for _, param := range params {
		if param == nil {
			continue
		}
		paramName := param.Name
		if paramName == "" {
			return errors.New("parameter name is empty")
		}
		paramRequired := false
		if param.Required != nil && *param.Required {
			paramRequired = true
		}
		paramPaths := append(fieldPaths, paramName)
		schemaType, apiSchema, _, err := oc.builder.getSchemaTypeFromProxy(param.Schema, !paramRequired, apiPath, true, paramPaths)
		if err != nil {
			return err
		}

		paramLocation, err := rest.ParseParameterLocation(param.In)
		if err != nil {
			return err
		}

		encoding := rest.EncodingObject{
			AllowReserved: param.AllowReserved,
			Explode:       param.Explode,
		}
		if param.Style != "" {
			style, err := rest.ParseParameterEncodingStyle(param.Style)
			if err != nil {
				return err
			}
			encoding.Style = style
		}
		oc.RequestParams = append(oc.RequestParams, rest.RequestParameter{
			Name:           paramName,
			In:             paramLocation,
			Schema:         apiSchema,
			EncodingObject: encoding,
		})

		argument := schema.ArgumentInfo{
			Type: schemaType.Encode(),
		}
		if param.Description != "" {
			argument.Description = &param.Description
		}

		oc.Arguments[paramName] = argument
	}

	return nil
}

// build a named type for JSON scalar
func (oc *openAPIv3Builder) buildScalarJSON() *schema.NamedType {
	scalarName := string(rest.ScalarJSON)
	if _, ok := oc.schema.ScalarTypes[scalarName]; !ok {
		oc.schema.ScalarTypes[scalarName] = *schema.NewScalarType()
	}
	return schema.NewNamedType(scalarName)
}
