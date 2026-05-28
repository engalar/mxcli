// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"strings"

	"github.com/mendixlabs/mxcli/model"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ============================================================================
// Consumed OData Service — Rest$ConsumedODataService
// ============================================================================

// SerializeConsumedODataService returns BSON bytes for a consumed OData service unit.
func SerializeConsumedODataService(svc *model.ConsumedODataService) ([]byte, error) {
	doc := bson.M{
		"$ID":                  idToBsonBinary(string(svc.ID)),
		"$Type":                "Rest$ConsumedODataService",
		"Name":                 svc.Name,
		"Documentation":        svc.Documentation,
		"Version":              svc.Version,
		"ServiceName":          svc.ServiceName,
		"ODataVersion":         svc.ODataVersion,
		"MetadataUrl":          svc.MetadataUrl,
		"TimeoutExpression":    svc.TimeoutExpression,
		"ProxyType":            svc.ProxyType,
		"Description":          svc.Description,
		"Validated":            svc.Validated,
		"Excluded":             svc.Excluded,
		"ExportLevel":          "Hidden",
		"Metadata":             svc.Metadata,
		"MetadataHash":         svc.MetadataHash,
		"MetadataReferences":   bson.A{int32(0)},
		"ValidatedEntities":    bson.A{int32(0)},
		"LastUpdated":          "",
		"UseQuerySegment":      false,
		"MinimumMxVersion":     "",
		"RecommendedMxVersion": "",
	}

	if svc.ConfigurationMicroflow != "" {
		doc["ConfigurationMicroflow"] = svc.ConfigurationMicroflow
	}
	if svc.ErrorHandlingMicroflow != "" {
		doc["ErrorHandlingMicroflow"] = svc.ErrorHandlingMicroflow
	}
	if svc.ProxyHost != "" {
		doc["ProxyHost"] = svc.ProxyHost
	}
	if svc.ProxyPort != "" {
		doc["ProxyPort"] = svc.ProxyPort
	}
	if svc.ProxyUsername != "" {
		doc["ProxyUsername"] = svc.ProxyUsername
	}
	if svc.ProxyPassword != "" {
		doc["ProxyPassword"] = svc.ProxyPassword
	}
	if svc.ApplicationId != "" {
		doc["ApplicationId"] = svc.ApplicationId
	}
	if svc.EndpointId != "" {
		doc["EndpointId"] = svc.EndpointId
	}
	if svc.CatalogUrl != "" {
		doc["CatalogUrl"] = svc.CatalogUrl
	}
	if svc.EnvironmentType != "" {
		doc["EnvironmentType"] = svc.EnvironmentType
	}

	doc["HttpConfiguration"] = serWebHttpConfiguration(svc.HttpConfiguration)

	return bson.Marshal(doc)
}

// serWebHttpConfiguration converts an HttpConfiguration to a BSON map.
func serWebHttpConfiguration(cfg *model.HttpConfiguration) bson.M {
	cfgID := generateUUID()
	if cfg != nil && cfg.ID != "" {
		cfgID = string(cfg.ID)
	}

	doc := bson.M{
		"$ID":                        idToBsonBinary(cfgID),
		"$Type":                      "Microflows$HttpConfiguration",
		"UseHttpAuthentication":      false,
		"HttpAuthenticationUserName": "",
		"HttpAuthenticationPassword": "",
		"HttpMethod":                 "Post",
		"OverrideLocation":           false,
		"CustomLocation":             "",
		"ClientCertificate":          "",
	}

	if cfg != nil {
		doc["UseHttpAuthentication"] = cfg.UseAuthentication
		doc["HttpAuthenticationUserName"] = cfg.Username
		doc["HttpAuthenticationPassword"] = cfg.Password
		if cfg.HttpMethod != "" {
			doc["HttpMethod"] = cfg.HttpMethod
		}
		doc["OverrideLocation"] = cfg.OverrideLocation
		doc["CustomLocation"] = cfg.CustomLocation
		doc["ClientCertificate"] = cfg.ClientCertificate

		if len(cfg.HeaderEntries) > 0 {
			headers := bson.A{int32(3)}
			for _, h := range cfg.HeaderEntries {
				hID := string(h.ID)
				if hID == "" {
					hID = generateUUID()
				}
				headers = append(headers, bson.M{
					"$ID":   idToBsonBinary(hID),
					"$Type": "Microflows$HttpHeaderEntry",
					"Key":   h.Key,
					"Value": h.Value,
				})
			}
			doc["HttpHeaderEntries"] = headers
		}
	}

	return doc
}

