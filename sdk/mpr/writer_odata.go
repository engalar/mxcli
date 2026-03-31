// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/model"

	"go.mongodb.org/mongo-driver/bson"
)

// ============================================================================
// Consumed OData Service (OData Client) — Rest$ConsumedODataService
// ============================================================================

// CreateConsumedODataService creates a new consumed OData service (client) document.
func (w *Writer) CreateConsumedODataService(svc *model.ConsumedODataService) error {
	if svc.ID == "" {
		svc.ID = model.ID(generateUUID())
	}
	svc.TypeName = "Rest$ConsumedODataService"

	contents, err := w.serializeConsumedODataService(svc)
	if err != nil {
		return fmt.Errorf("failed to serialize consumed OData service: %w", err)
	}

	return w.insertUnit(string(svc.ID), string(svc.ContainerID), "Documents", "Rest$ConsumedODataService", contents)
}

// UpdateConsumedODataService updates an existing consumed OData service.
func (w *Writer) UpdateConsumedODataService(svc *model.ConsumedODataService) error {
	contents, err := w.serializeConsumedODataService(svc)
	if err != nil {
		return fmt.Errorf("failed to serialize consumed OData service: %w", err)
	}

	return w.updateUnit(string(svc.ID), contents)
}

// DeleteConsumedODataService deletes a consumed OData service by ID.
func (w *Writer) DeleteConsumedODataService(id model.ID) error {
	return w.deleteUnit(string(id))
}

// serializeConsumedODataService converts a ConsumedODataService to BSON bytes.
func (w *Writer) serializeConsumedODataService(svc *model.ConsumedODataService) ([]byte, error) {
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
		"MetadataReferences":   bson.A{int64(0)}, // empty BSON array marker
		"ValidatedEntities":    bson.A{int64(0)}, // empty BSON array marker
		"LastUpdated":          "",
		"UseQuerySegment":      false,
		"MinimumMxVersion":     "",
		"RecommendedMxVersion": "",
	}

	// Microflow references (BY_NAME)
	if svc.ConfigurationMicroflow != "" {
		doc["ConfigurationMicroflow"] = svc.ConfigurationMicroflow
	}
	if svc.ErrorHandlingMicroflow != "" {
		doc["ErrorHandlingMicroflow"] = svc.ErrorHandlingMicroflow
	}

	// Proxy constant references (BY_NAME)
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

	// Mendix Catalog integration (optional)
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

	// HTTP configuration (required nested part)
	doc["HttpConfiguration"] = serializeHttpConfiguration(svc.HttpConfiguration)

	return bson.Marshal(doc)
}

