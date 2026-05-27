// SPDX-License-Identifier: Apache-2.0

package dtsparser

import (
	"os"
	"testing"
)

// findMendixModelSDKGenDir returns the path to the mendixmodelsdk src/gen
// directory from node_modules/. Calls t.Skip if not found.
func findMendixModelSDKGenDir(t *testing.T) string {
	t.Helper()
	p := "../../../node_modules/mendixmodelsdk/src/gen"
	if _, err := os.Stat(p); err != nil {
		t.Skip("mendixmodelsdk not available — run: npm install")
	}
	return p
}
