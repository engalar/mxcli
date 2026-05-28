// SPDX-License-Identifier: Apache-2.0

// odata_compat.go — OData service BSON parsing for ConsumedODataService and
// PublishedODataService. Mirrors the retired sdk/mpr/parser_odata.go logic
// against modelsdk/mpr.Reader's raw bytes.

package mprbackend

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/model"
)

const (
	consumedODataServiceBsonType  = "Rest$ConsumedODataService"
	publishedODataServiceBsonType = "ODataPublish$PublishedODataService2"
)

func (b *MprBackend) listConsumedODataServicesFromRaw() ([]*model.ConsumedODataService, error) {
	rawUnits, err := b.msdkReader.ListRawUnitsByType(consumedODataServiceBsonType)
	if err != nil {
		return nil, err
	}
	out := make([]*model.ConsumedODataService, 0, len(rawUnits))
	for _, ru := range rawUnits {
		if ru == nil {
			continue
		}
		svc, err := parseConsumedODataServiceRaw(string(ru.ID), string(ru.ContainerID), ru.Contents)
		if err != nil {
			return nil, fmt.Errorf("parse consumed OData service %s: %w", ru.ID, err)
		}
		out = append(out, svc)
	}
	return out, nil
}

func (b *MprBackend) listPublishedODataServicesFromRaw() ([]*model.PublishedODataService, error) {
	rawUnits, err := b.msdkReader.ListRawUnitsByType(publishedODataServiceBsonType)
	if err != nil {
		return nil, err
	}
	out := make([]*model.PublishedODataService, 0, len(rawUnits))
	for _, ru := range rawUnits {
		if ru == nil {
			continue
		}
		svc, err := parsePublishedODataServiceRaw(string(ru.ID), string(ru.ContainerID), ru.Contents)
		if err != nil {
			return nil, fmt.Errorf("parse published OData service %s: %w", ru.ID, err)
		}
		out = append(out, svc)
	}
	return out, nil
}

func parseConsumedODataServiceRaw(unitID, containerID string, contents []byte) (*model.ConsumedODataService, error) {
	if len(contents) < 4 {
		return nil, fmt.Errorf("contents too short")
	}
	var raw map[string]any
	if err := bson.Unmarshal(contents, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal BSON: %w", err)
	}
	svc := &model.ConsumedODataService{}
	svc.ID = model.ID(unitID)
	svc.TypeName = consumedODataServiceBsonType
	svc.ContainerID = model.ID(containerID)

	svc.Name = extractString(raw["Name"])
	svc.Documentation = extractString(raw["Documentation"])
	svc.Version = extractString(raw["Version"])
	svc.ServiceName = extractString(raw["ServiceName"])
	svc.ODataVersion = extractString(raw["ODataVersion"])
	svc.MetadataUrl = extractString(raw["MetadataUrl"])
	svc.TimeoutExpression = extractString(raw["TimeoutExpression"])
	svc.ProxyType = extractString(raw["ProxyType"])
	svc.Description = extractString(raw["Description"])
	svc.Validated = extractBool(raw["Validated"], false)
	svc.Excluded = extractBool(raw["Excluded"], false)

	svc.ConfigurationMicroflow = extractString(raw["ConfigurationMicroflow"])
	svc.ErrorHandlingMicroflow = extractString(raw["ErrorHandlingMicroflow"])

	svc.ProxyHost = extractString(raw["ProxyHost"])
	svc.ProxyPort = extractString(raw["ProxyPort"])
	svc.ProxyUsername = extractString(raw["ProxyUsername"])
	svc.ProxyPassword = extractString(raw["ProxyPassword"])

	svc.Metadata = extractString(raw["Metadata"])
	svc.MetadataHash = extractString(raw["MetadataHash"])

	svc.ApplicationId = extractString(raw["ApplicationId"])
	svc.EndpointId = extractString(raw["EndpointId"])
	svc.CatalogUrl = extractString(raw["CatalogUrl"])
	svc.EnvironmentType = extractString(raw["EnvironmentType"])

	if httpCfg, ok := raw["HttpConfiguration"].(map[string]any); ok {
		svc.HttpConfiguration = parseODataHttpConfiguration(httpCfg)
	}
	return svc, nil
}

func parseODataHttpConfiguration(raw map[string]any) *model.HttpConfiguration {
	cfg := &model.HttpConfiguration{}
	cfg.ID = model.ID(extractBsonID(raw["$ID"]))
	cfg.TypeName = extractString(raw["$Type"])
	cfg.UseAuthentication = extractBool(raw["UseHttpAuthentication"], false)
	cfg.Username = extractString(raw["HttpAuthenticationUserName"])
	cfg.Password = extractString(raw["HttpAuthenticationPassword"])
	cfg.HttpMethod = extractString(raw["HttpMethod"])
	cfg.OverrideLocation = extractBool(raw["OverrideLocation"], false)
	cfg.CustomLocation = extractString(raw["CustomLocation"])
	cfg.ClientCertificate = extractString(raw["ClientCertificate"])

	for _, h := range extractBsonArray(raw["HttpHeaderEntries"]) {
		if hMap, ok := h.(map[string]any); ok {
			entry := &model.HttpHeaderEntry{}
			entry.ID = model.ID(extractBsonID(hMap["$ID"]))
			entry.TypeName = extractString(hMap["$Type"])
			entry.Key = extractString(hMap["Key"])
			entry.Value = extractString(hMap["Value"])
			cfg.HeaderEntries = append(cfg.HeaderEntries, entry)
		}
	}
	return cfg
}

