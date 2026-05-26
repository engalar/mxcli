// SPDX-License-Identifier: Apache-2.0

// convert_reader.go — gen/* → model.* conversion helpers for msdkReader-based methods.
//
// Phase 4B: these converters extract the fields consumed by the executor
// from gen-typed reads (via mprread free functions), preserving the
// backend.* interface signatures that callers depend on.
//
// Keep each converter narrow: only populate fields actually read by the
// executor. New fields should be added on demand, not preemptively.

package mprbackend

import (
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genBE "github.com/mendixlabs/mxcli/modelsdk/gen/businessevents"
	genConst "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
	genDBC "github.com/mendixlabs/mxcli/modelsdk/gen/databaseconnector"
	genDTrans "github.com/mendixlabs/mxcli/modelsdk/gen/datatransformers"
	genDT "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genEnum "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
	genExpMap "github.com/mendixlabs/mxcli/modelsdk/gen/exportmappings"
	genImpMap "github.com/mendixlabs/mxcli/modelsdk/gen/importmappings"
	genJson "github.com/mendixlabs/mxcli/modelsdk/gen/jsonstructures"
	genProj "github.com/mendixlabs/mxcli/modelsdk/gen/projects"
	genSched "github.com/mendixlabs/mxcli/modelsdk/gen/scheduledevents"
	"github.com/mendixlabs/mxcli/modelsdk/mprread"
)

// ---------------------------------------------------------------------------
// Enumeration
// ---------------------------------------------------------------------------

func enumToModel(u mprread.Unit[*genEnum.Enumeration]) *model.Enumeration {
	e := u.Element
	out := &model.Enumeration{
		BaseElement: model.BaseElement{
			ID:       model.ID(e.ID()),
			TypeName: "Enumerations$Enumeration",
		},
		ContainerID:   u.ContainerID,
		Name:          e.Name(),
		Documentation: e.Documentation(),
	}
	for _, item := range e.ValuesItems() {
		ev, ok := item.(*genEnum.EnumerationValue)
		if !ok {
			continue
		}
		out.Values = append(out.Values, enumValueToModel(ev))
	}
	return out
}

func enumValueToModel(v *genEnum.EnumerationValue) model.EnumerationValue {
	out := model.EnumerationValue{
		BaseElement: model.BaseElement{
			ID:       model.ID(v.ID()),
			TypeName: "Enumerations$EnumerationValue",
		},
		Name: v.Name(),
	}
	if cap := v.Caption(); cap != nil {
		out.Caption = textElementToModel(cap)
	}
	return out
}

// textElementToModel extracts translations from a gen Texts$Text element.
// Returns nil if the element does not expose translations. Uses duck-typing
// to avoid importing the texts gen package directly.
func textElementToModel(el element.Element) *model.Text {
	type translationsAccessor interface {
		TranslationsItems() []element.Element
	}
	type translationAccessor interface {
		LanguageCode() string
		Text() string
	}
	ta, ok := el.(translationsAccessor)
	if !ok {
		return nil
	}
	out := &model.Text{Translations: map[string]string{}}
	for _, item := range ta.TranslationsItems() {
		tr, ok := item.(translationAccessor)
		if !ok {
			continue
		}
		if code := tr.LanguageCode(); code != "" {
			out.Translations[code] = tr.Text()
		}
	}
	return out
}

func enumUnitsToModel(units []mprread.Unit[*genEnum.Enumeration]) []*model.Enumeration {
	out := make([]*model.Enumeration, len(units))
	for i, u := range units {
		out[i] = enumToModel(u)
	}
	return out
}

// ---------------------------------------------------------------------------
// Constant
// ---------------------------------------------------------------------------

func constToModel(u mprread.Unit[*genConst.Constant]) *model.Constant {
	c := u.Element
	return &model.Constant{
		BaseElement: model.BaseElement{
			ID:       model.ID(c.ID()),
			TypeName: "Constants$Constant",
		},
		ContainerID:     u.ContainerID,
		Name:            c.Name(),
		Documentation:   c.Documentation(),
		Type:            constantDataTypeToModel(c.Type()),
		DefaultValue:    c.DefaultValue(),
		ExposedToClient: c.ExposedToClient(),
		Excluded:        c.Excluded(),
		ExportLevel:     c.ExportLevel(),
	}
}