// ============================================================================
// Published OData Service — ODataPublish$PublishedODataService2
// ============================================================================

// SerializePublishedODataService returns BSON bytes for a published OData service unit.
func SerializePublishedODataService(svc *model.PublishedODataService) ([]byte, error) {
	// Authentication types array (versioned: starts with int32(3))
	authTypes := bson.A{int32(3)}
	for _, at := range svc.AuthenticationTypes {
		authTypes = append(authTypes, at)
	}

	// Serialize entity types and build ID map for entity set pointers
	entityTypeIDMap := make(map[string]string)
	entityTypes := bson.A{}
	for _, et := range svc.EntityTypes {
		etID := string(et.ID)
		if etID == "" {
			etID = generateUUID()
			et.ID = model.ID(etID)
		}
		entityTypeIDMap[et.ExposedName] = etID
		entityTypes = append(entityTypes, serWebPublishedEntityType(et))
	}

	entitySets := bson.A{}
	for _, es := range svc.EntitySets {
		esID := string(es.ID)
		if esID == "" {
			esID = generateUUID()
			es.ID = model.ID(esID)
		}
		entityTypeID := entityTypeIDMap[es.EntityTypeName]
		entitySets = append(entitySets, serWebPublishedEntitySet(es, entityTypeID))
	}

	doc := bson.M{
		"$ID":                     idToBsonBinary(string(svc.ID)),
		"$Type":                   "ODataPublish$PublishedODataService2",
		"Name":                    svc.Name,
		"Documentation":           svc.Documentation,
		"Path":                    svc.Path,
		"Namespace":               svc.Namespace,
		"ServiceName":             svc.ServiceName,
		"Version":                 svc.Version,
		"ODataVersion":            svc.ODataVersion,
		"Summary":                 svc.Summary,
		"Description":             svc.Description,
		"PublishAssociations":     svc.PublishAssociations,
		"UseGeneralization":       svc.UseGeneralization,
		"AuthenticationMicroflow": svc.AuthMicroflow,
		"AuthenticationTypes":     authTypes,
		"EntityTypes":             entityTypes,
		"EntitySets":              entitySets,
		"Excluded":                svc.Excluded,
	}
	return bson.Marshal(doc)
}

func serWebPublishedEntityType(et *model.PublishedEntityType) bson.M {
	members := bson.A{}
	for _, m := range et.Members {
		members = append(members, serWebPublishedMember(m))
	}
	return bson.M{
		"$ID":          idToBsonBinary(string(et.ID)),
		"$Type":        "ODataPublish$EntityType",
		"Entity":       et.Entity,
		"ExposedName":  et.ExposedName,
		"Summary":      et.Summary,
		"Description":  et.Description,
		"ChildMembers": members,
	}
}

func serWebPublishedEntitySet(es *model.PublishedEntitySet, entityTypeID string) bson.M {
	doc := bson.M{
		"$ID":         idToBsonBinary(string(es.ID)),
		"$Type":       "ODataPublish$EntitySet",
		"ExposedName": es.ExposedName,
		"UsePaging":   es.UsePaging,
		"PageSize":    int64(es.PageSize),
	}

	if entityTypeID != "" {
		doc["EntityTypePointer"] = idToBsonBinary(entityTypeID)
	}
	if es.ReadMode != "" {
		doc["ReadMode"] = serWebReadMode(es.ReadMode)
	}
	if es.InsertMode != "" {
		doc["InsertMode"] = serWebChangeMode(es.InsertMode)
	}
	if es.UpdateMode != "" {
		doc["UpdateMode"] = serWebChangeMode(es.UpdateMode)
	}
	if es.DeleteMode != "" {
		doc["DeleteMode"] = serWebChangeMode(es.DeleteMode)
	}
	return doc
}

