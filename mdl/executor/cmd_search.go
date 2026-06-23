// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
)

// ensureGraphForShowFn checks that the in-memory project graph exists with HandlerDeps.
func ensureGraphForShowFn(ctx context.Context, deps *HandlerDeps) error {
	if deps.Graph != nil {
		return nil
	}
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	return mdlerrors.NewBackend("graph", fmt.Errorf("project graph not built — run REFRESH CATALOG FULL first"))
}

// ensureGraph builds the in-memory project graph if it has not been built yet.
func ensureGraph(ctx *ExecContext) error {
	if ctx.Graph != nil {
		return nil
	}
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}
	return buildGraph(ctx)
}

// ExecShowCallersFn handles SHOW CALLERS OF with HandlerDeps.
func ExecShowCallersFn(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
	if s.Name == nil {
		return mdlerrors.NewValidation("target name required for show callers")
	}
	if err := ensureGraphForShowFn(ctx, deps); err != nil {
		return err
	}

	targetName := s.Name.String()
	fmt.Fprintf(deps.Output, "\nCallers of %s", targetName)
	if s.Transitive {
		fmt.Fprintln(deps.Output, " (transitive)")
	} else {
		fmt.Fprintln(deps.Output, "")
	}

	callers := deps.Graph.Callers(targetName, s.Transitive)
	if len(callers) == 0 {
		fmt.Fprintln(deps.Output, "(no callers found)")
		return nil
	}
	sort.SliceStable(callers, func(i, j int) bool {
		if callers[i].Depth != callers[j].Depth {
			return callers[i].Depth < callers[j].Depth
		}
		return callers[i].Caller < callers[j].Caller
	})

	fmt.Fprintf(deps.Output, "Found %d caller(s)\n", len(callers))
	w := tabwriter.NewWriter(deps.Output, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CALLER\tDEPTH")
	for _, c := range callers {
		fmt.Fprintf(w, "%s\t%d\n", c.Caller, c.Depth)
	}
	return w.Flush()
}

// execShowCallers handles SHOW CALLERS OF Module.Microflow [TRANSITIVE].

// ExecShowCalleesFn handles SHOW CALLEES OF with HandlerDeps.
func ExecShowCalleesFn(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
	if s.Name == nil {
		return mdlerrors.NewValidation("target name required for show callees")
	}
	if err := ensureGraphForShowFn(ctx, deps); err != nil {
		return err
	}

	sourceName := s.Name.String()
	fmt.Fprintf(deps.Output, "\nCallees of %s", sourceName)
	if s.Transitive {
		fmt.Fprintln(deps.Output, " (transitive)")
	} else {
		fmt.Fprintln(deps.Output, "")
	}

	callees := deps.Graph.Callees(sourceName, s.Transitive)
	if len(callees) == 0 {
		fmt.Fprintln(deps.Output, "(no callees found)")
		return nil
	}
	sort.SliceStable(callees, func(i, j int) bool {
		if callees[i].Depth != callees[j].Depth {
			return callees[i].Depth < callees[j].Depth
		}
		return callees[i].Callee < callees[j].Callee
	})

	fmt.Fprintf(deps.Output, "Found %d callee(s)\n", len(callees))
	w := tabwriter.NewWriter(deps.Output, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CALLEE\tDEPTH")
	for _, c := range callees {
		fmt.Fprintf(w, "%s\t%d\n", c.Callee, c.Depth)
	}
	return w.Flush()
}

// execShowCallees handles SHOW CALLEES OF Module.Microflow [TRANSITIVE].

// ExecShowReferencesFn handles SHOW REFERENCES TO with HandlerDeps.
func ExecShowReferencesFn(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
	if s.Name == nil {
		return mdlerrors.NewValidation("target name required for show references")
	}
	if err := ensureGraphForShowFn(ctx, deps); err != nil {
		return err
	}

	targetName := s.Name.String()
	fmt.Fprintf(deps.Output, "\nReferences to %s\n", targetName)

	refs := deps.Graph.Impact(targetName)
	if len(refs) == 0 {
		fmt.Fprintln(deps.Output, "(no references found)")
		return nil
	}
	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].RefKind != refs[j].RefKind {
			return refs[i].RefKind < refs[j].RefKind
		}
		return refs[i].Source < refs[j].Source
	})

	fmt.Fprintf(deps.Output, "Found %d reference(s)\n", len(refs))
	w := tabwriter.NewWriter(deps.Output, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SOURCE\tREF KIND")
	for _, r := range refs {
		fmt.Fprintf(w, "%s\t%s\n", r.Source, r.RefKind)
	}
	return w.Flush()
}

// execShowReferences handles SHOW REFERENCES TO Module.Entity.

// ExecShowImpactFn handles SHOW IMPACT OF with HandlerDeps.
func ExecShowImpactFn(ctx context.Context, s *ast.ShowStmt, deps *HandlerDeps) error {
	if s.Name == nil {
		return mdlerrors.NewValidation("target name required for show impact")
	}
	if err := ensureGraphForShowFn(ctx, deps); err != nil {
		return err
	}

	targetName := s.Name.String()
	fmt.Fprintf(deps.Output, "\nImpact analysis for %s\n", targetName)

	refs := deps.Graph.Impact(targetName)
	if len(refs) == 0 {
		fmt.Fprintln(deps.Output, "(no impact - element is not referenced)")
		return nil
	}

	kindCounts := make(map[string]int)
	for _, r := range refs {
		kindCounts[r.RefKind]++
	}
	kinds := make([]string, 0, len(kindCounts))
	for k := range kindCounts {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)

	fmt.Fprintf(deps.Output, "\nSummary:\n")
	for _, k := range kinds {
		fmt.Fprintf(deps.Output, "  %s: %d\n", k, kindCounts[k])
	}
	fmt.Fprintln(deps.Output)

	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].RefKind != refs[j].RefKind {
			return refs[i].RefKind < refs[j].RefKind
		}
		return refs[i].Source < refs[j].Source
	})

	fmt.Fprintf(deps.Output, "Found %d affected element(s)\n", len(refs))
	w := tabwriter.NewWriter(deps.Output, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SOURCE\tREF KIND")
	for _, r := range refs {
		fmt.Fprintf(w, "%s\t%s\n", r.Source, r.RefKind)
	}
	return w.Flush()
}

// execShowImpact handles SHOW IMPACT OF Module.Entity.
