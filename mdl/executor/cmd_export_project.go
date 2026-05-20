// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
)

// ExportOptions controls the behaviour of ExportProject.
type ExportOptions struct {
	Module   string
	DryRun   bool
	Progress func(line string)
}

// documentFilePath returns the output file path for a single document.
func documentFilePath(outputDir, moduleName, folderPath, qname string) string {
	if folderPath != "" {
		return filepath.Join(outputDir, moduleName, filepath.FromSlash(folderPath), qname+".mdl")
	}
	return filepath.Join(outputDir, moduleName, qname+".mdl")
}

func classifyModules(mods []*model.Module) (regular, marketplace []*model.Module) {
	for _, m := range mods {
		if m.FromAppStore {
			marketplace = append(marketplace, m)
		} else {
			regular = append(regular, m)
		}
	}
	return
}

// captureDescribeFunc temporarily redirects ctx.Output to a buffer while fn
// runs and returns the captured text. ctx.Output is restored before returning,
// even when fn returns an error.
//
// Distinct from captureDescribe (cmd_catalog.go), which is a string-dispatch
// helper for SHOW CATALOG paths.
func captureDescribeFunc(ctx *ExecContext, fn func(*ExecContext) error) (string, error) {
	var buf bytes.Buffer
	saved := ctx.Output
	ctx.Output = &buf
	err := fn(ctx)
	ctx.Output = saved
	return buf.String(), err
}

func marketplaceFileContent(mods []*model.Module) string {
	var sb strings.Builder
	sb.WriteString("-- Marketplace modules detected in this project.\n")
	sb.WriteString("-- Reinstall these before running mxcli import.\n")
	sb.WriteString("--\n")
	for _, m := range mods {
		version := m.AppStoreVersion
		if version == "" {
			version = "unknown"
		}
		fmt.Fprintf(&sb, "-- Module: %-30s (version: %s)\n", m.Name, version)
	}
	return sb.String()
}

// writeFile writes content to path, creating parent directories as needed.
func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// ExportProject writes the project to outputDir as a tree of MDL files.
//
// Layout:
//
//	outputDir/
//	  _marketplace.mdl              -- marketplace module manifest (commentary)
//	  _project/
//	    settings.mdl                -- alter settings ...
//	    navigation.mdl              -- create or replace navigation ...
//	    security.mdl                -- project security (placeholder)
//	  <Module>/                     -- one directory per non-marketplace module
//	    _module.mdl                 -- create module Foo;
//	    ...                         -- per-document files (added in Task 5)
func (e *Executor) ExportProject(outputDir string, opts ExportOptions) error {
	ctx := e.newExecContext(context.Background())
	if !ctx.Connected() {
		return fmt.Errorf("not connected to a project")
	}

	progress := opts.Progress
	if progress == nil {
		progress = func(string) {}
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outputDir, err)
	}

	mods, err := ctx.Backend.ListModules()
	if err != nil {
		return fmt.Errorf("list modules: %w", err)
	}
	regular, marketplace := classifyModules(mods)

	marketContent := marketplaceFileContent(marketplace)
	marketPath := filepath.Join(outputDir, "_marketplace.mdl")
	if opts.DryRun {
		progress(fmt.Sprintf("  [dry-run]  %s", marketPath))
	} else {
		if err := writeFile(marketPath, marketContent); err != nil {
			return fmt.Errorf("write _marketplace.mdl: %w", err)
		}
		progress(fmt.Sprintf("  [write]    %s", marketPath))
	}

	if err := exportProjectLevel(ctx, outputDir, opts, progress); err != nil {
		return fmt.Errorf("export project-level: %w", err)
	}

	if opts.Module != "" {
		filtered := regular[:0]
		for _, m := range regular {
			if m.Name == opts.Module {
				filtered = append(filtered, m)
			}
		}
		regular = filtered
	}

	for _, m := range regular {
		if err := exportModule(ctx, outputDir, m, opts, progress); err != nil {
			return fmt.Errorf("export module %s: %w", m.Name, err)
		}
	}

	return nil
}

