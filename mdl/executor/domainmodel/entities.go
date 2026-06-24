package domainmodel

import (
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func ExecCreateEntityFn(ctx context.Context, s *ast.CreateEntityStmt, d DomainModelDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	if s == nil || s.Name.Module == "" || s.Name.Name == "" {
		return mdlerrors.NewValidation("CREATE ENTITY requires a qualified name (Module.Entity)")
	}

	module, err := d.FindOrCreateModule(s.Name.Module)
	if err != nil {
		return err
	}
	if err := validateCreateEntityTypeRefsFn(ctx, s, d); err != nil {
		return err
	}

	dm, err := d.GetDomainModelGenCached(module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewBackend("get domain model", nil)
	}

	var existingEntity *genDm.Entity
	for _, e := range dm.EntitiesItems() {
		ent, ok := e.(interface{ Name() string })
		if !ok {
			continue
		}
		if ent.Name() == s.Name.Name && !s.CreateOrModify {
			return mdlerrors.NewAlreadyExistsMsg("entity", s.Name.Module+"."+s.Name.Name,
				"entity already exists: "+s.Name.Module+"."+s.Name.Name+" (use create or modify to update)")
		}
		if ent.Name() == s.Name.Name && s.CreateOrModify {
			existingEntity, _ = e.(*genDm.Entity)
			break
		}
	}

	if err := persistEntityDirectFn(ctx, s, dm, existingEntity, module, d); err != nil {
		return err
	}
	return nil
}

func parseEntityLocation(loc string) (x, y int, ok bool) {
	if _, err := fmt.Sscanf(loc, "%d;%d", &x, &y); err == nil {
		return x, y, true
	}
	_, err := fmt.Sscanf(loc, "%d %d", &x, &y)
	return x, y, err == nil
}

func persistEntityDirectFn(ctx context.Context, s *ast.CreateEntityStmt, dm *genDm.DomainModel, existing *genDm.Entity, module *model.Module, d DomainModelDeps) error {
	stmt := *s
	if stmt.Position == nil {
		if existing != nil {
			if loc := existing.Location(); loc != "" {
				if x, y, parsed := parseEntityLocation(loc); parsed {
					stmt.Position = &ast.Position{X: x, Y: y}
				}
			}
		}
		if stmt.Position == nil {
			stmt.Position = &ast.Position{X: 0, Y: 0}
		}
	}

	autoTypeToSystemMember := map[ast.DataTypeKind]string{
		ast.TypeAutoOwner:       "owner",
		ast.TypeAutoChangedBy:   "changedBy",
		ast.TypeAutoCreatedDate: "createdDate",
		ast.TypeAutoChangedDate: "changedDate",
	}

	if len(stmt.Attributes) > 0 {
		existingSystemMembers := make(map[string]bool, len(stmt.SystemMembers))
		for _, m := range stmt.SystemMembers {
			existingSystemMembers[m] = true
		}

		var filteredAttrs []ast.Attribute
		extraSystemMembers := stmt.SystemMembers
		for _, attr := range stmt.Attributes {
			if memberName, isAuto := autoTypeToSystemMember[attr.Type.Kind]; isAuto {
				if !existingSystemMembers[memberName] {
					extraSystemMembers = append(extraSystemMembers, memberName)
					existingSystemMembers[memberName] = true
				}
				continue
			}
			filteredAttrs = append(filteredAttrs, attr)
		}
		stmt.SystemMembers = extraSystemMembers

		attrs := make([]ast.Attribute, len(filteredAttrs))
		copy(attrs, filteredAttrs)
		for i := range attrs {
			if attrs[i].Type.Kind == ast.TypeBoolean && !attrs[i].HasDefault {
				attrs[i].HasDefault = true
				attrs[i].DefaultValue = false
			}
		}
		stmt.Attributes = attrs
	}

	gen, err := executor.BuildEntityFromAST(stmt.Name.Module, &stmt)
	if err != nil {
		return mdlerrors.NewBackend("CREATE ENTITY: build gen entity", err)
	}

	if existing != nil {
		gen.SetID(existing.ID())
		if err := d.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), gen); err != nil {
			return mdlerrors.NewBackend("CREATE ENTITY: update", err)
		}
	} else {
		if err := d.DomainModelWriter.CreateEntityGen(model.ID(dm.ID()), gen); err != nil {
			return mdlerrors.NewBackend("CREATE ENTITY: create", err)
		}
	}

	d.InvalidateDomainModelGenCache(module.ID)
	d.InvalidateDomainModelsCache()
	d.InvalidateHierarchy()

	if s.Position == nil {
		if err := d.DomainModelWriter.RelayoutDomainModel(model.ID(dm.ID())); err != nil {
			fmt.Fprintf(d.Output, "warning: auto-layout failed: %v\n", err)
		} else {
			d.InvalidateDomainModelGenCache(module.ID)
		}
	}

	if existing != nil {
		fmt.Fprintf(d.Output, "Modified entity: %s\n", s.Name)
	} else {
		fmt.Fprintf(d.Output, "Created entity: %s\n", s.Name)
		isPersistent := false
		if g, ok := gen.Generalization().(*genDm.NoGeneralization); ok {
			isPersistent = g.Persistable()
		}
		if isPersistent && len(gen.AccessRulesItems()) == 0 {
			fmt.Fprintf(d.Output, "  \u26a0  %s has no access rules — run SHOW PROJECT SECURITY and GRANT to configure entity-level access\n", s.Name)
		}
	}
	d.TrackModifiedDomainModel(module.ID, module.Name)
	return nil
}

