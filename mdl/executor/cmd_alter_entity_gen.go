// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genRest "github.com/mendixlabs/mxcli/modelsdk/gen/rest"
)

func canExecAlterEntityGen(ctx *ExecContext, s *ast.AlterEntityStmt) bool {
	if ctx == nil || s == nil {
		return false
	}
	switch s.Operation {
	case ast.AlterEntityRenameAttribute,
		ast.AlterEntityModifyAttribute,
		ast.AlterEntitySetDocumentation,
		ast.AlterEntitySetComment,
		ast.AlterEntitySetPosition,
		ast.AlterEntityAddIndex,
		ast.AlterEntityDropIndex,
		ast.AlterEntityAddEventHandler,
		ast.AlterEntityDropEventHandler:
		return true
	case ast.AlterEntitySetAllowCreateChangeLocally:
		entity, _, _, ok := loadAlterEntityGenTarget(ctx, s)
		if !ok || entity == nil {
			return false
		}
		_, ok = entity.Source().(*genRest.ODataRemoteEntitySource)
		return ok
	default:
		return false
	}
}

func execAlterEntityGen(ctx *ExecContext, s *ast.AlterEntityStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}

	entity, dm, module, ok := loadAlterEntityGenTarget(ctx, s)
	if !ok {
		return mdlerrors.NewNotFoundMsg("entity", fmt.Sprint(s.Name), fmt.Sprintf("entity not found: %s", s.Name))
	}

	switch s.Operation {
	case ast.AlterEntityRenameAttribute:
		attr := findAttributeGenByName(entity, s.AttributeName)
		if attr == nil {
			return mdlerrors.NewNotFoundMsg("attribute", s.AttributeName, fmt.Sprintf("attribute '%s' not found on entity %s", s.AttributeName, s.Name))
		}
		attr.SetName(s.NewName)
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("rename attribute", err)
		}
		invalidateHierarchy(ctx)
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Renamed attribute '%s' to '%s' on entity %s\n", s.AttributeName, s.NewName, s.Name)

	case ast.AlterEntityModifyAttribute:
		if s.Calculated && !entityPersistableGen(entity) {
			return mdlerrors.NewValidationf("attribute '%s': calculated attributes are only supported on persistent entities", s.AttributeName)
		}
		attr := findAttributeGenByName(entity, s.AttributeName)
		if attr == nil {
			return mdlerrors.NewNotFoundMsg("attribute", s.AttributeName, fmt.Sprintf("attribute '%s' not found on entity %s", s.AttributeName, s.Name))
		}
		attr.SetType(astToAttributeTypeGen(s.DataType))
		if s.Calculated {
			cv := genDm.NewCalculatedValue()
			if s.CalculatedMicroflow != nil {
				cv.SetMicroflowQualifiedName(s.CalculatedMicroflow.String())
			}
			attr.SetValue(cv)
		}
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("modify attribute", err)
		}
		invalidateHierarchy(ctx)
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Modified attribute '%s' on entity %s\n", s.AttributeName, s.Name)

	case ast.AlterEntitySetDocumentation:
		entity.SetDocumentation(s.Documentation)
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("set documentation", err)
		}
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Set documentation on entity %s\n", s.Name)

	case ast.AlterEntitySetComment:
		entity.SetDocumentation(s.Comment)
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("set comment", err)
		}
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Set comment on entity %s\n", s.Name)

	case ast.AlterEntitySetPosition:
		if s.Position == nil {
			return mdlerrors.NewValidation("no position provided")
		}
		entity.SetLocation(fmt.Sprintf("%d,%d", s.Position.X, s.Position.Y))
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("set position", err)
		}
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Set position of entity %s to (%d, %d)\n", s.Name, s.Position.X, s.Position.Y)

	case ast.AlterEntityAddIndex:
		if s.Index == nil {
			return mdlerrors.NewValidation("no index definition provided")
		}
		attrNameToID := make(map[string]model.ID)
		for _, attrElem := range entity.AttributesItems() {
			attr, ok := attrElem.(*genDm.Attribute)
			if !ok {
				continue
			}
			attrNameToID[attr.Name()] = model.ID(attr.ID())
		}
		for _, col := range s.Index.Columns {
			if _, ok := attrNameToID[col.Name]; !ok {
				return mdlerrors.NewNotFoundMsg("attribute", col.Name, fmt.Sprintf("attribute '%s' not found for index on entity %s", col.Name, s.Name))
			}
		}
		index := astToIndexGen(s.Index, attrNameToID)
		if index == nil || len(index.AttributesItems()) == 0 {
			return mdlerrors.NewValidationf("no valid index columns on entity %s", s.Name)
		}
		entity.AddIndexes(index)
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("add index", err)
		}
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Added index to entity %s\n", s.Name)

	case ast.AlterEntityDropIndex:
		if len(entity.IndexesItems()) == 0 {
			return mdlerrors.NewValidationf("no indexes on entity %s", s.Name)
		}
		idx := -1
		for i := range entity.IndexesItems() {
			indexName := fmt.Sprintf("idx%d", i+1)
			if indexName == s.IndexName {
				idx = i
				break
			}
		}
		if idx < 0 {
			return mdlerrors.NewNotFoundMsg("index", s.IndexName, fmt.Sprintf("index '%s' not found on entity %s", s.IndexName, s.Name))
		}
		entity.RemoveIndexes(idx)
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("drop index", err)
		}
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Dropped index '%s' from entity %s\n", s.IndexName, s.Name)

	case ast.AlterEntityAddEventHandler:
		if s.EventHandler == nil {
			return mdlerrors.NewValidation("missing event handler definition")
		}
		for _, existingElem := range entity.EventHandlersItems() {
			existing, ok := existingElem.(*genDm.EventHandler)
			if !ok {
				continue
			}
			if existing.Moment() == stringTitleLowerASCII(s.EventHandler.Moment) &&
				existing.Event() == stringTitleLowerASCII(s.EventHandler.Event) {
				return mdlerrors.NewAlreadyExistsMsg("event handler",
					fmt.Sprintf("%s %s", s.EventHandler.Moment, s.EventHandler.Event),
					fmt.Sprintf("event handler already exists for %s %s on %s", s.EventHandler.Moment, s.EventHandler.Event, s.Name))
			}
		}
		entity.AddEventHandlers(astToEventHandlerGen(s.EventHandler))
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("add event handler", err)
		}
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Added event handler %s %s on %s\n",
			s.EventHandler.Moment, s.EventHandler.Event, s.Name)

	case ast.AlterEntityDropEventHandler:
		if s.EventHandler == nil {
			return mdlerrors.NewValidation("missing event handler reference")
		}
		targetMoment := stringTitleLowerASCII(s.EventHandler.Moment)
		targetEvent := stringTitleLowerASCII(s.EventHandler.Event)
		idx := -1
		for i, existingElem := range entity.EventHandlersItems() {
			existing, ok := existingElem.(*genDm.EventHandler)
			if !ok {
				continue
			}
			if existing.Moment() == targetMoment && existing.Event() == targetEvent {
				idx = i
				break
			}
		}
		if idx < 0 {
			return mdlerrors.NewNotFoundMsg("event handler",
				fmt.Sprintf("%s %s", s.EventHandler.Moment, s.EventHandler.Event),
				fmt.Sprintf("event handler %s %s not found on %s", s.EventHandler.Moment, s.EventHandler.Event, s.Name))
		}
		entity.RemoveEventHandlers(idx)
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("drop event handler", err)
		}
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Dropped event handler %s %s from %s\n",
			s.EventHandler.Moment, s.EventHandler.Event, s.Name)

	case ast.AlterEntitySetAllowCreateChangeLocally:
		src, ok := entity.Source().(*genRest.ODataRemoteEntitySource)
		if !ok || src == nil {
			return mdlerrors.NewUnsupported("allow create change locally is only supported for OData remote entities on the gen path")
		}
		src.SetCreateChangeLocally(s.BoolValue)
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("set allow create change locally", err)
		}
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Set AllowCreateChangeLocally = %v on entity %s\n", s.BoolValue, s.Name)

	default:
		return mdlerrors.NewUnsupported("unsupported alter entity operation")
	}

	ctx.trackModifiedDomainModel(module.ID, module.Name)
	return nil
}