func constantDataTypeToModel(typ element.Element) model.ConstantDataType {
	dt := model.ConstantDataType{Kind: "Unknown"}
	if typ == nil {
		return dt
	}
	switch typ.TypeName() {
	case "DataTypes$StringType":
		dt.Kind = "String"
	case "DataTypes$IntegerType":
		dt.Kind = "Integer"
	case "DataTypes$LongType":
		dt.Kind = "Long"
	case "DataTypes$DecimalType":
		dt.Kind = "Decimal"
	case "DataTypes$BooleanType":
		dt.Kind = "Boolean"
	case "DataTypes$DateTimeType":
		dt.Kind = "DateTime"
	case "DataTypes$BinaryType":
		dt.Kind = "Binary"
	case "DataTypes$FloatType":
		dt.Kind = "Float"
	case "DataTypes$EnumerationType":
		dt.Kind = "Enumeration"
		if et, ok := typ.(*genDT.EnumerationType); ok {
			dt.EnumRef = et.EnumerationQualifiedName()
		}
	case "DataTypes$ObjectType":
		dt.Kind = "Object"
		if ot, ok := typ.(*genDT.ObjectType); ok {
			dt.EntityRef = ot.EntityQualifiedName()
		}
	case "DataTypes$ListType":
		dt.Kind = "List"
		if lt, ok := typ.(*genDT.ListType); ok {
			dt.EntityRef = lt.EntityQualifiedName()
		}
	}
	return dt
}

func constUnitsToModel(units []mprread.Unit[*genConst.Constant]) []*model.Constant {
	out := make([]*model.Constant, len(units))
	for i, u := range units {
		out[i] = constToModel(u)
	}
	return out
}

// ---------------------------------------------------------------------------
// ScheduledEvent
// ---------------------------------------------------------------------------

func schedEventToModel(u mprread.Unit[*genSched.ScheduledEvent]) *model.ScheduledEvent {
	s := u.Element
	return &model.ScheduledEvent{
		BaseElement: model.BaseElement{
			ID:       model.ID(s.ID()),
			TypeName: "ScheduledEvents$ScheduledEvent",
		},
		ContainerID:   u.ContainerID,
		Name:          s.Name(),
		Documentation: s.Documentation(),
		Interval:      int(s.Interval()),
		IntervalType:  s.IntervalType(),
		Enabled:       s.Enabled(),
	}
}

func schedEventUnitsToModel(units []mprread.Unit[*genSched.ScheduledEvent]) []*model.ScheduledEvent {
	out := make([]*model.ScheduledEvent, len(units))
	for i, u := range units {
		out[i] = schedEventToModel(u)
	}
	return out
}

// ---------------------------------------------------------------------------
// ModuleSettings
// ---------------------------------------------------------------------------

func moduleSettingsToTypes(u mprread.Unit[*genProj.ModuleSettings]) *types.ModuleSettings {
	ms := u.Element
	out := &types.ModuleSettings{
		ID:                  model.ID(ms.ID()),
		ContainerID:         u.ContainerID,
		ExportLevel:         ms.ExportLevel(),
		ProtectedModuleType: ms.ProtectedModuleType(),
		Version:             ms.Version(),
		BasedOnVersion:      ms.BasedOnVersion(),
		ExtensionName:       ms.ExtensionName(),
		SolutionIdentifier:  ms.SolutionIdentifier(),
	}
	for _, item := range ms.JarDependenciesItems() {
		jd, ok := item.(*genProj.JarDependency)
		if !ok {
			continue
		}
		out.JarDependencies = append(out.JarDependencies, jarDepToTypes(jd))
	}
	return out
}

