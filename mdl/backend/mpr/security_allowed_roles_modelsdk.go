// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"bytes"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
)

func (b *MprBackend) updateAllowedRolesViaModelsdk(unitID model.ID, roles []string) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		return fmt.Errorf("read unit: %w", err)
	}
	rolesArr := make(bson.A, 0, len(roles)+1)
	rolesArr = append(rolesArr, int32(3))
	for _, r := range roles {
		rolesArr = append(rolesArr, r)
	}
	newBytes, err := codec.PatchBSONField(rawBytes, "AllowedRoles", rolesArr)
	if err != nil {
		return fmt.Errorf("patch AllowedRoles: %w", err)
	}
	return b.msdkWriteRaw(unitID, newBytes)
}

func (b *MprBackend) updatePublishedRestServiceRolesViaModelsdk(unitID model.ID, roles []string) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		return fmt.Errorf("read unit: %w", err)
	}
	rolesArr := make(bson.A, 0, len(roles)+1)
	rolesArr = append(rolesArr, int32(3))
	for _, r := range roles {
		rolesArr = append(rolesArr, r)
	}
	newBytes, err := codec.PatchBSONField(rawBytes, "AllowedRoles", rolesArr)
	if err != nil {
		return fmt.Errorf("patch AllowedRoles: %w", err)
	}
	return b.msdkWriteRaw(unitID, newBytes)
}

func (b *MprBackend) removeFromAllowedRolesViaModelsdk(unitID model.ID, roleName string) (bool, error) {
	if b.msdkWriter == nil {
		return false, fmt.Errorf("modelsdk writer not initialized")
	}
	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		return false, fmt.Errorf("read unit: %w", err)
	}
	newBytes, err := codec.PatchBSONArrayRemove(rawBytes, "AllowedRoles", roleName)
	if err != nil {
		return false, fmt.Errorf("remove from AllowedRoles: %w", err)
	}
	if bytes.Equal(rawBytes, newBytes) {
		return false, nil
	}
	if err := b.msdkWriteRaw(unitID, newBytes); err != nil {
		return false, err
	}
	return true, nil
}

// msdkWriteRaw writes pre-patched raw BSON bytes via WriteTransaction.
// After committing, it invalidates the sdk/mpr reader's unit cache so that
// consecutive writes via writeDomainModel always read fresh data (v2 unitCache
// stores metadata only, but invalidation ensures correctness after any unit
// structural change such as InsertUnit or cross-domain model operations).
func (b *MprBackend) msdkWriteRaw(unitID model.ID, contents []byte) error {
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
