// SPDX-License-Identifier: Apache-2.0

package exprcheck

import "testing"

func TestInterfaces_Compile(t *testing.T) {
	var (
		_ Parser
		_ SlotResolver
		_ CatalogReader
		_ HintSink
	)
	_ = t
}
