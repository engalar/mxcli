// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/gen/security"
	"go.mongodb.org/mongo-driver/bson"
)

// setSecurityLevelViaModelsdk patches the SecurityLevel field on the
// Security$ProjectSecurity unit using the modelsdk decode→mutate→encode→write
// pipeline instead of the legacy sdk/mpr BSON-patch path.
func (b *MprBackend) setSecurityLevelViaModelsdk(unitID model.ID, level string) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}

	raw, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		return fmt.Errorf("read security unit %s: %w", unitID, err)
	}

	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, err := dec.Decode(bson.Raw(raw))
	if err != nil {
		return fmt.Errorf("decode security unit %s: %w", unitID, err)
	}

	ps, ok := elem.(*security.ProjectSecurity)
	if !ok {
		return fmt.Errorf("unit %s is not Security$ProjectSecurity (got %s)", unitID, elem.TypeName())
	}

	ps.SetSecurityLevel(level)

	enc := &codec.Encoder{}
	encoded, err := enc.Encode(ps)
	if err != nil {
		return fmt.Errorf("encode security unit %s: %w", unitID, err)
	}

	if err := b.msdkWriter.UpdateRawUnit(string(unitID), encoded); err != nil {
		return fmt.Errorf("write security unit %s: %w", unitID, err)
	}
	return nil
}