func jarDepToTypes(jd *genProj.JarDependency) *types.JarDependency {
	out := &types.JarDependency{
		ID:         model.ID(jd.ID()),
		GroupID:    jd.GroupId(),
		ArtifactID: jd.ArtifactId(),
		Version:    jd.Version(),
		IsIncluded: jd.IsIncluded(),
	}
	for _, item := range jd.ExclusionsItems() {
		ex, ok := item.(*genProj.JarDependencyExclusion)
		if !ok {
			continue
		}
		out.Exclusions = append(out.Exclusions, &types.JarDependencyExclusion{
			ID:         model.ID(ex.ID()),
			GroupID:    ex.GroupId(),
			ArtifactID: ex.ArtifactId(),
		})
	}
	return out
}

func moduleSettingsUnitsToTypes(units []mprread.Unit[*genProj.ModuleSettings]) []*types.ModuleSettings {
	out := make([]*types.ModuleSettings, len(units))
	for i, u := range units {
		out[i] = moduleSettingsToTypes(u)
	}
	return out
}

// ---------------------------------------------------------------------------
// DataTransformer
// ---------------------------------------------------------------------------

func dataTransformerToModel(u mprread.Unit[*genDTrans.DataTransformer]) *model.DataTransformer {
	t := u.Element
	out := &model.DataTransformer{
		BaseElement: model.BaseElement{
			ID:       model.ID(t.ID()),
			TypeName: "DataTransformers$DataTransformer",
		},
		ContainerID: u.ContainerID,
		Name:        t.Name(),
		SourceType:  t.SourceType(),
		SourceJSON:  t.SourceJson(),
		Excluded:    t.Excluded(),
	}
	for _, item := range t.StepsItems() {
		st, ok := item.(*genDTrans.Step)
		if !ok {
			continue
		}
		out.Steps = append(out.Steps, &model.DataTransformerStep{
			Technology: st.Technology(),
			Expression: st.Expression(),
		})
	}
	return out
}

func dataTransformerUnitsToModel(units []mprread.Unit[*genDTrans.DataTransformer]) []*model.DataTransformer {
	out := make([]*model.DataTransformer, len(units))
	for i, u := range units {
		out[i] = dataTransformerToModel(u)
	}
	return out
}

// ---------------------------------------------------------------------------
// DatabaseConnection
// ---------------------------------------------------------------------------

func databaseConnectionToModel(u mprread.Unit[*genDBC.DatabaseConnection]) *model.DatabaseConnection {
	c := u.Element
	out := &model.DatabaseConnection{
		BaseElement: model.BaseElement{
			ID:       model.ID(c.ID()),
			TypeName: "DatabaseConnector$DatabaseConnection",
		},
		ContainerID:      u.ContainerID,
		Name:             c.Name(),
		Documentation:    c.Documentation(),
		DatabaseType:     c.DatabaseType(),
		ConnectionString: c.ConnectionStringQualifiedName(),
		UserName:         c.UserNameQualifiedName(),
		Password:         c.PasswordQualifiedName(),
		Excluded:         c.Excluded(),
		ExportLevel:      c.ExportLevel(),
	}
	if ci := c.ConnectionInput(); ci != nil {
		if cs, ok := ci.(*genDBC.ConnectionString); ok {
			out.ConnectionInputValue = cs.Value()
		}
	}
	for _, item := range c.QueriesItems() {
		q, ok := item.(*genDBC.DatabaseQuery)
		if !ok {
			continue
		}
		out.Queries = append(out.Queries, databaseQueryToModel(q))
	}
	return out
}

func databaseQueryToModel(q *genDBC.DatabaseQuery) *model.DatabaseQuery {
	out := &model.DatabaseQuery{
		BaseElement: model.BaseElement{
			ID:       model.ID(q.ID()),
			TypeName: q.TypeName(),
		},
		Name:      q.Name(),
		SQL:       q.Query(),
		QueryType: int(q.QueryType()),
	}
	for _, item := range q.TableMappingsItems() {
		tm, ok := item.(*genDBC.TableMapping)
		if !ok {
			continue
		}
		out.TableMappings = append(out.TableMappings, databaseTableMappingToModel(tm))
	}
	for _, item := range q.ParametersItems() {
		p, ok := item.(*genDBC.QueryParameter)
		if !ok {
			continue
		}
		out.Parameters = append(out.Parameters, databaseQueryParameterToModel(p))
	}
	return out
}

