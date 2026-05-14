// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.1.D5 — gen-typed demo user create/drop.
//
// Mirrors cmd_security_write.go:762–900 (execCreateDemoUser,
// detectUserEntity, joinCandidates, execDropDemoUser) but reads
// ProjectSecurity via getProjectSecurityGen and walks DemoUsersItems()
// with a type-assert to *genSec.DemoUser.
//
// detectUserEntityGen intentionally keeps using ctx.Backend.ListDomainModels()
// (sdk/domainmodel path) because the domain-model domain has not been
// migrated to gen-typed yet (Stage 3.3 priority #4). It accesses only
// string/ID fields on the returned structs — no sdk/domainmodel type
// names appear beyond what the legacy version already used.
//
// Backend mutations (AddDemoUser, RemoveDemoUser) are unchanged from
// legacy; they live in mdl/backend and are not part of this migration.

package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// execCreateDemoUserGen handles CREATE [OR MODIFY] DEMO USER 'name' PASSWORD 'pw'
// [ENTITY Module.Entity] (Roles) using the gen-typed ProjectSecurity path.
func execCreateDemoUserGen(ctx *ExecContext, s *ast.CreateDemoUserStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity not found"))
	}

	// Validate password against project password policy using gen accessor.
	// PasswordPolicySettings() returns element.Element; type-assert to access typed fields.
	if raw := ps.PasswordPolicySettings(); raw != nil {
		if pp, ok := raw.(*genSec.PasswordPolicySettings); ok && pp != nil {
			minLen := int(pp.MinimumLength())
			if minLen > 0 && len(s.Password) < minLen {
				return mdlerrors.NewValidationf("password policy violation for demo user '%s': password must be at least %d characters (got %d)\nhint: check your project's password policy with show project security", s.UserName, minLen, len(s.Password))
			}
		}
	}

	// Check if user already exists — walk DemoUsersItems() with type-assert.
	for _, du := range ps.DemoUsersItems() {
		typed, ok := du.(*genSec.DemoUser)
		if !ok {
			continue
		}
		if typed.UserName() != s.UserName {
			continue
		}
		if !s.CreateOrModify {
			return mdlerrors.NewAlreadyExists("demo user", s.UserName)
		}
		// Additive: merge roles, update password. Drop and re-create with merged roles.
		mergedRoles := typed.UserRolesQualifiedNames()
		existingSet := make(map[string]bool)
		for _, r := range mergedRoles {
			existingSet[r] = true
		}
		for _, r := range s.UserRoles {
			if !existingSet[r] {
				mergedRoles = append(mergedRoles, r)
			}
		}
		entity := typed.EntityQualifiedName()
		if s.Entity != "" {
			entity = s.Entity
		}
		if err := ctx.Backend.RemoveDemoUser(model.ID(ps.ID()), s.UserName); err != nil {
			return mdlerrors.NewBackend("update demo user", err)
		}
		if err := ctx.Backend.AddDemoUser(model.ID(ps.ID()), s.UserName, s.Password, entity, mergedRoles); err != nil {
			return mdlerrors.NewBackend("update demo user", err)
		}
		invalidateProjectSecurityCache(ctx)
		fmt.Fprintf(ctx.Output, "Modified demo user: %s\n", s.UserName)
		return nil
	}

	// Resolve entity: use explicit value or auto-detect from domain models.
	entity := s.Entity
	if entity == "" {
		detected, err := detectUserEntityGen(ctx)
		if err != nil {
			return err
		}
		entity = detected
	}

	if err := ctx.Backend.AddDemoUser(model.ID(ps.ID()), s.UserName, s.Password, entity, s.UserRoles); err != nil {
		return mdlerrors.NewBackend("create demo user", err)
	}
	invalidateProjectSecurityCache(ctx)

	fmt.Fprintf(ctx.Output, "Created demo user: %s (entity: %s)\n", s.UserName, entity)
	return nil
}

// execDropDemoUserGen handles DROP DEMO USER 'name' using the gen-typed
// ProjectSecurity path.
func execDropDemoUserGen(ctx *ExecContext, s *ast.DropDemoUserStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	ps, err := getProjectSecurityGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity not found"))
	}

	// Check if user exists — walk DemoUsersItems() with type-assert.
	found := false
	for _, du := range ps.DemoUsersItems() {
		typed, ok := du.(*genSec.DemoUser)
		if !ok {
			continue
		}
		if typed.UserName() == s.UserName {
			found = true
			break
		}
	}
	if !found {
		return mdlerrors.NewNotFound("demo user", s.UserName)
	}

	if err := ctx.Backend.RemoveDemoUser(model.ID(ps.ID()), s.UserName); err != nil {
		return mdlerrors.NewBackend("drop demo user", err)
	}
	invalidateProjectSecurityCache(ctx)

	fmt.Fprintf(ctx.Output, "Dropped demo user: %s\n", s.UserName)
	return nil
}

// detectUserEntityGen finds the entity that generalizes System.User.
// Intentionally keeps using ctx.Backend.ListDomainModels() (sdk/domainmodel path)
// because the domain-model domain has not been migrated to gen-typed yet.
// Accesses only string/ID fields on the returned structs.
func detectUserEntityGen(ctx *ExecContext) (string, error) {
	modules, err := ctx.Backend.ListModules()
	if err != nil {
		return "", mdlerrors.NewBackend("list modules", err)
	}
	moduleNameByID := make(map[model.ID]string, len(modules))
	for _, m := range modules {
		moduleNameByID[m.ID] = m.Name
	}

	dms, err := ctx.Backend.ListDomainModels()
	if err != nil {
		return "", mdlerrors.NewBackend("list domain models", err)
	}

	var candidates []string
	for _, dm := range dms {
		moduleName := moduleNameByID[dm.ContainerID]
		for _, ent := range dm.Entities {
			if ent.GeneralizationRef == "System.User" {
				candidates = append(candidates, moduleName+"."+ent.Name)
			}
		}
	}

	switch len(candidates) {
	case 0:
		return "", mdlerrors.NewValidation("no entity found that generalizes System.User; use entity clause to specify one")
	case 1:
		return candidates[0], nil
	default:
		return "", mdlerrors.NewValidationf("multiple entities generalize System.User: %s; use entity clause to specify one", joinCandidatesGen(candidates))
	}
}

// joinCandidatesGen formats a slice of candidate strings as a comma-separated
// list. Named with "Gen" suffix to avoid linker conflict with joinCandidates
// in cmd_security_write.go (both files compile in the same package).
func joinCandidatesGen(candidates []string) string {
	result := candidates[0]
	for i := 1; i < len(candidates); i++ {
		result += ", " + candidates[i]
	}
	return result
}
