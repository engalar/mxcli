// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5a — gen-typed autocomplete providers.
//
// LSP / REPL autocomplete for microflows and nanoflows runs through the
// modelsdk/gen-native repos (ctx.Microflows / ctx.Nanoflows) rather
// than Backend.ListMicroflows. The legacy sdk-typed providers in
// autocomplete.go remain wired into the dispatcher until Stage 3.2.6
// rotates the call sites; this file is the parallel implementation.
//
// Other element kinds (entity, page, snippet, layout, java action,
// javascript action, OData service, REST service, business event,
// JSON structure, database connection) do not touch sdk/microflows
// and stay in autocomplete.go.

package executor

import "github.com/mendixlabs/mxcli/model"

// getMicroflowNamesACGen returns qualified microflow names, optionally
// filtered by module. Mirrors getMicroflowNamesAC but consumes gen
// types via ctx.Microflows + the SQL-backed container chain.
func getMicroflowNamesACGen(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	mfs, err := listMicroflowsGen(ctx)
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(mf.ID()))
		if modName == "" {
			continue
		}
		if moduleFilter == "" || modName == moduleFilter {
			names = append(names, modName+"."+mf.Name())
		}
	}
	return names
}

// getNanoflowNamesACGen returns qualified nanoflow names, optionally
// filtered by module. Stage 3.2.5a adds this for parity even though
// the legacy autocomplete.go does not yet expose a nanoflow-specific
// AC entry — the dispatcher in 3.2.6 may want it for symmetry.
func getNanoflowNamesACGen(ctx *ExecContext, moduleFilter string) []string {
	if !ctx.Connected() {
		return nil
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	nfs, err := listNanoflowsGen(ctx)
	if err != nil {
		return nil
	}
	names := make([]string, 0)
	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(nf.ID()))
		if modName == "" {
			continue
		}
		if moduleFilter == "" || modName == moduleFilter {
			names = append(names, modName+"."+nf.Name())
		}
	}
	return names
}
