// SPDX-License-Identifier: Apache-2.0

// Package executor - Image collection commands (CREATE/DROP IMAGE COLLECTION)
package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
)

// ────────────────────────────────────────────────────────────
// Fn (HandlerDeps) versions
// ────────────────────────────────────────────────────────────

func execCreateImageCollectionFn(ctx context.Context, s *ast.CreateImageCollectionStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	module, err := findModuleFn(deps.ModuleLister, s.Name.Module)
	if err != nil {
		tmpCtx := NewExecContext(ctx, deps)
		if createErr := execCreateModule(tmpCtx, &ast.CreateModuleStmt{Name: s.Name.Module}); createErr != nil {
			return mdlerrors.NewBackend("auto-create module "+s.Name.Module, createErr)
		}
		module, err = findModuleFn(deps.ModuleLister, s.Name.Module)
		if err != nil {
			return err
		}
	}

	existing := findImageCollectionFn(deps, s.Name.Module, s.Name.Name)
	if existing != nil && !s.CreateOrModify {
		return mdlerrors.NewAlreadyExists("image collection", s.Name.Module+"."+s.Name.Name)
	}

	containerID := module.ID
	if existing != nil {
		containerID = existing.ContainerID
	}

	ic := &types.ImageCollection{
		ContainerID:   containerID,
		Name:          s.Name.Name,
		ExportLevel:   s.ExportLevel,
		Documentation: s.Comment,
	}
	if existing != nil {
		ic.ID = existing.ID
	}

	for _, item := range s.Images {
		filePath := item.FilePath
		if !filepath.IsAbs(filePath) {
			cwd, err := os.Getwd()
			if err != nil {
				return mdlerrors.NewBackend("get working directory", err)
			}
			filePath = filepath.Join(cwd, filePath)
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return mdlerrors.NewBackend(fmt.Sprintf("read image file %q", item.FilePath), err)
		}
		format := extToImageFormat(filepath.Ext(filePath))
		ic.Images = append(ic.Images, types.Image{
			Name:   item.Name,
			Data:   data,
			Format: format,
		})
	}

	if existing != nil {
		if err := deps.ImageCollectionWriter.UpdateImageCollection(ic); err != nil {
			return mdlerrors.NewBackend("update image collection", err)
		}
		fmt.Fprintf(deps.Output, "Modified image collection: %s\n", s.Name)
	} else {
		if err := deps.ImageCollectionWriter.CreateImageCollection(ic); err != nil {
			return mdlerrors.NewBackend("create image collection", err)
		}
		fmt.Fprintf(deps.Output, "Created image collection: %s\n", s.Name)
	}

	if deps.CacheInvalidator != nil {
		deps.CacheInvalidator.InvalidateCache()
	}
	return nil
}

func execDropImageCollectionFn(ctx context.Context, s *ast.DropImageCollectionStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	ic := findImageCollectionFn(deps, s.Name.Module, s.Name.Name)
	if ic == nil {
		return mdlerrors.NewNotFound("image collection", s.Name.String())
	}

	if err := deps.ImageCollectionWriter.DeleteImageCollection(string(ic.ID)); err != nil {
		return mdlerrors.NewBackend("delete image collection", err)
	}

	fmt.Fprintf(deps.Output, "Dropped image collection: %s\n", s.Name)
	return nil
}

func describeImageCollectionFn(ctx context.Context, output io.Writer, deps *HandlerDeps, name ast.QualifiedName) error {
	ic := findImageCollectionFn(deps, name.Module, name.Name)
	if ic == nil {
		return mdlerrors.NewNotFound("image collection", name.String())
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return err
	}
	modID := h.FindModuleID(ic.ContainerID)
	modName := h.GetModuleName(modID)

	if ic.Documentation != "" {
		fmt.Fprintf(output, "/**\n * %s\n */\n", ic.Documentation)
	}

	exportLevel := ic.ExportLevel
	if exportLevel == "" {
		exportLevel = "Hidden"
	}

	qualifiedName := fmt.Sprintf("%s.%s", modName, ic.Name)

	if len(ic.Images) == 0 {
		fmt.Fprintf(output, "create or modify image collection %s", qualifiedName)
		if exportLevel != "Hidden" {
			fmt.Fprintf(output, " export level '%s'", exportLevel)
		}
		fmt.Fprintln(output, ";")
		fmt.Fprintln(output, "/")
		return nil
	}

	previewDir := filepath.Join("/tmp/mxcli-preview", qualifiedName)
	if err := os.MkdirAll(previewDir, 0o755); err != nil {
		return mdlerrors.NewBackend("create preview directory", err)
	}

	fmt.Fprintf(output, "create or modify image collection %s", qualifiedName)
	if exportLevel != "Hidden" {
		fmt.Fprintf(output, " export level '%s'", exportLevel)
	}
	fmt.Fprintln(output, " (")

	for i, img := range ic.Images {
		ext := imageFormatToExt(img.Format)
		filePath := filepath.Join(previewDir, img.Name+ext)
		if len(img.Data) > 0 {
			if err := os.WriteFile(filePath, img.Data, 0o644); err != nil {
				return mdlerrors.NewBackend(fmt.Sprintf("write image %s", img.Name), err)
			}
		}

		comma := ","
		if i == len(ic.Images)-1 {
			comma = ""
		}
		fmt.Fprintf(output, "    image %s from file '%s'%s\n", img.Name, filePath, comma)
	}

	fmt.Fprintln(output, ");")
	fmt.Fprintln(output, "/")
	return nil
}

