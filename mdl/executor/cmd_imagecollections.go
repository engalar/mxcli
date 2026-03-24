// SPDX-License-Identifier: Apache-2.0

// Package executor - Image collection commands (CREATE/DROP IMAGE COLLECTION)
package executor

import (
	"fmt"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// execCreateImageCollection handles CREATE IMAGE COLLECTION statements.
func (e *Executor) execCreateImageCollection(s *ast.CreateImageCollectionStmt) error {
	if e.reader == nil {
		return fmt.Errorf("not connected to a project")
	}

	// Find module
	module, err := e.findModule(s.Name.Module)
	if err != nil {
		return err
	}

	// Check if image collection already exists
	existing := e.findImageCollection(s.Name.Module, s.Name.Name)
	if existing != nil {
		return fmt.Errorf("image collection already exists: %s.%s", s.Name.Module, s.Name.Name)
	}

	// Build ImageCollection
	ic := &mpr.ImageCollection{
		ContainerID:   module.ID,
		Name:          s.Name.Name,
		ExportLevel:   s.ExportLevel,
		Documentation: s.Comment,
	}

	if err := e.writer.CreateImageCollection(ic); err != nil {
		return fmt.Errorf("failed to create image collection: %w", err)
	}

	// Invalidate hierarchy cache so the new collection's container is visible
	e.invalidateHierarchy()

	fmt.Fprintf(e.output, "Created image collection: %s\n", s.Name)
	return nil
}

// execDropImageCollection handles DROP IMAGE COLLECTION statements.
func (e *Executor) execDropImageCollection(s *ast.DropImageCollectionStmt) error {
	if e.reader == nil {
		return fmt.Errorf("not connected to a project")
	}

	ic := e.findImageCollection(s.Name.Module, s.Name.Name)
	if ic == nil {
		return fmt.Errorf("image collection not found: %s", s.Name)
	}

	if err := e.writer.DeleteImageCollection(string(ic.ID)); err != nil {
		return fmt.Errorf("failed to delete image collection: %w", err)
	}

	fmt.Fprintf(e.output, "Dropped image collection: %s\n", s.Name)
	return nil
}

// findImageCollection finds an image collection by module and name.
func (e *Executor) findImageCollection(moduleName, collectionName string) *mpr.ImageCollection {
	collections, err := e.reader.ListImageCollections()
	if err != nil {
		return nil
	}

	h, err := e.getHierarchy()
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
