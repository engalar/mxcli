// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"

	"go.mongodb.org/mongo-driver/bson"
)

// parsePublishedRestService parses a published REST service from BSON.
func (r *Reader) parsePublishedRestService(unitID, containerID string, contents []byte) (*model.PublishedRestService, error) {
	contents, err := r.resolveContents(unitID, contents)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := bson.Unmarshal(contents, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal BSON: %w", err)
	}

	svc := &model.PublishedRestService{}
	svc.ID = model.ID(unitID)
	svc.TypeName = "Rest$PublishedRestService"
	svc.ContainerID = model.ID(containerID)

	svc.Name = extractString(raw["Name"])
	svc.Path = extractString(raw["Path"])
	svc.Version = extractString(raw["Version"])
	svc.ServiceName = extractString(raw["ServiceName"])
	svc.Excluded = extractBool(raw["Excluded"], false)

	// Parse resources
	resources := extractBsonArray(raw["Resources"])
	for _, res := range resources {
		if resMap, ok := res.(map[string]any); ok {
			resource := &model.PublishedRestResource{}
			resource.ID = model.ID(extractBsonID(resMap["$ID"]))
			resource.TypeName = extractString(resMap["$Type"])
			resource.Name = extractString(resMap["Name"])

			// Parse operations
			ops := extractBsonArray(resMap["Operations"])
			for _, op := range ops {
				if opMap, ok := op.(map[string]any); ok {
					operation := &model.PublishedRestOperation{}
					operation.ID = model.ID(extractBsonID(opMap["$ID"]))
					operation.TypeName = extractString(opMap["$Type"])
					operation.Path = extractString(opMap["Path"])
					operation.HTTPMethod = extractString(opMap["HttpMethod"])
					operation.Summary = extractString(opMap["Summary"])
					operation.Microflow = extractString(opMap["Microflow"])
					operation.Deprecated = extractBool(opMap["Deprecated"], false)
					resource.Operations = append(resource.Operations, operation)
				}
			}

			svc.Resources = append(svc.Resources, resource)
		}
	}

	return svc, nil
}

// parseConsumedRestService parses a consumed REST service from BSON.
func (r *Reader) parseConsumedRestService(unitID, containerID string, contents []byte) (*model.ConsumedRestService, error) {
	contents, err := r.resolveContents(unitID, contents)
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := bson.Unmarshal(contents, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal BSON: %w", err)
	}

	svc := &model.ConsumedRestService{}
	svc.ID = model.ID(unitID)
	svc.TypeName = "Rest$ConsumedRestService"
	svc.ContainerID = model.ID(containerID)

	svc.Name = extractString(raw["Name"])
	svc.Documentation = extractString(raw["Documentation"])
	svc.Excluded = extractBool(raw["Excluded"], false)

	// Parse BaseUrl from Rest$ValueTemplate
	if baseUrlMap := extractBsonMap(raw["BaseUrl"]); baseUrlMap != nil {
		svc.BaseUrl = extractString(baseUrlMap["Value"])
	}

	// Parse AuthenticationScheme (polymorphic: null or Rest$BasicAuthenticationScheme)
	if authMap := extractBsonMap(raw["AuthenticationScheme"]); authMap != nil {
		authType := extractString(authMap["$Type"])
		if authType == "Rest$BasicAuthenticationScheme" {
			auth := &model.RestAuthentication{Scheme: "Basic"}
			auth.Username = extractRestValue(authMap["Username"])
			auth.Password = extractRestValue(authMap["Password"])
			svc.Authentication = auth
		}
	}

	// Parse Operations
	ops := extractBsonArray(raw["Operations"])
	for _, op := range ops {
		opMap, ok := op.(map[string]any)
		if !ok {
			continue
		}
		operation := parseRestOperation(opMap)
		svc.Operations = append(svc.Operations, operation)
	}

	return svc, nil
}