// exportProjectLevel writes _project/settings.mdl, _project/navigation.mdl,
// and _project/security.mdl.
func exportProjectLevel(ctx *ExecContext, outputDir string, opts ExportOptions, progress func(string)) error {
	projDir := filepath.Join(outputDir, "_project")

	settings, err := captureDescribeFunc(ctx, func(c *ExecContext) error {
		return describeSettings(c)
	})
	if err != nil {
		settings = fmt.Sprintf("-- describe settings failed: %v\n", err)
	}
	settingsPath := filepath.Join(projDir, "settings.mdl")
	if opts.DryRun {
		progress(fmt.Sprintf("  [dry-run]  %s", settingsPath))
	} else {
		if err := writeFile(settingsPath, settings); err != nil {
			return err
		}
		progress(fmt.Sprintf("  [write]    %s", settingsPath))
	}

	nav, err := captureDescribeFunc(ctx, func(c *ExecContext) error {
		return describeNavigation(c, ast.QualifiedName{})
	})
	if err != nil {
		nav = fmt.Sprintf("-- describe navigation failed: %v\n", err)
	}
	navPath := filepath.Join(projDir, "navigation.mdl")
	if opts.DryRun {
		progress(fmt.Sprintf("  [dry-run]  %s", navPath))
	} else {
		if err := writeFile(navPath, nav); err != nil {
			return err
		}
		progress(fmt.Sprintf("  [write]    %s", navPath))
	}

	sec, err := captureDescribeFunc(ctx, func(c *ExecContext) error {
		return listProjectSecurityGen(c)
	})
	if err != nil {
		sec = fmt.Sprintf("-- describe project security failed: %v\n", err)
	}
	secPath := filepath.Join(projDir, "security.mdl")
	if opts.DryRun {
		progress(fmt.Sprintf("  [dry-run]  %s", secPath))
	} else {
		if err := writeFile(secPath, sec); err != nil {
			return err
		}
		progress(fmt.Sprintf("  [write]    %s", secPath))
	}

	return nil
}

// exportModule writes one MDL file per document in a module.
//
// Layout under outputDir/<Module>/:
//
//	_module.mdl              -- create module <Name>;
//	Domain/<qname>.mdl       -- entities + enumerations (one per file)
//	_associations.mdl        -- create association ... (one statement per assoc)
//	Constants/<qname>.mdl    -- create or modify constant ...
//	_module_roles.mdl        -- module security roles
//	Microflows/<folder>/<qname>.mdl
//	Nanoflows/<folder>/<qname>.mdl
//	JavaActions/<folder>/<qname>.mdl
//	JavaScriptActions/<folder>/<qname>.mdl
//	Pages/<folder>/<qname>.mdl
//	Layouts/<folder>/<qname>.mdl
//	Snippets/<folder>/<qname>.mdl
//	Workflows/<folder>/<qname>.mdl
//
// <folder> is the document's container path inside the module
// (BuildFolderPath); root-level documents land directly under the
// section directory.
func exportModule(ctx *ExecContext, outputDir string, m *model.Module, opts ExportOptions, progress func(string)) error {
	moduleContent, err := captureDescribeFunc(ctx, func(c *ExecContext) error {
		return describeModule(c, m.Name, false)
	})
	if err != nil {
		moduleContent = fmt.Sprintf("create module %s;\n", m.Name)
	}
	if err := writeOrLog(filepath.Join(outputDir, m.Name, "_module.mdl"), moduleContent, opts, progress); err != nil {
		return err
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return fmt.Errorf("hierarchy: %w", err)
	}

	if err := exportEnumerations(ctx, outputDir, m, h, opts, progress); err != nil {
		return err
	}
	if err := exportEntities(ctx, outputDir, m, h, opts, progress); err != nil {
		return err
	}
	if err := exportAssociations(ctx, outputDir, m, opts, progress); err != nil {
		return err
	}
	if err := exportConstants(ctx, outputDir, m, opts, progress); err != nil {
		return err
	}
	if err := exportModuleRoles(ctx, outputDir, m, opts, progress); err != nil {
		return err
	}
	if err := exportJavaActions(ctx, outputDir, m, h, opts, progress); err != nil {
		return err
	}
	if err := exportJavaScriptActions(ctx, outputDir, m, h, opts, progress); err != nil {
		return err
	}
	if err := exportMicroflows(ctx, outputDir, m, h, opts, progress); err != nil {
		return err
	}
	if err := exportNanoflows(ctx, outputDir, m, h, opts, progress); err != nil {
		return err
	}
	if err := exportLayouts(ctx, outputDir, m, h, opts, progress); err != nil {
		return err
	}
	if err := exportSnippets(ctx, outputDir, m, h, opts, progress); err != nil {
		return err
	}
	if err := exportPages(ctx, outputDir, m, h, opts, progress); err != nil {
		return err
	}
	if err := exportWorkflows(ctx, outputDir, m, h, opts, progress); err != nil {
		return err
	}
	return nil
}

