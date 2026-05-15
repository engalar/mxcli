// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.1 D6b — gen-typed entity access write paths.
//
// execGrantEntityAccessGen mirrors execGrantEntityAccess (cmd_security_write.go:292-458)
// but module-security validation uses ctx.Backend.GetModuleSecurityGen (gen path).
// Domain-model reads still go through ctx.Backend.GetDomainModel (passthrough until
// Stage 3.3 priority #4); mutations go through ctx.Backend.AddEntityAccessRule and
// ctx.Backend.RemoveEntityAccessRule (already wired to addEntityAccessRuleViaModelsdk
// and corrected by D6a for the versioned-array prefix).
//
// execRevokeEntityAccessGen mirrors execRevokeEntityAccess (cmd_security_write.go:461-553).

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// execGrantEntityAccessGen handles GRANT roles ON MODULE.ENTITY [(rights...)].
// Gen-typed security validation; domain-model read via GetDomainModel (passthrough).
func execGrantEntityAccessGen(ctx *ExecContext, s *ast.GrantEntityAccessStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	module, err := findModule(ctx, s.Entity.Module)
	if err != nil {
		return err
	}

	dm, err := ctx.Backend.GetDomainModelGen(module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewNotFound("entity", s.Entity.Module+"."+s.Entity.Name)
	}

	entity, _, err := findEntityGen(ctx, s.Entity)
	if err != nil {
		return mdlerrors.NewBackend("find entity", err)
	}
	if entity == nil {
		return mdlerrors.NewNotFound("entity", s.Entity.Module+"."+s.Entity.Name)
	}

	// validateModuleRole already uses GetModuleSecurityGen internally.
	for _, role := range s.Roles {
		if err := validateModuleRole(ctx, role); err != nil {
			return err
		}
	}

	var roleNames []string
	for _, role := range s.Roles {
		roleNames = append(roleNames, role.Module+"."+role.Name)
	}

	allowCreate, allowDelete := false, false
	defaultMemberAccess := "None"
	var readMembers, writeMembers []string
	for _, right := range s.Rights {
		switch right.Type {
		case ast.EntityAccessCreate:
			allowCreate = true
		case ast.EntityAccessDelete:
			allowDelete = true
		case ast.EntityAccessReadAll:
			if defaultMemberAccess == "None" {
				defaultMemberAccess = "ReadOnly"
			}
		case ast.EntityAccessReadMembers:
			readMembers = right.Members
		case ast.EntityAccessWriteAll:
			defaultMemberAccess = "ReadWrite"
		case ast.EntityAccessWriteMembers:
			writeMembers = right.Members
		}
	}

	var memberAccesses []types.EntityMemberAccess

	writeMemberSet := make(map[string]bool)
	for _, m := range writeMembers {
		writeMemberSet[m] = true
	}
	readMemberSet := make(map[string]bool)
	for _, m := range readMembers {
		readMemberSet[m] = true
	}

	for _, attrElem := range entity.AttributesItems() {
		attr, ok := attrElem.(*genDm.Attribute)
		if !ok {
			continue
		}
		rights := defaultMemberAccess
		if writeMemberSet[attr.Name()] {
			rights = "ReadWrite"
		} else if readMemberSet[attr.Name()] {
			rights = "ReadOnly"
		}
		// Calculated attributes cannot have write rights (CE6592).
		isCalculated := false
		if val := attr.Value(); val != nil && val.TypeName() == "DomainModels$CalculatedValue" {
			isCalculated = true
		}
		if isCalculated && (rights == "ReadWrite" || rights == "WriteOnly") {
			rights = "ReadOnly"
		}
		memberAccesses = append(memberAccesses, types.EntityMemberAccess{
			AttributeRef: module.Name + "." + s.Entity.Name + "." + attr.Name(),
			AccessRights: rights,
		})
	}

	// MemberAccess for associations is only required on the FROM entity
	// (ParentID = FROM entity / FK owner). Adding to the TO side triggers CE0066.
	for _, assocElem := range dm.AssociationsItems() {
		assoc, ok := assocElem.(*genDm.Association)
		if !ok || model.ID(assoc.ParentRefID()) != model.ID(entity.ID()) {
			continue
		}
		rights := defaultMemberAccess
		if writeMemberSet[assoc.Name()] {
			rights = "ReadWrite"
		} else if readMemberSet[assoc.Name()] {
			rights = "ReadOnly"
		}
		memberAccesses = append(memberAccesses, types.EntityMemberAccess{
			AssociationRef: module.Name + "." + assoc.Name(),
			AccessRights:   rights,
		})
	}
	for _, crossElem := range dm.CrossAssociationsItems() {
		ca, ok := crossElem.(*genDm.CrossAssociation)
		if !ok || model.ID(ca.ParentRefID()) != model.ID(entity.ID()) {
			continue
		}
		rights := defaultMemberAccess
		if writeMemberSet[ca.Name()] {
			rights = "ReadWrite"
		} else if readMemberSet[ca.Name()] {
			rights = "ReadOnly"
		}
		memberAccesses = append(memberAccesses, types.EntityMemberAccess{
			AssociationRef: module.Name + "." + ca.Name(),
			AccessRights:   rights,
		})
	}

	if ng, ok := entity.Generalization().(*genDm.NoGeneralization); ok && ng.HasOwner() {
		memberAccesses = append(memberAccesses, types.EntityMemberAccess{
			AssociationRef: "System.owner",
			AccessRights:   defaultMemberAccess,
		})
	}
	if ng, ok := entity.Generalization().(*genDm.NoGeneralization); ok && ng.HasChangedBy() {
		memberAccesses = append(memberAccesses, types.EntityMemberAccess{
			AssociationRef: "System.changedBy",
			AccessRights:   defaultMemberAccess,
		})
	}

	if err := ctx.Backend.AddEntityAccessRule(backend.EntityAccessRuleParams{
		UnitID:              model.ID(dm.ID()),
		EntityName:          s.Entity.Name,
		RoleNames:           roleNames,
		AllowCreate:         allowCreate,
		AllowDelete:         allowDelete,
		DefaultMemberAccess: defaultMemberAccess,
		XPathConstraint:     s.XPathConstraint,
		MemberAccesses:      memberAccesses,
	}); err != nil {
		return mdlerrors.NewBackend("grant entity access", err)
	}

	if count, err := ctx.Backend.ReconcileMemberAccesses(model.ID(dm.ID()), module.Name); err != nil {
		return mdlerrors.NewBackend("reconcile member accesses", err)
	} else if count > 0 && !ctx.Quiet {
		fmt.Fprintf(ctx.Output, "Reconciled %d access rule(s) in module %s\n", count, module.Name)
	}

	ctx.trackModifiedDomainModel(module.ID, module.Name)
	fmt.Fprintf(ctx.Output, "Granted access on %s.%s to %s\n", s.Entity.Module, s.Entity.Name, strings.Join(roleNames, ", "))
	if !ctx.Quiet {
		fmt.Fprint(ctx.Output, formatAccessRuleResult(ctx, s.Entity.Module, s.Entity.Name, roleNames))
	}
	return nil
}