func stringTitleLowerASCII(s string) string {
	if s == "" {
		return ""
	}
	b := []byte(s)
	for i := range b {
		if 'A' <= b[i] && b[i] <= 'Z' {
			b[i] = b[i] - 'A' + 'a'
		}
	}
	if 'a' <= b[0] && b[0] <= 'z' {
		b[0] = b[0] - 'a' + 'A'
	}
	return string(b)
}

func loadAlterEntityGenTarget(ctx *ExecContext, s *ast.AlterEntityStmt) (*genDm.Entity, *genDm.DomainModel, *model.Module, bool) {
	module, err := findModule(ctx, s.Name.Module)
	if err != nil {
		return nil, nil, nil, false
	}
	dm, err := ctx.Backend.GetDomainModelGen(module.ID)
	if err != nil || dm == nil {
		return nil, nil, nil, false
	}
	entity := findEntityInDMGenByName(dm, s.Name.Name)
	if entity == nil {
		return nil, nil, nil, false
	}
	return entity, dm, module, true
}

func findAttributeGenByName(entity *genDm.Entity, name string) *genDm.Attribute {
	if entity == nil {
		return nil
	}
	for _, attrElem := range entity.AttributesItems() {
		attr, ok := attrElem.(*genDm.Attribute)
		if ok && attr.Name() == name {
			return attr
		}
	}
	return nil
}
