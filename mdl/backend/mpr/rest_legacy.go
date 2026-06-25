// SPDX-License-Identifier: Apache-2.0

// rest_compat.go — REST service BSON parsing for ConsumedRestService and
// PublishedRestService. Mirrors the retired sdk/mpr/parser_rest.go logic
// against modelsdk/mpr.Reader's raw bytes.

package mprbackend

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/model"
)

const (
	consumedRestServiceBsonType  = "Rest$ConsumedRestService"
	publishedRestServiceBsonType = "Rest$PublishedRestService"
)

func (b *MprBackend) listConsumedRestServicesFromRaw() ([]*model.ConsumedRestService, error) {
	rawUnits, err := b.msdkReader.ListRawUnitsByType(consumedRestServiceBsonType)
	if err != nil {
		return nil, err
	}
	out := make([]*model.ConsumedRestService, 0, len(rawUnits))
	for _, ru := range rawUnits {
		if ru == nil {
			continue
		}
		svc, err := parseConsumedRestServiceRaw(string(ru.ID), string(ru.ContainerID), ru.Contents)
		if err != nil {
			return nil, fmt.Errorf("parse consumed REST service %s: %w", ru.ID, err)
		}
		out = append(out, svc)
	}
	return out, nil
}

func (b *MprBackend) listPublishedRestServicesFromRaw() ([]*model.PublishedRestService, error) {
	rawUnits, err := b.msdkReader.ListRawUnitsByType(publishedRestServiceBsonType)
	if err != nil {
		return nil, err
	}
	out := make([]*model.PublishedRestService, 0, len(rawUnits))
	for _, ru := range rawUnits {
		if ru == nil {
			continue
		}
		svc, err := parsePublishedRestServiceRaw(string(ru.ID), string(ru.ContainerID), ru.Contents)
		if err != nil {
			return nil, fmt.Errorf("parse published REST service %s: %w", ru.ID, err)
		}
		out = append(out, svc)
	}
	return out, nil
}

