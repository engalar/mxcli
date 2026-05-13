// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/model"

	"go.mongodb.org/mongo-driver/bson"
)

// CreateConsumedRestService creates a new consumed REST service document.
func (w *Writer) CreateConsumedRestService(svc *model.ConsumedRestService) error {
	if svc.ID == "" {
		svc.ID = model.ID(generateUUID())
	}
	svc.TypeName = "Rest$ConsumedRestService"

	contents, err := serializeConsumedRestService(svc)
	if err != nil {
		return fmt.Errorf("failed to serialize consumed REST service: %w", err)
	}

	return w.insertUnit(string(svc.ID), string(svc.ContainerID), "Documents", "Rest$ConsumedRestService", contents)
}

// UpdateConsumedRestService updates an existing consumed REST service.
func (w *Writer) UpdateConsumedRestService(svc *model.ConsumedRestService) error {
	contents, err := serializeConsumedRestService(svc)
	if err != nil {
		return fmt.Errorf("failed to serialize consumed REST service: %w", err)
	}

	return w.updateUnit(string(svc.ID), contents)
}

// DeleteConsumedRestService deletes a consumed REST service by ID.
func (w *Writer) DeleteConsumedRestService(id model.ID) error {
	return w.deleteUnit(string(id))
}

// serializeConsumedRestService converts a ConsumedRestService to BSON bytes.
func serializeConsumedRestService(svc *model.ConsumedRestService) ([]byte, error) {
	doc := bson.M{
		"$ID":           idToBsonBinary(string(svc.ID)),
		"$Type":         "Rest$ConsumedRestService",
		"Name":          svc.Name,
		"Documentation": svc.Documentation,
		"Excluded":      svc.Excluded,
		// ExportLevel: whether the document is exposed to other modules/projects.
		// Studio Pro defaults to "Hidden". Missing this field has been observed
		// to cause runtime auth issues (#200).
		"ExportLevel":      "Hidden",
		"BaseUrlParameter": nil,
	}

	// OpenApiFile: only present when the service was created from an OpenAPI spec.
	// Field name and subfield are PascalCase to match Studio Pro serialization.
	// Do NOT write a null entry for manually-created services — Studio Pro omits this field entirely.
	if svc.OpenApiContent != "" {
		doc["OpenApiFile"] = bson.M{
			"$ID":     idToBsonBinary(generateUUID()),
			"$Type":   "Rest$OpenApiFile",
			"Content": svc.OpenApiContent,
		}
	}

	// BaseUrl as Rest$ValueTemplate
	doc["BaseUrl"] = serializeValueTemplate(svc.BaseUrl)

	// AuthenticationScheme: polymorphic (null or Rest$BasicAuthenticationScheme)
	if svc.Authentication == nil {
		doc["AuthenticationScheme"] = nil
	} else {
		doc["AuthenticationScheme"] = serializeRestAuthScheme(svc.Authentication)
	}

	// Operations: versioned array
	ops := bson.A{int32(2)}
	for _, op := range svc.Operations {
		ops = append(ops, serializeRestOperation(op))
	}
	doc["Operations"] = ops

	return bson.Marshal(doc)
}

// serializeValueTemplate creates a Rest$ValueTemplate BSON object.
func serializeValueTemplate(value string) bson.M {
	return bson.M{
		"$ID":   idToBsonBinary(generateUUID()),
		"$Type": "Rest$ValueTemplate",
		"Value": value,
	}
}

// serializeRestAuthScheme converts authentication config to a BSON map.
func serializeRestAuthScheme(auth *model.RestAuthentication) bson.M {
	doc := bson.M{
		"$ID":   idToBsonBinary(generateUUID()),
		"$Type": "Rest$BasicAuthenticationScheme",
	}

	doc["Username"] = serializeRestValue(auth.Username)
	doc["Password"] = serializeRestValue(auth.Password)

	return doc
}

// serializeRestValue creates a polymorphic Rest$Value (StringValue or ConstantValue).
// Values starting with "$" are treated as constant references; others as string literals.
func serializeRestValue(value string) bson.M {
	if strings.HasPrefix(value, "$") {
		// Constant reference — the BSON field is "Value" (QualifiedName of the constant).
		constRef := strings.TrimPrefix(value, "$")
		return bson.M{
			"$ID":   idToBsonBinary(generateUUID()),
			"$Type": "Rest$ConstantValue",
			"Value": constRef,
		}
	}
	return bson.M{
		"$ID":   idToBsonBinary(generateUUID()),
		"$Type": "Rest$StringValue",
		"Value": value,
	}
}

