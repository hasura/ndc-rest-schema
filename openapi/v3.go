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

type openAPIv3Converter struct {
	schema *rest.NDCRestSchema
}

// OpenAPIv3ToNDCSchema converts OpenAPI v3 JSON bytes to NDC REST schema
func OpenAPIv3ToNDCSchema(input []byte) (*rest.NDCRestSchema, []error) {
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

	converter := &openAPIv3Converter{
		schema: rest.NewNDCRestSchema(),
	}
	if docModel.Model.Info != nil {
		converter.schema.Settings.Version = docModel.Model.Info.Version
	}
	for _, server := range docModel.Model.Servers {
		if server.URL != "" {
			converter.schema.Settings.Request.URL = server.URL
			break
		}
	}

	for iterPath := docModel.Model.Paths.PathItems.First(); iterPath != nil; iterPath = iterPath.Next() {
		if err := converter.pathToNDCOperations(iterPath); err != nil {
			return nil, append(errs, err)
		}
	}

	for cSchema := docModel.Model.Components.Schemas.First(); cSchema != nil; cSchema = cSchema.Next() {
		if err := converter.convertComponentSchemas(cSchema); err != nil {
			return nil, append(errs, err)
		}
	}

	return converter.schema, nil
}

func (oc *openAPIv3Converter) pathToNDCOperations(pathItem orderedmap.Pair[string, *v3.PathItem]) error {
	pathKey := pathItem.Key()
	pathValue := pathItem.Value()
	if pathValue.Get != nil {
		itemGet := pathValue.Get
		funcName := itemGet.OperationId
		if funcName == "" {
			funcName = buildPathMethodName(pathKey, "get")
		}
		resultType, err := oc.convertResponse(itemGet.Responses, []string{funcName, "Result"})
		if err != nil {
			return fmt.Errorf("%s: %s", pathKey, err)
		}
		if resultType != nil {
			arguments, reqParams, err := oc.convertParameters(itemGet.Parameters, []string{funcName})
			if err != nil {
				return fmt.Errorf("%s: %s", funcName, err)
			}

			function := rest.RESTFunctionInfo{
				Request: &rest.Request{
					URL:        pathKey,
					Method:     "get",
					Parameters: reqParams,
				},
				FunctionInfo: schema.FunctionInfo{
					Name:       funcName,
					Arguments:  arguments,
					ResultType: resultType.Encode(),
				},
			}

			if itemGet.Summary != "" {
				function.Description = &itemGet.Summary
			}

			oc.schema.Functions = append(oc.schema.Functions, &function)
		}
	}

	procPost, err := oc.convertProcedureOperation(pathKey, "post", pathValue.Post)
	if err != nil {
		return err
	}
	if procPost != nil {
		oc.schema.Procedures = append(oc.schema.Procedures, procPost)
	}

	procPut, err := oc.convertProcedureOperation(pathKey, "put", pathValue.Put)
	if err != nil {
		return err
	}
	if procPut != nil {
		oc.schema.Procedures = append(oc.schema.Procedures, procPut)
	}

	procPatch, err := oc.convertProcedureOperation(pathKey, "patch", pathValue.Patch)
	if err != nil {
		return err
	}
	if procPatch != nil {
		oc.schema.Procedures = append(oc.schema.Procedures, procPatch)
	}

	procDelete, err := oc.convertProcedureOperation(pathKey, "delete", pathValue.Delete)
	if err != nil {
		return err
	}
	if procDelete != nil {
		oc.schema.Procedures = append(oc.schema.Procedures, procDelete)
	}
	return nil
}