// parseRestOperation parses a single Rest$RestOperation from BSON.
func parseRestOperation(opMap map[string]any) *model.RestClientOperation {
	op := &model.RestClientOperation{}
	op.Name = extractString(opMap["Name"])
	op.Timeout = extractInt(opMap["Timeout"])

	// Parse Method (polymorphic: WithBody or WithoutBody)
	if methodMap := extractBsonMap(opMap["Method"]); methodMap != nil {
		methodType := extractString(methodMap["$Type"])
		httpMethod := extractString(methodMap["HttpMethod"])
		op.HttpMethod = httpMethodToUpper(httpMethod)

		if methodType == "Rest$RestOperationMethodWithBody" {
			parseRestBody(methodMap["Body"], op)
		}
	}

	// Parse Path from Rest$ValueTemplate
	if pathMap := extractBsonMap(opMap["Path"]); pathMap != nil {
		op.Path = extractString(pathMap["Value"])
	}

	// Parse Headers
	headers := extractBsonArray(opMap["Headers"])
	for _, h := range headers {
		if hMap, ok := h.(map[string]any); ok {
			header := &model.RestClientHeader{
				Name: extractString(hMap["Name"]),
			}
			if valMap := extractBsonMap(hMap["Value"]); valMap != nil {
				header.Value = extractString(valMap["Value"])
			}
			op.Headers = append(op.Headers, header)
		}
	}

	// Parse Parameters (path parameters)
	params := extractBsonArray(opMap["Parameters"])
	for _, p := range params {
		if pMap, ok := p.(map[string]any); ok {
			param := &model.RestClientParameter{
				Name:     extractString(pMap["Name"]),
				DataType: extractRestDataType(pMap["DataType"]),
			}
			op.Parameters = append(op.Parameters, param)
		}
	}

	// Parse QueryParameters
	queryParams := extractBsonArray(opMap["QueryParameters"])
	for _, q := range queryParams {
		if qMap, ok := q.(map[string]any); ok {
			param := &model.RestClientParameter{
				Name:     extractString(qMap["Name"]),
				DataType: "String", // query parameters default to String
			}
			op.QueryParameters = append(op.QueryParameters, param)
		}
	}

	// Parse ResponseHandling (polymorphic)
	if respMap := extractBsonMap(opMap["ResponseHandling"]); respMap != nil {
		respType := extractString(respMap["$Type"])
		switch respType {
		case "Rest$NoResponseHandling":
			op.ResponseType = "NONE"
		case "Rest$ImplicitMappingResponseHandling":
			contentType := extractString(respMap["ContentType"])
			switch contentType {
			case "application/json":
				op.ResponseType = "JSON"
			default:
				op.ResponseType = "JSON" // default for implicit mapping
			}
		}
	}

	return op
}

// parseRestBody extracts body information from the Method's Body field.
func parseRestBody(bodyVal any, op *model.RestClientOperation) {
	bodyMap := extractBsonMap(bodyVal)
	if bodyMap == nil {
		return
	}
	bodyType := extractString(bodyMap["$Type"])
	switch bodyType {
	case "Rest$ImplicitMappingBody":
		op.BodyType = "JSON"
	case "Rest$JsonBody":
		op.BodyType = "JSON"
	case "Rest$StringBody":
		op.BodyType = "FILE" // StringBody with template is used for file uploads
	}
}

// extractRestValue extracts a value from a polymorphic Rest$Value (StringValue or ConstantValue).
func extractRestValue(v any) string {
	valMap := extractBsonMap(v)
	if valMap == nil {
		return ""
	}
	valType := extractString(valMap["$Type"])
	switch valType {
	case "Rest$StringValue":
		return extractString(valMap["Value"])
	case "Rest$ConstantValue":
		return extractString(valMap["Constant"])
	}
	return ""
}

// extractRestDataType extracts a data type name from a DataTypes$DataType BSON object.
// Handles both DataTypes$IntegerType (consumed REST) and DataTypes$IntegerAttributeType formats.
func extractRestDataType(v any) string {
	dtMap := extractBsonMap(v)
	if dtMap == nil {
		return "String"
	}
	dtType := extractString(dtMap["$Type"])
	switch dtType {
	case "DataTypes$IntegerType", "DataTypes$IntegerAttributeType":
		return "Integer"
	case "DataTypes$LongAttributeType":
		return "Long"
	case "DataTypes$DecimalType", "DataTypes$DecimalAttributeType":
		return "Decimal"
	case "DataTypes$BooleanType", "DataTypes$BooleanAttributeType":
		return "Boolean"
	case "DataTypes$StringType", "DataTypes$StringAttributeType":
		return "String"
	default:
		return "String"
	}
}

// httpMethodToUpper converts Mendix HTTP method names to uppercase.
func httpMethodToUpper(method string) string {
	switch method {
	case "Get":
		return "GET"
	case "Post":
		return "POST"
	case "Put":
		return "PUT"
	case "Patch":
		return "PATCH"
	case "Delete":
		return "DELETE"
	case "Head":
		return "HEAD"
	case "Options":
		return "OPTIONS"
	default:
		return method
	}
}
