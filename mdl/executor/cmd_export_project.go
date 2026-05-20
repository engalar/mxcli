// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// ExportOptions controls the behaviour of ExportProject.
type ExportOptions struct {
	Module   string
	DryRun   bool
	// Force bypasses the cache check and always re-exports every document.
	Force    bool
	Progress func(line string)
}

// exportCache holds pre-loaded unit hash data for cache-based skip logic.
type exportCache struct {
	unitHashes map[string]string  // unit UUID → ContentsHash (nil = unavailable)
	allUnits   []*types.UnitInfo  // all project units with ContainerID (nil = unavailable)
}

const cacheCommentPrefix = "-- @cache: "

// moduleHash computes a stable combined hash covering every unit that belongs
// to the given module. Returns "" when hash data is unavailable.
func (ec *exportCache) moduleHash(h *ContainerHierarchy, moduleID model.ID) string {
	if ec == nil || ec.unitHashes == nil || ec.allUnits == nil {
		return ""
	}
	var hashes []string
	for _, u := range ec.allUnits {
		if h.FindModuleID(u.ContainerID) != moduleID {
			continue
		}
		if hash, ok := ec.unitHashes[string(u.ID)]; ok && hash != "" {
			hashes = append(hashes, hash)
		}
	}
	if len(hashes) == 0 {
		return ""
	}
	sort.Strings(hashes)
	sum := sha256.Sum256([]byte(strings.Join(hashes, ",")))
	return base64.RawURLEncoding.EncodeToString(sum[:])[:12]
}

// docHash returns the ContentsHash for a top-level document unit using its
// gen element ID string (e.g. string(mf.ID())). Returns "" when unavailable.
func (ec *exportCache) docHash(elemID string) string {
	if ec == nil || ec.unitHashes == nil {
		return ""
	}
	return ec.unitHashes[elemID]
}

// containmentHash returns the ContentsHash of the first unit with the given
// ContainmentName belonging to moduleID. Used for embedded units like
// DomainModel (entities/enums/assocs) and ModuleSecurity (module roles).
func (ec *exportCache) containmentHash(containmentName string, h *ContainerHierarchy, moduleID model.ID) string {
	if ec == nil || ec.unitHashes == nil || ec.allUnits == nil {
		return ""
	}
	for _, u := range ec.allUnits {
		if u.ContainmentName != containmentName {
			continue
		}
		if h.FindModuleID(u.ContainerID) == moduleID {
			return ec.unitHashes[string(u.ID)]
		}
	}
	return ""
}

// readCacheMarker reads the first line of path and returns the hash value
// after the cache comment prefix, or "" if absent/unreadable.
func readCacheMarker(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, cacheCommentPrefix) {
			return strings.TrimPrefix(line, cacheCommentPrefix)
		}
	}
	return ""
}

// cacheHit returns true when the existing file at path has a matching cache
// marker, meaning the underlying model hasn't changed.
func cacheHit(path, hash string) bool {
	if hash == "" {
		return false
	}
	return readCacheMarker(path) == hash
}

// documentFilePath returns the output file path for a single document.
func documentFilePath(outputDir, moduleName, folderPath, qname string) string {
	if folderPath != "" {
		return filepath.Join(outputDir, moduleName, filepath.FromSlash(folderPath), qname+".mdl")
	}
	return filepath.Join(outputDir, moduleName, qname+".mdl")
}

// builtinModuleNames lists Mendix built-in read-only modules that must not
// be exported or imported (they are always present in every project).
var builtinModuleNames = map[string]bool{
	"System": true,
}