func listImageCollectionsFn(ctx context.Context, output io.Writer, format OutputFormat, deps *HandlerDeps, moduleName string) error {
	collections, err := deps.ImageCollectionWriter.ListImageCollections()
	if err != nil {
		return mdlerrors.NewBackend("list image collections", err)
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return err
	}

	result := &TableResult{
		Columns: []string{"Image Collection", "Export Level", "Images"},
	}

	for _, ic := range collections {
		modID := h.FindModuleID(ic.ContainerID)
		modName := h.GetModuleName(modID)
		if moduleName != "" && modName != moduleName {
			continue
		}

		qualifiedName := fmt.Sprintf("%s.%s", modName, ic.Name)
		exportLevel := ic.ExportLevel
		if exportLevel == "" {
			exportLevel = "Hidden"
		}
		result.Rows = append(result.Rows, []any{qualifiedName, exportLevel, len(ic.Images)})
	}

	result.Summary = fmt.Sprintf("(%d image collection(s))", len(result.Rows))
	return writeResultTo(output, format, result)
}

func findImageCollectionFn(deps *HandlerDeps, moduleName, collectionName string) *types.ImageCollection {
	collections, err := deps.ImageCollectionWriter.ListImageCollections()
	if err != nil {
		return nil
	}

	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return nil
	}

	for _, ic := range collections {
		modID := h.FindModuleID(ic.ContainerID)
		modName := h.GetModuleName(modID)
		if ic.Name == collectionName && modName == moduleName {
			return ic
		}
	}
	return nil
}

func execAlterImageCollectionFn(ctx context.Context, s *ast.AlterImageCollectionStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}

	ic := findImageCollectionFn(deps, s.Name.Module, s.Name.Name)
	if ic == nil {
		return mdlerrors.NewNotFound("image collection", s.Name.String())
	}

	dirty := false

	for _, rawAction := range s.Actions {
		switch action := rawAction.(type) {

		case *ast.AddImageAction:
			data, format, err := readImageFile(action.FilePath)
			if err != nil {
				return err
			}
			ic.Images = append(ic.Images, types.Image{
				Name:   action.ImageName,
				Data:   data,
				Format: format,
			})
			dirty = true
			fmt.Fprintf(deps.Output, "Added image %q to %s\n", action.ImageName, s.Name)

		case *ast.DropImageAction:
			idx := findImageIndex(ic, action.ImageName)
			if idx < 0 {
				return mdlerrors.NewNotFound("image", action.ImageName)
			}
			ic.Images = append(ic.Images[:idx], ic.Images[idx+1:]...)
			dirty = true
			fmt.Fprintf(deps.Output, "Dropped image %q from %s\n", action.ImageName, s.Name)

		case *ast.RenameImageAction:
			idx := findImageIndex(ic, action.From)
			if idx < 0 {
				return mdlerrors.NewNotFound("image", action.From)
			}
			if findImageIndex(ic, action.To) >= 0 {
				return mdlerrors.NewAlreadyExists("image", action.To)
			}
			ic.Images[idx].Name = action.To
			dirty = true
			fmt.Fprintf(deps.Output, "Renamed image %q to %q in %s\n", action.From, action.To, s.Name)

		case *ast.SetImageAction:
			idx := findImageIndex(ic, action.ImageName)
			if idx < 0 {
				return mdlerrors.NewNotFound("image", action.ImageName)
			}
			data, format, err := readImageFile(action.FilePath)
			if err != nil {
				return err
			}
			ic.Images[idx].Data = data
			ic.Images[idx].Format = format
			dirty = true
			fmt.Fprintf(deps.Output, "Updated image %q in %s\n", action.ImageName, s.Name)

		case *ast.MoveImageCollectionAction:
			if dirty {
				if err := deps.ImageCollectionWriter.UpdateImageCollection(ic); err != nil {
					return mdlerrors.NewBackend("update image collection before move", err)
				}
				dirty = false
			}
			targetMod, err := findModuleFn(deps.ModuleLister, action.Target.Module)
			if err != nil {
				return mdlerrors.NewNotFound("module", action.Target.Module)
			}
			ic.ContainerID = targetMod.ID
			if err := deps.ImageCollectionWriter.MoveImageCollection(ic); err != nil {
				return mdlerrors.NewBackend("move image collection", err)
			}
			if deps.CacheInvalidator != nil {
				deps.CacheInvalidator.InvalidateCache()
			}
			fmt.Fprintf(deps.Output, "Moved image collection %s to module %s\n", s.Name, action.Target.Module)

		case *ast.ExportImageAction:
			idx := findImageIndex(ic, action.ImageName)
			if idx < 0 {
				return mdlerrors.NewNotFound("image", action.ImageName)
			}
			filePath := action.FilePath
			if !filepath.IsAbs(filePath) {
				cwd, err := os.Getwd()
				if err != nil {
					return mdlerrors.NewBackend("get working directory", err)
				}
				filePath = filepath.Join(cwd, filePath)
			}
			if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
				return mdlerrors.NewBackend("create output directory", err)
			}
			if err := os.WriteFile(filePath, ic.Images[idx].Data, 0o644); err != nil {
				return mdlerrors.NewBackend(fmt.Sprintf("write image file %q", filePath), err)
			}
			fmt.Fprintf(deps.Output, "Exported image %q to %s\n", action.ImageName, filePath)
		}
	}

	if dirty {
		if err := deps.ImageCollectionWriter.UpdateImageCollection(ic); err != nil {
			return mdlerrors.NewBackend("update image collection", err)
		}
	}

	return nil
}

