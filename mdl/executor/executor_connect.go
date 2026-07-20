// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"errors"
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
)

func execConnect(ctx *ExecContext, s *ast.ConnectStmt) error {
	if ctx.Backend != nil && ctx.ConnectionManager.IsConnected() {
		if err := ctx.ConnectionManager.Disconnect(); err != nil {
			fmt.Fprintf(ctx.Output, "Warning: disconnect error: %v\n", err)
		}
	}

	if ctx.BackendFactory != nil {
		b := ctx.BackendFactory()
		if err := b.Connect(s.Path); err != nil {
			return mdlerrors.NewBackend("connect", err)
		}
		ctx.Backend = b.(backend.FullBackend)
		ctx.backendFactory = b.(backend.BackendFactory)
		ctx.initRoles()
	} else if ctx.Backend != nil {
		// Persistent backend (per-MPR daemon): Connect is a no-op on noOpConnectBackend.
		if err := ctx.ConnectionManager.Connect(s.Path); err != nil {
			return mdlerrors.NewBackend("connect", err)
		}
	} else {
		return mdlerrors.NewBackend("connect", errors.New("no backend factory configured"))
	}

	ctx.MprPath = s.Path
	ctx.Cache = newExecutorCache() // Initialize fresh cache

	// Build project graph and load into ctx.Graph.
	// The graph is built from a fresh read-only model, synced to the backend
	// via SetProjectGraph so that GetMxGraph() is available for warmup.
	if graph, buildErr := BuildGraphAtPath(s.Path); buildErr == nil && graph != nil {
		ctx.Graph = graph
		warmCacheFromGraph(ctx.Cache, graph)
		if bg, ok := ctx.Backend.(interface {
			SetProjectGraph(*graphcatalog.ProjectGraph)
		}); ok {
			bg.SetProjectGraph(graph)
		}
	}

	// Reset project-scoped caches — previous project's theme
	// registry is invalid for the new connection.
	ctx.ThemeRegistry = nil

	// Display connection info with version.
	// Written to StatusOutput (stderr by default) so it never pollutes stdout
	// when the caller redirects stdout to a file (e.g. > describe-snapshot.mdl).
	pv := ctx.ConnectionManager.ProjectVersion()
	if !ctx.Quiet {
		fmt.Fprintf(ctx.statusWriter(), "Connected to: %s (Mendix %s)\n", s.Path, pv.ProductVersion)
	}
	if ctx.Logger != nil {
		ctx.Logger.Connect(s.Path, pv.ProductVersion, pv.FormatVersion)
	}
	return nil
}

// reconnect closes the current connection and reopens it.
// This is needed when the project file has been modified externally.
func reconnect(ctx *ExecContext) error {
	if ctx.MprPath == "" {
		return mdlerrors.NewNotConnected()
	}

	// Close existing connection
	if ctx.Backend != nil && ctx.ConnectionManager.IsConnected() {
		if err := ctx.ConnectionManager.Disconnect(); err != nil {
			fmt.Fprintf(ctx.Output, "Warning: disconnect error: %v\n", err)
		}
	}

	// Reopen connection
	if ctx.BackendFactory != nil {
		b := ctx.BackendFactory()
		if err := b.Connect(ctx.MprPath); err != nil {
			return mdlerrors.NewBackend("reconnect", err)
		}
		ctx.Backend = b.(backend.FullBackend)
		ctx.backendFactory = b.(backend.BackendFactory)
		ctx.initRoles()
	} else if ctx.Backend != nil {
		// Persistent backend: Connect is a no-op on noOpConnectBackend.
		if err := ctx.ConnectionManager.Connect(ctx.MprPath); err != nil {
			return mdlerrors.NewBackend("reconnect", err)
		}
	} else {
		return mdlerrors.NewBackend("reconnect", fmt.Errorf("no backend factory configured"))
	}

	ctx.Cache = newExecutorCache() // Reset cache

	// Reset project-scoped caches — file may have changed externally.
	ctx.ThemeRegistry = nil

	return nil
}

func execDisconnect(ctx *ExecContext) error {
	if ctx.Backend == nil || !ctx.ConnectionManager.IsConnected() {
		fmt.Fprintln(ctx.Output, "Not connected")
		return nil
	}

	// Reconcile any pending security changes before closing
	if ctx.FinalizeFn != nil {
		if err := ctx.FinalizeFn(); err != nil {
			fmt.Fprintf(ctx.Output, "Warning: finalization error: %v\n", err)
		}
	}

	if err := ctx.ConnectionManager.Disconnect(); err != nil {
		fmt.Fprintf(ctx.Output, "Warning: disconnect error: %v\n", err)
	}
	fmt.Fprintf(ctx.Output, "Disconnected from: %s\n", ctx.MprPath)
	ctx.MprPath = ""
	ctx.Cache = nil
	ctx.Backend = nil

	return nil
}

func execStatus(ctx *ExecContext) error {
	if ctx.Backend == nil || !ctx.ConnectionManager.IsConnected() {
		fmt.Fprintln(ctx.Output, "Status: Not connected")
		return nil
	}

	pv := ctx.ConnectionManager.ProjectVersion()
	fmt.Fprintf(ctx.Output, "Status: Connected\n")
	fmt.Fprintf(ctx.Output, "Project: %s\n", ctx.MprPath)
	fmt.Fprintf(ctx.Output, "Mendix Version: %s\n", pv.ProductVersion)
	fmt.Fprintf(ctx.Output, "MPR Format: v%d\n", pv.FormatVersion)

	// Show module count
	modules, err := ctx.ModuleLister.ListModules()
	if err == nil {
		fmt.Fprintf(ctx.Output, "Modules: %d\n", len(modules))
	}

	return nil
}
