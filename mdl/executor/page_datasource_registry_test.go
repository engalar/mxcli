// SPDX-License-Identifier: Apache-2.0
package executor

import (
	"testing"
)

// TestDataSourceBuilderRegistryCoverage 确认所有 DataSourceV3.Type 值都有注册 handler。
func TestDataSourceBuilderRegistryCoverage(t *testing.T) {
	knownTypes := []string{
		"parameter", "database", "microflow", "nanoflow", "association", "selection",
	}
	for _, typ := range knownTypes {
		if _, ok := dataSourceBuilders[typ]; !ok {
			t.Errorf("datasource type %q has no handler in dataSourceBuilders — add it to page_datasource_registry.go", typ)
		}
	}
}
