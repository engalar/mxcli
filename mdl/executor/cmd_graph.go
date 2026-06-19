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
	designdprops "github.com/mendixlabs/mxcli/internal/mxgraph/adapter/designdprops"
	themescss "github.com/mendixlabs/mxcli/internal/mxgraph/adapter/themescss"
	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/modelsdk"
)

// buildGraph constructs the in-memory project graph from the connected project,
// registering all five domain adapters, and installs it as ctx.Graph.
// It also persists a gob snapshot + delta log to <projectDir>/.mxcli/ so a
// later session can reload without a full rebuild.
//
// The graph is built from a fresh read-only modelsdk.Model opened from MprPath
// (matching cmd/mxcli/serve.go's buildProjectGraph). This avoids coupling the
// graph adapters, which consume *modelsdk.Model, to the executor's write-path
// MprBackend, which exposes only a modelsdkmpr.Reader.
func buildGraph(ctx *ExecContext) error {
	if ctx.MprPath == "" {
		return mdlerrors.NewNotConnected()
	}

	projectDir := filepath.Dir(ctx.MprPath)
	snapPath := graphcatalog.SnapshotPath(projectDir)
	deltaPath := graphcatalog.DeltaPath(projectDir)

	// Fast path: restore from cached snapshot + delta log.
	if g, err := mxgraph.RestoreFromSnapshot(snapPath, deltaPath); err != nil {
		return mdlerrors.NewBackend("restore graph cache", err)
	} else if g != nil {
		mgr := mxgraph.NewIndexManagerFromGraph(g)
		pg := graphcatalog.NewProjectGraph(mgr)
		ctx.Graph = pg
		if ctx.SyncGraph != nil {
			ctx.SyncGraph(pg)
		}
		if !ctx.Quiet {
			fmt.Fprintf(ctx.Output, "Graph restored: %d nodes, %d edges (from cache)\n",
				len(g.AllNodes()), len(g.AllEdges()))
		}
		return nil
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
	mgr.RegisterAdapter(&mpradapter.WidgetAdapter{ProjectDir: projectDir})
	mgr.RegisterAdapter(&themescss.ThemeScssAdapter{ProjectDir: projectDir})
	mgr.RegisterAdapter(&designdprops.DesignPropertyAdapter{ProjectDir: projectDir})
	mgr.RegisterAdapter(&mpradapter.WidgetInstanceAdapter{Source: &mpradapter.ModelsdkUnitSource{Model: m}})
	mgr.RegisterAdapter(&mpradapter.AccessRuleAdapter{Model: m})
	mgr.RegisterAdapter(&mpradapter.DocumentGrantAdapter{Model: m})
	mgr.RegisterAdapter(&mpradapter.PageRefAdapter{Model: m})
	mgr.RegisterAdapter(&mpradapter.NavigationAdapter{
		Source: &mpradapter.ModelsdkUnitSource{Model: m},
	})
	mgr.RegisterAdapter(&mpradapter.DataContainerAdapter{
		Source: &mpradapter.ModelsdkUnitSource{Model: m},
		Model:  m,
	})

	// Open delta log for event persistence during build.
	if err := os.MkdirAll(filepath.Dir(deltaPath), 0700); err != nil {
		return mdlerrors.NewBackend("create cache directory", err)
	}
	deltaLog, err := mxgraph.OpenDeltaLog(deltaPath)
	if err != nil {
		return mdlerrors.NewBackend("open delta log", err)
	}
	defer deltaLog.Close()

	sink := mxgraph.NewLoggingSink(mgr, deltaLog)

	buildCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := mgr.BuildAll(buildCtx, sink); err != nil {
		return mdlerrors.NewBackend("build graph", err)
	}

	pg := graphcatalog.NewProjectGraph(mgr)
	ctx.Graph = pg
	if ctx.SyncGraph != nil {
		ctx.SyncGraph(pg)
	}

	// Persist snapshot and reset delta log so future restores are fast.
	if data, err := pg.MarshalSnapshot(); err == nil {
		if mkErr := os.MkdirAll(filepath.Dir(snapPath), 0700); mkErr == nil {
			_ = os.WriteFile(snapPath, data, 0600)
		}
	}
	// After snapshot is saved the delta log is no longer needed; reset it.
	_ = deltaLog.Reset()

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
