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
	case ast.AlterEntityAddAttribute,
		ast.AlterEntityDropAttribute,
		ast.AlterEntityRenameAttribute,
		ast.AlterEntityModifyAttribute,
		ast.AlterEntitySetDocumentation,
		ast.AlterEntitySetComment,
		ast.AlterEntitySetPosition,
		ast.AlterEntityAddIndex,
		ast.AlterEntityDropIndex,
		ast.AlterEntityAddEventHandler,
		ast.AlterEntityDropEventHandler,
		ast.AlterEntitySetSystemMembers:
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
	case ast.AlterEntityAddAttribute:
		a := s.Attribute
		if a == nil {
			return mdlerrors.NewValidation("no attribute definition provided")
		}
		if a.Calculated && !entityPersistableGen(entity) {
			return mdlerrors.NewValidationf("attribute '%s': calculated attributes are only supported on persistent entities", a.Name)
		}
		if findAttributeGenByName(entity, a.Name) != nil {
			return mdlerrors.NewAlreadyExistsMsg("attribute", a.Name, fmt.Sprintf("attribute '%s' already exists on entity %s", a.Name, s.Name))
		}
		ac := *a
		if ac.Type.Kind == ast.TypeBoolean && !ac.HasDefault {
			ac.HasDefault = true
			ac.DefaultValue = false
		}
		attr := astToAttributeGen(&ac)
		if attr == nil {
			return mdlerrors.NewValidation("failed to build attribute")
		}
		entity.AddAttributes(attr)
		for _, vr := range astToValidationRulesGen(&ac, s.Name.String()) {
			entity.AddValidationRules(vr)
		}
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("add attribute", err)
		}
		invalidateHierarchy(ctx)
		invalidateDomainModelGenForModule(ctx, module.ID)
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Added attribute '%s' to entity %s\n", a.Name, s.Name)

	case ast.AlterEntityDropAttribute:
		if handled, err := applyPseudoAttributeDropGen(entity, s.AttributeName); handled {
			if err != nil {
				return err
			}
			if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
				return mdlerrors.NewBackend("drop attribute", err)
			}
			invalidateHierarchy(ctx)
			invalidateDomainModelGenForModule(ctx, module.ID)
			invalidateDomainModelsCache(ctx)
			fmt.Fprintf(ctx.Output, "Dropped attribute '%s' from entity %s\n", s.AttributeName, s.Name)
			ctx.trackModifiedDomainModel(module.ID, module.Name)
			return nil
		}
		attr, attrIdx := findAttributeGenWithIndexByName(entity, s.AttributeName)
		if attr == nil || attrIdx < 0 {
			return mdlerrors.NewNotFoundMsg("attribute", s.AttributeName, fmt.Sprintf("attribute '%s' not found on entity %s", s.AttributeName, s.Name))
		}
		droppedID := model.ID(attr.ID())
		attrQN := s.Name.String() + "." + s.AttributeName

		origValidationCount := len(entity.ValidationRulesItems())
		origIndexCount := len(entity.IndexesItems())
		removedMemberAccess := cleanupDroppedAttributeReferencesGen(entity, droppedID, attrQN)

		entity.RemoveAttributes(attrIdx)
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("drop attribute", err)
		}
		invalidateHierarchy(ctx)
		invalidateDomainModelGenForModule(ctx, module.ID)
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Dropped attribute '%s' from entity %s\n", s.AttributeName, s.Name)
		if n := origValidationCount - len(entity.ValidationRulesItems()); n > 0 {
			fmt.Fprintf(ctx.Output, "  Removed %d validation rule(s)\n", n)
		}
		if removedMemberAccess > 0 {
			fmt.Fprintf(ctx.Output, "  Removed %d access rule member reference(s)\n", removedMemberAccess)
		}
		if n := origIndexCount - len(entity.IndexesItems()); n > 0 {
			fmt.Fprintf(ctx.Output, "  Removed %d index(es)\n", n)
		}
		entityQName := s.Name.String()
		fmt.Fprintf(ctx.Output, "  Warning: pages, microflows, and other documents may still reference '%s'. Update them manually.\n", s.AttributeName)
		fmt.Fprintf(ctx.Output, "  Use show references to %s to find usages (requires refresh catalog full).\n", entityQName)

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
		invalidateDomainModelGenForModule(ctx, module.ID)
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
		invalidateDomainModelGenForModule(ctx, module.ID)
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Modified attribute '%s' on entity %s\n", s.AttributeName, s.Name)

	case ast.AlterEntitySetDocumentation:
		entity.SetDocumentation(s.Documentation)
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("set documentation", err)
		}
		invalidateDomainModelGenForModule(ctx, module.ID)
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Set documentation on entity %s\n", s.Name)

	case ast.AlterEntitySetComment:
		entity.SetDocumentation(s.Comment)
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("set comment", err)
		}
		invalidateDomainModelGenForModule(ctx, module.ID)
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Set comment on entity %s\n", s.Name)

	case ast.AlterEntitySetPosition:
		if s.Position == nil {
			return mdlerrors.NewValidation("no position provided")
		}
		entity.SetLocation(layoutPos(s.Position.X, s.Position.Y))
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("set position", err)
		}
		invalidateDomainModelGenForModule(ctx, module.ID)
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
		invalidateDomainModelGenForModule(ctx, module.ID)
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
		invalidateDomainModelGenForModule(ctx, module.ID)
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
		invalidateDomainModelGenForModule(ctx, module.ID)
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
		invalidateDomainModelGenForModule(ctx, module.ID)
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
		invalidateDomainModelGenForModule(ctx, module.ID)
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Set AllowCreateChangeLocally = %v on entity %s\n", s.BoolValue, s.Name)

	case ast.AlterEntitySetSystemMembers:
		noGen, ok := entity.Generalization().(*genDm.NoGeneralization)
		if !ok {
			return mdlerrors.NewUnsupported("system members are only supported on root (non-specialization) entities")
		}
		noGen.SetHasOwner(false)
		noGen.SetHasChangedBy(false)
		noGen.SetHasCreatedDate(false)
		noGen.SetHasChangedDate(false)
		for _, m := range s.SystemMembers {
			switch m {
			case "owner":
				noGen.SetHasOwner(true)
			case "changedBy":
				noGen.SetHasChangedBy(true)
			case "createdDate":
				noGen.SetHasCreatedDate(true)
			case "changedDate":
				noGen.SetHasChangedDate(true)
			}
		}
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("set system members", err)
		}
		invalidateDomainModelGenForModule(ctx, module.ID)
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Set system members (%v) on entity %s\n", s.SystemMembers, s.Name)

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
	dm, err := getDomainModelGenCached(ctx, module.ID)
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
	attr, _ := findAttributeGenWithIndexByName(entity, name)
	return attr
}