func serWebPublishedMember(m *model.PublishedMember) bson.M {
	memberID := string(m.ID)
	if memberID == "" {
		memberID = generateUUID()
	}

	doc := bson.M{
		"$ID":         idToBsonBinary(memberID),
		"ExposedName": m.ExposedName,
		"Filterable":  m.Filterable,
		"Sortable":    m.Sortable,
		"IsPartOfKey": m.IsPartOfKey,
	}

	switch m.Kind {
	case "attribute":
		doc["$Type"] = "ODataPublish$PublishedAttribute"
		doc["Attribute"] = m.Name
	case "association":
		doc["$Type"] = "ODataPublish$PublishedAssociationEnd"
		doc["Association"] = m.Name
	case "id":
		doc["$Type"] = "ODataPublish$PublishedId"
		doc["Attribute"] = m.Name
	default:
		doc["$Type"] = "ODataPublish$PublishedAttribute"
		doc["Attribute"] = m.Name
	}
	return doc
}

func serWebReadMode(mode string) bson.M {
	modeID := idToBsonBinary(generateUUID())
	switch {
	case strings.EqualFold(mode, "ReadFromDatabase") || strings.EqualFold(mode, "SOURCE"):
		return bson.M{"$ID": modeID, "$Type": "ODataPublish$ReadSource"}
	case strings.HasPrefix(mode, "CallMicroflow:"):
		return bson.M{"$ID": modeID, "$Type": "ODataPublish$CallMicroflowToRead", "Microflow": strings.TrimPrefix(mode, "CallMicroflow:")}
	case strings.HasPrefix(mode, "MICROFLOW "):
		return bson.M{"$ID": modeID, "$Type": "ODataPublish$CallMicroflowToRead", "Microflow": strings.TrimPrefix(mode, "MICROFLOW ")}
	default:
		return bson.M{"$ID": modeID, "$Type": "ODataPublish$ReadSource"}
	}
}

func serWebChangeMode(mode string) bson.M {
	modeID := idToBsonBinary(generateUUID())
	switch {
	case strings.EqualFold(mode, "ChangeFromDatabase") || strings.EqualFold(mode, "SOURCE"):
		return bson.M{"$ID": modeID, "$Type": "ODataPublish$ChangeSource"}
	case strings.EqualFold(mode, "NotSupported") || strings.EqualFold(mode, "NOT_SUPPORTED"):
		return bson.M{"$ID": modeID, "$Type": "ODataPublish$ChangeNotSupported"}
	case strings.HasPrefix(mode, "CallMicroflow:"):
		return bson.M{"$ID": modeID, "$Type": "ODataPublish$CallMicroflowToChange", "Microflow": strings.TrimPrefix(mode, "CallMicroflow:")}
	case strings.HasPrefix(mode, "MICROFLOW "):
		return bson.M{"$ID": modeID, "$Type": "ODataPublish$CallMicroflowToChange", "Microflow": strings.TrimPrefix(mode, "MICROFLOW ")}
	default:
		return bson.M{"$ID": modeID, "$Type": "ODataPublish$ChangeNotSupported"}
	}
}

// ============================================================================
// Consumed REST Service — Rest$ConsumedRestService
// ============================================================================

// SerializeConsumedRestService returns BSON bytes for a consumed REST service unit.
func SerializeConsumedRestService(svc *model.ConsumedRestService) ([]byte, error) {
	doc := bson.M{
		"$ID":              idToBsonBinary(string(svc.ID)),
		"$Type":            "Rest$ConsumedRestService",
		"Name":             svc.Name,
		"Documentation":    svc.Documentation,
		"Excluded":         svc.Excluded,
		"ExportLevel":      "Hidden",
		"BaseUrlParameter": nil,
	}

	if svc.OpenApiContent != "" {
		doc["OpenApiFile"] = bson.M{
			"$ID":     idToBsonBinary(generateUUID()),
			"$Type":   "Rest$OpenApiFile",
			"Content": svc.OpenApiContent,
		}
	}

	doc["BaseUrl"] = serWebValueTemplate(svc.BaseUrl)

	if svc.Authentication == nil {
		doc["AuthenticationScheme"] = nil
	} else {
		doc["AuthenticationScheme"] = serWebRestAuthScheme(svc.Authentication)
	}

	ops := bson.A{int32(2)}
	for _, op := range svc.Operations {
		ops = append(ops, serWebRestOperation(op))
	}
	doc["Operations"] = ops

	return bson.Marshal(doc)
}

func serWebValueTemplate(value string) bson.M {
	return bson.M{
		"$ID":   idToBsonBinary(generateUUID()),
		"$Type": "Rest$ValueTemplate",
		"Value": value,
	}
}