// findModuleFn finds a module by name via ModuleLister.
func findModuleFn(ml backend.ModuleLister, name string) (*model.Module, error) {
	modules, err := ml.ListModules()
	if err != nil {
		return nil, mdlerrors.NewBackend("list modules", err)
	}
	for _, m := range modules {
		if m.Name == name {
			return m, nil
		}
	}
	return nil, mdlerrors.NewNotFound("module", name)
}

// ────────────────────────────────────────────────────────────
// Old ExecContext wrappers (delegate to Fn versions)
// ────────────────────────────────────────────────────────────



func describeImageCollection(ctx *ExecContext, name ast.QualifiedName) error {
	deps := execContextToDeps(ctx)
	return describeImageCollectionFn(ctx, deps.Output, deps, name)
}

func listImageCollections(ctx *ExecContext, moduleName string) error {
	deps := execContextToDeps(ctx)
	return listImageCollectionsFn(ctx, deps.Output, deps.Format, deps, moduleName)
}

func findImageCollection(ctx *ExecContext, moduleName, collectionName string) *types.ImageCollection {
	return findImageCollectionFn(execContextToDeps(ctx), moduleName, collectionName)
}


// ────────────────────────────────────────────────────────────
// Stateless helpers (no ctx/deps needed)
// ────────────────────────────────────────────────────────────

// imageFormatToExt converts a Mendix ImageFormat value to a file extension.
func imageFormatToExt(format string) string {
	switch format {
	case "Svg":
		return ".svg"
	case "Gif":
		return ".gif"
	case "Jpg":
		return ".jpg"
	case "Bmp":
		return ".bmp"
	case "Webp":
		return ".webp"
	default:
		return ".png"
	}
}

// extToImageFormat converts a file extension to a Mendix ImageFormat value.
func extToImageFormat(ext string) string {
	switch strings.ToLower(ext) {
	case ".svg":
		return "Svg"
	case ".gif":
		return "Gif"
	case ".jpg", ".jpeg":
		return "Jpg"
	case ".bmp":
		return "Bmp"
	case ".webp":
		return "Webp"
	default:
		return "Png"
	}
}

// readImageFile reads an image file and returns (data, format, error).
func readImageFile(filePath string) ([]byte, string, error) {
	if !filepath.IsAbs(filePath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, "", mdlerrors.NewBackend("get working directory", err)
		}
		filePath = filepath.Join(cwd, filePath)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", mdlerrors.NewBackend(fmt.Sprintf("read image file %q", filePath), err)
	}
	return data, extToImageFormat(filepath.Ext(filePath)), nil
}

// findImageIndex returns the index of the image with the given name, or -1.
func findImageIndex(ic *types.ImageCollection, name string) int {
	for i, img := range ic.Images {
		if img.Name == name {
			return i
		}
	}
	return -1
}
