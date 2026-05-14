// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.5 Phase A: gen-typed SHOW PAGES / LAYOUTS / SNIPPETS
// commands. This file is the gen-typed twin of cmd_pages_show.go,
// cmd_layouts.go and cmd_snippets.go. It walks gen-typed units via the
// listPagesWithContainerGen / listLayoutsWithContainerGen /
// listSnippetsWithContainerGen cache helpers (Stage 3.3.5 A0) and
// renders the same TableResult shape as the legacy listings.

package executor

import (
	"fmt"
	"sort"
	"strings"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genTx "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
)

// listPagesGen handles SHOW PAGES via gen-typed Page units.
func listPagesGen(ctx *ExecContext, moduleName string) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list pages", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		excluded      bool
		folderPath    string
		title         string
		url           string
		params        int
	}
	var rows []row

	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modName := ""
		folderPath := ""
		if h != nil {
			modID := h.FindModuleID(model.ID(p.ContainerID))
			modName = h.GetModuleName(modID)
			folderPath = h.BuildFolderPath(model.ID(p.ContainerID))
		}
		if moduleName != "" && modName != moduleName {
			continue
		}
		qualifiedName := modName + "." + p.Elem.Name()
		title := pickPageTitleGen(p.Elem)
		url := p.Elem.Url()
		params := len(p.Elem.ParametersItems())

		rows = append(rows, row{
			qualifiedName: qualifiedName,
			module:        modName,
			name:          p.Elem.Name(),
			excluded:      p.Elem.Excluded(),
			folderPath:    folderPath,
			title:         title,
			url:           url,
			params:        params,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Excluded", "Folder", "Title", "url", "Params"},
		Summary: fmt.Sprintf("(%d pages)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.excluded, r.folderPath, r.title, r.url, r.params})
	}
	return writeResult(ctx, result)
}

// listLayoutsGen handles SHOW LAYOUTS via gen-typed Layout units.
func listLayoutsGen(ctx *ExecContext, moduleName string) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listLayoutsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list layouts", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		folderPath    string
		layoutType    string
	}
	var rows []row

	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modName := ""
		folderPath := ""
		if h != nil {
			modID := h.FindModuleID(model.ID(p.ContainerID))
			modName = h.GetModuleName(modID)
			folderPath = h.BuildFolderPath(model.ID(p.ContainerID))
		}
		if moduleName != "" && modName != moduleName {
			continue
		}
		qualifiedName := modName + "." + p.Elem.Name()
		rows = append(rows, row{
			qualifiedName: qualifiedName,
			module:        modName,
			name:          p.Elem.Name(),
			folderPath:    folderPath,
			layoutType:    p.Elem.LayoutType(),
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Folder", "Type"},
		Summary: fmt.Sprintf("(%d layouts)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.folderPath, r.layoutType})
	}
	return writeResult(ctx, result)
}

// listSnippetsGen handles SHOW SNIPPETS via gen-typed Snippet units.
func listSnippetsGen(ctx *ExecContext, moduleName string) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listSnippetsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list snippets", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		folderPath    string
		params        int
	}
	var rows []row

	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modName := ""
		folderPath := ""
		if h != nil {
			modID := h.FindModuleID(model.ID(p.ContainerID))
			modName = h.GetModuleName(modID)
			folderPath = h.BuildFolderPath(model.ID(p.ContainerID))
		}
		if moduleName != "" && modName != moduleName {
			continue
		}
		qualifiedName := modName + "." + p.Elem.Name()
		rows = append(rows, row{
			qualifiedName: qualifiedName,
			module:        modName,
			name:          p.Elem.Name(),
			folderPath:    folderPath,
			params:        len(p.Elem.ParametersItems()),
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Folder", "Params"},
		Summary: fmt.Sprintf("(%d snippets)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.folderPath, r.params})
	}
	return writeResult(ctx, result)
}

// pickPageTitleGen extracts the localized title from a gen-typed
// Page. Page.Title is typed as element.Element but Mendix decodes it
// as *Texts$Text. Returns "" when the title is empty or not a Text.
func pickPageTitleGen(p *genPg.Page) string {
	if p == nil {
		return ""
	}
	titleElem := p.Title()
	if titleElem == nil {
		return ""
	}
	t, ok := titleElem.(*genTx.Text)
	if !ok || t == nil {
		return ""
	}
	s, _ := pickTextTranslationGen(t)
	return s
}