// serializeRestOperation converts a RestClientOperation to a BSON map.
func serializeRestOperation(op *model.RestClientOperation) bson.M {
	doc := bson.M{
		"$ID":   idToBsonBinary(generateUUID()),
		"$Type": "Rest$RestOperation",
		"Name":  op.Name,
	}

	// Timeout: Studio Pro always writes this field; default is 300 seconds.
	timeout := int64(op.Timeout)
	if timeout <= 0 {
		timeout = 300
	}
	doc["Timeout"] = timeout

	// Tags: versioned string array; used by Studio Pro as resource group labels.
	tags := bson.A{int32(1)}
	for _, t := range op.Tags {
		tags = append(tags, t)
	}
	doc["Tags"] = tags

	// Method: polymorphic (WithBody or WithoutBody)
	doc["Method"] = serializeRestMethod(op)

	// Path as Rest$ValueTemplate
	doc["Path"] = serializeValueTemplate(op.Path)

	// Headers: versioned array of Rest$HeaderWithValueTemplate
	headers := bson.A{int32(2)}
	hasAccept := false
	for _, h := range op.Headers {
		headers = append(headers, serializeRestHeader(h))
		if strings.EqualFold(h.Name, "Accept") {
			hasAccept = true
		}
	}
	// Mendix requires an Accept header on every consumed REST operation (CE7062)
	if !hasAccept {
		headers = append(headers, serializeRestHeader(&model.RestClientHeader{Name: "Accept", Value: "*/*"}))
	}
	doc["Headers"] = headers

	// Parameters: versioned array of Rest$RestOperationParameter (path params)
	params := bson.A{int32(2)}
	for _, p := range op.Parameters {
		params = append(params, serializeRestParameter(p))
	}
	doc["Parameters"] = params

	// QueryParameters: versioned array of Rest$QueryParameter
	queryParams := bson.A{int32(2)}
	for _, q := range op.QueryParameters {
		queryParams = append(queryParams, serializeRestQueryParameter(q))
	}
	doc["QueryParameters"] = queryParams

	// ResponseHandling: polymorphic
	if op.ResponseType == "MAPPING" && op.ResponseEntity != "" && len(op.ResponseMappings) > 0 {
		doc["ResponseHandling"] = serializeRestImplicitMappingResponse(op.ResponseEntity, op.ResponseMappings)
	} else {
		doc["ResponseHandling"] = serializeRestResponseHandling(op.ResponseType)
	}

	return doc
}

// serializeRestMethod creates the polymorphic Method field.
// Methods with bodies (POST, PUT, PATCH) use Rest$RestOperationMethodWithBody;
// others use Rest$RestOperationMethodWithoutBody.
func serializeRestMethod(op *model.RestClientOperation) bson.M {
	httpMethod := httpMethodToMendix(op.HttpMethod)

	if op.BodyType != "" {
		// Method with explicit body
		doc := bson.M{
			"$ID":        idToBsonBinary(generateUUID()),
			"$Type":      "Rest$RestOperationMethodWithBody",
			"HttpMethod": httpMethod,
		}
		if op.BodyType == "EXPORT_MAPPING" && len(op.BodyMappings) > 0 {
			doc["Body"] = serializeRestImplicitMappingBody(op.BodyVariable, op.BodyMappings)
		} else {
			doc["Body"] = serializeRestBody(op.BodyType, op.BodyVariable)
		}
		return doc
	}

	// POST, PUT, PATCH must include a body even if not explicitly specified (CE7064)
	methodUpper := strings.ToUpper(op.HttpMethod)
	if methodUpper == "POST" || methodUpper == "PUT" || methodUpper == "PATCH" {
		doc := bson.M{
			"$ID":        idToBsonBinary(generateUUID()),
			"$Type":      "Rest$RestOperationMethodWithBody",
			"HttpMethod": httpMethod,
		}
		doc["Body"] = serializeRestBody("JSON", op.BodyVariable)
		return doc
	}

	// Method without body
	return bson.M{
		"$ID":        idToBsonBinary(generateUUID()),
		"$Type":      "Rest$RestOperationMethodWithoutBody",
		"HttpMethod": httpMethod,
	}
}

