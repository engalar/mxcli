// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	mpradapter "github.com/mendixlabs/mxcli/internal/mxgraph/adapter/mpr"
	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/modelsdk"
)

// buildGraph constructs the in-memory project graph from the connected project,
// registering all five domain adapters, and installs it as ctx.Graph.
// It also persists a gob snapshot to <projectDir>/.mxcli/graph.gob so a later
// session can reload without rebuilding.
//
// The graph is built from a fresh read-only modelsdk.Model opened from MprPath
// (matching cmd/mxcli/serve.go's buildProjectGraph). This avoids coupling the
// graph adapters, which consume *modelsdk.Model, to the executor's write-path
// MprBackend, which exposes only a modelsdkmpr.Reader.
func buildGraph(ctx *ExecContext) error {
	if ctx.MprPath == "" {
		return mdlerrors.NewNotConnected()
	}

	m, err := modelsdk.Open(ctx.MprPath)
	if err != nil {
		return mdlerrors.NewBackend("open project for graph build", err)
	}
	defer m.Close()

	mgr := mxgraph.NewIndexManager()
	mgr.RegisterAdapter(&mpradapter.DomainModelAdapter{Model: m})
	mgr.RegisterAdapter(&mpradapter.MicroflowAdapter{Model: m})
	mgr.RegisterAdapter(&mpradapter.PageAdapter{Model: m})
	mgr.RegisterAdapter(&mpradapter.SecurityAdapter{Model: m})
	mgr.RegisterAdapter(&mpradapter.EnumerationAdapter{Model: m})
	mgr.RegisterAdapter(&mpradapter.WorkflowAdapter{Model: m})
	mgr.RegisterAdapter(&mpradapter.WidgetAdapter{ProjectDir: filepath.Dir(ctx.MprPath)})

	buildCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := mgr.BuildAll(buildCtx); err != nil {
		return mdlerrors.NewBackend("build graph", err)
	}

	pg := graphcatalog.NewProjectGraph(mgr)
	ctx.Graph = pg
	if ctx.SyncGraph != nil {
		ctx.SyncGraph(pg)
	}

	// Persist a gob snapshot next to the project (best-effort).
	if data, err := pg.MarshalSnapshot(); err == nil {
		snapPath := graphcatalog.SnapshotPath(filepath.Dir(ctx.MprPath))
		if mkErr := os.MkdirAll(filepath.Dir(snapPath), 0700); mkErr == nil {
			_ = os.WriteFile(snapPath, data, 0600)
		}
	}

	if !ctx.Quiet {
		g := mgr.Query()
		fmt.Fprintf(ctx.Output, "Graph built: %d nodes, %d edges\n",
			len(g.AllNodes()), len(g.AllEdges()))
	}
	return nil
}

// execRefreshGraphStmt handles REFRESH GRAPH [FULL] — rebuilds the in-memory
// project graph. The Full flag is accepted for compatibility with the existing
// RefreshCatalogStmt AST but the graph is always built completely.
func execRefreshGraphStmt(ctx *ExecContext, _ *ast.RefreshCatalogStmt) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	return buildGraph(ctx)
}