func databaseQueryParameterToModel(p *genDBC.QueryParameter) *model.DatabaseQueryParameter {
	out := &model.DatabaseQueryParameter{
		BaseElement: model.BaseElement{
			ID:       model.ID(p.ID()),
			TypeName: p.TypeName(),
		},
		ParameterName:         p.ParameterName(),
		DefaultValue:          p.DefaultValue(),
		EmptyValueBecomesNull: p.EmptyValueBecomesNull(),
	}
	if dt := p.DataType(); dt != nil {
		out.DataType = dt.TypeName()
	}
	return out
}

func databaseTableMappingToModel(m *genDBC.TableMapping) *model.DatabaseTableMapping {
	out := &model.DatabaseTableMapping{
		BaseElement: model.BaseElement{
			ID:       model.ID(m.ID()),
			TypeName: m.TypeName(),
		},
		Entity:    m.EntityQualifiedName(),
		TableName: m.TableName(),
	}
	for _, item := range m.ColumnsItems() {
		c, ok := item.(*genDBC.ColumnMapping)
		if !ok {
			continue
		}
		out.Columns = append(out.Columns, databaseColumnMappingToModel(c))
	}
	return out
}

func databaseColumnMappingToModel(c *genDBC.ColumnMapping) *model.DatabaseColumnMapping {
	out := &model.DatabaseColumnMapping{
		BaseElement: model.BaseElement{
			ID:       model.ID(c.ID()),
			TypeName: c.TypeName(),
		},
		Attribute:  c.AttributeQualifiedName(),
		ColumnName: c.ColumnName(),
	}
	if dt := c.SqlDataType(); dt != nil {
		out.SqlDataType = dt.TypeName()
	}
	return out
}

func databaseConnectionUnitsToModel(units []mprread.Unit[*genDBC.DatabaseConnection]) []*model.DatabaseConnection {
	out := make([]*model.DatabaseConnection, len(units))
	for i, u := range units {
		out[i] = databaseConnectionToModel(u)
	}
	return out
}

// ---------------------------------------------------------------------------
// JsonStructure
// ---------------------------------------------------------------------------

func jsonStructureToTypes(u mprread.Unit[*genJson.JsonStructure]) *types.JsonStructure {
	js := u.Element
	out := &types.JsonStructure{
		BaseElement: model.BaseElement{
			ID:       model.ID(js.ID()),
			TypeName: "JsonStructures$JsonStructure",
		},
		ContainerID:   u.ContainerID,
		Name:          js.Name(),
		Documentation: js.Documentation(),
		JsonSnippet:   js.JsonSnippet(),
		Excluded:      js.Excluded(),
		ExportLevel:   js.ExportLevel(),
	}
	for _, item := range js.ElementsItems() {
		je, ok := item.(*genJson.JsonElement)
		if !ok {
			continue
		}
		out.Elements = append(out.Elements, jsonElementToTypes(je))
	}
	return out
}

func jsonElementToTypes(e *genJson.JsonElement) *types.JsonElement {
	out := &types.JsonElement{
		ExposedName:     e.ExposedName(),
		ExposedItemName: e.ExposedItemName(),
		Path:            e.Path(),
		ElementType:     e.ElementType(),
		PrimitiveType:   e.PrimitiveType(),
		MinOccurs:       int(e.MinOccurs()),
		MaxOccurs:       int(e.MaxOccurs()),
		Nillable:        e.Nillable(),
		IsDefaultType:   e.IsDefaultType(),
		MaxLength:       int(e.MaxLength()),
		FractionDigits:  int(e.FractionDigits()),
		TotalDigits:     int(e.TotalDigits()),
		OriginalValue:   e.OriginalValue(),
	}
	for _, item := range e.ChildrenItems() {
		child, ok := item.(*genJson.JsonElement)
		if !ok {
			continue
		}
		out.Children = append(out.Children, jsonElementToTypes(child))
	}
	return out
}