func validateCreateEntityTypeRefsFn(ctx context.Context, s *ast.CreateEntityStmt, d DomainModelDeps) error {
	if s == nil {
		return nil
	}
	for _, a := range s.Attributes {
		if a.Type.Kind != ast.TypeEnumeration || a.Type.EnumRef == nil {
			continue
		}
		refModule := a.Type.EnumRef.Module
		refName := a.Type.EnumRef.Name
		if d.FindEnumeration(refModule, refName) != nil {
			continue
		}
		if _, _, err := d.FindEntityGen(ast.QualifiedName{Module: refModule, Name: refName}); err == nil {
			continue
		}
		return mdlerrors.NewValidationf(
			"attribute '%s': unknown type '%s' — not a primitive, enumeration, or entity",
			a.Name, a.Type.EnumRef.String())
	}
	return nil
}

func ExecDropEntityFn(ctx context.Context, s *ast.DropEntityStmt, d DomainModelDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	if s == nil || s.Name.Module == "" || s.Name.Name == "" {
		return mdlerrors.NewValidation("DROP ENTITY requires a qualified name (Module.Entity)")
	}

	module, err := d.FindModule(s.Name.Module)
	if err != nil {
		return err
	}

	dm, err := d.GetDomainModelGenCached(module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewNotFoundMsg("entity", fmt.Sprint(s.Name), fmt.Sprintf("entity not found: %s", s.Name))
	}

	for _, e := range dm.EntitiesItems() {
		ent, ok := e.(*genDm.Entity)
		if !ok {
			continue
		}
		if ent.Name() != s.Name.Name {
			continue
		}
		d.WarnEntityReferences(s.Name.String())
		d.WarnMicroflowEntityParamRefs(s.Name.String())

		if src := ent.Source(); src != nil && src.TypeName() == "DomainModels$OqlViewEntitySource" {
			if err := d.DomainModelWriter.DeleteViewEntitySourceDocumentByName(s.Name.Module, s.Name.Name); err != nil {
				return mdlerrors.NewBackend("delete view entity source document", err)
			}
		}
		if err := d.DomainModelWriter.DeleteEntity(model.ID(dm.ID()), model.ID(ent.ID())); err != nil {
			return mdlerrors.NewBackend("delete entity", err)
		}
		d.InvalidateDomainModelGenCache(module.ID)
		d.InvalidateDomainModelsCache()
		fmt.Fprintf(d.Output, "Dropped entity: %s\n", s.Name)
		return nil
	}

	return mdlerrors.NewNotFoundMsg("entity", fmt.Sprint(s.Name), fmt.Sprintf("entity not found: %s", s.Name))
}

