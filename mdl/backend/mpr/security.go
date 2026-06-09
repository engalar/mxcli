// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	msdksecurity "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// setSecurityLevelViaModelsdk writes the SecurityLevel field using the modelsdk
// decode→mutate→encode roundtrip. This avoids the sdk/mpr updateTransactionID()
// call that triggers SQLITE_READONLY_DBMOVED (1544) on hard-linked MPR files.
func (b *MprBackend) setSecurityLevelViaModelsdk(unitID model.ID, level string) error {
	if b.msdkWriter == nil {
		return fmt.Errorf("modelsdk writer not initialized")
	}

	rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
	if err != nil {
		return fmt.Errorf("read security unit: %w", err)
	}

	elem, err := codec.NewDecoder(codec.DefaultRegistry).Decode(bson.Raw(rawBytes))
	if err != nil {
		return fmt.Errorf("decode security unit: %w", err)
	}

	ps, ok := elem.(*msdksecurity.ProjectSecurity)
	if !ok {
		return fmt.Errorf("unexpected type %T for security unit (want *security.ProjectSecurity)", elem)
	}

	ps.SetSecurityLevel(level)

	newBytes, err := (&codec.Encoder{}).Encode(ps)
	if err != nil {
		return fmt.Errorf("encode security unit: %w", err)
	}

	wtx, err := b.msdkWriter.BeginWriteTransaction()
	if err != nil {
		return fmt.Errorf("begin write transaction: %w", err)
	}
	if err := wtx.WriteUnit(string(unitID), newBytes); err != nil {
		_ = wtx.Rollback()
		return fmt.Errorf("write security unit: %w", err)
	}
	return wtx.Commit()
}
