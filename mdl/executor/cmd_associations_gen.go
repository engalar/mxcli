// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 A4: gen-typed read functions for association queries.

package executor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// listAssociationsGen handles SHOW ASSOCIATIONS on the gen-typed read
// path (Stage 3.3.4 A4 — replaces listAssociations in cmd_associations.go).
// Walks every DomainModel in the project, intra-module Associations and
// CrossAssociations both included.
func listAssociationsGen(ctx *ExecContext, moduleName string) error {
	pairs, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list domain models", err)
	}

	mods, err := ctx.Backend.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}
	moduleNames := make(map[model.ID]string, len(mods))
	for _, m := range mods {
		moduleNames[m.ID] = m.Name
	}

	// Build entity ID -> qualified name map across all modules so we can
	// translate ParentRefID / ChildRefID into "Module.EntityName".
	entityNames := make(map[model.ID]string)
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		modName := moduleNames[p.ContainerID]
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			entityNames[model.ID(entity.ID())] = modName + "." + entity.Name()
		}
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		parent        string
		child         string
		assocType     string
		owner         string
		storage       string
	}
	var rows []row

	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		modName := moduleNames[p.ContainerID]
		if moduleName != "" && modName != moduleName {
			continue
		}
		for _, a := range p.DM.AssociationsItems() {
			assoc, ok := a.(*genDm.Association)
			if !ok {
				continue
			}
			parentID := model.ID(assoc.ParentRefID())
			childID := model.ID(assoc.ChildRefID())
			parent := entityNames[parentID]
			if parent == "" {
				parent = string(parentID)
			}
			child := entityNames[childID]
			if child == "" {
				child = string(childID)
			}
			rows = append(rows, row{
				qualifiedName: modName + "." + assoc.Name(),
				module:        modName,
				name:          assoc.Name(),
				parent:        parent,
				child:         child,
				assocType:     assoc.Type(),
				owner:         assoc.Owner(),
				storage:       assoc.StorageFormat(),
			})
		}
		for _, c := range p.DM.CrossAssociationsItems() {
			ca, ok := c.(*genDm.CrossAssociation)
			if !ok {
				continue
			}
			parentID := model.ID(ca.ParentRefID())
			parent := entityNames[parentID]
			if parent == "" {
				parent = string(parentID)
			}
			rows = append(rows, row{
				qualifiedName: modName + "." + ca.Name(),
				module:        modName,
				name:          ca.Name(),
				parent:        parent,
				child:         ca.ChildQualifiedName(),
				assocType:     ca.Type(),
				owner:         ca.Owner(),
				storage:       ca.StorageFormat(),
			})
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Parent", "Child", "Type", "Owner", "Storage"},
		Summary: fmt.Sprintf("(%d associations)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.parent, r.child, r.assocType, r.owner, r.storage})
	}
	return writeResult(ctx, result)
}

// describeAssociationGen handles DESCRIBE ASSOCIATION on the gen-typed
// read path (Stage 3.3.4 A4). Renders MDL `create association ... from
// X to Y` form including type / owner / storage / delete_behavior.
func describeAssociationGen(ctx *ExecContext, name ast.QualifiedName) error {
	pairs, err := listDomainModelsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list domain models", err)
	}
	mods, err := ctx.Backend.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}
	moduleNames := make(map[model.ID]string, len(mods))
	for _, m := range mods {
		moduleNames[m.ID] = m.Name
	}
	entityNames := make(map[model.ID]string)
	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		modName := moduleNames[p.ContainerID]
		for _, e := range p.DM.EntitiesItems() {
			entity, ok := e.(*genDm.Entity)
			if !ok {
				continue
			}
			entityNames[model.ID(entity.ID())] = modName + "." + entity.Name()
		}
	}

	formatAssocDetails := func(assocType, owner, storage string, deleteBehavior interface{}) {
		typeName := "Reference"
		if assocType == "ReferenceSet" {
			typeName = "ReferenceSet"
		}
		fmt.Fprintf(ctx.Output, "type %s\n", typeName)
		ownerStr := "Default"
		if owner == "Both" {
			ownerStr = "Both"
		}
		fmt.Fprintf(ctx.Output, "owner %s\n", ownerStr)
		if storage == "Column" {
			fmt.Fprintln(ctx.Output, "storage column")
		}
		// Delete behavior for the child side — gen carries this on the
		// AssociationDeleteBehavior element with ParentDeleteBehavior /
		// ChildDeleteBehavior strings (no enum on read).
		childDel := "DELETE_BUT_KEEP_REFERENCES"
		if dbe, ok := deleteBehavior.(*genDm.AssociationDeleteBehavior); ok && dbe != nil {
			switch dbe.ChildDeleteBehavior() {
			case "DeleteMeAndReferences":
				childDel = "DELETE_CASCADE"
			case "DeleteMeIfNoReferences":
				childDel = "DELETE_IF_NO_REFERENCES"
			case "DeleteMeButKeepReferences", "":
				childDel = "DELETE_BUT_KEEP_REFERENCES"
			}
		}
		fmt.Fprintf(ctx.Output, "delete_behavior %s;\n", childDel)
	}

	for _, p := range pairs {
		if p.DM == nil {
			continue
		}
		modName := moduleNames[p.ContainerID]
		if modName != name.Module {
			continue
		}
		for _, a := range p.DM.AssociationsItems() {
			assoc, ok := a.(*genDm.Association)
			if !ok || assoc.Name() != name.Name {
				continue
			}
			fromQN := entityNames[model.ID(assoc.ParentRefID())]
			toQN := entityNames[model.ID(assoc.ChildRefID())]
			if doc := assoc.Documentation(); doc != "" {
				fmt.Fprintf(ctx.Output, "/**\n * %s\n */\n", doc)
			}
			fmt.Fprintf(ctx.Output, "create association %s.%s\n", modName, assoc.Name())
			fmt.Fprintf(ctx.Output, "from %s to %s\n", fromQN, toQN)
			formatAssocDetails(assoc.Type(), assoc.Owner(), assoc.StorageFormat(), assoc.DeleteBehavior())
			fmt.Fprintln(ctx.Output, "/")
			return nil
		}
		for _, c := range p.DM.CrossAssociationsItems() {
			ca, ok := c.(*genDm.CrossAssociation)
			if !ok || ca.Name() != name.Name {
				continue
			}
			fromQN := entityNames[model.ID(ca.ParentRefID())]
			if fromQN == "" {
				fromQN = string(ca.ParentRefID())
			}
			if doc := ca.Documentation(); doc != "" {
				fmt.Fprintf(ctx.Output, "/**\n * %s\n */\n", doc)
			}
			fmt.Fprintf(ctx.Output, "create association %s.%s\n", modName, ca.Name())
			fmt.Fprintf(ctx.Output, "from %s to %s\n", fromQN, ca.ChildQualifiedName())
			formatAssocDetails(ca.Type(), ca.Owner(), ca.StorageFormat(), ca.DeleteBehavior())
			fmt.Fprintln(ctx.Output, "/")
			return nil
		}
	}
	return mdlerrors.NewNotFound("association", name.String())
}