func ExecAlterEntityFn(ctx context.Context, s *ast.AlterEntityStmt, d DomainModelDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	entity, dm, module, ok := loadAlterEntityTargetFn(ctx, s, d)
	if !ok {
		return mdlerrors.NewNotFoundMsg("entity", fmt.Sprint(s.Name), fmt.Sprintf("entity not found: %s", s.Name))
	}

	switch s.Operation {
	case ast.AlterEntityAddAttribute:
		a := s.Attribute
		if a == nil {
			return mdlerrors.NewValidation("no attribute definition provided")
		}
		if a.Calculated && !d.EntityPersistable(entity) {
			return mdlerrors.NewValidationf("attribute '%s': calculated attributes are only supported on persistent entities", a.Name)
		}
		if executor.FindAttributeGenByName(entity, a.Name) != nil {
			return mdlerrors.NewAlreadyExistsMsg("attribute", a.Name, fmt.Sprintf("attribute '%s' already exists on entity %s", a.Name, s.Name))
		}
		ac := *a
		if ac.Type.Kind == ast.TypeBoolean && !ac.HasDefault {
			ac.HasDefault = true
			ac.DefaultValue = false
		}
		attr := executor.AstToAttributeGen(&ac)
		if attr == nil {
			return mdlerrors.NewValidation("failed to build attribute")
		}
		entity.AddAttributes(attr)
		for _, vr := range executor.AstToValidationRulesGen(&ac, s.Name.String()) {
			entity.AddValidationRules(vr)
		}
		if err := d.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("add attribute", err)
		}
		d.InvalidateHierarchy()
		d.InvalidateDomainModelGenCache(module.ID)
		d.InvalidateDomainModelsCache()
		fmt.Fprintf(d.Output, "Added attribute '%s' to entity %s\n", a.Name, s.Name)

	case ast.AlterEntityDropAttribute:
		if handled, err := executor.ApplyPseudoAttributeDropGen(entity, s.AttributeName); handled {
			if err != nil {
				return err
			}
			if err := d.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
				return mdlerrors.NewBackend("drop attribute", err)
			}
			d.InvalidateHierarchy()
			d.InvalidateDomainModelGenCache(module.ID)
			d.InvalidateDomainModelsCache()
			fmt.Fprintf(d.Output, "Dropped attribute '%s' from entity %s\n", s.AttributeName, s.Name)
			d.TrackModifiedDomainModel(module.ID, module.Name)
			return nil
		}
		attr, attrIdx := executor.FindAttributeGenWithIndexByName(entity, s.AttributeName)
		if attr == nil || attrIdx < 0 {
			return mdlerrors.NewNotFoundMsg("attribute", s.AttributeName, fmt.Sprintf("attribute '%s' not found on entity %s", s.AttributeName, s.Name))
		}
		droppedID := model.ID(attr.ID())
		attrQN := s.Name.String() + "." + s.AttributeName

		origValidationCount := len(entity.ValidationRulesItems())
		origIndexCount := len(entity.IndexesItems())
		removedMemberAccess := executor.CleanupDroppedAttributeReferencesGen(entity, droppedID, attrQN)

		entity.RemoveAttributes(attrIdx)
		if err := d.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("drop attribute", err)
		}
		d.InvalidateHierarchy()
		d.InvalidateDomainModelGenCache(module.ID)
		d.InvalidateDomainModelsCache()
		fmt.Fprintf(d.Output, "Dropped attribute '%s' from entity %s\n", s.AttributeName, s.Name)
		if n := origValidationCount - len(entity.ValidationRulesItems()); n > 0 {
			fmt.Fprintf(d.Output, "  Removed %d validation rule(s)\n", n)
		}
		if removedMemberAccess > 0 {
			fmt.Fprintf(d.Output, "  Removed %d access rule member reference(s)\n", removedMemberAccess)
		}
		if n := origIndexCount - len(entity.IndexesItems()); n > 0 {
			fmt.Fprintf(d.Output, "  Removed %d index(es)\n", n)
		}
		entityQName := s.Name.String()
		fmt.Fprintf(d.Output, "  Warning: pages, microflows, and other documents may still reference '%s'. Update them manually.\n", s.AttributeName)
		fmt.Fprintf(d.Output, "  Use show references to %s to find usages (requires refresh catalog full).\n", entityQName)

	case ast.AlterEntityRenameAttribute:
		attr := executor.FindAttributeGenByName(entity, s.AttributeName)
		if attr == nil {
			return mdlerrors.NewNotFoundMsg("attribute", s.AttributeName, fmt.Sprintf("attribute '%s' not found on entity %s", s.AttributeName, s.Name))
		}
		attr.SetName(s.NewName)
		if err := d.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("rename attribute", err)
		}
		d.InvalidateHierarchy()
		d.InvalidateDomainModelGenCache(module.ID)
		d.InvalidateDomainModelsCache()
		fmt.Fprintf(d.Output, "Renamed attribute '%s' to '%s' on entity %s\n", s.AttributeName, s.NewName, s.Name)

	case ast.AlterEntityModifyAttribute:
		if s.Calculated && !d.EntityPersistable(entity) {
			return mdlerrors.NewValidationf("attribute '%s': calculated attributes are only supported on persistent entities", s.AttributeName)
		}
		attr := executor.FindAttributeGenByName(entity, s.AttributeName)
		if attr == nil {
			return mdlerrors.NewNotFoundMsg("attribute", s.AttributeName, fmt.Sprintf("attribute '%s' not found on entity %s", s.AttributeName, s.Name))
		}
		attr.SetType(executor.AstToAttributeTypeGen(s.DataType))
		if s.Calculated {
			cv := genDm.NewCalculatedValue()
			if s.CalculatedMicroflow != nil {
				cv.SetMicroflowQualifiedName(s.CalculatedMicroflow.String())
			}
			attr.SetValue(cv)
		}
		if err := d.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("modify attribute", err)
		}
		d.InvalidateHierarchy()
		d.InvalidateDomainModelGenCache(module.ID)
		d.InvalidateDomainModelsCache()
		fmt.Fprintf(d.Output, "Modified attribute '%s' on entity %s\n", s.AttributeName, s.Name)

	case ast.AlterEntitySetDocumentation:
		entity.SetDocumentation(s.Documentation)
		if err := d.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("set documentation", err)
		}
		d.InvalidateDomainModelGenCache(module.ID)
		d.InvalidateDomainModelsCache()
		fmt.Fprintf(d.Output, "Set documentation on entity %s\n", s.Name)

	case ast.AlterEntitySetComment:
		entity.SetDocumentation(s.Comment)
		if err := d.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("set comment", err)
		}
		d.InvalidateDomainModelGenCache(module.ID)
		d.InvalidateDomainModelsCache()
		fmt.Fprintf(d.Output, "Set comment on entity %s\n", s.Name)

	case ast.AlterEntitySetPosition:
		if s.Position == nil {
			return mdlerrors.NewValidation("no position provided")
		}
		entity.SetLocation(executor.LayoutPos(s.Position.X, s.Position.Y))
		if err := d.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("set position", err)
		}
		d.InvalidateDomainModelGenCache(module.ID)
		d.InvalidateDomainModelsCache()
		fmt.Fprintf(d.Output, "Set position of entity %s to (%d, %d)\n", s.Name, s.Position.X, s.Position.Y)

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
		index := executor.AstToIndexGen(s.Index, attrNameToID)
		if index == nil || len(index.AttributesItems()) == 0 {
			return mdlerrors.NewValidationf("no valid index columns on entity %s", s.Name)
		}
		entity.AddIndexes(index)
		if err := d.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("add index", err)
		}
		d.InvalidateDomainModelGenCache(module.ID)
		d.InvalidateDomainModelsCache()
		fmt.Fprintf(d.Output, "Added index to entity %s\n", s.Name)

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
		if err := d.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("drop index", err)
		}
		d.InvalidateDomainModelGenCache(module.ID)
		d.InvalidateDomainModelsCache()
		fmt.Fprintf(d.Output, "Dropped index '%s' from entity %s\n", s.IndexName, s.Name)

	case ast.AlterEntityAddEventHandler:
		if s.EventHandler == nil {
			return mdlerrors.NewValidation("missing event handler definition")
		}
		for _, existingElem := range entity.EventHandlersItems() {
			existing, ok := existingElem.(*genDm.EventHandler)
			if !ok {
				continue
			}
			if existing.Moment() == executor.StringTitleLowerASCII(s.EventHandler.Moment) &&
				existing.Event() == executor.StringTitleLowerASCII(s.EventHandler.Event) {
				return mdlerrors.NewAlreadyExistsMsg("event handler",
					fmt.Sprintf("%s %s", s.EventHandler.Moment, s.EventHandler.Event),
					fmt.Sprintf("event handler already exists for %s %s on %s", s.EventHandler.Moment, s.EventHandler.Event, s.Name))
			}
		}
		entity.AddEventHandlers(executor.AstToEventHandlerGen(s.EventHandler))
		if err := d.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("add event handler", err)
		}
		d.InvalidateDomainModelGenCache(module.ID)
		d.InvalidateDomainModelsCache()
		fmt.Fprintf(d.Output, "Added event handler %s %s on %s\n",
			s.EventHandler.Moment, s.EventHandler.Event, s.Name)

	case ast.AlterEntityDropEventHandler:
		if s.EventHandler == nil {
			return mdlerrors.NewValidation("missing event handler reference")
		}
		targetMoment := executor.StringTitleLowerASCII(s.EventHandler.Moment)
		targetEvent := executor.StringTitleLowerASCII(s.EventHandler.Event)
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
		if err := d.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("drop event handler", err)
		}
		d.InvalidateDomainModelGenCache(module.ID)
		d.InvalidateDomainModelsCache()
		fmt.Fprintf(d.Output, "Dropped event handler %s %s from %s\n",
			s.EventHandler.Moment, s.EventHandler.Event, s.Name)

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
		if err := d.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("set system members", err)
		}
		d.InvalidateDomainModelGenCache(module.ID)
		d.InvalidateDomainModelsCache()
		fmt.Fprintf(d.Output, "Set system members (%v) on entity %s\n", s.SystemMembers, s.Name)

	default:
		return mdlerrors.NewUnsupported("unsupported alter entity operation")
	}

	d.TrackModifiedDomainModel(module.ID, module.Name)
	return nil
}