func (oc *openAPIv3Converter) convertProcedureOperation(pathKey string, method string, operation *v3.Operation) (*rest.RESTProcedureInfo, error) {
	if operation == nil {
		return nil, nil
	}

	procName := operation.OperationId
	if procName == "" {
		procName = buildPathMethodName(pathKey, "get")
	}

	resultType, err := oc.convertResponse(operation.Responses, []string{procName, "Result"})
	if err != nil {
		return nil, fmt.Errorf("%s: %s", pathKey, err)
	}

	if resultType == nil {
		return nil, nil
	}

	arguments, reqParams, err := oc.convertParameters(operation.Parameters, []string{procName})
	if err != nil {
		return nil, fmt.Errorf("%s: %s", pathKey, err)
	}

	reqBody, contentType, err := oc.convertRequestBody(operation.RequestBody, []string{procName})
	if err != nil {
		return nil, fmt.Errorf("%s: %s", pathKey, err)
	}
	if reqBody != nil {
		description := fmt.Sprintf("Request body of %s", pathKey)
		// renaming query parameter name `data` if exist to avoid conflicts
		if paramData, ok := arguments["body"]; ok {
			arguments["paramBody"] = paramData
		}

		arguments["body"] = schema.ArgumentInfo{
			Description: &description,
			Type:        reqBody.Encode(),
		}
	}

	procedure := rest.RESTProcedureInfo{
		Request: &rest.Request{
			URL:        pathKey,
			Method:     method,
			Parameters: reqParams,
			Headers: map[string]string{
				ContentTypeHeader: contentType,
			},
		},
		ProcedureInfo: schema.ProcedureInfo{
			Name:       procName,
			Arguments:  arguments,
			ResultType: resultType.Encode(),
		},
	}

	if operation.Summary != "" {
		procedure.Description = &operation.Summary
	}

	return &procedure, nil
}

func (oc *openAPIv3Converter) convertParameters(params []*v3.Parameter, fieldPaths []string) (map[string]schema.ArgumentInfo, []rest.RequestParameter, error) {

	arguments := make(map[string]schema.ArgumentInfo)
	if len(params) == 0 {
		return arguments, nil, nil
	}

	var reqParams []rest.RequestParameter

	for _, param := range params {
		if param == nil {
			continue
		}
		paramName := param.Name
		if paramName == "" {
			return nil, nil, errors.New("parameter name is empty")
		}
		paramRequired := false
		if param.Required != nil && *param.Required {
			paramRequired = true
		}
		schemaType, apiSchema, err := oc.getSchemaTypeFromProxy(param.Schema, !paramRequired, fieldPaths)
		if err != nil {
			return nil, nil, err
		}

		paramLocation, err := rest.ParseParameterLocation(param.In)
		if err != nil {
			return nil, nil, err
		}
		var scalarName string
		if apiSchema != nil && len(apiSchema.Type) > 0 {
			scalarName = getScalarNameFromType(apiSchema.Type[0])
		}
		reqParams = append(reqParams, rest.RequestParameter{
			Name:     paramName,
			In:       paramLocation,
			Required: paramRequired,
			Schema:   ParseTypeSchemaFromOpenAPISchema(apiSchema, scalarName),
		})

		argument := schema.ArgumentInfo{
			Type: schemaType.Encode(),
		}
		if param.Description != "" {
			argument.Description = &param.Description
		}

		arguments[paramName] = argument
	}

	return arguments, reqParams, nil

}

// get and convert an OpenAPI data type to a NDC type
func (oc *openAPIv3Converter) getSchemaTypeFromProxy(schemaProxy *base.SchemaProxy, nullable bool, fieldPaths []string) (schema.TypeEncoder, *base.Schema, error) {
	if schemaProxy == nil {
		return nil, nil, errParameterSchemaEmpty
	}
	innerSchema := schemaProxy.Schema()
	if innerSchema == nil {
		return nil, nil, fmt.Errorf("cannot get schema from proxy: %s", schemaProxy.GetReference())
	}
	refName := getSchemaRefTypeName(schemaProxy.GetReference())
	var ndcType schema.TypeEncoder
	var err error
	// return early object from ref
	if refName != "" && len(innerSchema.Type) > 0 && innerSchema.Type[0] == "object" {
		ndcType = schema.NewNamedType(refName)
	} else {
		if innerSchema.Title != "" && !strings.Contains(innerSchema.Title, " ") {
			fieldPaths = []string{utils.ToPascalCase(innerSchema.Title)}
		}
		ndcType, err = oc.getSchemaType(innerSchema, fieldPaths)
		if err != nil {
			return nil, nil, err
		}
	}
	if nullable {
		ndcType = schema.NewNullableType(ndcType)
	}
	return ndcType, innerSchema, nil
}