func serWebRestAuthScheme(auth *model.RestAuthentication) bson.M {
	doc := bson.M{
		"$ID":   idToBsonBinary(generateUUID()),
		"$Type": "Rest$BasicAuthenticationScheme",
	}
	doc["Username"] = serWebRestValue(auth.Username)
	doc["Password"] = serWebRestValue(auth.Password)
	return doc
}

func serWebRestValue(value string) bson.M {
	if strings.HasPrefix(value, "$") {
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

func serWebRestOperation(op *model.RestClientOperation) bson.M {
	doc := bson.M{
		"$ID":   idToBsonBinary(generateUUID()),
		"$Type": "Rest$RestOperation",
		"Name":  op.Name,
	}

	timeout := int64(op.Timeout)
	if timeout <= 0 {
		timeout = 300
	}
	doc["Timeout"] = timeout

	tags := bson.A{int32(1)}
	for _, t := range op.Tags {
		tags = append(tags, t)
	}
	doc["Tags"] = tags

	doc["Method"] = serWebRestMethod(op)
	doc["Path"] = serWebValueTemplate(op.Path)

	headers := bson.A{int32(2)}
	hasAccept := false
	for _, h := range op.Headers {
		headers = append(headers, serWebRestHeader(h))
		if strings.EqualFold(h.Name, "Accept") {
			hasAccept = true
		}
	}
	if !hasAccept {
		headers = append(headers, serWebRestHeader(&model.RestClientHeader{Name: "Accept", Value: "*/*"}))
	}
	doc["Headers"] = headers

	params := bson.A{int32(2)}
	for _, p := range op.Parameters {
		params = append(params, serWebRestParameter(p))
	}
	doc["Parameters"] = params

	queryParams := bson.A{int32(2)}
	for _, q := range op.QueryParameters {
		queryParams = append(queryParams, serWebRestQueryParameter(q))
	}
	doc["QueryParameters"] = queryParams

	if op.ResponseType == "MAPPING" && op.ResponseEntity != "" && len(op.ResponseMappings) > 0 {
		doc["ResponseHandling"] = serWebRestImplicitMappingResponse(op.ResponseEntity, op.ResponseMappings)
	} else {
		doc["ResponseHandling"] = serWebRestResponseHandling(op.ResponseType)
	}

	return doc
}

func serWebRestMethod(op *model.RestClientOperation) bson.M {
	httpMethod := serWebHttpMethodToMendix(op.HttpMethod)

	if op.BodyType != "" {
		bodyDoc := bson.M{
			"$ID":        idToBsonBinary(generateUUID()),
			"$Type":      "Rest$RestOperationMethodWithBody",
			"HttpMethod": httpMethod,
		}
		if op.BodyType == "EXPORT_MAPPING" && len(op.BodyMappings) > 0 {
			bodyDoc["Body"] = serWebRestImplicitMappingBody(op.BodyVariable, op.BodyMappings)
		} else {
			bodyDoc["Body"] = serWebRestBody(op.BodyType, op.BodyVariable)
		}
		return bodyDoc
	}

	methodUpper := strings.ToUpper(op.HttpMethod)
	if methodUpper == "POST" || methodUpper == "PUT" || methodUpper == "PATCH" {
		bodyDoc := bson.M{
			"$ID":        idToBsonBinary(generateUUID()),
			"$Type":      "Rest$RestOperationMethodWithBody",
			"HttpMethod": httpMethod,
		}
		bodyDoc["Body"] = serWebRestBody("JSON", op.BodyVariable)
		return bodyDoc
	}

	return bson.M{
		"$ID":        idToBsonBinary(generateUUID()),
		"$Type":      "Rest$RestOperationMethodWithoutBody",
		"HttpMethod": httpMethod,
	}
}

func serWebRestBody(bodyType, bodyExpr string) bson.M {
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
			"ValueTemplate": serWebValueTemplate(bodyExpr),
		}
	default:
		return bson.M{
			"$ID":   idToBsonBinary(generateUUID()),
			"$Type": "Rest$JsonBody",
			"Value": bodyExpr,
		}
	}
}