func classifyModules(mods []*model.Module) (regular, marketplace []*model.Module) {
	for _, m := range mods {
		if builtinModuleNames[m.Name] {
			continue // skip built-in read-only modules
		}
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

	// Pre-load unit hash data once for the whole export run.
	ec := &exportCache{}
	if !opts.Force {
		ec.unitHashes, _ = ctx.Backend.ListUnitHashes()
		ec.allUnits, _ = ctx.Backend.ListUnits()
	}

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
		if err := exportModule(ctx, outputDir, m, opts, ec, progress); err != nil {
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

	// Export user roles as executable MDL (create user role ... ;)
	sec, serr := exportUserRoles(ctx)
	if serr == nil && sec != "" {
		secPath := filepath.Join(projDir, "security.mdl")
		if opts.DryRun {
			progress(fmt.Sprintf("  [dry-run]  %s", secPath))
		} else {
			if err := writeFile(secPath, sec); err != nil {
				return err
			}
			progress(fmt.Sprintf("  [write]    %s", secPath))
		}
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
func exportModule(ctx *ExecContext, outputDir string, m *model.Module, opts ExportOptions, ec *exportCache, progress func(string)) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return fmt.Errorf("hierarchy: %w", err)
	}

	// Module-level cache check: if all units in this module are unchanged, skip.
	moduleMDLPath := filepath.Join(outputDir, m.Name, "_module.mdl")
	modHash := ec.moduleHash(h, m.ID)
	if !opts.Force && modHash != "" && cacheHit(moduleMDLPath, modHash) {
		progress(fmt.Sprintf("  [cached]   %s (unchanged)", m.Name))
		return nil
	}

	moduleContent, err := captureDescribeFunc(ctx, func(c *ExecContext) error {
		return describeModule(c, m.Name, false)
	})
	if err != nil {
		moduleContent = fmt.Sprintf("create module %s;\n", m.Name)
	}
	if err := writeOrLog(moduleMDLPath, modHash, moduleContent, opts, progress); err != nil {
		return err
	}

	// domHash covers all entities, enumerations, associations (embedded in DomainModel unit).
	// secHash covers module roles (embedded in ModuleSecurity unit).
	domHash := ec.containmentHash("DomainModel", h, m.ID)
	secHash := ec.containmentHash("ModuleSecurity", h, m.ID)

	if err := exportEnumerations(ctx, outputDir, m, h, opts, ec, domHash, progress); err != nil {
		return err
	}
	if err := exportEntities(ctx, outputDir, m, h, opts, ec, domHash, progress); err != nil {
		return err
	}
	if err := exportAssociations(ctx, outputDir, m, opts, ec, domHash, progress); err != nil {
		return err
	}
	if err := exportConstants(ctx, outputDir, m, opts, ec, progress); err != nil {
		return err
	}
	if err := exportModuleRoles(ctx, outputDir, m, opts, ec, secHash, progress); err != nil {
		return err
	}
	if err := exportJavaActions(ctx, outputDir, m, h, opts, ec, progress); err != nil {
		return err
	}
	if err := exportJavaScriptActions(ctx, outputDir, m, h, opts, ec, progress); err != nil {
		return err
	}
	if err := exportMicroflows(ctx, outputDir, m, h, opts, ec, progress); err != nil {
		return err
	}
	if err := exportNanoflows(ctx, outputDir, m, h, opts, ec, progress); err != nil {
		return err
	}
	if err := exportLayouts(ctx, outputDir, m, h, opts, ec, progress); err != nil {
		return err
	}
	if err := exportSnippets(ctx, outputDir, m, h, opts, ec, progress); err != nil {
		return err
	}
	if err := exportPages(ctx, outputDir, m, h, opts, ec, progress); err != nil {
		return err
	}
	if err := exportWorkflows(ctx, outputDir, m, h, opts, ec, progress); err != nil {
		return err
	}
	return nil
}

// writeOrLog writes content to path with an optional cache marker.
// hash is the ContentsHash of the source unit; "" means no caching.
// If the existing file already has a matching marker, the write is skipped.
func writeOrLog(path, hash, content string, opts ExportOptions, progress func(string)) error {
	if !opts.Force && cacheHit(path, hash) {
		progress(fmt.Sprintf("  [cached]   %s", path))
		return nil
	}
	if opts.DryRun {
		progress(fmt.Sprintf("  [dry-run]  %s", path))
		return nil
	}
	if hash != "" {
		content = cacheCommentPrefix + hash + "\n" + content
	}
	if err := writeFile(path, content); err != nil {
		return err
	}
	progress(fmt.Sprintf("  [write]    %s", path))
	return nil
}

func exportEnumerations(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, ec *exportCache, domHash string, progress func(string)) error {
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
		path := documentFilePath(outputDir, m.Name, joinSection("Enumerations", folder), qname)
		if err := writeOrLog(path, domHash, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportEntities(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, ec *exportCache, domHash string, progress func(string)) error {
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
		folder := h.BuildFolderPath(model.ID(e.ID()))
		path := documentFilePath(outputDir, m.Name, joinSection("Domain", folder), qname)
		if err := writeOrLog(path, domHash, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportAssociations(ctx *ExecContext, outputDir string, m *model.Module, opts ExportOptions, ec *exportCache, domHash string, progress func(string)) error {
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
	return writeOrLog(path, domHash, sb.String(), opts, progress)
}

func exportConstants(ctx *ExecContext, outputDir string, m *model.Module, opts ExportOptions, ec *exportCache, progress func(string)) error {
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
		content, derr := captureDescribeFunc(ctx, func(inner *ExecContext) error {
			return outputConstantMDL(inner, c, m.Name)
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe constant %s failed: %v\n", qname, derr)
		}
		folder := h.BuildFolderPath(c.ContainerID)
		path := documentFilePath(outputDir, m.Name, joinSection("Constants", folder), qname)
		// Constants have no separate unit in v1; use "" (always rewrite).
		if err := writeOrLog(path, "", content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportModuleRoles(ctx *ExecContext, outputDir string, m *model.Module, opts ExportOptions, ec *exportCache, secHash string, progress func(string)) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return nil
	}
	pairs, err := listModuleSecurityWithContainerGen(ctx)
	if err != nil {
		return nil
	}

	var sb strings.Builder
	for _, p := range pairs {
		if h.GetModuleName(p.ContainerID) != m.Name {
			continue
		}
		for _, mr := range p.MS.ModuleRolesItems() {
			named, ok := mr.(interface{ Name() string })
			if !ok {
				continue
			}
			content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
				return describeModuleRoleGen(c, ast.QualifiedName{Module: m.Name, Name: named.Name()})
			})
			if derr != nil {
				continue
			}
			sb.WriteString(content)
		}
	}

	if sb.Len() == 0 {
		return nil
	}
	path := filepath.Join(outputDir, m.Name, "_module_roles.mdl")
	return writeOrLog(path, secHash, sb.String(), opts, progress)
}

func exportMicroflows(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, ec *exportCache, progress func(string)) error {
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
		hash := ec.docHash(string(it.MF.ID()))
		folder := h.BuildFolderPath(model.ID(it.ContainerUUID))
		path := documentFilePath(outputDir, m.Name, joinSection("Microflows", folder), qname)
		if !opts.Force && cacheHit(path, hash) {
			progress(fmt.Sprintf("  [cached]   %s", path))
			continue
		}
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeMicroflowGen(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe microflow %s failed: %v\n", qname, derr)
		}
		if err := writeOrLog(path, hash, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportNanoflows(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, ec *exportCache, progress func(string)) error {
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
		hash := ec.docHash(string(it.NF.ID()))
		folder := h.BuildFolderPath(model.ID(it.ContainerUUID))
		path := documentFilePath(outputDir, m.Name, joinSection("Nanoflows", folder), qname)
		if !opts.Force && cacheHit(path, hash) {
			progress(fmt.Sprintf("  [cached]   %s", path))
			continue
		}
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeNanoflowGen(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe nanoflow %s failed: %v\n", qname, derr)
		}
		if err := writeOrLog(path, hash, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportJavaActions(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, ec *exportCache, progress func(string)) error {
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
		hash := ec.docHash(string(it.Elem.ID()))
		folder := h.BuildFolderPath(model.ID(it.ContainerID))
		path := documentFilePath(outputDir, m.Name, joinSection("JavaActions", folder), qname)
		if !opts.Force && cacheHit(path, hash) {
			progress(fmt.Sprintf("  [cached]   %s", path))
			continue
		}
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeJavaActionGen(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe java action %s failed: %v\n", qname, derr)
		}
		if err := writeOrLog(path, hash, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportJavaScriptActions(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, ec *exportCache, progress func(string)) error {
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
		hash := ec.docHash(string(it.Elem.ID()))
		folder := h.BuildFolderPath(model.ID(it.ContainerID))
		path := documentFilePath(outputDir, m.Name, joinSection("JavaScriptActions", folder), qname)
		if !opts.Force && cacheHit(path, hash) {
			progress(fmt.Sprintf("  [cached]   %s", path))
			continue
		}
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeJavaScriptActionGen(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe javascript action %s failed: %v\n", qname, derr)
		}
		if err := writeOrLog(path, hash, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportPages(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, ec *exportCache, progress func(string)) error {
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
		hash := ec.docHash(string(it.Elem.ID()))
		folder := h.BuildFolderPath(model.ID(it.ContainerID))
		path := documentFilePath(outputDir, m.Name, joinSection("Pages", folder), qname)
		if !opts.Force && cacheHit(path, hash) {
			progress(fmt.Sprintf("  [cached]   %s", path))
			continue
		}
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describePage(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe page %s failed: %v\n", qname, derr)
		}
		if err := writeOrLog(path, hash, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportLayouts(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, ec *exportCache, progress func(string)) error {
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
		hash := ec.docHash(string(it.Elem.ID()))
		folder := h.BuildFolderPath(model.ID(it.ContainerID))
		path := documentFilePath(outputDir, m.Name, joinSection("Layouts", folder), qname)
		if !opts.Force && cacheHit(path, hash) {
			progress(fmt.Sprintf("  [cached]   %s", path))
			continue
		}
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeLayout(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe layout %s failed: %v\n", qname, derr)
		}
		if err := writeOrLog(path, hash, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportSnippets(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, ec *exportCache, progress func(string)) error {
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
		hash := ec.docHash(string(it.Elem.ID()))
		folder := h.BuildFolderPath(model.ID(it.ContainerID))
		path := documentFilePath(outputDir, m.Name, joinSection("Snippets", folder), qname)
		if !opts.Force && cacheHit(path, hash) {
			progress(fmt.Sprintf("  [cached]   %s", path))
			continue
		}
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeSnippet(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe snippet %s failed: %v\n", qname, derr)
		}
		if err := writeOrLog(path, hash, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

func exportWorkflows(ctx *ExecContext, outputDir string, m *model.Module, h *ContainerHierarchy, opts ExportOptions, ec *exportCache, progress func(string)) error {
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
		hash := ec.docHash(string(it.Elem.ID()))
		folder := h.BuildFolderPath(model.ID(it.ContainerID))
		path := documentFilePath(outputDir, m.Name, joinSection("Workflows", folder), qname)
		if !opts.Force && cacheHit(path, hash) {
			progress(fmt.Sprintf("  [cached]   %s", path))
			continue
		}
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeWorkflowGen(c, ast.QualifiedName{Module: m.Name, Name: name})
		})
		if derr != nil {
			content = fmt.Sprintf("-- describe workflow %s failed: %v\n", qname, derr)
		}
		if err := writeOrLog(path, hash, content, opts, progress); err != nil {
			return err
		}
	}
	return nil
}

// exportUserRoles returns executable MDL for every user role in the project.
// Each role becomes a "create user role ..." statement, which is importable.
func exportUserRoles(ctx *ExecContext) (string, error) {
	ps, err := getProjectSecurityGen(ctx)
	if err != nil || ps == nil {
		return "", err
	}
	var sb strings.Builder
	for _, ur := range ps.UserRolesItems() {
		named, ok := ur.(interface{ Name() string })
		if !ok {
			continue
		}
		content, derr := captureDescribeFunc(ctx, func(c *ExecContext) error {
			return describeUserRoleGen(c, ast.QualifiedName{Name: named.Name()})
		})
		if derr != nil {
			continue
		}
		sb.WriteString(content)
	}
	return sb.String(), nil
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