func parsePublishedRestServiceRaw(unitID, containerID string, contents []byte) (*model.PublishedRestService, error) {
	if len(contents) < 4 {
		return nil, fmt.Errorf("contents too short")
	}
	var raw map[string]any
	if err := bson.Unmarshal(contents, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal BSON: %w", err)
	}
	svc := &model.PublishedRestService{}
	svc.ID = model.ID(unitID)
	svc.TypeName = publishedRestServiceBsonType
	svc.ContainerID = model.ID(containerID)

	svc.Name = extractString(raw["Name"])
	svc.Path = extractString(raw["Path"])
	svc.Version = extractString(raw["Version"])
	svc.ServiceName = extractString(raw["ServiceName"])
	svc.Excluded = extractBool(raw["Excluded"], false)

	for _, r := range extractBsonArray(raw["AllowedRoles"]) {
		if name, ok := r.(string); ok {
			svc.AllowedRoles = append(svc.AllowedRoles, name)
		}
	}

	for _, res := range extractBsonArray(raw["Resources"]) {
		resMap := extractBsonMap(res)
		if resMap == nil {
			continue
		}
		resource := &model.PublishedRestResource{}
		resource.ID = model.ID(extractBsonID(resMap["$ID"]))
		resource.TypeName = extractString(resMap["$Type"])
		resource.Name = extractString(resMap["Name"])

		for _, op := range extractBsonArray(resMap["Operations"]) {
			opMap := extractBsonMap(op)
			if opMap == nil {
				continue
			}
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
		svc.Resources = append(svc.Resources, resource)
	}
	return svc, nil
}

func parseConsumedRestServiceRaw(unitID, containerID string, contents []byte) (*model.ConsumedRestService, error) {
	if len(contents) < 4 {
		return nil, fmt.Errorf("contents too short")
	}
	var raw map[string]any
	if err := bson.Unmarshal(contents, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal BSON: %w", err)
	}
	svc := &model.ConsumedRestService{}
	svc.ID = model.ID(unitID)
	svc.TypeName = consumedRestServiceBsonType
	svc.ContainerID = model.ID(containerID)

	svc.Name = extractString(raw["Name"])
	svc.Documentation = extractString(raw["Documentation"])
	svc.Excluded = extractBool(raw["Excluded"], false)

	if baseUrlMap := extractBsonMap(raw["BaseUrl"]); baseUrlMap != nil {
		svc.BaseUrl = extractString(baseUrlMap["Value"])
	}

	if authMap := extractBsonMap(raw["AuthenticationScheme"]); authMap != nil {
		if extractString(authMap["$Type"]) == "Rest$BasicAuthenticationScheme" {
			auth := &model.RestAuthentication{Scheme: "Basic"}
			auth.Username = extractRestValue(authMap["Username"])
			auth.Password = extractRestValue(authMap["Password"])
			svc.Authentication = auth
		}
	}

	if openApiFile := extractBsonMap(raw["OpenApiFile"]); openApiFile != nil {
		svc.OpenApiContent = extractString(openApiFile["Content"])
	}

	for _, op := range extractBsonArray(raw["Operations"]) {
		if opMap := extractBsonMap(op); opMap != nil {
			svc.Operations = append(svc.Operations, parseRestOperation(opMap))
		}
	}
	return svc, nil
}

func parseRestOperation(opMap map[string]any) *model.RestClientOperation {
	op := &model.RestClientOperation{}
	op.Name = extractString(opMap["Name"])
	op.Timeout = extractInt(opMap["Timeout"])

	for _, t := range extractBsonArray(opMap["Tags"]) {
		if s, ok := t.(string); ok {
			op.Tags = append(op.Tags, s)
		}
	}

	if methodMap := extractBsonMap(opMap["Method"]); methodMap != nil {
		methodType := extractString(methodMap["$Type"])
		httpMethod := extractString(methodMap["HttpMethod"])
		op.HttpMethod = httpMethodToUpper(httpMethod)
		if methodType == "Rest$RestOperationMethodWithBody" {
			parseRestBody(methodMap["Body"], op)
		}
	}

	if pathMap := extractBsonMap(opMap["Path"]); pathMap != nil {
		op.Path = extractString(pathMap["Value"])
	}

	for _, h := range extractBsonArray(opMap["Headers"]) {
		if hMap := extractBsonMap(h); hMap != nil {
			header := &model.RestClientHeader{Name: extractString(hMap["Name"])}
			if valMap := extractBsonMap(hMap["Value"]); valMap != nil {
				header.Value = extractString(valMap["Value"])
			}
			op.Headers = append(op.Headers, header)
		}
	}

	for _, p := range extractBsonArray(opMap["Parameters"]) {
		if pMap := extractBsonMap(p); pMap != nil {
			op.Parameters = append(op.Parameters, &model.RestClientParameter{
				Name:     extractString(pMap["Name"]),
				DataType: extractRestDataType(pMap["DataType"]),
			})
		}
	}

	for _, q := range extractBsonArray(opMap["QueryParameters"]) {
		if qMap := extractBsonMap(q); qMap != nil {
			op.QueryParameters = append(op.QueryParameters, &model.RestClientParameter{
				Name:     extractString(qMap["Name"]),
				DataType: extractRestDataType(qMap["DataType"]),
			})
		}
	}

	if respMap := extractBsonMap(opMap["ResponseHandling"]); respMap != nil {
		respType := extractString(respMap["$Type"])
		switch respType {
		case "Rest$NoResponseHandling":
			contentType := extractString(respMap["ContentType"])
			switch contentType {
			case "application/json":
				op.ResponseType = "JSON"
			case "text/plain":
				op.ResponseType = "STRING"
			case "application/octet-stream":
				op.ResponseType = "FILE"
			default:
				op.ResponseType = "NONE"
			}
		case "Rest$ImplicitMappingResponseHandling":
			op.ResponseType = "MAPPING"
			if rootMap := extractBsonMap(respMap["RootMappingElement"]); rootMap != nil {
				op.ResponseEntity = extractString(rootMap["Entity"])
				op.ResponseMappings = parseMappingChildren(rootMap)
			}
		}
	}
	return op
}

func parseRestBody(bodyVal any, op *model.RestClientOperation) {
	bodyMap := extractBsonMap(bodyVal)
	if bodyMap == nil {
		return
	}
	bodyType := extractString(bodyMap["$Type"])
	switch bodyType {
	case "Rest$ImplicitMappingBody":
		op.BodyType = "EXPORT_MAPPING"
		if rootMap := extractBsonMap(bodyMap["RootMappingElement"]); rootMap != nil {
			op.BodyVariable = extractString(rootMap["Entity"])
			op.BodyMappings = parseExportMappingChildren(rootMap)
		}
	case "Rest$JsonBody":
		op.BodyType = "JSON"
		op.BodyVariable = extractString(bodyMap["Value"])
	case "Rest$StringBody":
		op.BodyType = "TEMPLATE"
		if vt := extractBsonMap(bodyMap["ValueTemplate"]); vt != nil {
			op.BodyVariable = extractString(vt["Value"])
		}
	}
}

func parseMappingChildren(parentMap map[string]any) []*model.RestResponseMapping {
	parentEntity := extractString(parentMap["Entity"])
	entityPrefix := parentEntity + "."

	var mappings []*model.RestResponseMapping
	for _, child := range extractBsonArray(parentMap["Children"]) {
		childMap := extractBsonMap(child)
		if childMap == nil {
			continue
		}
		switch extractString(childMap["$Type"]) {
		case "ImportMappings$ValueMappingElement":
			attr := extractString(childMap["Attribute"])
			exposed := extractString(childMap["ExposedName"])
			if attr == "" || exposed == "" {
				continue
			}
			mappings = append(mappings, &model.RestResponseMapping{
				Attribute:   strings.TrimPrefix(attr, entityPrefix),
				ExposedName: exposed,
				JsonPath:    extractString(childMap["JsonPath"]),
			})
		case "ImportMappings$ObjectMappingElement":
			mappings = append(mappings, &model.RestResponseMapping{
				Entity:      extractString(childMap["Entity"]),
				Association: extractString(childMap["Association"]),
				ExposedName: extractString(childMap["ExposedName"]),
				JsonPath:    extractString(childMap["JsonPath"]),
				Children:    parseMappingChildren(childMap),
			})
		}
	}
	return mappings
}

func parseExportMappingChildren(parentMap map[string]any) []*model.RestResponseMapping {
	parentEntity := extractString(parentMap["Entity"])
	entityPrefix := parentEntity + "."

	var mappings []*model.RestResponseMapping
	for _, child := range extractBsonArray(parentMap["Children"]) {
		childMap := extractBsonMap(child)
		if childMap == nil {
			continue
		}
		switch extractString(childMap["$Type"]) {
		case "ExportMappings$ValueMappingElement":
			attr := extractString(childMap["Attribute"])
			exposed := extractString(childMap["ExposedName"])
			if attr == "" || exposed == "" {
				continue
			}
			mappings = append(mappings, &model.RestResponseMapping{
				Attribute:   strings.TrimPrefix(attr, entityPrefix),
				ExposedName: exposed,
				JsonPath:    extractString(childMap["JsonPath"]),
			})
		case "ExportMappings$ObjectMappingElement":
			mappings = append(mappings, &model.RestResponseMapping{
				Entity:      extractString(childMap["Entity"]),
				Association: extractString(childMap["Association"]),
				ExposedName: extractString(childMap["ExposedName"]),
				JsonPath:    extractString(childMap["JsonPath"]),
				Children:    parseExportMappingChildren(childMap),
			})
		}
	}
	return mappings
}

func extractRestValue(v any) string {
	valMap := extractBsonMap(v)
	if valMap == nil {
		return ""
	}
	switch extractString(valMap["$Type"]) {
	case "Rest$StringValue":
		return extractString(valMap["Value"])
	case "Rest$ConstantValue":
		if v := extractString(valMap["Value"]); v != "" {
			return "$" + v
		}
		if v := extractString(valMap["Constant"]); v != "" {
			return "$" + v
		}
		return ""
	}
	return ""
}

func extractRestDataType(v any) string {
	dtMap := extractBsonMap(v)
	if dtMap == nil {
		return "String"
	}
	switch extractString(dtMap["$Type"]) {
	case "DataTypes$IntegerType", "DataTypes$IntegerAttributeType":
		return "Integer"
	case "DataTypes$LongType", "DataTypes$LongAttributeType":
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
