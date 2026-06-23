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
	"context"
	"fmt"
	"unicode"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// ExecCreateDemoUserGenFn is the HandlerDeps version of execCreateDemoUserGen.
func ExecCreateDemoUserGenFn(ctx context.Context, s *ast.CreateDemoUserStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ectx := NewExecContext(ctx, deps)
	ps, err := getProjectSecurityGen(ectx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity not found"))
	}
	if raw := ps.PasswordPolicySettings(); raw != nil {
		if pp, ok := raw.(*genSec.PasswordPolicySettings); ok && pp != nil {
			if err := validatePasswordPolicy(s.UserName, s.Password, pp); err != nil {
				return err
			}
		}
	}
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
		if err := deps.SecurityProjectManager.RemoveDemoUser(model.ID(ps.ID()), s.UserName); err != nil {
			return mdlerrors.NewBackend("update demo user", err)
		}
		if err := deps.SecurityProjectManager.AddDemoUser(model.ID(ps.ID()), s.UserName, s.Password, entity, mergedRoles); err != nil {
			return mdlerrors.NewBackend("update demo user", err)
		}
		invalidateProjectSecurityCache(ectx)
		fmt.Fprintf(deps.Output, "Modified demo user: %s\n", s.UserName)
		return nil
	}
	entity := s.Entity
	if entity == "" {
		detected, err := detectUserEntityGenFn(ctx, deps)
		if err != nil {
			return err
		}
		entity = detected
	}
	if err := deps.SecurityProjectManager.AddDemoUser(model.ID(ps.ID()), s.UserName, s.Password, entity, s.UserRoles); err != nil {
		return mdlerrors.NewBackend("create demo user", err)
	}
	invalidateProjectSecurityCache(ectx)
	fmt.Fprintf(deps.Output, "Created demo user: %s (entity: %s)\n", s.UserName, entity)
	return nil
}

// ExecDropDemoUserGenFn is the HandlerDeps version of execDropDemoUserGen.
func ExecDropDemoUserGenFn(ctx context.Context, s *ast.DropDemoUserStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnectedWrite()
	}
	ectx := NewExecContext(ctx, deps)
	ps, err := getProjectSecurityGen(ectx)
	if err != nil {
		return mdlerrors.NewBackend("read project security", err)
	}
	if ps == nil {
		return mdlerrors.NewBackend("read project security", fmt.Errorf("ProjectSecurity not found"))
	}
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
	if err := deps.SecurityProjectManager.RemoveDemoUser(model.ID(ps.ID()), s.UserName); err != nil {
		return mdlerrors.NewBackend("drop demo user", err)
	}
	invalidateProjectSecurityCache(ectx)
	fmt.Fprintf(deps.Output, "Dropped demo user: %s\n", s.UserName)
	return nil
}

// detectUserEntityGenFn is the HandlerDeps version of detectUserEntityGen.
func detectUserEntityGenFn(ctx context.Context, deps *HandlerDeps) (string, error) {
	ectx := NewExecContext(ctx, deps)
	return detectUserEntityGen(ectx)
}

// validatePasswordPolicy checks the password against all configured rules.
// Returns a validation error with an actionable fix hint if any rule is violated.
func validatePasswordPolicy(userName, password string, pp *genSec.PasswordPolicySettings) error {
	if pp == nil {
		return nil
	}
	if min := int(pp.MinimumLength()); min > 0 && len(password) < min {
		return mdlerrors.NewValidationf(
			"demo user '%s': password is %d characters, policy requires at least %d\n"+
				"hint: either use a longer password, or relax the policy first:\n"+
				"  alter project security password policy (min_length: %d, require_digit: %v, require_mixed_case: %v, require_symbol: %v);\n"+
				"  create or modify demo user '%s' password '<new-password>' ...;",
			userName, len(password), min,
			len(password), pp.RequireDigit(), pp.RequireMixedCase(), pp.RequireSymbol(),
			userName,
		)
	}
	if pp.RequireDigit() && !passwordContainsDigit(password) {
		return mdlerrors.NewValidationf(
			"demo user '%s': password must contain at least one digit (policy: require_digit: true)\n"+
				"hint: add a digit to the password, or disable the requirement:\n"+
				"  alter project security password policy (require_digit: false);",
			userName,
		)
	}
	if pp.RequireMixedCase() && (!passwordContainsUpper(password) || !passwordContainsLower(password)) {
		return mdlerrors.NewValidationf(
			"demo user '%s': password must contain both uppercase and lowercase letters (policy: require_mixed_case: true)\n"+
				"hint: add mixed-case letters, or disable the requirement:\n"+
				"  alter project security password policy (require_mixed_case: false);",
			userName,
		)
	}
	if pp.RequireSymbol() && !passwordContainsSymbol(password) {
		return mdlerrors.NewValidationf(
			"demo user '%s': password must contain at least one symbol (policy: require_symbol: true)\n"+
				"hint: add a symbol such as '!' or '@', or disable the requirement:\n"+
				"  alter project security password policy (require_symbol: false);",
			userName,
		)
	}
	return nil
}

func passwordContainsDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func passwordContainsUpper(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

func passwordContainsLower(s string) bool {
	for _, r := range s {
		if unicode.IsLower(r) {
			return true
		}
	}
	return false
}

func passwordContainsSymbol(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// execCreateDemoUserGen handles CREATE [OR MODIFY] DEMO USER. Delegates to Fn version.

// execDropDemoUserGen handles DROP DEMO USER. Delegates to Fn version.

// detectUserEntityGen finds the entity that generalizes System.User.
func detectUserEntityGen(ctx *ExecContext) (string, error) {
	modules, err := ctx.ModuleLister.ListModules()
	if err != nil {
		return "", mdlerrors.NewBackend("list modules", err)
	}
	moduleNameByID := make(map[model.ID]string, len(modules))
	for _, m := range modules {
		moduleNameByID[m.ID] = m.Name
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("build hierarchy", err)
	}

	dms, err := cachedDomainModelsGen(ctx)
	if err != nil {
		return "", mdlerrors.NewBackend("list domain models", err)
	}

	var candidates []string
	for _, dm := range dms {
		if dm == nil {
			continue
		}
		moduleName := moduleNameByID[h.FindModuleID(model.ID(dm.ID()))]
		for _, entityElem := range dm.EntitiesItems() {
			ent, ok := entityElem.(*genDm.Entity)
			if !ok {
				continue
			}
			if entityGeneralizationQNGen(ent) == "System.User" {
				candidates = append(candidates, moduleName+"."+ent.Name())
			}
		}
	}

	switch len(candidates) {
	case 0:
		// No custom user entity — project uses System.User directly (no
		// Administration.Account-style extension). Fall back to System.User so
		// demo users can still be created in minimal projects.
		return "System.User", nil
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