func writeOrLog(path, content string, opts ExportOptions, progress func(string)) error {
	if opts.DryRun {
		progress(fmt.Sprintf("  [dry-run]  %s", path))
		return nil
	}
	if err := writeFile(path, content); err != nil {
		return err
	}
	progress(fmt.Sprintf("  [write]    %s", path))
	return nil
}

func exportEnumerations(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, progress func(string)) error {
	enums, err := ctx.Backend.ListEnumerations()
	if err != nil {
		return nil
	}
	for _, e := range enums {
		if h.FindModuleID(e.ContainerID) != m.ID {
			continue
		}
		qname := m.Name + "." + e.Name
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeEnumeration(c, ast.QualifiedName{Module: m.Name, Name: e.Name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe enumeration %s failed: %v\n", qname, derr)
		}
		folder := h.BuildFolderPath(e.ContainerID)
		path := documentFilePath(outputDir, m.Name, joinSection("Domain", folder), qname)
		if err := writeOrLog(path, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportEntities(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, progress func(string)) error {
	entities, err := listEntitiesForModuleGen(ctx, m.Name)
	if err != nil || len(entities) == 0 {
		return nil
	}
	sorted := sortEntitiesByGeneralizationGen(entities, m.Name)
	for _, e := range sorted {
		name := e.Name()
		qname := m.Name + "." + name
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeEntity(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe entity %s failed: %v\n", qname, derr)
		}
		// Entities live inside the DomainModel; use the DM's folder path.
		folder := h.BuildFolderPath(model.ID(e.ID()))
		path := documentFilePath(outputDir, m.Name, joinSection("Domain", folder), qname)
		if err := writeOrLog(path, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportAssociations(ctx *ExecContext, outputDir string, m *model.Module, opts ExportOptions, progress func(string)) error {
	assocs, err := listAssociationsForModuleGen(ctx, m.Name)
	if err != nil || len(assocs) == 0 {
		return nil
	}
	var sb strings.Builder
	for _, a := range assocs {
		name := a.Name()
		qname := m.Name + "." + name
		text, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeAssociation(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			sb.WriteString(fmt.Sprintf("-- describe association %s failed: %v\n", qname, derr))
			continue
		}
		sb.WriteString(text)
		if !strings.HasSuffix(text, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	path := filepath.Join(outputDir, m.Name, "_associations.mdl")
	return writeOrLog(path, sb.String(), opts, progress)
}

func exportConstants(ctx *ExecContext, outputDir string, m *model.Module, opts ExportOptions, progress func(string)) error {
	consts, err := ctx.Backend.ListConstants()
	if err != nil {
		return nil
	}
	h, _ := getHierarchy(ctx)
	for _, c := range consts {
		if h == nil || h.FindModuleID(c.ContainerID) != m.ID {
			continue
		}
		qname := m.Name + "." + c.Name
		content, derr := captureDescribeFunc(ctx, func(ec *ExecContext) error {
			return outputConstantMDL(ec, c, m.Name)
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe constant %s failed: %v\n", qname, derr)
		}
		folder := h.BuildFolderPath(c.ContainerID)
		path := documentFilePath(outputDir, m.Name, joinSection("Constants", folder), qname)
		if err := writeOrLog(path, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportModuleRoles(ctx *ExecContext, outputDir string, m *model.Module, opts ExportOptions, progress func(string)) error {
	rolesContent, err := captureDescribeFunc(ctx, func(c *ExecContext) error {
		return listModuleRolesGen(c, m.Name)
	})
	if err != nil || strings.TrimSpace(rolesContent) == "" {
		return nil
	}
	path := filepath.Join(outputDir, m.Name, "_module_roles.mdl")
	return writeOrLog(path, rolesContent, opts, progress)
}

func exportMicroflows(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, progress func(string)) error {
	items, err := listMicroflowsWithContainerGen(ctx)
	if err != nil {
		return nil
	}
	for _, it := range items {
		if it.MF == nil {
			continue
		}
		if h.FindModuleID(model.ID(it.ContainerUUID)) != m.ID {
			continue
		}
		name := it.MF.Name()
		qname := m.Name + "." + name
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeMicroflowGen(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe microflow %s failed: %v\n", qname, derr)
		}
		folder := h.BuildFolderPath(model.ID(it.ContainerUUID))
		path := documentFilePath(outputDir, m.Name, joinSection("Microflows", folder), qname)
		if err := writeOrLog(path, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportNanoflows(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, progress func(string)) error {
	items, err := listNanoflowsWithContainerGen(ctx)
	if err != nil {
		return nil
	}
	for _, it := range items {
		if it.NF == nil {
			continue
		}
		if h.FindModuleID(model.ID(it.ContainerUUID)) != m.ID {
			continue
		}
		name := it.NF.Name()
		qname := m.Name + "." + name
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeNanoflowGen(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe nanoflow %s failed: %v\n", qname, derr)
		}
		folder := h.BuildFolderPath(model.ID(it.ContainerUUID))
		path := documentFilePath(outputDir, m.Name, joinSection("Nanoflows", folder), qname)
		if err := writeOrLog(path, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportJavaActions(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, progress func(string)) error {
	items, err := listJavaActionsWithContainerGen(ctx)
	if err != nil {
		return nil
	}
	for _, it := range items {
		if h.FindModuleID(model.ID(it.ContainerID)) != m.ID {
			continue
		}
		name := it.Elem.Name()
		qname := m.Name + "." + name
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeJavaActionGen(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe java action %s failed: %v\n", qname, derr)
		}
		folder := h.BuildFolderPath(model.ID(it.ContainerID))
		path := documentFilePath(outputDir, m.Name, joinSection("JavaActions", folder), qname)
		if err := writeOrLog(path, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportJavaScriptActions(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, progress func(string)) error {
	items, err := listJavaScriptActionsWithContainerGen(ctx)
	if err != nil {
		return nil
	}
	for _, it := range items {
		if h.FindModuleID(model.ID(it.ContainerID)) != m.ID {
			continue
		}
		name := it.Elem.Name()
		qname := m.Name + "." + name
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeJavaScriptActionGen(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe javascript action %s failed: %v\n", qname, derr)
		}
		folder := h.BuildFolderPath(model.ID(it.ContainerID))
		path := documentFilePath(outputDir, m.Name, joinSection("JavaScriptActions", folder), qname)
		if err := writeOrLog(path, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportPages(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, progress func(string)) error {
	items, err := listPagesWithContainerGen(ctx)
	if err != nil {
		return nil
	}
	for _, it := range items {
		if h.FindModuleID(model.ID(it.ContainerID)) != m.ID {
			continue
		}
		name := it.Elem.Name()
		qname := m.Name + "." + name
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describePage(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe page %s failed: %v\n", qname, derr)
		}
		folder := h.BuildFolderPath(model.ID(it.ContainerID))
		path := documentFilePath(outputDir, m.Name, joinSection("Pages", folder), qname)
		if err := writeOrLog(path, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportLayouts(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, progress func(string)) error {
	items, err := listLayoutsWithContainerGen(ctx)
	if err != nil {
		return nil
	}
	for _, it := range items {
		if h.FindModuleID(model.ID(it.ContainerID)) != m.ID {
			continue
		}
		name := it.Elem.Name()
		qname := m.Name + "." + name
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeLayout(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe layout %s failed: %v\n", qname, derr)
		}
		folder := h.BuildFolderPath(model.ID(it.ContainerID))
		path := documentFilePath(outputDir, m.Name, joinSection("Layouts", folder), qname)
		if err := writeOrLog(path, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportSnippets(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, progress func(string)) error {
	items, err := listSnippetsWithContainerGen(ctx)
	if err != nil {
		return nil
	}
	for _, it := range items {
		if h.FindModuleID(model.ID(it.ContainerID)) != m.ID {
			continue
		}
		name := it.Elem.Name()
		qname := m.Name + "." + name
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeSnippet(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe snippet %s failed: %v\n", qname, derr)
		}
		folder := h.BuildFolderPath(model.ID(it.ContainerID))
		path := documentFilePath(outputDir, m.Name, joinSection("Snippets", folder), qname)
		if err := writeOrLog(path, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportWorkflows(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, progress func(string)) error {
	items, err := listWorkflowsWithContainerGen(ctx)
	if err != nil {
		return nil
	}
	for _, it := range items {
		if h.FindModuleID(model.ID(it.ContainerID)) != m.ID {
			continue
		}
		name := it.Elem.Name()
		qname := m.Name + "." + name
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeWorkflowGen(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe workflow %s failed: %v\n", qname, derr)
		}
		folder := h.BuildFolderPath(model.ID(it.ContainerID))
		path := documentFilePath(outputDir, m.Name, joinSection("Workflows", folder), qname)
		if err := writeOrLog(path, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

// joinSection joins a section name ("Microflows", "Domain", ...) with an
// optional folder sub-path, returning a slash-separated relative path
// suitable for documentFilePath.
func joinSection(section, folder string) string {
	if folder == "" {
		return section
	}
	return section + "/" + folder
}
