// SPDX-License-Identifier: Apache-2.0
package executor

import (
	"testing"
)

// TestActionBuilderRegistryCoverage 确认所有已知 action type 都有注册 handler。
func TestActionBuilderRegistryCoverage(t *testing.T) {
	knownTypes := []string{
		"save", "cancel", "close", "delete", "create",
		"showPage", "microflow", "nanoflow",
		"openLink", "signOut", "completeTask",
	}
	handlers := ActionBuilders()
	for _, typ := range knownTypes {
		if _, ok := handlers[typ]; !ok {
			t.Errorf("action type %q has no handler in actionBuilders — add it to page_action_registry.go", typ)
		}
	}
}
