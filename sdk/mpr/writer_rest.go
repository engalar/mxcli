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

	contents, err := w.serializeConsumedRestService(svc)
	if err != nil {
		return fmt.Errorf("failed to serialize consumed REST service: %w", err)
	}

	return w.insertUnit(string(svc.ID), string(svc.ContainerID), "Documents", "Rest$ConsumedRestService", contents)
}

// UpdateConsumedRestService updates an existing consumed REST service.
func (w *Writer) UpdateConsumedRestService(svc *model.ConsumedRestService) error {
	contents, err := w.serializeConsumedRestService(svc)
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
func (w *Writer) serializeConsumedRestService(svc *model.ConsumedRestService) ([]byte, error) {
	doc := bson.M{
		"$ID":           idToBsonBinary(string(svc.ID)),
		"$Type":         "Rest$ConsumedRestService",
		"Name":          svc.Name,
		"Documentation": svc.Documentation,
		"Excluded":      svc.Excluded,
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
		// Constant reference — strip the $ prefix for the constant name
		return bson.M{
			"$ID":      idToBsonBinary(generateUUID()),
			"$Type":    "Rest$ConstantValue",
			"Constant": strings.TrimPrefix(value, "$"),
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

	if op.Timeout > 0 {
		doc["Timeout"] = int64(op.Timeout)
	}

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
	doc["ResponseHandling"] = serializeRestResponseHandling(op.ResponseType)

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
		doc["Body"] = serializeRestBody(op.BodyType)
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
		doc["Body"] = serializeRestBody("JSON")
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
func serializeRestBody(bodyType string) bson.M {
	switch strings.ToUpper(bodyType) {
	case "JSON":
		return bson.M{
			"$ID":   idToBsonBinary(generateUUID()),
			"$Type": "Rest$JsonBody",
			"Value": "",
		}
	case "FILE":
		return bson.M{
			"$ID":           idToBsonBinary(generateUUID()),
			"$Type":         "Rest$StringBody",
			"ValueTemplate": serializeValueTemplate(""),
		}
	default:
		return bson.M{
			"$ID":   idToBsonBinary(generateUUID()),
			"$Type": "Rest$JsonBody",
			"Value": "",
		}
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

	contents, err := w.serializePublishedRestService(svc)
	if err != nil {
		return fmt.Errorf("failed to serialize published REST service: %w", err)
	}

	return w.insertUnit(string(svc.ID), string(svc.ContainerID), "Documents", "Rest$PublishedRestService", contents)
}

// DeletePublishedRestService deletes a published REST service by ID.
func (w *Writer) DeletePublishedRestService(id model.ID) error {
	return w.deleteUnit(string(id))
}

func (w *Writer) serializePublishedRestService(svc *model.PublishedRestService) ([]byte, error) {
	resources := bson.A{int32(2)}
	for _, res := range svc.Resources {
		ops := bson.A{int32(2)}
		for _, op := range res.Operations {
			opDoc := bson.M{
				"$ID":         idToBsonBinary(GenerateID()),
				"$Type":       "Rest$PublishedRestServiceOperation",
				"HttpMethod":  httpMethodToMendix(op.HTTPMethod),
				"Path":        op.Path,
				"Microflow":   op.Microflow,
				"Summary":     op.Summary,
				"Deprecated":  op.Deprecated,
				"Commit":      "Yes",
				"Documentation": "",
				"ExportMapping": "",
				"ImportMapping": "",
				"ObjectHandlingBackup": "Create",
				"Parameters":  serializePublishedRestParams(op.Path, op.Microflow),
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
		"$ID":                    idToBsonBinary(string(svc.ID)),
		"$Type":                  "Rest$PublishedRestService",
		"Name":                   svc.Name,
		"Documentation":          "",
		"Excluded":               svc.Excluded,
		"ExportLevel":            "Hidden",
		"Path":                   svc.Path,
		"Version":                svc.Version,
		"ServiceName":            svc.ServiceName,
		"AllowedRoles":           bson.A{int32(2)},
		"AuthenticationTypes":    bson.A{int32(2)},
		"AuthenticationMicroflow": "",
		"CorsConfiguration":      nil,
		"Parameters":             bson.A{int32(2)},
		"Resources":              resources,
	}

	return bson.Marshal(doc)
}

// serializePublishedRestParams builds the Parameters array for a published REST operation.
// It auto-extracts path parameters from {paramName} placeholders in the path string,
// then appends any explicitly declared parameters.
// serializePublishedRestParams builds the Parameters array for a published REST operation.
// Auto-extracts path parameters from {paramName} placeholders in the path string.
// microflowQN is the qualified name of the backing microflow (e.g., "Module.MF_Name").
func serializePublishedRestParams(path string, microflowQN string) bson.A {
	names := extractPathParams(path)
	if len(names) == 0 {
		return bson.A{int32(2)}
	}
	// Version marker: 3 for non-empty arrays (Mendix 11.x convention)
	params := bson.A{int32(3)}
	for _, name := range names {
		params = append(params, bson.M{
			"$ID":                idToBsonBinary(generateUUID()),
			"$Type":              "Rest$RestOperationParameter",
			"Name":               name,
			"Description":        "",
			"MicroflowParameter": microflowQN + "." + name,
			"ParameterType":      "Path",
			"Type": bson.M{
				"$ID":   idToBsonBinary(generateUUID()),
				"$Type": "DataTypes$StringType",
			},
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