// serializeRestBody creates a polymorphic Body field.
// Uses Rest$JsonBody instead of Rest$ImplicitMappingBody to avoid CE7247/CE0061
// (ImplicitMappingBody requires entity mapping which isn't supported yet).
//
// bodyExpr is a Mendix expression (typically "$variableName") that produces
// the JSON or file body at call time. It is stored verbatim in the BSON Value
// field so a roundtrip preserves it.
func serializeRestBody(bodyType, bodyExpr string) bson.M {
	switch strings.ToUpper(bodyType) {
	case "JSON":
		return bson.M{
			"$ID":   idToBsonBinary(generateUUID()),
			"$Type": "Rest$JsonBody",
			"Value": bodyExpr,
		}
	case "FILE", "TEMPLATE":
		return bson.M{
			"$ID":           idToBsonBinary(generateUUID()),
			"$Type":         "Rest$StringBody",
			"ValueTemplate": serializeValueTemplate(bodyExpr),
		}
	default:
		return bson.M{
			"$ID":   idToBsonBinary(generateUUID()),
			"$Type": "Rest$JsonBody",
			"Value": bodyExpr,
		}
	}
}

// serializeRestImplicitMappingBody creates a Rest$ImplicitMappingBody with an inline
// export mapping tree (ExportMappings$ObjectMappingElement). Used for Body: MAPPING Entity { ... }.
func serializeRestImplicitMappingBody(entity string, mappings []*model.RestResponseMapping) bson.M {
	rootElement := serializeInlineMappingElement(entity, "", "", "(Object)", mappings, "ExportMappings", "Parameter")

	return bson.M{
		"$ID":                idToBsonBinary(generateUUID()),
		"$Type":              "Rest$ImplicitMappingBody",
		"RootMappingElement": rootElement,
		"TestValue": bson.M{
			"$ID":   idToBsonBinary(generateUUID()),
			"$Type": "Rest$StringValue",
			"Value": "",
		},
	}
}

// serializeRestImplicitMappingResponse creates a Rest$ImplicitMappingResponseHandling with an
// inline import mapping tree (ImportMappings$ObjectMappingElement). Used for Response: MAPPING Entity { ... }.
func serializeRestImplicitMappingResponse(entity string, mappings []*model.RestResponseMapping) bson.M {
	rootElement := serializeInlineMappingElement(entity, "", "", "(Object)", mappings, "ImportMappings", "Create")

	return bson.M{
		"$ID":                idToBsonBinary(generateUUID()),
		"$Type":              "Rest$ImplicitMappingResponseHandling",
		"ContentType":        "application/json",
		"RootMappingElement": rootElement,
		"StatusCode":         int32(200),
	}
}

// serializeInlineMappingElement creates a single ObjectMappingElement with children for inline REST mappings.
// namespace is "ImportMappings" or "ExportMappings". objectHandling is "Create" or "Parameter".
func serializeInlineMappingElement(entity, association, exposedName, jsonPath string, mappings []*model.RestResponseMapping, namespace, objectHandling string) bson.M {
	children := bson.A{int32(2)}
	for _, m := range mappings {
		if m.Entity != "" {
			// Nested object mapping
			childJsonPath := jsonPath + "|" + m.ExposedName
			child := serializeInlineMappingElement(m.Entity, m.Association, m.ExposedName, childJsonPath, m.Children, namespace, "Create")
			children = append(children, child)
		} else {
			// Value mapping
			valueJsonPath := m.JsonPath
			if valueJsonPath == "" {
				valueJsonPath = jsonPath + "|" + m.ExposedName
			}
			attrQN := entity + "." + m.Attribute
			children = append(children, bson.M{
				"$ID":              idToBsonBinary(generateUUID()),
				"$Type":            namespace + "$ValueMappingElement",
				"Attribute":        attrQN,
				"ExposedName":      m.ExposedName,
				"JsonPath":         valueJsonPath,
				"XmlPath":          "",
				"IsKey":            false,
				"Type":             bson.M{"$ID": idToBsonBinary(generateUUID()), "$Type": "DataTypes$StringType"},
				"MinOccurs":        int32(0),
				"MaxOccurs":        int32(1),
				"Nillable":         true,
				"IsDefaultType":    false,
				"ElementType":      "Value",
				"Documentation":    "",
				"Converter":        "",
				"FractionDigits":   int32(-1),
				"TotalDigits":      int32(-1),
				"MaxLength":        int32(0),
				"IsContent":        false,
				"IsXmlAttribute":   false,
				"OriginalValue":    "",
				"XmlPrimitiveType": "String",
			})
		}
	}

	minOccurs := int32(1)
	if association != "" {
		minOccurs = 0
	}

	return bson.M{
		"$ID":                               idToBsonBinary(generateUUID()),
		"$Type":                             namespace + "$ObjectMappingElement",
		"Entity":                            entity,
		"ExposedName":                       exposedName,
		"JsonPath":                          jsonPath,
		"XmlPath":                           "",
		"ObjectHandling":                    objectHandling,
		"ObjectHandlingBackup":              "Create",
		"ObjectHandlingBackupAllowOverride": false,
		"Association":                       association,
		"Children":                          children,
		"MinOccurs":                         minOccurs,
		"MaxOccurs":                         int32(1),
		"Nillable":                          true,
		"IsDefaultType":                     false,
		"ElementType":                       "Object",
		"Documentation":                     "",
		"CustomHandlerCall":                 nil,
	}
}