// execRevokeEntityAccessGen handles REVOKE roles ON MODULE.ENTITY [(rights...)].
// Gen-typed security validation; domain-model read via GetDomainModel (passthrough).
func execRevokeEntityAccessGen(ctx *ExecContext, s *ast.RevokeEntityAccessStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	module, err := findModule(ctx, s.Entity.Module)
	if err != nil {
		return err
	}

	dm, err := ctx.Backend.GetDomainModelGen(module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewNotFound("entity", s.Entity.Module+"."+s.Entity.Name)
	}

	entity, _, err := findEntityGen(ctx, s.Entity)
	if err != nil {
		return mdlerrors.NewBackend("find entity", err)
	}
	if entity == nil {
		return mdlerrors.NewNotFound("entity", s.Entity.Module+"."+s.Entity.Name)
	}

	// validateModuleRole already uses GetModuleSecurityGen internally.
	for _, role := range s.Roles {
		if err := validateModuleRole(ctx, role); err != nil {
			return err
		}
	}

	var roleNames []string
	for _, role := range s.Roles {
		roleNames = append(roleNames, role.Module+"."+role.Name)
	}

	if len(s.Rights) > 0 {
		// Partial revoke — downgrade specific rights.
		revocation := types.EntityAccessRevocation{}
		for _, right := range s.Rights {
			switch right.Type {
			case ast.EntityAccessCreate:
				revocation.RevokeCreate = true
			case ast.EntityAccessDelete:
				revocation.RevokeDelete = true
			case ast.EntityAccessReadAll:
				revocation.RevokeReadAll = true
			case ast.EntityAccessWriteAll:
				revocation.RevokeWriteAll = true
			case ast.EntityAccessReadMembers:
				for _, m := range right.Members {
					revocation.RevokeReadMembers = append(revocation.RevokeReadMembers,
						module.Name+"."+s.Entity.Name+"."+m)
				}
			case ast.EntityAccessWriteMembers:
				for _, m := range right.Members {
					revocation.RevokeWriteMembers = append(revocation.RevokeWriteMembers,
						module.Name+"."+s.Entity.Name+"."+m)
				}
			}
		}

		modified, err := ctx.Backend.RevokeEntityMemberAccess(model.ID(dm.ID()), s.Entity.Name, roleNames, revocation)
		if err != nil {
			return mdlerrors.NewBackend("revoke entity access", err)
		}

		if modified == 0 {
			fmt.Fprintf(ctx.Output, "No access rules found matching %s on %s.%s\n",
				strings.Join(roleNames, ", "), s.Entity.Module, s.Entity.Name)
		} else {
			fmt.Fprintf(ctx.Output, "Revoked partial access on %s.%s from %s\n",
				s.Entity.Module, s.Entity.Name, strings.Join(roleNames, ", "))
			if !ctx.Quiet {
				fmt.Fprint(ctx.Output, formatAccessRuleResult(ctx, s.Entity.Module, s.Entity.Name, roleNames))
			}
		}
	} else {
		// Full revoke — remove entire access rule.
		modified, err := ctx.Backend.RemoveEntityAccessRule(model.ID(dm.ID()), s.Entity.Name, roleNames)
		if err != nil {
			return mdlerrors.NewBackend("revoke entity access", err)
		}

		if modified == 0 {
			fmt.Fprintf(ctx.Output, "No access rules found matching %s on %s.%s\n",
				strings.Join(roleNames, ", "), s.Entity.Module, s.Entity.Name)
		} else {
			fmt.Fprintf(ctx.Output, "Revoked access on %s.%s from %s\n",
				s.Entity.Module, s.Entity.Name, strings.Join(roleNames, ", "))
			if !ctx.Quiet {
				fmt.Fprint(ctx.Output, "  Result: (no access)\n")
			}
		}
	}

	ctx.trackModifiedDomainModel(module.ID, module.Name)
	return nil
}