// serializeHttpConfiguration converts an HttpConfiguration to a BSON map.
// If cfg is nil, a minimal default configuration is created.
func serializeHttpConfiguration(cfg *model.HttpConfiguration) bson.M {
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

		// Serialize header entries
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

// CreatePublishedODataService creates a new published OData service document.
func (w *Writer) CreatePublishedODataService(svc *model.PublishedODataService) error {
	if svc.ID == "" {
		svc.ID = model.ID(generateUUID())
	}
	svc.TypeName = "ODataPublish$PublishedODataService2"

	contents, err := w.serializePublishedODataService(svc)
	if err != nil {
		return fmt.Errorf("failed to serialize published OData service: %w", err)
	}

	return w.insertUnit(string(svc.ID), string(svc.ContainerID), "Documents", "ODataPublish$PublishedODataService2", contents)
}

// UpdatePublishedODataService updates an existing published OData service.
func (w *Writer) UpdatePublishedODataService(svc *model.PublishedODataService) error {
	contents, err := w.serializePublishedODataService(svc)
	if err != nil {
		return fmt.Errorf("failed to serialize published OData service: %w", err)
	}

	return w.updateUnit(string(svc.ID), contents)
}

// DeletePublishedODataService deletes a published OData service by ID.
func (w *Writer) DeletePublishedODataService(id model.ID) error {
	return w.deleteUnit(string(id))
}

// serializePublishedODataService converts a PublishedODataService to BSON bytes.
func (w *Writer) serializePublishedODataService(svc *model.PublishedODataService) ([]byte, error) {
	// Authentication types array (versioned: starts with int64(3))
	authTypes := bson.A{int32(3)}
	for _, at := range svc.AuthenticationTypes {
		authTypes = append(authTypes, at)
	}

	// Serialize entity types and build ID map for entity set pointers
	entityTypeIDMap := make(map[string]string) // ExposedName -> entity type ID
	entityTypes := bson.A{}
	for _, et := range svc.EntityTypes {
		etID := string(et.ID)
		if etID == "" {
			etID = generateUUID()
			et.ID = model.ID(etID)
		}
		entityTypeIDMap[et.ExposedName] = etID
		entityTypes = append(entityTypes, serializePublishedEntityType(et))
	}

	// Serialize entity sets with BY_ID pointers to entity types
	entitySets := bson.A{}
	for _, es := range svc.EntitySets {
		esID := string(es.ID)
		if esID == "" {
			esID = generateUUID()
			es.ID = model.ID(esID)
		}
		// Resolve EntityTypeName to EntityType ID
		entityTypeID := entityTypeIDMap[es.EntityTypeName]
		entitySets = append(entitySets, serializePublishedEntitySet(es, entityTypeID))
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

// serializePublishedEntityType converts a PublishedEntityType to a BSON map.
func serializePublishedEntityType(et *model.PublishedEntityType) bson.M {
	// Serialize child members
	members := bson.A{}
	for _, m := range et.Members {
		members = append(members, serializePublishedMember(m))
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

// serializePublishedEntitySet converts a PublishedEntitySet to a BSON map.
func serializePublishedEntitySet(es *model.PublishedEntitySet, entityTypeID string) bson.M {
	doc := bson.M{
		"$ID":         idToBsonBinary(string(es.ID)),
		"$Type":       "ODataPublish$EntitySet",
		"ExposedName": es.ExposedName,
		"UsePaging":   es.UsePaging,
		"PageSize":    int64(es.PageSize),
	}

	// EntityTypePointer is a BY_ID reference
	if entityTypeID != "" {
		doc["EntityTypePointer"] = idToBsonBinary(entityTypeID)
	}

	// Serialize mode objects
	if es.ReadMode != "" {
		doc["ReadMode"] = serializeReadMode(es.ReadMode)
	}
	if es.InsertMode != "" {
		doc["InsertMode"] = serializeChangeMode(es.InsertMode)
	}
	if es.UpdateMode != "" {
		doc["UpdateMode"] = serializeChangeMode(es.UpdateMode)
	}
	if es.DeleteMode != "" {
		doc["DeleteMode"] = serializeChangeMode(es.DeleteMode)
	}

	return doc
}

// serializePublishedMember converts a PublishedMember to a BSON map.
func serializePublishedMember(m *model.PublishedMember) bson.M {
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
		// Default to attribute for unknown kinds
		doc["$Type"] = "ODataPublish$PublishedAttribute"
		doc["Attribute"] = m.Name
	}

	return doc
}

// serializeReadMode converts a read mode string to a BSON mode object.
// Accepts both parsed format ("ReadFromDatabase") and MDL format ("SOURCE").
func serializeReadMode(mode string) bson.M {
	modeID := idToBsonBinary(generateUUID())

	switch {
	case strings.EqualFold(mode, "ReadFromDatabase") || strings.EqualFold(mode, "SOURCE"):
		return bson.M{
			"$ID":   modeID,
			"$Type": "ODataPublish$ReadSource",
		}
	case strings.HasPrefix(mode, "CallMicroflow:"):
		return bson.M{
			"$ID":       modeID,
			"$Type":     "ODataPublish$CallMicroflowToRead",
			"Microflow": strings.TrimPrefix(mode, "CallMicroflow:"),
		}
	case strings.HasPrefix(mode, "MICROFLOW "):
		return bson.M{
			"$ID":       modeID,
			"$Type":     "ODataPublish$CallMicroflowToRead",
			"Microflow": strings.TrimPrefix(mode, "MICROFLOW "),
		}
	default:
		// Unknown mode — store as ReadSource
		return bson.M{
			"$ID":   modeID,
			"$Type": "ODataPublish$ReadSource",
		}
	}
}

// serializeChangeMode converts a change mode string to a BSON mode object.
// Accepts both parsed format ("ChangeFromDatabase", "NotSupported") and MDL format ("SOURCE", "NOT_SUPPORTED").
func serializeChangeMode(mode string) bson.M {
	modeID := idToBsonBinary(generateUUID())

	switch {
	case strings.EqualFold(mode, "ChangeFromDatabase") || strings.EqualFold(mode, "SOURCE"):
		return bson.M{
			"$ID":   modeID,
			"$Type": "ODataPublish$ChangeSource",
		}
	case strings.EqualFold(mode, "NotSupported") || strings.EqualFold(mode, "NOT_SUPPORTED"):
		return bson.M{
			"$ID":   modeID,
			"$Type": "ODataPublish$ChangeNotSupported",
		}
	case strings.HasPrefix(mode, "CallMicroflow:"):
		return bson.M{
			"$ID":       modeID,
			"$Type":     "ODataPublish$CallMicroflowToChange",
			"Microflow": strings.TrimPrefix(mode, "CallMicroflow:"),
		}
	case strings.HasPrefix(mode, "MICROFLOW "):
		return bson.M{
			"$ID":       modeID,
			"$Type":     "ODataPublish$CallMicroflowToChange",
			"Microflow": strings.TrimPrefix(mode, "MICROFLOW "),
		}
	default:
		// Unknown mode — store as ChangeNotSupported
		return bson.M{
			"$ID":   modeID,
			"$Type": "ODataPublish$ChangeNotSupported",
		}
	}
}