// serializeRestHeader creates a Rest$HeaderWithValueTemplate BSON object.
func serializeRestHeader(h *model.RestClientHeader) bson.M {
	return bson.M{
		"$ID":   idToBsonBinary(generateUUID()),
		"$Type": "Rest$HeaderWithValueTemplate",
		"Name":  h.Name,
		"Value": serializeValueTemplate(h.Value),
	}
}

// serializeRestParameter creates a Rest$OperationParameter BSON object.
// This is the correct type for consumed REST operation parameters
// (distinct from Rest$RestOperationParameter used in published REST services).
func serializeRestParameter(p *model.RestClientParameter) bson.M {
	return bson.M{
		"$ID":      idToBsonBinary(generateUUID()),
		"$Type":    "Rest$OperationParameter",
		"Name":     p.Name,
		"DataType": serializeRestDataType(p.DataType),
	}
}

// serializeRestQueryParameter creates a Rest$QueryParameter BSON object.
func serializeRestQueryParameter(p *model.RestClientParameter) bson.M {
	return bson.M{
		"$ID":   idToBsonBinary(generateUUID()),
		"$Type": "Rest$QueryParameter",
		"Name":  p.Name,
		"ParameterUsage": bson.M{
			"$ID":   idToBsonBinary(generateUUID()),
			"$Type": "Rest$RequiredQueryParameterUsage",
		},
	}
}

// serializeRestResponseHandling creates a polymorphic ResponseHandling BSON object.
// Uses Rest$NoResponseHandling for all types to avoid CE0061 (ImplicitMappingResponseHandling
// requires entity mapping which isn't supported yet). ContentType is set to enable roundtripping.
func serializeRestResponseHandling(responseType string) bson.M {
	doc := bson.M{
		"$ID":   idToBsonBinary(generateUUID()),
		"$Type": "Rest$NoResponseHandling",
	}
	switch strings.ToUpper(responseType) {
	case "JSON":
		doc["ContentType"] = "application/json"
	case "STRING":
		doc["ContentType"] = "text/plain"
	case "FILE":
		doc["ContentType"] = "application/octet-stream"
	}
	return doc
}

// serializeRestDataType converts a simple type name to a BSON DataType object.
// REST operation parameters use the DataTypes$ namespace with simple type names
// (e.g., DataTypes$IntegerType, not DataTypes$IntegerAttributeType).
func serializeRestDataType(typeName string) bson.M {
	bsonType := "DataTypes$StringType"
	switch typeName {
	case "Integer":
		bsonType = "DataTypes$IntegerType"
	case "Long":
		bsonType = "DataTypes$IntegerType" // Long maps to IntegerType in DataTypes
	case "Decimal":
		bsonType = "DataTypes$DecimalType"
	case "Boolean":
		bsonType = "DataTypes$BooleanType"
	case "String":
		bsonType = "DataTypes$StringType"
	}
	return bson.M{
		"$ID":   idToBsonBinary(generateUUID()),
		"$Type": bsonType,
	}
}

// CreatePublishedRestService creates a new published REST service document.
func (w *Writer) CreatePublishedRestService(svc *model.PublishedRestService) error {
	if svc.ID == "" {
		svc.ID = model.ID(generateUUID())
	}
	svc.TypeName = "Rest$PublishedRestService"

	contents, err := serializePublishedRestService(svc)
	if err != nil {
		return fmt.Errorf("failed to serialize published REST service: %w", err)
	}

	return w.insertUnit(string(svc.ID), string(svc.ContainerID), "Documents", "Rest$PublishedRestService", contents)
}

// DeletePublishedRestService deletes a published REST service by ID.
func (w *Writer) DeletePublishedRestService(id model.ID) error {
	return w.deleteUnit(string(id))
}

