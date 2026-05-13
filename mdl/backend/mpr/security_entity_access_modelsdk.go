// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// writeUnitContents commits already-serialized BSON bytes for a unit through a
// modelsdk WriteTransaction. Mirrors the tail of msdkWrite (Begin → WriteUnit
// → Commit → InvalidateCache) and avoids sdk/mpr's updateTransactionID() which
// fails with SQLITE_READONLY_DBMOVED 1544 on hard-linked MPR files. Used by
// the entity-access Patch* helpers, whose encode logic stays in sdk/mpr to
// preserve the AllowedModuleRoles BSON key (gen AccessRule still binds to the
// older "ModuleRoles" key — gen-native migration deferred to a separate plan).
func (b *MprBackend) writeUnitContents(unitID model.ID, contents []byte) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	wtx, err := b.msdkWriter.BeginWriteTransaction()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := wtx.WriteUnit(string(unitID), contents); err != nil {
		_ = wtx.Rollback()
		return fmt.Errorf("write unit: %w", err)
	}
	if err := wtx.Commit(); err != nil {
		return err
	}
	b.reader.InvalidateCache()
	return nil
}

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
