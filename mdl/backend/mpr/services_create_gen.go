// SPDX-License-Identifier: Apache-2.0

// Gen-native CREATE path for service document types.
// Uses ServiceRepository.Create instead of sdk/mpr.Serialize* + InsertUnit.
// Only types where the gen encoder safely produces the correct BSON are here;
// complex serializers (DataTransformer, ImportMapping, OData, REST) stay in
// create_services_modelsdk.go until field-mapping is verified against Studio Pro.

package mprbackend

import (
	"fmt"

	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genBE "github.com/mendixlabs/mxcli/modelsdk/gen/businessevents"
	genDB "github.com/mendixlabs/mxcli/modelsdk/gen/databaseconnector"
	genDt "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genDM "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genJS "github.com/mendixlabs/mxcli/modelsdk/gen/jsonstructures"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// ── JsonStructure ─────────────────────────────────────────────────────────

func (b *MprBackend) createJsonStructureGen(js *types.JsonStructure) error {
	if js.ID == "" {
		js.ID = model.ID(modelsdkmpr.GenerateID())
	}
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	g := genJS.NewJsonStructure()
	g.SetID(element.ID(string(js.ID)))
	g.SetName(js.Name)
	g.SetDocumentation(js.Documentation)
	g.SetExcluded(js.Excluded)
	if js.ExportLevel != "" {
		g.SetExportLevel(js.ExportLevel)
	}
	g.SetJsonSnippet(js.JsonSnippet)
	return mprrepos.NewServiceRepository(w).Create(string(js.ContainerID), "Documents", g)
}

// ── DatabaseConnection ────────────────────────────────────────────────────

func (b *MprBackend) createDatabaseConnectionGen(conn *model.DatabaseConnection) error {
	if conn.ID == "" {
		conn.ID = model.ID(modelsdkmpr.GenerateID())
	}
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	g := genDB.NewDatabaseConnection()
	g.SetID(element.ID(string(conn.ID)))
	g.SetName(conn.Name)
	g.SetDocumentation(conn.Documentation)
	g.SetExcluded(conn.Excluded)
	g.SetExportLevel(conn.ExportLevel)
	g.SetDatabaseType(conn.DatabaseType)
	g.SetConnectionStringQualifiedName(conn.ConnectionString)
	g.SetUserNameQualifiedName(conn.UserName)
	g.SetPasswordQualifiedName(conn.Password)

	// ConnectionInput holds the literal JDBC URL for Studio Pro dev mode.
	connStr := genDB.NewConnectionString()
	connStr.SetValue(conn.ConnectionInputValue)
	g.SetConnectionInput(connStr)

	// Serialize queries (name + SQL + parameters + return entity + column mappings).
	for _, q := range conn.Queries {
		gq := genDB.NewDatabaseQuery()
		gq.SetName(q.Name)
		gq.SetQuery(q.SQL)
		gq.SetQueryType(int32(q.QueryType))
		// Parameters
		for _, p := range q.Parameters {
			gp := genDB.NewQueryParameter()
			gp.SetParameterName(p.ParameterName)
			if dt := dbDataTypeElemFromString(p.DataType); dt != nil {
				gp.SetDataType(dt)
			}
			if p.DefaultValue != "" {
				gp.SetDefaultValue(p.DefaultValue)
			}
			gp.SetEmptyValueBecomesNull(p.EmptyValueBecomesNull)
			gq.AddParameters(gp)
		}
		// Table mappings (entity + column map)
		for _, tm := range q.TableMappings {
			gtm := genDB.NewTableMapping()
			gtm.SetEntityQualifiedName(tm.Entity)
			for _, cm := range tm.Columns {
				gcm := genDB.NewColumnMapping()
				gcm.SetColumnName(cm.ColumnName)
				gcm.SetAttributeQualifiedName(cm.Attribute)
				gtm.AddColumns(gcm)
			}
			gq.AddTableMappings(gtm)
		}
		g.AddQueries(gq)
	}

	return mprrepos.NewServiceRepository(w).Create(string(conn.ContainerID), "Documents", g)
}

// ── BusinessEventService ──────────────────────────────────────────────────

func (b *MprBackend) createBusinessEventServiceGen(svc *model.BusinessEventService) error {
	if svc.ID == "" {
		svc.ID = model.ID(modelsdkmpr.GenerateID())
	}
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	g := genBE.NewBusinessEventService()
	g.SetID(element.ID(string(svc.ID)))
	g.SetName(svc.Name)
	g.SetDocumentation(svc.Documentation)
	g.SetExcluded(svc.Excluded)
	g.SetExportLevel(svc.ExportLevel)
	g.SetDocument(svc.Document)

	if svc.Definition != nil {
		g.SetDefinition(buildBEDefinition(svc.Definition))
	}
	for _, op := range svc.OperationImplementations {
		g.AddOperationImplementations(buildBEServiceOperation(op))
	}

	return mprrepos.NewServiceRepository(w).Create(string(svc.ContainerID), "Documents", g)
}

// buildBEDefinition converts a model.BusinessEventDefinition to its gen element.
func buildBEDefinition(def *model.BusinessEventDefinition) *genBE.BusinessEventDefinition {
	g := genBE.NewBusinessEventDefinition()
	g.SetServiceName(def.ServiceName)
	g.SetEventNamePrefix(def.EventNamePrefix)
	for _, ch := range def.Channels {
		g.AddChannels(buildBEChannel(ch))
	}
	return g
}

// buildBEChannel converts a model.BusinessEventChannel to its gen element.
func buildBEChannel(ch *model.BusinessEventChannel) *genBE.Channel {
	g := genBE.NewChannel()
	g.SetChannelName(ch.ChannelName)
	for _, msg := range ch.Messages {
		g.AddMessages(buildBEMessage(msg))
	}
	return g
}

// buildBEMessage converts a model.BusinessEventMessage to its gen element.
func buildBEMessage(msg *model.BusinessEventMessage) *genBE.Message {
	g := genBE.NewMessage()
	g.SetMessageName(msg.MessageName)
	g.SetCanPublish(msg.CanPublish)
	g.SetCanSubscribe(msg.CanSubscribe)
	for _, attr := range msg.Attributes {
		g.AddAttributes(buildBEMessageAttribute(attr))
	}
	return g
}

// buildBEMessageAttribute converts a model.BusinessEventAttribute to its gen element.
func buildBEMessageAttribute(attr *model.BusinessEventAttribute) *genBE.MessageAttribute {
	g := genBE.NewMessageAttribute()
	g.SetAttributeName(attr.AttributeName)
	g.SetAttributeType(beAttrTypeElem(attr.AttributeType))
	return g
}

// buildBEServiceOperation converts a model.ServiceOperation to its gen element.
func buildBEServiceOperation(op *model.ServiceOperation) *genBE.ServiceOperation {
	g := genBE.NewServiceOperation()
	g.SetMessageName(op.MessageName)
	g.SetOperation(op.Operation)
	if op.Entity != "" {
		g.SetEntityQualifiedName(op.Entity)
	}
	if op.Microflow != "" {
		g.SetMicroflowQualifiedName(op.Microflow)
	}
	return g
}

// beAttrTypeElem converts a Business Events attribute type string ("Long",
// "String", etc.) to the matching gen DomainModels AttributeType element.
//
// Business Events MessageAttribute.AttributeType is a DomainModels$*AttributeType,
// NOT a DataTypes$*Type — Mendix refuses to load the MPR if the wrong type
// family is used ("cannot be converted to type AttributeTypeBase").
func beAttrTypeElem(t string) element.Element {
	switch t {
	case "Integer":
		return genDM.NewIntegerAttributeType()
	case "Long":
		return genDM.NewLongAttributeType()
	case "Decimal":
		return genDM.NewDecimalAttributeType()
	case "Boolean":
		return genDM.NewBooleanAttributeType()
	case "DateTime":
		return genDM.NewDateTimeAttributeType()
	default: // "String" and unknown
		return genDM.NewStringAttributeType()
	}
}

// dbDataTypeElemFromString converts a DataTypes$* type-name string (as stored in
// model.DatabaseQueryParameter.DataType) to the matching gen DataType element.
func dbDataTypeElemFromString(dt string) element.Element {
	switch dt {
	case "DataTypes$IntegerType":
		return genDt.NewIntegerType()
	case "DataTypes$DecimalType":
		return genDt.NewDecimalType()
	case "DataTypes$BooleanType":
		return genDt.NewBooleanType()
	case "DataTypes$DateTimeType":
		return genDt.NewDateTimeType()
	default: // DataTypes$StringType or unknown
		return genDt.NewStringType()
	}
}