// UpdatePublishedRestService re-serializes an existing published REST
// service. Used by ALTER PUBLISHED REST SERVICE.
func (w *Writer) UpdatePublishedRestService(svc *model.PublishedRestService) error {
	contents, err := serializePublishedRestService(svc)
	if err != nil {
		return fmt.Errorf("failed to serialize published REST service: %w", err)
	}
	return w.updateUnit(string(svc.ID), contents)
}

func serializePublishedRestService(svc *model.PublishedRestService) ([]byte, error) {
	resources := bson.A{int32(2)}
	for _, res := range svc.Resources {
		ops := bson.A{int32(2)}
		for _, op := range res.Operations {
			opDoc := bson.M{
				"$ID":                  idToBsonBinary(GenerateID()),
				"$Type":                "Rest$PublishedRestServiceOperation",
				"HttpMethod":           httpMethodToMendix(op.HTTPMethod),
				"Path":                 op.Path,
				"Microflow":            op.Microflow,
				"Summary":              op.Summary,
				"Deprecated":           op.Deprecated,
				"Commit":               "Yes",
				"Documentation":        "",
				"ExportMapping":        "",
				"ImportMapping":        "",
				"ObjectHandlingBackup": "Create",
				"Parameters":           serializePublishedRestParams(op.Path, op.Microflow, op.Parameters),
			}
			ops = append(ops, opDoc)
		}
		resDoc := bson.M{
			"$ID":           idToBsonBinary(GenerateID()),
			"$Type":         "Rest$PublishedRestServiceResource",
			"Name":          res.Name,
			"Documentation": "",
			"Operations":    ops,
		}
		resources = append(resources, resDoc)
	}

	doc := bson.M{
		"$ID":                     idToBsonBinary(string(svc.ID)),
		"$Type":                   "Rest$PublishedRestService",
		"Name":                    svc.Name,
		"Documentation":           "",
		"Excluded":                svc.Excluded,
		"ExportLevel":             "Hidden",
		"Path":                    svc.Path,
		"Version":                 svc.Version,
		"ServiceName":             svc.ServiceName,
		"AllowedRoles":            makeMendixStringArray(svc.AllowedRoles),
		"AuthenticationTypes":     bson.A{int32(2)},
		"AuthenticationMicroflow": "",
		"CorsConfiguration":       nil,
		"Parameters":              bson.A{int32(2)},
		"Resources":               resources,
	}

	return bson.Marshal(doc)
}

// serializePublishedRestParams builds the Parameters array for a published REST operation.
// It auto-extracts path parameters from {paramName} placeholders in the path string,
// then appends any explicitly declared parameters.
//
// Each parameter must include:
//   - Type: a structured DataTypes$StringType object (not a bare string)
//   - ParameterType: "Path" (vs Query/Header/Body)
//   - MicroflowParameter: qualified name of the matching microflow parameter,
//     so Mendix wires the path value to the handler. Without this, mx check
//     reports CE6538 "Parameter is not passed to a microflow parameter" and
//     CE0350 "Microflow has a parameter that is not a parameter of the operation".
func serializePublishedRestParams(path string, microflowQN string, _ []string) bson.A {
	params := bson.A{int32(2)}
	// Extract {paramName} from path
	for _, name := range extractPathParams(path) {
		// MicroflowParameter format: "Module.Microflow.parameterName"
		mfParam := ""
		if microflowQN != "" {
			mfParam = microflowQN + "." + name
		}
		params = append(params, bson.M{
			"$ID":   idToBsonBinary(generateUUID()),
			"$Type": "Rest$RestOperationParameter",
			"Name":  name,
			"Type": bson.M{
				"$ID":   idToBsonBinary(generateUUID()),
				"$Type": "DataTypes$StringType",
			},
			"ParameterType":      "Path",
			"MicroflowParameter": mfParam,
			"Description":        "",
		})
	}
	return params
}

// extractPathParams returns parameter names from {param} placeholders in a path.
func extractPathParams(path string) []string {
	var names []string
	for {
		start := strings.Index(path, "{")
		if start < 0 {
			break
		}
		end := strings.Index(path[start:], "}")
		if end < 0 {
			break
		}
		names = append(names, path[start+1:start+end])
		path = path[start+end+1:]
	}
	return names
}

// httpMethodToMendix converts uppercase HTTP method names to Mendix casing.
func httpMethodToMendix(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return "Get"
	case "POST":
		return "Post"
	case "PUT":
		return "Put"
	case "PATCH":
		return "Patch"
	case "DELETE":
		return "Delete"
	case "HEAD":
		return "Head"
	case "OPTIONS":
		return "Options"
	default:
		return method
	}
}
