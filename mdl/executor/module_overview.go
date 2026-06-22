// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"encoding/json"
	"fmt"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

type moduleOverviewData struct {
	Format  string               `json:"format"`
	Modules []moduleOverviewNode `json:"modules"`
}

type moduleOverviewNode struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	IsSystem       bool   `json:"isSystem"`
	EntityCount    int    `json:"entityCount"`
	MicroflowCount int    `json:"microflowCount"`
	PageCount      int    `json:"pageCount"`
}

type moduleOverviewEdge struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	RefKind  string `json:"refKind"`
	RefCount int    `json:"refCount"`
}

var systemModuleNames = map[string]bool{
	"System":                  true,
	"Atlas_Core":              true,
	"Atlas_UI_Resources":      true,
	"Atlas_Native_Mobile_Content": true,
}

// ModuleOverview builds the module dependency overview using backend sources.
func ModuleOverview(ctx *ExecContext) error {
	if !ctx.Connected() {
		return mdlerrors.NewNotConnected()
	}

	modules, err := ctx.ModuleLister.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("list modules", err)
	}

	modNames := make([]string, 0, len(modules))
	modMap := make(map[string]model.ID)
	for _, m := range modules {
		modNames = append(modNames, m.Name)
		modMap[m.Name] = m.ID
	}
	sortStrings(modNames)

	// Count entities via domain models
	entityCounts := make(map[string]int)
	if dms, err := ctx.DomainModelReader.ListDomainModelsGen(); err == nil {
		for _, dm := range dms {
			if dm == nil {
				continue
			}
			modName := findModuleNameByContainer(ctx, model.ID(dm.ID()))
			if modName != "" {
				for range dm.EntitiesItems() {
					entityCounts[modName]++
				}
			}
		}
	}

	// Count microflows per module via hierarchy
	mfCounts := make(map[string]int)
	if mfs, err := ctx.Microflows.ListAll(); err == nil {
		for _, mf := range mfs {
			if mf == nil {
				continue
			}
			modName := findModuleNameByContainer(ctx, model.ID(mf.ID()))
			if modName != "" {
				mfCounts[modName]++
			}
		}
	}

	pageCounts := make(map[string]int)
	if pages, err := ctx.Pages.ListAll(); err == nil {
		for _, p := range pages {
			if p == nil {
				continue
			}
			modName := findModuleNameByContainer(ctx, model.ID(p.ID()))
			if modName != "" {
				pageCounts[modName]++
			}
		}
	}

	var nodes []moduleOverviewNode
	for _, name := range modNames {
		nodes = append(nodes, moduleOverviewNode{
			ID:             name,
			Name:           name,
			IsSystem:       systemModuleNames[name],
			EntityCount:    entityCounts[name],
			MicroflowCount: mfCounts[name],
			PageCount:      pageCounts[name],
		})
	}

	data := moduleOverviewData{Format: "module-overview", Modules: nodes}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("encode module overview: %w", err)
	}
	fmt.Fprintln(ctx.Output, string(out))
	return nil
}

// findModuleNameByContainer resolves the module name for an element ID.
func findModuleNameByContainer(ctx *ExecContext, elemID model.ID) string {
	h, err := getHierarchy(ctx)
	if err != nil {
		return ""
	}
	modID := h.FindModuleID(elemID)
	return h.GetModuleName(modID)
}