func serWebRestImplicitMappingBody(entity string, mappings []*model.RestResponseMapping) bson.M {
	rootElement := serWebInlineMappingElement(entity, "", "", "(Object)", mappings, "ExportMappings", "Parameter")
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

func serWebRestImplicitMappingResponse(entity string, mappings []*model.RestResponseMapping) bson.M {
	rootElement := serWebInlineMappingElement(entity, "", "", "(Object)", mappings, "ImportMappings", "Create")
	return bson.M{
		"$ID":                idToBsonBinary(generateUUID()),
		"$Type":              "Rest$ImplicitMappingResponseHandling",
		"ContentType":        "application/json",
		"RootMappingElement": rootElement,
		"StatusCode":         int32(200),
	}
}

func serWebInlineMappingElement(entity, association, exposedName, jsonPath string, mappings []*model.RestResponseMapping, namespace, objectHandling string) bson.M {
	children := bson.A{int32(2)}
	for _, m := range mappings {
		if m.Entity != "" {
			childJsonPath := jsonPath + "|" + m.ExposedName
			child := serWebInlineMappingElement(m.Entity, m.Association, m.ExposedName, childJsonPath, m.Children, namespace, "Create")
			children = append(children, child)
		} else {
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

func serWebRestHeader(h *model.RestClientHeader) bson.M {
	return bson.M{
		"$ID":   idToBsonBinary(generateUUID()),
		"$Type": "Rest$HeaderWithValueTemplate",
		"Name":  h.Name,
		"Value": serWebValueTemplate(h.Value),
	}
}

func serWebRestParameter(p *model.RestClientParameter) bson.M {
	return bson.M{
		"$ID":      idToBsonBinary(generateUUID()),
		"$Type":    "Rest$OperationParameter",
		"Name":     p.Name,
		"DataType": serWebRestDataType(p.DataType),
	}
}

func serWebRestQueryParameter(p *model.RestClientParameter) bson.M {
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

func serWebRestResponseHandling(responseType string) bson.M {
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

func serWebRestDataType(typeName string) bson.M {
	bsonType := "DataTypes$StringType"
	switch typeName {
	case "Integer":
		bsonType = "DataTypes$IntegerType"
	case "Long":
		bsonType = "DataTypes$IntegerType"
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

// ============================================================================
// Published REST Service — Rest$PublishedRestService
// ============================================================================

// SerializePublishedRestService returns BSON bytes for a published REST service unit.
func SerializePublishedRestService(svc *model.PublishedRestService) ([]byte, error) {
	resources := bson.A{int32(2)}
	for _, res := range svc.Resources {
		ops := bson.A{int32(2)}
		for _, op := range res.Operations {
			opDoc := bson.M{
				"$ID":                  idToBsonBinary(GenerateID()),
				"$Type":                "Rest$PublishedRestServiceOperation",
				"HttpMethod":           serWebHttpMethodToMendix(op.HTTPMethod),
				"Path":                 op.Path,
				"Microflow":            op.Microflow,
				"Summary":              op.Summary,
				"Deprecated":           op.Deprecated,
				"Commit":               "Yes",
				"Documentation":        "",
				"ExportMapping":        "",
				"ImportMapping":        "",
				"ObjectHandlingBackup": "Create",
				"Parameters":           serWebPublishedRestParams(op.Path, op.Microflow, op.Parameters),
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
		"AllowedRoles":            serWebMendixStringArray(svc.AllowedRoles),
		"AuthenticationTypes":     bson.A{int32(2)},
		"AuthenticationMicroflow": "",
		"CorsConfiguration":       nil,
		"Parameters":              bson.A{int32(2)},
		"Resources":               resources,
	}

	return bson.Marshal(doc)
}

// serWebPublishedRestParams builds the Parameters array for a published REST operation.
func serWebPublishedRestParams(path string, microflowQN string, _ []string) bson.A {
	params := bson.A{int32(2)}
	for _, name := range serWebExtractPathParams(path) {
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

// serWebExtractPathParams returns parameter names from {param} placeholders in a path.
func serWebExtractPathParams(path string) []string {
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

// serWebHttpMethodToMendix converts uppercase HTTP method names to Mendix casing.
func serWebHttpMethodToMendix(method string) string {
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

// serWebMendixStringArray builds a Mendix-style versioned string array (int32(1) prefix).
func serWebMendixStringArray(items []string) bson.A {
	arr := bson.A{int32(1)}
	for _, s := range items {
		arr = append(arr, s)
	}
	return arr
}
