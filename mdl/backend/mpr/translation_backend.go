// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// SetEnumerationTranslation loads the enumeration unit, sets the langCode
// translation on the caption of the value named valueName, and saves the unit.
func (b *MprBackend) SetEnumerationTranslation(enumQN, valueName, langCode, text string) error {
	info, err := b.GetRawUnitByName("enumeration", enumQN)
	if err != nil {
		return fmt.Errorf("locate enumeration %q: %w", enumQN, err)
	}
	if info == nil {
		return mdlerrors.NewValidationf("enumeration %q not found", enumQN)
	}

	rawBytes, err := b.msdkReader.GetRawUnitBytes(info.ID)
	if err != nil {
		return fmt.Errorf("read enumeration unit: %w", err)
	}
	var enumDoc bson.D
	if err := bson.Unmarshal(rawBytes, &enumDoc); err != nil {
		return fmt.Errorf("unmarshal enumeration BSON: %w", err)
	}

	if !setEnumValueTranslation(enumDoc, valueName, langCode, text) {
		return mdlerrors.NewValidationf(
			"enumeration value %q not found in %q", valueName, enumQN)
	}

	contents, err := bson.Marshal(enumDoc)
	if err != nil {
		return fmt.Errorf("marshal enumeration BSON: %w", err)
	}
	return b.writeUnitContents(model.ID(info.ID), contents)
}
