// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// ---------------------------------------------------------------------------
// SecurityBackend (ProjectSecurityBackend + ModuleSecurityBackend + EntityAccessBackend)
// ---------------------------------------------------------------------------

func (b *MprBackend) GetProjectSecurityGen() (*genSec.ProjectSecurity, error) {
	b.initSubBackends()
	return b.security.GetProjectSecurityGen()
}
func (b *MprBackend) SetProjectSecurityLevel(unitID model.ID, level string) error {
	return b.setSecurityLevelViaModelsdk(unitID, level)
}
func (b *MprBackend) SetProjectDemoUsersEnabled(unitID model.ID, enabled bool) error {
	return b.setProjectDemoUsersEnabledViaModelsdk(unitID, enabled)
}
func (b *MprBackend) AddUserRole(unitID model.ID, name string, moduleRoles []string, manageAllRoles bool) error {
	return b.addUserRoleViaModelsdk(unitID, name, moduleRoles, manageAllRoles)
}
func (b *MprBackend) AlterUserRoleModuleRoles(unitID model.ID, userRoleName string, add bool, moduleRoles []string) error {
	return b.alterUserRoleModuleRolesViaModelsdk(unitID, userRoleName, add, moduleRoles)
}
func (b *MprBackend) RemoveUserRole(unitID model.ID, name string) error {
	return b.removeUserRoleViaModelsdk(unitID, name)
}
func (b *MprBackend) AddDemoUser(unitID model.ID, userName, password, entity string, userRoles []string) error {
	return b.addDemoUserViaModelsdk(unitID, userName, password, entity, userRoles)
}
func (b *MprBackend) RemoveDemoUser(unitID model.ID, userName string) error {
	return b.removeDemoUserViaModelsdk(unitID, userName)
}
func (b *MprBackend) SetPasswordPolicy(unitID model.ID, minLength *int32, requireDigit, requireMixedCase, requireSymbol *bool) error {
	return b.setPasswordPolicyViaModelsdk(unitID, minLength, requireDigit, requireMixedCase, requireSymbol)
}

func (b *MprBackend) GetModuleSecurityGen(moduleID model.ID) (*genSec.ModuleSecurity, error) {
	b.initSubBackends()
	return b.security.GetModuleSecurityGen(moduleID)
}
func (b *MprBackend) ListModuleSecurityGen() ([]*genSec.ModuleSecurity, error) {
	b.initSubBackends()
	return b.security.ListModuleSecurityGen()
}
func (b *MprBackend) AddModuleRole(unitID model.ID, roleName, description string) error {
	return b.addModuleRoleViaModelsdk(unitID, roleName, description)
}
func (b *MprBackend) RemoveModuleRole(unitID model.ID, roleName string) error {
	return b.removeModuleRoleViaModelsdk(unitID, roleName)
}
func (b *MprBackend) RemoveModuleRoleFromAllUserRoles(unitID model.ID, qualifiedRole string) (int, error) {
	return 0, b.removeModuleRoleFromAllUserRolesViaModelsdk(unitID, qualifiedRole)
}

func (b *MprBackend) UpdateAllowedRoles(unitID model.ID, roles []string) error {
	return b.updateAllowedRolesViaModelsdk(unitID, roles)
}
func (b *MprBackend) UpdatePublishedRestServiceRoles(unitID model.ID, roles []string) error {
	return b.updatePublishedRestServiceRolesViaModelsdk(unitID, roles)
}
func (b *MprBackend) RemoveFromAllowedRoles(unitID model.ID, roleName string) (bool, error) {
	return b.removeFromAllowedRolesViaModelsdk(unitID, roleName)
}
func (b *MprBackend) AddEntityAccessRule(params backend.EntityAccessRuleParams) error {
	return b.addEntityAccessRuleViaModelsdk(params.UnitID, params.EntityName, params.RoleNames, params.AllowCreate, params.AllowDelete, params.DefaultMemberAccess, params.XPathConstraint, params.MemberAccesses)
}
func (b *MprBackend) RemoveEntityAccessRule(unitID model.ID, entityName string, roleNames []string) (int, error) {
	return b.removeEntityAccessRuleViaModelsdk(unitID, entityName, roleNames)
}
func (b *MprBackend) RevokeEntityMemberAccess(unitID model.ID, entityName string, roleNames []string, revocation types.EntityAccessRevocation) (int, error) {
	return b.revokeEntityMemberAccessViaModelsdk(unitID, entityName, roleNames, revocation)
}
func (b *MprBackend) RemoveRoleFromAllEntities(unitID model.ID, roleName string) (int, error) {
	return b.removeRoleFromAllEntitiesViaModelsdk(unitID, roleName)
}
func (b *MprBackend) ReconcileMemberAccesses(unitID model.ID, moduleName string) ([]string, error) {
	changes, err := b.reconcileMemberAccessesViaModelsdk(unitID, moduleName)
	if err != nil {
		return nil, err
	}
	msgs := make([]string, len(changes))
	for i, ch := range changes {
		msgs[i] = ch.String()
	}
	return msgs, nil
}
