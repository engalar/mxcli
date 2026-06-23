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
	"context"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// ExecGrantEntityAccessGenFn is the HandlerDeps version of execGrantEntityAccessGen.
func ExecGrantEntityAccessGenFn(ctx context.Context, s *ast.GrantEntityAccessStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ectx := NewExecContext(ctx, deps)
	module, err := findModule(ectx, s.Entity.Module)
	if err != nil {
		return err
	}

	dm, err := getDomainModelGenCached(ectx, module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewNotFound("entity", s.Entity.Module+"."+s.Entity.Name)
	}

	entity, _, err := findEntityGen(ectx, s.Entity)
	if err != nil {
		return mdlerrors.NewBackend("find entity", err)
	}
	if entity == nil {
		return mdlerrors.NewNotFound("entity", s.Entity.Module+"."+s.Entity.Name)
	}

	var roleNames []string
	for _, role := range s.Roles {
		found, err := validateModuleRole(ectx, role)
		if err != nil {
			return err
		}
		if found {
			roleNames = append(roleNames, role.Module+"."+role.Name)
		}
	}
	if len(roleNames) == 0 {
		return nil
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

	memberSetFrom := func(members []string) map[string]bool {
		s := make(map[string]bool, len(members)*2)
		for _, m := range members {
			s[m] = true
			if idx := strings.LastIndex(m, "."); idx >= 0 {
				s[m[idx+1:]] = true
			}
		}
		return s
	}
	writeMemberSet := memberSetFrom(writeMembers)
	readMemberSet := memberSetFrom(readMembers)

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

	if err := deps.SecurityEntityAccessManager.AddEntityAccessRule(backend.EntityAccessRuleParams{
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

	invalidateDomainModelGenForModule(ectx, module.ID)
	invalidateDomainModelsCache(ectx)

	if msgs, err := deps.SecurityEntityAccessManager.ReconcileMemberAccesses(model.ID(dm.ID()), module.Name); err != nil {
		return mdlerrors.NewBackend("reconcile member accesses", err)
	} else if len(msgs) > 0 && !deps.Quiet {
		for _, msg := range msgs {
			fmt.Fprintf(deps.Output, "  reconciled: %s\n", msg)
		}
	}

	ectx.trackModifiedDomainModel(module.ID, module.Name)
	fmt.Fprintf(deps.Output, "Granted access on %s.%s to %s\n", s.Entity.Module, s.Entity.Name, strings.Join(roleNames, ", "))
	if !deps.Quiet {
		fmt.Fprint(deps.Output, formatAccessRuleResult(ectx, s.Entity.Module, s.Entity.Name, roleNames))
	}
	return nil
}

// ExecRevokeEntityAccessGenFn is the HandlerDeps version of execRevokeEntityAccessGen.
func ExecRevokeEntityAccessGenFn(ctx context.Context, s *ast.RevokeEntityAccessStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ectx := NewExecContext(ctx, deps)
	module, err := findModule(ectx, s.Entity.Module)
	if err != nil {
		return err
	}

	dm, err := getDomainModelGenCached(ectx, module.ID)
	if err != nil {
		return mdlerrors.NewBackend("get domain model", err)
	}
	if dm == nil {
		return mdlerrors.NewNotFound("entity", s.Entity.Module+"."+s.Entity.Name)
	}

	entity, _, err := findEntityGen(ectx, s.Entity)
	if err != nil {
		return mdlerrors.NewBackend("find entity", err)
	}
	if entity == nil {
		return mdlerrors.NewNotFound("entity", s.Entity.Module+"."+s.Entity.Name)
	}

	var roleNames []string
	for _, role := range s.Roles {
		found, err := validateModuleRole(ectx, role)
		if err != nil {
			return err
		}
		if found {
			roleNames = append(roleNames, role.Module+"."+role.Name)
		}
	}
	if len(roleNames) == 0 {
		return nil
	}

	if len(s.Rights) > 0 {
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

		modified, err := deps.SecurityEntityAccessManager.RevokeEntityMemberAccess(model.ID(dm.ID()), s.Entity.Name, roleNames, revocation)
		if err != nil {
			return mdlerrors.NewBackend("revoke entity access", err)
		}

		if modified == 0 {
			fmt.Fprintf(deps.Output, "No access rules found matching %s on %s.%s\n",
				strings.Join(roleNames, ", "), s.Entity.Module, s.Entity.Name)
		} else {
			fmt.Fprintf(deps.Output, "Revoked partial access on %s.%s from %s\n",
				s.Entity.Module, s.Entity.Name, strings.Join(roleNames, ", "))
			if !deps.Quiet {
				fmt.Fprint(deps.Output, formatAccessRuleResult(ectx, s.Entity.Module, s.Entity.Name, roleNames))
			}
		}
	} else {
		modified, err := deps.SecurityEntityAccessManager.RemoveEntityAccessRule(model.ID(dm.ID()), s.Entity.Name, roleNames)
		if err != nil {
			return mdlerrors.NewBackend("revoke entity access", err)
		}

		if modified == 0 {
			fmt.Fprintf(deps.Output, "No access rules found matching %s on %s.%s\n",
				strings.Join(roleNames, ", "), s.Entity.Module, s.Entity.Name)
		} else {
			fmt.Fprintf(deps.Output, "Revoked access on %s.%s from %s\n",
				s.Entity.Module, s.Entity.Name, strings.Join(roleNames, ", "))
			if !deps.Quiet {
				fmt.Fprint(deps.Output, "  Result: (no access)\n")
			}
		}
	}

	ectx.trackModifiedDomainModel(module.ID, module.Name)
	return nil
}

// execGrantEntityAccessGen handles GRANT roles ON MODULE.ENTITY. Delegates to Fn version.

// execRevokeEntityAccessGen handles REVOKE roles ON MODULE.ENTITY. Delegates to Fn version.