func findAttributeGenWithIndexByName(entity *genDm.Entity, name string) (*genDm.Attribute, int) {
	if entity == nil {
		return nil, -1
	}
	for i, attrElem := range entity.AttributesItems() {
		attr, ok := attrElem.(*genDm.Attribute)
		if ok && attr.Name() == name {
			return attr, i
		}
	}
	return nil, -1
}

func applyPseudoAttributeDropGen(entity *genDm.Entity, attrName string) (bool, error) {
	if entity == nil {
		return false, nil
	}
	noGen, ok := entity.Generalization().(*genDm.NoGeneralization)
	if !ok {
		return false, nil
	}
	switch lowerASCII(attrName) {
	case "owner":
		if noGen.HasOwner() {
			noGen.SetHasOwner(false)
			return true, nil
		}
	case "changedby":
		if noGen.HasChangedBy() {
			noGen.SetHasChangedBy(false)
			return true, nil
		}
	case "createddate":
		if noGen.HasCreatedDate() {
			noGen.SetHasCreatedDate(false)
			return true, nil
		}
	case "changeddate":
		if noGen.HasChangedDate() {
			noGen.SetHasChangedDate(false)
			return true, nil
		}
	}
	return false, nil
}

func cleanupDroppedAttributeReferencesGen(entity *genDm.Entity, droppedID model.ID, attrQN string) int {
	if entity == nil {
		return 0
	}
	for i := len(entity.ValidationRulesItems()) - 1; i >= 0; i-- {
		vr, ok := entity.ValidationRulesItems()[i].(*genDm.ValidationRule)
		if ok && vr.AttributeQualifiedName() == attrQN {
			entity.RemoveValidationRules(i)
		}
	}

	removedMemberAccess := 0
	for _, ruleElem := range entity.AccessRulesItems() {
		rule, ok := ruleElem.(*genDm.AccessRule)
		if !ok {
			continue
		}
		for i := len(rule.MemberAccessesItems()) - 1; i >= 0; i-- {
			ma, ok := rule.MemberAccessesItems()[i].(*genDm.MemberAccess)
			if ok && ma.AttributeQualifiedName() == attrQN {
				rule.RemoveMemberAccesses(i)
				removedMemberAccess++
			}
		}
	}

	for i := len(entity.IndexesItems()) - 1; i >= 0; i-- {
		idx, ok := entity.IndexesItems()[i].(*genDm.Index)
		if !ok {
			continue
		}
		for j := len(idx.AttributesItems()) - 1; j >= 0; j-- {
			ia, ok := idx.AttributesItems()[j].(*genDm.IndexedAttribute)
			if ok && model.ID(ia.AttributeRefID()) == droppedID {
				idx.RemoveAttributes(j)
			}
		}
		if len(idx.AttributesItems()) == 0 {
			entity.RemoveIndexes(i)
		}
	}

	return removedMemberAccess
}

func lowerASCII(s string) string {
	if s == "" {
		return ""
	}
	b := []byte(s)
	for i := range b {
		if 'A' <= b[i] && b[i] <= 'Z' {
			b[i] = b[i] - 'A' + 'a'
		}
	}
	return string(b)
}
