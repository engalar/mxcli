// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// addEntityAccessRuleViaModelsdk patches an entity access rule through the
// modelsdk write path. Bypasses sdk/mpr.Writer.updateUnit (1544 bug).
func (b *MprBackend) addEntityAccessRuleViaModelsdk(unitID model.ID, entityName string, roleNames []string,
	allowCreate, allowDelete bool, defaultMemberAccess, xpathConstraint string,
	memberAccesses []mpr.EntityMemberAccess) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		return fmt.Errorf("read unit: %w", err)
	}
	patched, err := b.writer.PatchAddEntityAccessRule(rawBytes, entityName, roleNames, allowCreate, allowDelete, defaultMemberAccess, xpathConstraint, memberAccesses)
	if err != nil {
		return err
	}
	return b.writeUnitContents(unitID, patched)
}

func (b *MprBackend) removeEntityAccessRuleViaModelsdk(unitID model.ID, entityName string, roleNames []string) (int, error) {
	if b.msdkWriter == nil {
		return 0, fmt.Errorf("modelsdk writer not initialized")
	}
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		return 0, fmt.Errorf("read unit: %w", err)
	}
	patched, modified, err := b.writer.PatchRemoveEntityAccessRule(rawBytes, entityName, roleNames)
	if err != nil {
		return 0, err
	}
	if err := b.writeUnitContents(unitID, patched); err != nil {
		return 0, err
	}
	return modified, nil
}

func (b *MprBackend) revokeEntityMemberAccessViaModelsdk(unitID model.ID, entityName string, roleNames []string, revocation mpr.EntityAccessRevocation) (int, error) {
	if b.msdkWriter == nil {
		return 0, fmt.Errorf("modelsdk writer not initialized")
	}
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		return 0, fmt.Errorf("read unit: %w", err)
	}
	patched, modified, err := b.writer.PatchRevokeEntityMemberAccess(rawBytes, entityName, roleNames, revocation)
	if err != nil {
		return 0, err
	}
	if err := b.writeUnitContents(unitID, patched); err != nil {
		return 0, err
	}
	return modified, nil
}

func (b *MprBackend) removeRoleFromAllEntitiesViaModelsdk(unitID model.ID, roleName string) (int, error) {
	if b.msdkWriter == nil {
		return 0, fmt.Errorf("modelsdk writer not initialized")
	}
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		return 0, fmt.Errorf("read unit: %w", err)
	}
	patched, modified, err := b.writer.PatchRemoveRoleFromAllEntities(rawBytes, roleName)
	if err != nil {
		return 0, err
	}
	if err := b.writeUnitContents(unitID, patched); err != nil {
		return 0, err
	}
	return modified, nil
}

func (b *MprBackend) reconcileMemberAccessesViaModelsdk(unitID model.ID, moduleName string) (int, error) {
	if b.msdkWriter == nil {
		return 0, fmt.Errorf("modelsdk writer not initialized")
	}
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		return 0, fmt.Errorf("read unit: %w", err)
	}
	patched, modified, err := b.writer.PatchReconcileMemberAccesses(rawBytes, moduleName)
	if err != nil {
		return 0, err
	}
	if err := b.writeUnitContents(unitID, patched); err != nil {
		return 0, err
	}
	return modified, nil
}
