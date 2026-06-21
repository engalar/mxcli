// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"log"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/bsonutil"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genCW "github.com/mendixlabs/mxcli/modelsdk/gen/customwidgets"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// ============================================================================
// Gen-native helper functions (V3 builder support)
// ============================================================================

// buildNanoflowSourceGen constructs a NanoflowSource gen element from a nanoflow
// DataSourceV3, including parameter mappings for any explicit datasource arguments.
// DataGrid2 callers convert to bson.D via genElementToBSONDoc; Forms callers
// return the element directly.
func (pb *pageBuilder) buildNanoflowSourceGen(ds *ast.DataSourceV3) (*genPg.NanoflowSource, string, error) {
	nfID, err := pb.resolveNanoflowByName(ds.Reference)
	if err != nil {
		return nil, "", mdlerrors.NewBackend("resolve nanoflow", err)
	}
	_ = nfID
	entityName := pb.getNanoflowReturnEntityName(ds.Reference)
	ns := genPg.NewNanoflowSource()
	assignFreshID(ns)
	ns.SetNanoflowQualifiedName(ds.Reference)
	for _, arg := range ds.Args {
		pm := genPg.NewNanoflowParameterMapping()
		assignFreshID(pm)
		pm.SetParameterQualifiedName(ds.Reference + "." + arg.Name)
		if expr, ok := arg.Value.(string); ok {
			pm.SetExpression(expr)
		}
		ns.AddParameterMappings(pm)
	}
	return ns, entityName, nil
}

// =============================================================================
// V3 DataSource Builders
// =============================================================================

// buildDataSourceV3 converts a V3 DataSource AST to an element.Element.
// Returns the datasource element, the entity name for context, and any error.
// For DataView context (database type), produces Forms$DataViewSource.
// For ListView context (database type), use buildListViewDataSourceV3 instead.
func (pb *pageBuilder) buildDataSourceV3(ds *ast.DataSourceV3) (element.Element, string, error) {
	if fn, ok := dataSourceBuilders[ds.Type]; ok {
		return fn(pb, ds)
	}
	return nil, "", mdlerrors.NewUnsupported("unsupported datasource type: " + ds.Type)
}

// buildListViewDataSourceV3 builds a datasource suitable for ListView context.
// For database type, produces Forms$ListViewXPathSource instead of DataViewSource.
func (pb *pageBuilder) buildListViewDataSourceV3(ds *ast.DataSourceV3) (element.Element, string, error) {
	if ds.Type != "database" {
		return pb.buildDataSourceV3(ds)
	}

	entityID, err := pb.resolveEntity(ast.QualifiedName{
		Module: pb.extractModule(ds.Reference),
		Name:   pb.extractName(ds.Reference),
	})
	if err != nil {
		return nil, "", mdlerrors.NewBackend("resolve entity", err)
	}
	_ = entityID

	lvs := genPg.NewListViewXPathSource()
	assignFreshID(lvs)
	lvs.SetEntityPath(ds.Reference)
	ref := genDm.NewDirectEntityRef()
	assignFreshID(ref)
	ref.SetEntityQualifiedName(ds.Reference)
	lvs.SetEntityRef(ref)
	if ds.Where != "" {
		lvs.SetXPathConstraint(ds.Where)
	}
	return lvs, ds.Reference, nil
}

