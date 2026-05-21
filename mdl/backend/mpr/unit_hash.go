// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"crypto/sha256"
	"encoding/hex"
)

// mprUnitSHA256Hex returns the SHA-256 hex digest of data.
// Used by MprUnitPersistence.BatchHash for @cache: marker computation.
func mprUnitSHA256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