func parsePublishedODataServiceRaw(unitID, containerID string, contents []byte) (*model.PublishedODataService, error) {
	if len(contents) < 4 {
		return nil, fmt.Errorf("contents too short")
	}
	var raw map[string]any
	if err := bson.Unmarshal(contents, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal BSON: %w", err)
	}
	svc := &model.PublishedODataService{}
	svc.ID = model.ID(unitID)
	svc.TypeName = publishedODataServiceBsonType
	svc.ContainerID = model.ID(containerID)

	svc.Name = extractString(raw["Name"])
	svc.Documentation = extractString(raw["Documentation"])
	svc.Path = extractString(raw["Path"])
	svc.Namespace = extractString(raw["Namespace"])
	svc.ServiceName = extractString(raw["ServiceName"])
	svc.Version = extractString(raw["Version"])
	svc.ODataVersion = extractString(raw["ODataVersion"])
	svc.Summary = extractString(raw["Summary"])
	svc.Description = extractString(raw["Description"])
	svc.PublishAssociations = extractBool(raw["PublishAssociations"], false)
	svc.UseGeneralization = extractBool(raw["UseGeneralization"], false)
	svc.Excluded = extractBool(raw["Excluded"], false)
	svc.AuthMicroflow = extractString(raw["AuthenticationMicroflow"])

	for _, at := range extractBsonArray(raw["AuthenticationTypes"]) {
		if s, ok := at.(string); ok {
			svc.AuthenticationTypes = append(svc.AuthenticationTypes, s)
		}
	}
	for _, r := range extractBsonArray(raw["AllowedModuleRoles"]) {
		if name, ok := r.(string); ok {
			svc.AllowedModuleRoles = append(svc.AllowedModuleRoles, name)
		}
	}

	entityTypeMap := make(map[string]*model.PublishedEntityType)
	for _, et := range extractBsonArray(raw["EntityTypes"]) {
		if etMap, ok := et.(map[string]any); ok {
			entityType := parsePublishedEntityType(etMap)
			svc.EntityTypes = append(svc.EntityTypes, entityType)
			entityTypeMap[string(entityType.ID)] = entityType
		}
	}
	for _, es := range extractBsonArray(raw["EntitySets"]) {
		if esMap, ok := es.(map[string]any); ok {
			svc.EntitySets = append(svc.EntitySets, parsePublishedEntitySet(esMap, entityTypeMap))
		}
	}
	return svc, nil
}

func parsePublishedEntityType(raw map[string]any) *model.PublishedEntityType {
	et := &model.PublishedEntityType{}
	et.ID = model.ID(extractBsonID(raw["$ID"]))
	et.TypeName = extractString(raw["$Type"])
	et.Entity = extractString(raw["Entity"])
	et.ExposedName = extractString(raw["ExposedName"])
	et.Summary = extractString(raw["Summary"])
	et.Description = extractString(raw["Description"])

	for _, m := range extractBsonArray(raw["ChildMembers"]) {
		if mMap, ok := m.(map[string]any); ok {
			et.Members = append(et.Members, parsePublishedMember(mMap))
		}
	}
	return et
}

func parsePublishedEntitySet(raw map[string]any, entityTypeMap map[string]*model.PublishedEntityType) *model.PublishedEntitySet {
	es := &model.PublishedEntitySet{}
	es.ID = model.ID(extractBsonID(raw["$ID"]))
	es.TypeName = extractString(raw["$Type"])
	es.ExposedName = extractString(raw["ExposedName"])
	es.UsePaging = extractBool(raw["UsePaging"], false)
	es.PageSize = extractInt(raw["PageSize"])

	entityTypeID := extractBsonID(raw["EntityTypePointer"])
	if entityTypeID != "" {
		if et, ok := entityTypeMap[entityTypeID]; ok {
			es.EntityTypeName = et.Entity
		}
	}

	es.ReadMode = parseChangeMode(raw["ReadMode"])
	es.InsertMode = parseChangeMode(raw["InsertMode"])
	es.UpdateMode = parseChangeMode(raw["UpdateMode"])
	es.DeleteMode = parseChangeMode(raw["DeleteMode"])
	return es
}

func parsePublishedMember(raw map[string]any) *model.PublishedMember {
	m := &model.PublishedMember{}
	m.ID = model.ID(extractBsonID(raw["$ID"]))
	m.TypeName = extractString(raw["$Type"])
	m.ExposedName = extractString(raw["ExposedName"])
	m.Filterable = extractBool(raw["Filterable"], false)
	m.Sortable = extractBool(raw["Sortable"], false)
	m.IsPartOfKey = extractBool(raw["IsPartOfKey"], false)

	switch m.TypeName {
	case "ODataPublish$PublishedAttribute":
		m.Kind = "attribute"
		m.Name = extractString(raw["Attribute"])
	case "ODataPublish$PublishedAssociationEnd":
		m.Kind = "association"
		m.Name = extractString(raw["Association"])
	case "ODataPublish$PublishedId":
		m.Kind = "id"
		m.Name = extractString(raw["Attribute"])
	default:
		m.Kind = "unknown"
	}
	return m
}

func parseChangeMode(v any) string {
	if v == nil {
		return ""
	}
	modeMap, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	typeName := extractString(modeMap["$Type"])
	switch typeName {
	case "ODataPublish$ReadSource":
		return "ReadFromDatabase"
	case "ODataPublish$CallMicroflowToRead":
		if mfName := extractString(modeMap["Microflow"]); mfName != "" {
			return "CallMicroflow:" + mfName
		}
		return "CallMicroflow"
	case "ODataPublish$ChangeSource":
		return "ChangeFromDatabase"
	case "ODataPublish$ChangeNotSupported":
		return "NotSupported"
	case "ODataPublish$CallMicroflowToChange":
		if mfName := extractString(modeMap["Microflow"]); mfName != "" {
			return "CallMicroflow:" + mfName
		}
		return "CallMicroflow"
	default:
		return typeName
	}
}