// buildDataGridDataSourceBSON builds a pre-serialized bson.D datasource for use in DataGridSpec.
// Returns the datasource BSON document, the resolved entity name, and any error.
func (pb *pageBuilder) buildDataGridDataSourceBSON(ds *ast.DataSourceV3) (bson.D, string, error) {
	switch ds.Type {
	case "parameter":
		paramName := strings.TrimPrefix(ds.Reference, "$")
		entityID, ok := pb.paramScope[paramName]
		entityName := pb.paramEntityNames[paramName]
		if !ok {
			entityID, ok = pb.paramScope["$"+paramName]
			entityName = pb.paramEntityNames["$"+paramName]
		}
		if !ok {
			return nil, "", mdlerrors.NewNotFound("parameter", ds.Reference)
		}
		if entityName == "" {
			var err error
			entityName, err = pb.getEntityNameByID(entityID)
			if err != nil {
				log.Printf("warning: could not resolve entity name for ID %s: %v", entityID, err)
			}
		}
		var entityRef any
		if entityName != "" {
			entityRef = bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "DomainModels$DirectEntityRef"},
				{Key: "Entity", Value: entityName},
			}
		}
		doc := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$DataViewSource"},
			{Key: "EntityRef", Value: entityRef},
			{Key: "ForceFullObjects", Value: false},
			{Key: "SourceVariable", Value: nil},
		}
		return doc, entityName, nil

	case "database":
		_, err := pb.resolveEntity(ast.QualifiedName{
			Module: pb.extractModule(ds.Reference),
			Name:   pb.extractName(ds.Reference),
		})
		if err != nil {
			return nil, "", mdlerrors.NewBackend("resolve entity", err)
		}
		var entityRef any
		if ds.Reference != "" {
			entityRef = bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "DomainModels$DirectEntityRef"},
				{Key: "Entity", Value: ds.Reference},
			}
		}
		sortItems := bson.A{int32(2)}
		for _, ob := range ds.OrderBy {
			direction := "Ascending"
			if strings.ToLower(ob.Direction) == "desc" {
				direction = "Descending"
			}
			sortItem := bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "Forms$GridSortItem"},
				{Key: "AttributeRef", Value: bson.D{
					{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
					{Key: "$Type", Value: "DomainModels$AttributeRef"},
					{Key: "Attribute", Value: pb.resolveAttributePathForEntity(ob.Attribute, ds.Reference)},
					{Key: "EntityRef", Value: nil},
				}},
				{Key: "SortOrder", Value: direction},
			}
			sortItems = append(sortItems, sortItem)
		}
		doc := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "CustomWidgets$CustomWidgetXPathSource"},
			{Key: "EntityRef", Value: entityRef},
			{Key: "ForceFullObjects", Value: false},
			{Key: "SortBar", Value: bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "Forms$GridSortBar"},
				{Key: "SortItems", Value: sortItems},
			}},
			{Key: "SourceVariable", Value: nil},
			{Key: "XPathConstraint", Value: ds.Where},
		}
		return doc, ds.Reference, nil

	case "microflow":
		mfID, err := pb.resolveMicroflow(ds.Reference)
		if err != nil {
			return nil, "", mdlerrors.NewBackend("resolve microflow", err)
		}
		_ = mfID
		entityName := pb.getMicroflowReturnEntityName(ds.Reference)
		doc := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$MicroflowSource"},
			{Key: "MicroflowSettings", Value: bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "Forms$MicroflowSettings"},
				{Key: "Asynchronous", Value: false},
				{Key: "ConfirmationInfo", Value: nil},
				{Key: "FormValidations", Value: "All"},
				{Key: "Microflow", Value: ds.Reference},
				{Key: "ParameterMappings", Value: bson.A{int32(3)}},
				{Key: "ProgressBar", Value: "None"},
				{Key: "ProgressMessage", Value: nil},
			}},
		}
		return doc, entityName, nil

	case "nanoflow":
		ns, entityName, err := pb.buildNanoflowSourceGen(ds)
		if err != nil {
			return nil, "", err
		}
		doc, err := pb.genElementToBSONDoc(ns)
		if err != nil {
			return nil, "", mdlerrors.NewBackend("encode nanoflow source", err)
		}
		return doc, entityName, nil

	case "association":
		ctxVar := ds.ContextVariable
		if ctxVar == "currentObject" {
			ctxVar = ""
		}
		path := ds.Reference
		destEntity := ""
		if idx := strings.Index(path, "/"); idx >= 0 {
			destEntity = path[idx+1:]
			path = path[:idx]
		} else {
			destEntity = pb.resolveAssociationDestination(path, pb.entityContext)
		}
		step := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "DomainModels$EntityRefStep"},
			{Key: "Association", Value: path},
			{Key: "DestinationEntity", Value: destEntity},
		}
		entityRef := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "DomainModels$IndirectEntityRef"},
			{Key: "Steps", Value: bson.A{int32(2), step}},
		}
		var sourceVar any
		if ctxVar != "" {
			sourceVar = bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "Forms$PageVariable"},
				{Key: "LocalVariable", Value: ""},
				{Key: "PageParameter", Value: ctxVar},
				{Key: "SnippetParameter", Value: ""},
				{Key: "SubKey", Value: ""},
				{Key: "UseAllPages", Value: false},
				{Key: "Widget", Value: ""},
			}
		}
		doc := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$AssociationSource"},
			{Key: "EntityRef", Value: entityRef},
			{Key: "ForceFullObjects", Value: false},
			{Key: "SourceVariable", Value: sourceVar},
		}
		return doc, destEntity, nil

	case "selection":
		widgetName := ds.Reference
		widgetID, ok := pb.widgetScope[widgetName]
		if !ok {
			return nil, "", mdlerrors.NewNotFound("widget", widgetName)
		}
		_ = widgetID
		entityName := pb.paramEntityNames[widgetName]
		doc := bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$ListenTargetSource"},
			{Key: "ListenTarget", Value: widgetName},
		}
		return doc, entityName, nil

	default:
		return nil, "", mdlerrors.NewUnsupported("unsupported datasource type: " + ds.Type)
	}
}

