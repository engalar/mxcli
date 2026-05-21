// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// execCreateViewEntityGen handles CREATE VIEW ENTITY on the gen-native read +
// write path, preserving IDs on MODIFY so the sdk-backed writer can update the
// embedded source and OQL view values without churn.
func execCreateViewEntityGen(ctx *ExecContext, s *ast.CreateViewEntityStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}

	if err := checkFeature(ctx, "domain_model", "view_entities",
		"create view entity",
		"upgrade your project to 10.18+ or use a regular entity with a microflow data source"); err != nil {
		return err
	}

	if s.Query.RawQuery != "" {
		if oqlViolations := ValidateOQLSyntax(s.Query.RawQuery); len(oqlViolations) > 0 {
			var msgs []string
			for _, v := range oqlViolations {
				msgs = append(msgs, v.Message)
			}
			return mdlerrors.NewValidationf("invalid OQL in view entity '%s':\n  - %s",
				s.Name.String(), strings.Join(msgs, "\n  - "))
		}
	}

	module, err := findModule(ctx, s.Name.Module)
	if err != nil {
		return err
	}

	dm, err := ctx.Backend.GetDomainModelGen(module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewBackend("get domain model", nil)
	}

	existingEntity := findEntityInDMGenByName(dm, s.Name.Name)
	if existingEntity != nil && !s.CreateOrModify && !s.CreateOrReplace {
		return mdlerrors.NewAlreadyExistsMsg("entity", s.Name.Module+"."+s.Name.Name,
			fmt.Sprintf("entity already exists: %s.%s (use create or modify to update, or create or replace to drop and recreate)", s.Name.Module, s.Name.Name))
	}

	if existingEntity != nil && s.CreateOrReplace {
		if s.Position == nil {
			x, y, ok := parseLocationBSON(existingEntity.Location())
			if ok {
				s.Position = &ast.Position{X: x, Y: y}
			}
		}
		if err := ctx.Backend.DeleteViewEntitySourceDocumentByName(s.Name.Module, s.Name.Name); err != nil {
			return mdlerrors.NewBackend("delete existing ViewEntitySourceDocument", err)
		}
		if err := ctx.Backend.DeleteEntity(model.ID(dm.ID()), model.ID(existingEntity.ID())); err != nil {
			return mdlerrors.NewBackend("delete existing entity for replace", err)
		}
		dm, err = ctx.Backend.GetDomainModelGen(module.ID)
		if err != nil {
			return mdlerrors.NewBackend("get domain model after delete", err)
		}
		if dm == nil {
			return mdlerrors.NewBackend("get domain model after delete", nil)
		}
		existingEntity = nil
	}

	location := autoLayoutLocationGen(s.Position, existingEntity, dm)
	sourceDocRef := s.Name.Module + "." + s.Name.Name

	if err := ctx.Backend.DeleteViewEntitySourceDocumentByName(s.Name.Module, s.Name.Name); err != nil {
		return mdlerrors.NewBackend("delete existing ViewEntitySourceDocument", err)
	}
	if _, err := ctx.Backend.CreateViewEntitySourceDocument(
		module.ID,
		s.Name.Module,
		s.Name.Name,
		s.Query.RawQuery,
		s.Documentation,
	); err != nil {
		return mdlerrors.NewBackend("create ViewEntitySourceDocument", err)
	}

	entity := astToViewEntityGen(s, sourceDocRef, location)
	if entity == nil {
		return mdlerrors.NewValidation("failed to build gen view entity from AST")
	}
	preserveViewEntityIDs(entity, existingEntity)

	if existingEntity != nil && s.CreateOrModify {
		if err := ctx.Backend.UpdateEntityGen(model.ID(dm.ID()), entity); err != nil {
			return mdlerrors.NewBackend("update view entity", err)
		}
		invalidateHierarchy(ctx)
		invalidateDomainModelsCache(ctx)
		fmt.Fprintf(ctx.Output, "Modified view entity: %s\n", s.Name)
		ctx.trackModifiedDomainModel(module.ID, module.Name)
		return nil
	}

	if err := ctx.Backend.CreateEntityGen(model.ID(dm.ID()), entity); err != nil {
		return mdlerrors.NewBackend("create view entity", err)
	}
	invalidateHierarchy(ctx)
	invalidateDomainModelsCache(ctx)
	fmt.Fprintf(ctx.Output, "Created view entity: %s\n", s.Name)
	ctx.trackModifiedDomainModel(module.ID, module.Name)
	return nil
}

func astToViewEntityGen(s *ast.CreateViewEntityStmt, sourceDocRef string, location model.Point) *genDm.Entity {
	if s == nil {
		return nil
	}
	entity := genDm.NewEntity()
	entity.SetName(s.Name.Name)
	entity.SetDocumentation(s.Documentation)
	entity.SetLocation(layoutPos(location.X, location.Y))

	noGen := genDm.NewNoGeneralization()
	noGen.SetPersistable(true)
	entity.SetGeneralization(noGen)

	src := genDm.NewOqlViewEntitySource()
	src.SetSourceDocumentQualifiedName(sourceDocRef)
	src.SetOql(s.Query.RawQuery)
	entity.SetSource(src)

	for _, a := range s.Attributes {
		attr := genDm.NewAttribute()
		attr.SetName(a.Name)
		attr.SetType(astToAttributeTypeGen(a.Type))
		val := genDm.NewOqlViewValue()
		val.SetReference(a.Name)
		attr.SetValue(val)
		entity.AddAttributes(attr)
	}
	return entity
}

func preserveViewEntityIDs(entity, existing *genDm.Entity) {
	if entity == nil || existing == nil {
		return
	}
	entity.SetID(existing.ID())
	if src, ok := entity.Source().(*genDm.OqlViewEntitySource); ok && src != nil {
		if existingSrc, ok := existing.Source().(*genDm.OqlViewEntitySource); ok && existingSrc != nil {
			src.SetID(existingSrc.ID())
		}
	}

	existingAttrs := make(map[string]*genDm.Attribute)
	for _, a := range existing.AttributesItems() {
		attr, ok := a.(*genDm.Attribute)
		if !ok {
			continue
		}
		existingAttrs[attr.Name()] = attr
	}
	for _, a := range entity.AttributesItems() {
		attr, ok := a.(*genDm.Attribute)
		if !ok {
			continue
		}
		existingAttr := existingAttrs[attr.Name()]
		if existingAttr == nil {
			continue
		}
		attr.SetID(existingAttr.ID())
		if val, ok := attr.Value().(*genDm.OqlViewValue); ok && val != nil {
			if existingVal, ok := existingAttr.Value().(*genDm.OqlViewValue); ok && existingVal != nil {
				val.SetID(existingVal.ID())
			}
		}
	}
}

func autoLayoutLocationGen(pos *ast.Position, existing *genDm.Entity, dm *genDm.DomainModel) model.Point {
	if pos != nil {
		return model.Point{X: pos.X, Y: pos.Y}
	}
	if existing != nil {
		if x, y, ok := parseLocationBSON(existing.Location()); ok {
			return model.Point{X: x, Y: y}
		}
	}
	count := 0
	if dm != nil {
		count = len(dm.EntitiesItems())
	}
	return model.Point{X: 100 + count*150, Y: 100}
}

func findEntityInDMGenByName(dm *genDm.DomainModel, name string) *genDm.Entity {
	if dm == nil {
		return nil
	}
	for _, e := range dm.EntitiesItems() {
		entity, ok := e.(*genDm.Entity)
		if ok && entity.Name() == name {
			return entity
		}
	}
	return nil
}