func loadAlterEntityTargetFn(ctx context.Context, s *ast.AlterEntityStmt, d DomainModelDeps) (*genDm.Entity, *genDm.DomainModel, *model.Module, bool) {
	module, err := d.FindModule(s.Name.Module)
	if err != nil {
		return nil, nil, nil, false
	}
	dm, err := d.GetDomainModelGenCached(module.ID)
	if err != nil || dm == nil {
		return nil, nil, nil, false
	}
	entity := executor.FindEntityInDMGenByName(dm, s.Name.Name)
	if entity == nil {
		return nil, nil, nil, false
	}
	return entity, dm, module, true
}

func ExecCreateViewEntityFn(ctx context.Context, s *ast.CreateViewEntityStmt, d DomainModelDeps) error {
	if !d.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	if err := d.CheckFeature("domain_model", "view_entities",
		"create view entity",
		"upgrade your project to 10.18+ or use a regular entity with a microflow data source"); err != nil {
		return err
	}

	if s.Query.RawQuery != "" {
		if oqlViolations := executor.ValidateOQLSyntax(s.Query.RawQuery); len(oqlViolations) > 0 {
			var msgs []string
			for _, v := range oqlViolations {
				msgs = append(msgs, v.Message)
			}
			return mdlerrors.NewValidationf("invalid OQL in view entity '%s':\n  - %s",
				s.Name.String(), strings.Join(msgs, "\n  - "))
		}
	}

	module, err := d.FindModule(s.Name.Module)
	if err != nil {
		return err
	}

	dm, err := d.GetDomainModelGenCached(module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewBackend("get domain model", nil)
	}

	existingEntity := executor.FindEntityInDMGenByName(dm, s.Name.Name)
	if existingEntity != nil && !s.CreateOrModify && !s.CreateOrReplace {
		return mdlerrors.NewAlreadyExistsMsg("entity", s.Name.Module+"."+s.Name.Name,
			fmt.Sprintf("entity already exists: %s.%s (use create or modify to update, or create or replace to drop and recreate)", s.Name.Module, s.Name.Name))
	}

	if existingEntity != nil && s.CreateOrReplace {
		if s.Position == nil {
			x, y, ok := executor.ParseLocationBSON(existingEntity.Location())
			if ok {
				s.Position = &ast.Position{X: x, Y: y}
			}
		}
		if err := d.DomainModelWriter.DeleteViewEntitySourceDocumentByName(s.Name.Module, s.Name.Name); err != nil {
			return mdlerrors.NewBackend("delete existing ViewEntitySourceDocument", err)
		}
		if err := d.DomainModelWriter.DeleteEntity(model.ID(dm.ID()), model.ID(existingEntity.ID())); err != nil {
			return mdlerrors.NewBackend("delete existing entity for replace", err)
		}
		dm, err = d.GetDomainModelGenCached(module.ID)
		if err != nil {
			return mdlerrors.NewBackend("get domain model after delete", err)
		}
		if dm == nil {
			return mdlerrors.NewBackend("get domain model after delete", nil)
		}
		existingEntity = nil
	}

	location := executor.AutoLayoutLocationGen(s.Position, existingEntity, dm)
	sourceDocRef := s.Name.Module + "." + s.Name.Name

	if existingEntity != nil && s.CreateOrModify {
		if err := d.DomainModelWriter.UpdateViewEntitySourceDocument(
			s.Name.Module, s.Name.Name, s.Query.RawQuery, s.Documentation,
		); err != nil {
			return mdlerrors.NewBackend("update ViewEntitySourceDocument", err)
		}
	} else {
		if err := d.DomainModelWriter.DeleteViewEntitySourceDocumentByName(s.Name.Module, s.Name.Name); err != nil {
			return mdlerrors.NewBackend("delete existing ViewEntitySourceDocument", err)
		}
		if _, err := d.DomainModelWriter.CreateViewEntitySourceDocument(
			module.ID,
			s.Name.Module,
			s.Name.Name,
			s.Query.RawQuery,
			s.Documentation,
		); err != nil {
			return mdlerrors.NewBackend("create ViewEntitySourceDocument", err)
		}
	}

	entity := executor.AstToViewEntityGen(s, sourceDocRef, location)
	if entity == nil {
		return mdlerrors.NewValidation("failed to build gen view entity from AST")
	}

	executor.PreserveViewEntityIDs(entity, existingEntity)

	if existingEntity != nil && s.CreateOrModify {
		if err := d.DomainModelWriter.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("update view entity", err)
		}
		d.InvalidateDomainModelGenCache(module.ID)
		d.InvalidateHierarchy()
		d.InvalidateDomainModelsCache()
		fmt.Fprintf(d.Output, "Modified view entity: %s\n", s.Name)
		d.TrackModifiedDomainModel(module.ID, module.Name)
		return nil
	}

	if err := d.DomainModelWriter.CreateEntityGen(model.ID(dm.ID()), entity); err != nil {
		return mdlerrors.NewBackend("create view entity", err)
	}
	d.InvalidateDomainModelGenCache(module.ID)
	d.InvalidateHierarchy()
	d.InvalidateDomainModelsCache()
	fmt.Fprintf(d.Output, "Created view entity: %s\n", s.Name)
	d.TrackModifiedDomainModel(module.ID, module.Name)
	return nil
}