// get and convert an OpenAPI data type to a NDC type
func (oc *openAPIv3Converter) getSchemaType(typeSchema *base.Schema, fieldPaths []string) (schema.TypeEncoder, error) {

	if typeSchema == nil {
		return nil, errParameterSchemaEmpty
	}
	if len(typeSchema.AnyOf) > 0 || typeSchema.AdditionalProperties != nil {
		scalarName := "JSON"
		if _, ok := oc.schema.ScalarTypes[scalarName]; !ok {
			oc.schema.ScalarTypes[scalarName] = *schema.NewScalarType()
		}
		return schema.NewNamedType(scalarName), nil
	}

	if len(typeSchema.Type) == 0 {
		return nil, errParameterSchemaEmpty
	}

	var result schema.TypeEncoder
	typeName := typeSchema.Type[0]
	switch typeName {
	case "boolean", "integer", "number", "string":
		scalarName := getScalarNameFromType(typeName)
		if _, ok := oc.schema.ScalarTypes[scalarName]; !ok {
			oc.schema.ScalarTypes[scalarName] = *schema.NewScalarType()
		}
		result = schema.NewNamedType(scalarName)
	// case "null":
	case "object":
		refName := utils.StringSliceToPascalCase(fieldPaths)

		if typeSchema.Properties == nil || typeSchema.Properties.IsZero() {
			// treat no-property objects as a JSON scalar
			oc.schema.ScalarTypes[refName] = *schema.NewScalarType()
		} else {
			object := schema.ObjectType{
				Fields: make(schema.ObjectTypeFields),
			}
			if typeSchema.Description != "" {
				object.Description = &typeSchema.Description
			}
			for prop := typeSchema.Properties.First(); prop != nil; prop = prop.Next() {
				propName := prop.Key()
				propType, propApiSchema, err := oc.getSchemaTypeFromProxy(prop.Value(), !slices.Contains(typeSchema.Required, propName), append(fieldPaths, propName))
				if err != nil {
					return nil, err
				}
				objField := schema.ObjectField{
					Type: propType.Encode(),
				}
				if propApiSchema.Description != "" {
					objField.Description = &propApiSchema.Description
				}
				object.Fields[propName] = objField
			}

			oc.schema.ObjectTypes[refName] = object
		}
		result = schema.NewNamedType(refName)
	case "array":
		if typeSchema.Items == nil || typeSchema.Items.A == nil {
			return nil, errors.New("array item is empty")
		}

		itemName := getSchemaRefTypeName(typeSchema.Items.A.GetReference())
		if itemName != "" {
			result = schema.NewArrayType(schema.NewNamedType(itemName))
		} else {
			itemSchemaA := typeSchema.Items.A.Schema()
			if itemSchemaA != nil {
				itemSchema, err := oc.getSchemaType(itemSchemaA, fieldPaths)
				if err != nil {
					return nil, err
				}

				result = schema.NewArrayType(itemSchema)
			}
		}

		if result == nil {
			return nil, fmt.Errorf("cannot parse type reference name: %s", typeSchema.Items.A.GetReference())
		}
	default:
		return nil, fmt.Errorf("unsupported schema type %s", typeName)
	}

	if typeSchema.Nullable != nil && *typeSchema.Nullable {
		return schema.NewNullableType(result), nil
	}
	return result, nil
}

func (oc *openAPIv3Converter) convertRequestBody(reqBody *v3.RequestBody, fieldPaths []string) (schema.TypeEncoder, string, error) {
	if reqBody == nil || reqBody.Content == nil {
		return nil, "", nil
	}

	jsonContent, ok := reqBody.Content.Get(ContentTypeJSON)
	if !ok {
		return nil, reqBody.Content.First().Key(), nil
	}

	schemaType, _, err := oc.getSchemaTypeFromProxy(jsonContent.Schema, false, fieldPaths)
	if err != nil {
		return nil, "", err
	}
	return schemaType, ContentTypeJSON, nil
}

func (oc *openAPIv3Converter) convertResponse(responses *v3.Responses, fieldPaths []string) (schema.TypeEncoder, error) {
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

	schemaType, _, err := oc.getSchemaTypeFromProxy(jsonContent.Schema, false, fieldPaths)
	if err != nil {
		return nil, err
	}
	return schemaType, nil
}

func (oc *openAPIv3Converter) convertComponentSchemas(schemaItem orderedmap.Pair[string, *base.SchemaProxy]) error {
	typeValue := schemaItem.Value()
	typeSchema := typeValue.Schema()

	if typeSchema == nil || !slices.Contains(typeSchema.Type, "object") {
		return nil
	}
	_, err := oc.getSchemaType(typeSchema, []string{schemaItem.Key()})
	return err
}