func jsonStructureUnitsToTypes(units []mprread.Unit[*genJson.JsonStructure]) []*types.JsonStructure {
	out := make([]*types.JsonStructure, len(units))
	for i, u := range units {
		out[i] = jsonStructureToTypes(u)
	}
	return out
}

// ---------------------------------------------------------------------------
// ImportMapping / ExportMapping
// ---------------------------------------------------------------------------

func importMappingToModel(u mprread.Unit[*genImpMap.ImportMapping]) *model.ImportMapping {
	m := u.Element
	out := &model.ImportMapping{
		BaseElement:       model.BaseElement{ID: model.ID(m.ID()), TypeName: "ImportMappings$ImportMapping"},
		ContainerID:       u.ContainerID,
		Name:              m.Name(),
		Documentation:     m.Documentation(),
		Excluded:          m.Excluded(),
		ExportLevel:       m.ExportLevel(),
		JsonStructure:     m.JsonStructureQualifiedName(),
		XmlSchema:         m.XmlSchemaQualifiedName(),
		MessageDefinition: m.MessageDefinitionQualifiedName(),
	}
	for _, raw := range m.RootMappingElementsItems() {
		if elem := convertImportElement(raw); elem != nil {
			out.Elements = append(out.Elements, elem)
		}
	}
	return out
}

func convertImportElement(raw element.Element) *model.ImportMappingElement {
	switch e := raw.(type) {
	case *genImpMap.ImportObjectMappingElement:
		out := &model.ImportMappingElement{
			BaseElement:    model.BaseElement{ID: model.ID(e.ID())},
			Kind:           "Object",
			Entity:         e.EntityQualifiedName(),
			Association:    e.AssociationQualifiedName(),
			ObjectHandling: e.ObjectHandling(),
			ExposedName:    e.ExposedName(),
			JsonPath:       e.JsonPath(),
			MinOccurs:      int(e.MinOccurs()),
			MaxOccurs:      int(e.MaxOccurs()),
			Nillable:       e.Nillable(),
		}
		for _, child := range e.ChildrenItems() {
			if c := convertImportElement(child); c != nil {
				out.Children = append(out.Children, c)
			}
		}
		return out
	case *genImpMap.ImportValueMappingElement:
		return &model.ImportMappingElement{
			BaseElement: model.BaseElement{ID: model.ID(e.ID())},
			Kind:        "Value",
			Attribute:   e.AttributeQualifiedName(),
			IsKey:       e.IsKey(),
			ExposedName: e.ExposedName(),
			JsonPath:    e.JsonPath(),
			MinOccurs:   int(e.MinOccurs()),
			MaxOccurs:   int(e.MaxOccurs()),
			Nillable:    e.Nillable(),
		}
	default:
		return nil
	}
}

func importMappingUnitsToModel(units []mprread.Unit[*genImpMap.ImportMapping]) []*model.ImportMapping {
	out := make([]*model.ImportMapping, len(units))
	for i, u := range units {
		out[i] = importMappingToModel(u)
	}
	return out
}

func exportMappingToModel(u mprread.Unit[*genExpMap.ExportMapping]) *model.ExportMapping {
	m := u.Element
	out := &model.ExportMapping{
		BaseElement:       model.BaseElement{ID: model.ID(m.ID()), TypeName: "ExportMappings$ExportMapping"},
		ContainerID:       u.ContainerID,
		Name:              m.Name(),
		Documentation:     m.Documentation(),
		Excluded:          m.Excluded(),
		ExportLevel:       m.ExportLevel(),
		JsonStructure:     m.JsonStructureQualifiedName(),
		XmlSchema:         m.XmlSchemaQualifiedName(),
		MessageDefinition: m.MessageDefinitionQualifiedName(),
		NullValueOption:   m.NullValueOption(),
	}
	for _, raw := range m.RootMappingElementsItems() {
		if elem := convertExportElement(raw); elem != nil {
			out.Elements = append(out.Elements, elem)
		}
	}
	return out
}