// resolveAssociationDestination looks up an association by qualified name and returns
// the qualified name of the entity OPPOSITE to contextEntity. Returns "" if the
// association can't be resolved or the context isn't on either end.
//
// Convention (per CLAUDE.md): ParentID = FROM entity, ChildID = TO entity.
// For `Module.OrderLine_Order` (`FROM OrderLine TO Order`), context=Order → dest=OrderLine (parent side).
func (pb *pageBuilder) resolveAssociationDestination(assocQN, contextEntity string) string {
	if assocQN == "" {
		return ""
	}
	parts := strings.SplitN(assocQN, ".", 2)
	if len(parts) != 2 {
		return ""
	}
	modName, assocName := parts[0], parts[1]

	pairs, err := pb.getDomainModelsWithContainer()
	if err != nil {
		return ""
	}
	for _, pair := range pairs {
		if pair.DM == nil {
			continue
		}
		if pb.moduleNameByID(pair.ContainerID) != modName {
			continue
		}
		for _, a := range pair.DM.AssociationsItems() {
			assoc, ok := a.(*genDm.Association)
			if !ok || assoc.Name() != assocName {
				continue
			}
			parentEntity := pb.entityQNByID(model.ID(assoc.ParentRefID()))
			childEntity := pb.entityQNByID(model.ID(assoc.ChildRefID()))
			if contextEntity != "" {
				if contextEntity == childEntity {
					return parentEntity
				}
				if contextEntity == parentEntity {
					return childEntity
				}
			}
			return childEntity
		}
	}
	return ""
}

// buildPluggableDataSourceOpaque builds the datasource for pluggable widgets (Gallery, etc.)
// using modelsdk gen types (CustomWidgetXPathSource) serialised via the codec path so no
// direct bson usage is needed here or in widget_engine.go.
func (pb *pageBuilder) buildPluggableDataSourceOpaque(ds *ast.DataSourceV3) (backend.OpaqueWidget, string, error) {
	src := genCW.NewCustomWidgetXPathSource()
	assignFreshID(src)
	src.SetForceFullObjects(false)

	entityName := ""
	switch ds.Type {
	case "database":
		_, err := pb.resolveEntity(ast.QualifiedName{
			Module: pb.extractModule(ds.Reference),
			Name:   pb.extractName(ds.Reference),
		})
		if err != nil {
			log.Printf("warning: buildPluggableDataSourceOpaque: entity %s not found: %v", ds.Reference, err)
		}
		entityName = ds.Reference
		ref := genDm.NewDirectEntityRef()
		assignFreshID(ref)
		ref.SetEntityQualifiedName(ds.Reference)
		src.SetEntityRef(ref)

		// Build sort bar if sort order is specified.
		sortBar := genPg.NewGridSortBar()
		assignFreshID(sortBar)
		for _, ob := range ds.OrderBy {
			item := genPg.NewGridSortItem()
			assignFreshID(item)
			attrRef := genDm.NewAttributeRef()
			assignFreshID(attrRef)
			attrRef.SetAttributeQualifiedName(pb.resolveAttributePathForEntity(ob.Attribute, ds.Reference))
			item.SetAttributeRef(attrRef)
			direction := "Ascending"
			if ob.Direction == "DESC" {
				direction = "Descending"
			}
			item.SetSortDirection(direction)
			sortBar.AddSortItems(item)
		}
		src.SetSortBar(sortBar)
		if ds.Where != "" {
			src.SetXPathConstraint(ds.Where)
		}
	default:
		// For non-database types (parameter, microflow, selection) fall back to
		// the gen DataViewSource path via the standard builder.
		elem, eName, err := pb.buildDataSourceV3(ds)
		if err != nil {
			return nil, "", err
		}
		opaque := pb.widgetBackend.SerializeGenElemToOpaque(elem)
		return opaque, eName, nil
	}

	opaque := pb.widgetBackend.SerializeGenElemToOpaque(src)
	if opaque == nil {
		return nil, "", fmt.Errorf("buildPluggableDataSourceOpaque: serialize returned nil")
	}
	return opaque, entityName, nil
}
