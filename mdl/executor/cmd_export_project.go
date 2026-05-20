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

// captureDescribe temporarily redirects ctx.Output to a buffer while fn runs
// and returns the captured text. ctx.Output is restored before returning,
// even when fn returns an error.
func captureDescribe(ctx *ExecContext, fn func(*ExecContext) error) (string, error) {
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

	settings, err := captureDescribe(ctx, func(c *ExecContext) error {
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

	nav, err := captureDescribe(ctx, func(c *ExecContext) error {
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

	sec, err := captureDescribe(ctx, func(c *ExecContext) error {
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

// exportModule writes per-module files. The body is filled in by Task 5;
// for Task 4 we just write _module.mdl so the test can verify a module
// directory is created.
func exportModule(ctx *ExecContext, outputDir string, m *model.Module, opts ExportOptions, progress func(string)) error {
	moduleContent, err := captureDescribe(ctx, func(c *ExecContext) error {
		return describeModule(c, m.Name, false)
	})
	if err != nil {
		moduleContent = fmt.Sprintf("create module %s;\n", m.Name)
	}
	modulePath := filepath.Join(outputDir, m.Name, "_module.mdl")
	if opts.DryRun {
		progress(fmt.Sprintf("  [dry-run]  %s", modulePath))
	} else {
		if err := writeFile(modulePath, moduleContent); err != nil {
			return err
		}
		progress(fmt.Sprintf("  [write]    %s", modulePath))
	}
	return nil
}
