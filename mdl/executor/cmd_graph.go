// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	designdprops "github.com/mendixlabs/mxcli/internal/mxgraph/adapter/designdprops"
	mpradapter "github.com/mendixlabs/mxcli/internal/mxgraph/adapter/mpr"
	themescss "github.com/mendixlabs/mxcli/internal/mxgraph/adapter/themescss"
	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/modelsdk"
)

// BuildGraphAtPath constructs the in-memory project graph from projectPath,
// returning a *graphcatalog.ProjectGraph ready for queries.
// It persists a gob snapshot + delta log to <projectDir>/.mxcli/ so a
// later session can reload without a full rebuild.
//
// The graph is built from a fresh read-only modelsdk.Model opened from the path,
// avoiding coupling to the write-path backend.
func BuildGraphAtPath(projectPath string) (*graphcatalog.ProjectGraph, error) {
	projectDir := filepath.Dir(projectPath)
	snapPath := graphcatalog.SnapshotPath(projectDir)
	deltaPath := graphcatalog.DeltaPath(projectDir)

	if g, err := mxgraph.RestoreFromSnapshot(snapPath, deltaPath); err == nil && g != nil {
		mgr := mxgraph.NewIndexManagerFromGraph(g)
		return graphcatalog.NewProjectGraph(mgr), nil
	}

	m, err := modelsdk.Open(projectPath)
	if err != nil {
		return nil, fmt.Errorf("open project for graph: %w", err)
	}
	defer m.Close()

	pg, err := buildGraphFromModel(m, projectDir, snapPath, deltaPath)
	if err != nil {
		return nil, err
	}
	return pg, nil
}

// buildGraphDeps constructs the in-memory project graph using HandlerDeps.
func buildGraphDeps(deps *HandlerDeps) error {
	if deps.MprPath == "" {
		return mdlerrors.NewNotConnected()
	}

	projectDir := filepath.Dir(deps.MprPath)
	snapPath := graphcatalog.SnapshotPath(projectDir)
	deltaPath := graphcatalog.DeltaPath(projectDir)

	// Fast path: restore from cached snapshot + delta log.
	if g, err := mxgraph.RestoreFromSnapshot(snapPath, deltaPath); err != nil {
		return mdlerrors.NewBackend("restore graph cache", err)
	} else if g != nil {
		mgr := mxgraph.NewIndexManagerFromGraph(g)
		pg := graphcatalog.NewProjectGraph(mgr)
		deps.Graph = pg
		if deps.SyncGraph != nil {
			deps.SyncGraph(pg)
		}
		if !deps.Quiet {
			fmt.Fprintf(deps.Output, "Graph restored: %d nodes, %d edges (from cache)\n",
				len(g.AllNodes()), len(g.AllEdges()))
		}
		return nil
	}

	m, err := modelsdk.Open(deps.MprPath)
	if err != nil {
		return mdlerrors.NewBackend("open project for graph build", err)
	}
	defer m.Close()

	pg, err := buildGraphFromModel(m, projectDir, snapPath, deltaPath)
	if err != nil {
		return err
	}

	deps.Graph = pg
	if deps.SyncGraph != nil {
		deps.SyncGraph(pg)
	}

	if !deps.Quiet {
		g := pg.MxGraph()
		fmt.Fprintf(deps.Output, "Graph built: %d nodes, %d edges\n",
			len(g.AllNodes()), len(g.AllEdges()))
	}
	return nil
}

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
	return buildGraphDeps(ctx.Deps)
}

// buildGraphFromModel constructs a ProjectGraph from an opened modelsdk.Model.
// Shared between buildGraph (executor path) and BuildGraphAtPath (standalone).
func buildGraphFromModel(m *modelsdk.Model, projectDir, snapPath, deltaPath string) (*graphcatalog.ProjectGraph, error) {
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
	docCache := mpradapter.NewBsonDocCache()

	mgr.RegisterAdapter(&mpradapter.WidgetInstanceAdapter{
		Source:   &mpradapter.ModelsdkUnitSource{Model: m},
		DocCache: docCache,
	})
	mgr.RegisterAdapter(&mpradapter.AccessRuleAdapter{Model: m})
	mgr.RegisterAdapter(&mpradapter.DocumentGrantAdapter{Model: m})
	mgr.RegisterAdapter(&mpradapter.PageRefAdapter{
		Model:    m,
		DocCache: docCache,
	})
	mgr.RegisterAdapter(&mpradapter.NavigationAdapter{
		Source: &mpradapter.ModelsdkUnitSource{Model: m},
	})
	mgr.RegisterAdapter(&mpradapter.DataContainerAdapter{
		Source:   &mpradapter.ModelsdkUnitSource{Model: m},
		Model:    m,
		DocCache: docCache,
	})

	os.MkdirAll(filepath.Dir(deltaPath), 0700)
	deltaLog, err := mxgraph.OpenDeltaLog(deltaPath)
	if err != nil {
		return nil, fmt.Errorf("open delta log: %w", err)
	}
	defer deltaLog.Close()

	sink := mxgraph.NewLoggingSink(mgr, deltaLog)

	buildCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := mgr.BuildAll(buildCtx, sink); err != nil {
		return nil, fmt.Errorf("build graph: %w", err)
	}

	pg := graphcatalog.NewProjectGraph(mgr)

	if data, err := pg.MarshalSnapshot(); err == nil {
		if mkErr := os.MkdirAll(filepath.Dir(snapPath), 0700); mkErr == nil {
			os.WriteFile(snapPath, data, 0600)
		}
	}
	deltaLog.Reset()

	return pg, nil
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
