// SPDX-License-Identifier: Apache-2.0

package executor

// warnEntityReferences is a no-op — catalog SQLite system has been replaced by MXGraph.
// Use SHOW REFERENCES TO for cross-element reference checking.
func warnEntityReferences(_ *ExecContext, _ string) {
	// Reference checking is available via MXGraph graphcatalog.TraversalReader.References().
	// This function previously queried the catalog's refs table.
}
