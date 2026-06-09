// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"
)

// warnEntityReferences prints a warning if the entity is referenced by other elements.
// Uses the catalog if available; silently skips if catalog is not built.
func warnEntityReferences(ctx *ExecContext, entityName string) {
	if ctx.Catalog == nil || !ctx.Catalog.IsBuilt() {
		return
	}

	query := fmt.Sprintf(
		"select SourceType, SourceName, RefKind from refs where TargetName = '%s'",
		strings.ReplaceAll(entityName, "'", "''"),
	)
	result, err := ctx.Catalog.Query(query)
	if err != nil || result.Count == 0 {
		return
	}

	fmt.Fprintf(ctx.Output, "warning: %s is referenced by %d element(s):\n", entityName, result.Count)
	for _, row := range result.Rows {
		sourceType, _ := row[0].(string)
		sourceName, _ := row[1].(string)
		refKind, _ := row[2].(string)
		fmt.Fprintf(ctx.Output, "  - %s %s (%s)\n", sourceType, sourceName, refKind)
	}
}
