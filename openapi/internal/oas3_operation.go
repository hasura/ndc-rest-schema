package internal

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	rest "github.com/hasura/ndc-rest-schema/schema"
	"github.com/hasura/ndc-rest-schema/utils"
	"github.com/hasura/ndc-sdk-go/schema"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

type oas3OperationBuilder struct {
	builder       *OAS3Builder
	pathKey       string
	method        string
	Arguments     map[string]schema.ArgumentInfo
	RequestParams []rest.RequestParameter
}

func newOAS3OperationBuilder(builder *OAS3Builder, pathKey string, method string) *oas3OperationBuilder {
	return &oas3OperationBuilder{
		builder:   builder,
		pathKey:   pathKey,
		method:    method,
		Arguments: make(map[string]schema.ArgumentInfo),
	}
}

// BuildFunction build a REST NDC function information from OpenAPI v3 operation
func (oc *oas3OperationBuilder) BuildFunction(itemGet *v3.Operation) (*rest.RESTFunctionInfo, error) {
	funcName := itemGet.OperationId
	if funcName == "" {
		funcName = buildPathMethodName(oc.pathKey, "get", oc.builder.ConvertOptions)
	}

	oc.builder.Logger.Debug("function",
		slog.String("name", funcName),
		slog.String("path", oc.pathKey),
		slog.String("method", oc.method),
	)
	resultType, schemaResponse, err := oc.convertResponse(itemGet.Responses, oc.pathKey, []string{funcName, "Result"})
	if err != nil {
		return nil, fmt.Errorf("%s: %s", oc.pathKey, err)
	}
	if resultType == nil {
		return nil, nil
	}

	err = oc.convertParameters(itemGet.Parameters, oc.pathKey, []string{funcName})
	if err != nil {
		return nil, fmt.Errorf("%s: %s", funcName, err)
	}

	function := rest.RESTFunctionInfo{
		Request: &rest.Request{
			URL:        oc.pathKey,
			Method:     "get",
			Parameters: sortRequestParameters(oc.RequestParams),
			Security:   convertSecurities(itemGet.Security),
			Servers:    oc.builder.convertServers(itemGet.Servers),
			Response:   *schemaResponse,
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

func (oc *oas3OperationBuilder) BuildProcedure(operation *v3.Operation) (*rest.RESTProcedureInfo, error) {
	if operation == nil {
		return nil, nil
	}

	procName := operation.OperationId
	if procName == "" {
		procName = buildPathMethodName(oc.pathKey, oc.method, oc.builder.ConvertOptions)
	}

	oc.builder.Logger.Debug("procedure",
		slog.String("name", procName),
		slog.String("path", oc.pathKey),
		slog.String("method", oc.method),
	)
	resultType, schemaResponse, err := oc.convertResponse(operation.Responses, oc.pathKey, []string{procName, "Result"})
	if err != nil {
		return nil, fmt.Errorf("%s: %s", oc.pathKey, err)
	}

	if resultType == nil {
		return nil, nil
	}

	err = oc.convertParameters(operation.Parameters, oc.pathKey, []string{procName})
	if err != nil {
		return nil, fmt.Errorf("%s: %s", oc.pathKey, err)
	}

	reqBody, schemaType, err := oc.convertRequestBody(operation.RequestBody, oc.pathKey, []string{procName, "Body"})
	if err != nil {
		return nil, fmt.Errorf("%s: %s", oc.pathKey, err)
	}
	if reqBody != nil {
		description := fmt.Sprintf("Request body of %s %s", strings.ToUpper(oc.method), oc.pathKey)
		// renaming query parameter name `body` if exist to avoid conflicts
		if paramData, ok := oc.Arguments["body"]; ok {
			oc.Arguments["paramBody"] = paramData
		}

		oc.Arguments["body"] = schema.ArgumentInfo{
			Description: &description,
			Type:        schemaType.Encode(),
		}
	}

	procedure := rest.RESTProcedureInfo{
		Request: &rest.Request{
			URL:         oc.pathKey,
			Method:      oc.method,
			Parameters:  sortRequestParameters(oc.RequestParams),
			Security:    convertSecurities(operation.Security),
			Servers:     oc.builder.convertServers(operation.Servers),
			RequestBody: reqBody,
			Response:    *schemaResponse,
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

func (oc *oas3OperationBuilder) convertParameters(params []*v3.Parameter, apiPath string, fieldPaths []string) error {

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
		schemaType, apiSchema, _, err := newOAS3SchemaBuilder(oc.builder, apiPath, rest.ParameterLocation(param.In), true).
			getSchemaTypeFromProxy(param.Schema, !paramRequired, paramPaths)
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

		oc.builder.typeUsageCounter.Add(getNamedType(schemaType, true, ""), 1)
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

func (oc *oas3OperationBuilder) convertRequestBody(reqBody *v3.RequestBody, apiPath string, fieldPaths []string) (*rest.RequestBody, schema.TypeEncoder, error) {
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
	location := rest.InBody
	if contentType == rest.ContentTypeFormURLEncoded {
		location = rest.InQuery
	}
	schemaType, typeSchema, _, err := newOAS3SchemaBuilder(oc.builder, apiPath, location, true).
		getSchemaTypeFromProxy(content.Schema, !bodyRequired, fieldPaths)
	if err != nil {
		return nil, nil, err
	}

	if typeSchema == nil {
		return nil, nil, nil
	}

	oc.builder.typeUsageCounter.Add(getNamedType(schemaType, true, ""), 1)
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

					ndcType, typeSchema, _, err := newOAS3SchemaBuilder(oc.builder, apiPath, rest.InHeader, true).
						getSchemaTypeFromProxy(header.Schema, header.AllowEmptyValue, append(fieldPaths, key))
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

					oc.builder.typeUsageCounter.Add(getNamedType(ndcType, true, ""), 1)
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

func (oc *oas3OperationBuilder) convertResponse(responses *v3.Responses, apiPath string, fieldPaths []string) (schema.TypeEncoder, *rest.Response, error) {
	if responses == nil || responses.Codes == nil || responses.Codes.IsZero() {
		return nil, nil, nil
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
		scalarName := string(rest.ScalarBoolean)
		oc.builder.typeUsageCounter.Add(scalarName, 1)
		return schema.NewNullableNamedType(scalarName), &rest.Response{
			ContentType: rest.ContentTypeJSON,
		}, nil
	}

	contentType := rest.ContentTypeJSON
	bodyContent, present := resp.Content.Get(contentType)
	if !present {
		if len(oc.builder.AllowedContentTypes) == 0 {
			firstContent := resp.Content.First()
			bodyContent = firstContent.Value()
			contentType = firstContent.Key()
			present = true
		} else {
			for _, ct := range oc.builder.AllowedContentTypes {
				bodyContent, present = resp.Content.Get(ct)
				if present {
					contentType = ct
					break
				}
			}
		}
	}

	if !present {
		return nil, nil, nil
	}

	schemaType, _, _, err := newOAS3SchemaBuilder(oc.builder, apiPath, rest.InBody, false).
		getSchemaTypeFromProxy(bodyContent.Schema, false, fieldPaths)
	if err != nil {
		return nil, nil, err
	}
	oc.builder.typeUsageCounter.Add(getNamedType(schemaType, true, ""), 1)

	schemaResponse := &rest.Response{
		ContentType: contentType,
	}
	switch contentType {
	case rest.ContentTypeNdJSON:
		// Newline Delimited JSON (ndjson) format represents a stream of structured objects
		// so the response would be wrapped with an array
		return schema.NewArrayType(schemaType), schemaResponse, nil
	default:
		return schemaType, schemaResponse, nil
	}
}
