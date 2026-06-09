// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.6.3a: gen-typed SHOW MICROFLOWS / SHOW NANOFLOWS table
// emitters. These replace the deleted legacy `listMicroflows` /
// `listNanoflows` (in cmd_microflows_show.go) — they consume the
// modelsdk repos via ctx.Microflows / ctx.Nanoflows and walk the
// gen ObjectCollection for activity counts and McCabe complexity.
//
// Output schema is unchanged (`Qualified Name`, `Module`, `Name`,
// `Excluded`, `Folder`, `Params`, `Actions`, `McCabe`, `Returns`) so
// downstream tooling (LSP, REPL `show microflows`, dashboards) does
// not see a behavioral change.

package executor

import (
	"fmt"
	"sort"
	"strings"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// listMicroflows handles SHOW MICROFLOWS.
func listMicroflows(ctx *ExecContext, moduleName string) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}
	if moduleName != "" {
		if _, err := findModule(ctx, moduleName); err != nil {
			return err
		}
	}

	mfs, err := listMicroflowsGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		excluded      bool
		folderPath    string
		params        int
		activities    int
		complexity    int
		returnType    string
	}
	var rows []row

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		containerID, _ := ctx.Microflows.GetContainerUUID(model.ID(mf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if moduleName != "" && modName != moduleName {
			continue
		}
		qualifiedName := modName + "." + mf.Name()
		folderPath := h.BuildFolderPath(containerID)
		returnType := strings.TrimSpace(mf.ReturnType())
		oc, _ := mf.ObjectCollection().(*genMf.MicroflowObjectCollection)
		params := genFlowParameterElems(mf.ObjectCollection())
		activities := countGenFlowActivities(oc)
		complexity := calculateGenFlowComplexity(oc)
		rows = append(rows, row{
			qualifiedName: qualifiedName,
			module:        modName,
			name:          mf.Name(),
			excluded:      mf.Excluded(),
			folderPath:    folderPath,
			params:        len(params),
			activities:    activities,
			complexity:    complexity,
			returnType:    returnType,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Excluded", "Folder", "Params", "Actions", "McCabe", "Returns"},
		Summary: fmt.Sprintf("(%d microflows)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.excluded, r.folderPath, r.params, r.activities, r.complexity, r.returnType})
	}
	return writeResult(ctx, result)
}

// listNanoflows handles SHOW NANOFLOWS.
func listNanoflows(ctx *ExecContext, moduleName string) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}
	if moduleName != "" {
		if _, err := findModule(ctx, moduleName); err != nil {
			return err
		}
	}

	nfs, err := listNanoflowsGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list nanoflows", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		excluded      bool
		folderPath    string
		params        int
		activities    int
		complexity    int
		returnType    string
	}
	var rows []row

	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		// Nanoflows share the unit table with microflows for container lookup.
		containerID, _ := ctx.Microflows.GetContainerUUID(model.ID(nf.ID()))
		modName := h.GetModuleName(h.FindModuleID(containerID))
		if moduleName != "" && modName != moduleName {
			continue
		}
		qualifiedName := modName + "." + nf.Name()
		folderPath := h.BuildFolderPath(containerID)
		returnType := strings.TrimSpace(nf.ReturnType())
		oc, _ := nf.ObjectCollection().(*genMf.MicroflowObjectCollection)
		params := genFlowParameterElems(nf.ObjectCollection())
		activities := countGenFlowActivities(oc)
		complexity := calculateGenFlowComplexity(oc)
		rows = append(rows, row{
			qualifiedName: qualifiedName,
			module:        modName,
			name:          nf.Name(),
			excluded:      nf.Excluded(),
			folderPath:    folderPath,
			params:        len(params),
			activities:    activities,
			complexity:    complexity,
			returnType:    returnType,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Excluded", "Folder", "Params", "Actions", "McCabe", "Returns"},
		Summary: fmt.Sprintf("(%d nanoflows)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.excluded, r.folderPath, r.params, r.activities, r.complexity, r.returnType})
	}
	return writeResult(ctx, result)
}

// countGenFlowActivities counts meaningful activities — anything that
// is not a structural element (start/end events, exclusive merges).
// Mirrors the legacy `countMicroflowActivities` / `countNanoflowActivities`
// helpers but iterates the gen ObjectCollection's element list.
func countGenFlowActivities(oc *genMf.MicroflowObjectCollection) int {
	if oc == nil {
		return 0
	}
	count := 0
	for _, obj := range oc.ObjectsItems() {
		switch obj.TypeName() {
		case "Microflows$StartEvent",
			"Microflows$EndEvent",
			"Microflows$ExclusiveMerge",
			"Microflows$MicroflowParameter":
			// structural / parameter elements — skip
		default:
			count++
		}
	}
	return count
}

// calculateGenFlowComplexity computes McCabe cyclomatic complexity for
// a gen flow. Each conditional branch (ExclusiveSplit / InheritanceSplit /
// LoopedActivity) adds 1; baseline = 1 (the entry path).
func calculateGenFlowComplexity(oc *genMf.MicroflowObjectCollection) int {
	if oc == nil {
		return 1
	}
	complexity := 1
	for _, obj := range oc.ObjectsItems() {
		switch obj.TypeName() {
		case "Microflows$ExclusiveSplit",
			"Microflows$InheritanceSplit",
			"Microflows$LoopedActivity":
			complexity++
		}
	}
	return complexity
}