func convertExportElement(raw element.Element) *model.ExportMappingElement {
	switch e := raw.(type) {
	case *genExpMap.ExportObjectMappingElement:
		out := &model.ExportMappingElement{
			BaseElement:    model.BaseElement{ID: model.ID(e.ID())},
			Kind:           "Object",
			Entity:         e.EntityQualifiedName(),
			Association:    e.AssociationQualifiedName(),
			ObjectHandling: e.ObjectHandling(),
			ExposedName:    e.ExposedName(),
			JsonPath:       e.JsonPath(),
			MaxOccurs:      int(e.MaxOccurs()),
		}
		for _, child := range e.ChildrenItems() {
			if c := convertExportElement(child); c != nil {
				out.Children = append(out.Children, c)
			}
		}
		return out
	case *genExpMap.ExportValueMappingElement:
		return &model.ExportMappingElement{
			BaseElement: model.BaseElement{ID: model.ID(e.ID())},
			Kind:        "Value",
			Attribute:   e.AttributeQualifiedName(),
			ExposedName: e.ExposedName(),
			JsonPath:    e.JsonPath(),
			MaxOccurs:   int(e.MaxOccurs()),
		}
	default:
		return nil
	}
}

func exportMappingUnitsToModel(units []mprread.Unit[*genExpMap.ExportMapping]) []*model.ExportMapping {
	out := make([]*model.ExportMapping, len(units))
	for i, u := range units {
		out[i] = exportMappingToModel(u)
	}
	return out
}

// ---------------------------------------------------------------------------
// BusinessEventService
// ---------------------------------------------------------------------------

func businessEventServiceToModel(u mprread.Unit[*genBE.BusinessEventService]) *model.BusinessEventService {
	s := u.Element
	out := &model.BusinessEventService{
		BaseElement: model.BaseElement{
			ID:       model.ID(s.ID()),
			TypeName: "BusinessEvents$BusinessEventService",
		},
		ContainerID:   u.ContainerID,
		Name:          s.Name(),
		Documentation: s.Documentation(),
		Excluded:      s.Excluded(),
		ExportLevel:   s.ExportLevel(),
		Document:      s.Document(),
	}
	if def := s.Definition(); def != nil {
		if d, ok := def.(*genBE.BusinessEventDefinition); ok {
			out.Definition = businessEventDefinitionToModel(d)
		}
	}
	for _, item := range s.OperationImplementationsItems() {
		op, ok := item.(*genBE.ServiceOperation)
		if !ok {
			continue
		}
		out.OperationImplementations = append(out.OperationImplementations, serviceOperationToModel(op))
	}
	return out
}

func businessEventDefinitionToModel(d *genBE.BusinessEventDefinition) *model.BusinessEventDefinition {
	out := &model.BusinessEventDefinition{
		BaseElement: model.BaseElement{
			ID:       model.ID(d.ID()),
			TypeName: "BusinessEvents$BusinessEventDefinition",
		},
		ServiceName:     d.ServiceName(),
		EventNamePrefix: d.EventNamePrefix(),
		Description:     d.Description(),
		Summary:         d.Summary(),
	}
	for _, item := range d.ChannelsItems() {
		ch, ok := item.(*genBE.Channel)
		if !ok {
			continue
		}
		out.Channels = append(out.Channels, businessEventChannelToModel(ch))
	}
	return out
}

func businessEventChannelToModel(ch *genBE.Channel) *model.BusinessEventChannel {
	out := &model.BusinessEventChannel{
		BaseElement: model.BaseElement{
			ID:       model.ID(ch.ID()),
			TypeName: "BusinessEvents$Channel",
		},
		ChannelName: ch.ChannelName(),
		Description: ch.Description(),
	}
	for _, item := range ch.MessagesItems() {
		msg, ok := item.(*genBE.Message)
		if !ok {
			continue
		}
		out.Messages = append(out.Messages, businessEventMessageToModel(msg))
	}
	return out
}

