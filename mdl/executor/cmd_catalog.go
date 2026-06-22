// SPDX-License-Identifier: Apache-2.0

// Package executor - Catalog commands (removed, replaced by MXGraph).
package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

// execShowCatalogTables was removed with the catalog SQLite system.
func execShowCatalogTables(ctx *ExecContext) error {
	fmt.Fprintf(ctx.Output, "Catalog SQLite system has been replaced by MXGraph.\n")
	fmt.Fprintf(ctx.Output, "Use SHOW MODULES / SHOW ENTITIES / SHOW PAGES / etc. directly.\n")
	return nil
}

// execCatalogQuery was removed with the catalog SQLite system.
func execCatalogQuery(ctx *ExecContext, query string) error {
	return mdlerrors.NewUnsupported(
		"SELECT FROM CATALOG is no longer available. " +
			"The catalog SQLite system has been replaced by MXGraph. " +
			"Use SHOW commands (SHOW MODULES, SHOW ENTITIES, SHOW PAGES) directly.")
}

// execDescribeCatalogTable was removed with the catalog SQLite system.
func execDescribeCatalogTable(ctx *ExecContext, stmt *ast.DescribeCatalogTableStmt) error {
	return mdlerrors.NewUnsupported(
		"DESCRIBE CATALOG is no longer available. " +
			"The catalog SQLite system has been replaced by MXGraph.")
}

// execShowCatalogStatus was removed with the catalog SQLite system.
func execShowCatalogStatus(ctx *ExecContext) error {
	fmt.Fprintf(ctx.Output, "Catalog SQLite system has been replaced by MXGraph.\n")
	return nil
}

// execRefreshCatalogStmt handles REFRESH CATALOG.
// Catalog building is removed; the graph is built separately.
func execRefreshCatalogStmt(ctx *ExecContext, stmt *ast.RefreshCatalogStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	fmt.Fprintf(ctx.Output, "Catalog system has been replaced by MXGraph.\n")
	fmt.Fprintf(ctx.Output, "Index building is handled automatically.\n")
	return nil
}

// execSearch handles SEARCH 'keyword'.
// Searches project metadata via backend instead of catalog FTS5.
func execSearch(ctx *ExecContext, stmt *ast.SearchStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	return searchBackend(ctx, stmt.Query)
}

// searchBackend implements SEARCH by iterating backend data sources.
func searchBackend(ctx *ExecContext, keyword string) error {
	type match struct{ kind, name, location string }
	var matches []match
	upperKw := strings.ToUpper(keyword)

	// Search microflows
	if r := ctx.Microflows; r != nil {
		mfs, err := r.ListAll()
		if err == nil {
			for _, mf := range mfs {
				if mf == nil {
					continue
				}
				if strings.Contains(strings.ToUpper(mf.Name()), upperKw) {
					matches = append(matches, match{"microflow", mf.Name(), ""})
				}
			}
		}
	}

	// Search pages
	if r := ctx.Pages; r != nil {
		pages, err := r.ListAll()
		if err == nil {
			for _, p := range pages {
				if p == nil {
					continue
				}
				if strings.Contains(strings.ToUpper(p.Name()), upperKw) {
					matches = append(matches, match{"page", p.Name(), ""})
				}
			}
		}
	}

	// Search nanoflows
	if r := ctx.Nanoflows; r != nil {
		nfs, err := r.ListAll()
		if err == nil {
			for _, nf := range nfs {
				if nf == nil {
					continue
				}
				if strings.Contains(strings.ToUpper(nf.Name()), upperKw) {
					matches = append(matches, match{"nanoflow", nf.Name(), ""})
				}
			}
		}
	}

	if len(matches) == 0 {
		fmt.Fprintf(ctx.Output, "No results found for %q\n", keyword)
		return nil
	}

	fmt.Fprintf(ctx.Output, "Search results for %q:\n", keyword)
	fmt.Fprintln(ctx.Output, strings.Repeat("-", 80))
	for _, m := range matches {
		fmt.Fprintf(ctx.Output, "  %-12s %s\n", m.kind, m.name)
	}
	fmt.Fprintf(ctx.Output, "\n%d match(es) found\n", len(matches))
	return nil
}

// ensureCatalog is a no-op — catalog SQLite system has been removed.
// All data queries should use MXGraph or backend directly.
func ensureCatalog(_ *ExecContext, _ bool) error { return nil }