func businessEventMessageToModel(m *genBE.Message) *model.BusinessEventMessage {
	out := &model.BusinessEventMessage{
		BaseElement: model.BaseElement{
			ID:       model.ID(m.ID()),
			TypeName: "BusinessEvents$Message",
		},
		MessageName:  m.MessageName(),
		Description:  m.Description(),
		CanPublish:   m.CanPublish(),
		CanSubscribe: m.CanSubscribe(),
	}
	for _, item := range m.AttributesItems() {
		a, ok := item.(*genBE.MessageAttribute)
		if !ok {
			continue
		}
		out.Attributes = append(out.Attributes, businessEventAttributeToModel(a))
	}
	return out
}

func businessEventAttributeToModel(a *genBE.MessageAttribute) *model.BusinessEventAttribute {
	out := &model.BusinessEventAttribute{
		BaseElement: model.BaseElement{
			ID:       model.ID(a.ID()),
			TypeName: "BusinessEvents$MessageAttribute",
		},
		AttributeName: a.AttributeName(),
		Description:   a.Description(),
	}
	if at := a.AttributeType(); at != nil {
		out.AttributeType = attributeTypeFromBsonType(at.TypeName())
	}
	return out
}

func attributeTypeFromBsonType(bsonType string) string {
	switch bsonType {
	case "DomainModels$LongAttributeType":
		return "Long"
	case "DomainModels$StringAttributeType":
		return "String"
	case "DomainModels$IntegerAttributeType":
		return "Integer"
	case "DomainModels$BooleanAttributeType":
		return "Boolean"
	case "DomainModels$DateTimeAttributeType":
		return "DateTime"
	case "DomainModels$DecimalAttributeType":
		return "Decimal"
	case "DomainModels$AutoNumberAttributeType":
		return "AutoNumber"
	case "DomainModels$BinaryAttributeType":
		return "Binary"
	}
	return bsonType
}

func serviceOperationToModel(op *genBE.ServiceOperation) *model.ServiceOperation {
	return &model.ServiceOperation{
		BaseElement: model.BaseElement{
			ID:       model.ID(op.ID()),
			TypeName: "BusinessEvents$ServiceOperation",
		},
		MessageName: op.MessageName(),
		Operation:   op.Operation(),
		Entity:      op.EntityQualifiedName(),
		Microflow:   op.MicroflowQualifiedName(),
	}
}

func businessEventServiceUnitsToModel(units []mprread.Unit[*genBE.BusinessEventService]) []*model.BusinessEventService {
	out := make([]*model.BusinessEventService, len(units))
	for i, u := range units {
		out[i] = businessEventServiceToModel(u)
	}
	return out
}

// ---------------------------------------------------------------------------
// Module and Folder
// ---------------------------------------------------------------------------
//
// gen/projects.Module does not expose Documentation / Excluded fields
// (the underlying BSON schema does not declare them on Projects$Module),
// so those fields are left zero on the resulting model.Module. All other
// fields consumed by the executor (Name, FromAppStore, AppStoreVersion,
// AppStoreGuid, IsReusableComponent) round-trip correctly.

func moduleToModel(u mprread.Unit[*genProj.Module]) *model.Module {
	m := u.Element
	out := &model.Module{
		BaseElement: model.BaseElement{
			ID:       model.ID(m.ID()),
			TypeName: "Projects$Module",
		},
		Name:                m.Name(),
		FromAppStore:        m.FromAppStore(),
		AppStoreVersion:     m.AppStoreVersion(),
		AppStoreGuid:        m.AppStoreGuid(),
		IsReusableComponent: m.IsReusableComponent(),
	}
	return out
}

func moduleUnitsToModel(units []mprread.Unit[*genProj.Module]) []*model.Module {
	out := make([]*model.Module, 0, len(units)+1)
	for _, u := range units {
		out = append(out, moduleToModel(u))
	}
	return out
}

func folderToTypes(u mprread.Unit[*genProj.Folder]) *types.FolderInfo {
	f := u.Element
	return &types.FolderInfo{
		ID:          model.ID(f.ID()),
		ContainerID: u.ContainerID,
		Name:        f.Name(),
	}
}

func folderUnitsToTypes(units []mprread.Unit[*genProj.Folder]) []*types.FolderInfo {
	out := make([]*types.FolderInfo, 0, len(units))
	for _, u := range units {
		out = append(out, folderToTypes(u))
	}
	return out
}
