// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mendixlabs/mxcli/sdk/widgets/definitions"
)

// WidgetRegistry holds loaded widget definitions keyed by uppercase MDL name.
type WidgetRegistry struct {
	byMDLName  map[string]*WidgetDefinition // keyed by uppercase MDLName
	byWidgetID map[string]*WidgetDefinition // keyed by widgetId
}

// NewWidgetRegistry creates a registry pre-loaded with embedded definitions.
func NewWidgetRegistry() (*WidgetRegistry, error) {
	reg := &WidgetRegistry{
		byMDLName:  make(map[string]*WidgetDefinition),
		byWidgetID: make(map[string]*WidgetDefinition),
	}

	entries, err := definitions.EmbeddedFS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read embedded definitions: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".def.json") {
			continue
		}

		data, err := definitions.EmbeddedFS.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read definition %s: %w", entry.Name(), err)
		}

		var def WidgetDefinition
		if err := json.Unmarshal(data, &def); err != nil {
			return nil, fmt.Errorf("parse definition %s: %w", entry.Name(), err)
		}

		reg.byMDLName[strings.ToUpper(def.MDLName)] = &def
		reg.byWidgetID[def.WidgetID] = &def
	}

	return reg, nil
}

// Get returns a widget definition by MDL name (case-insensitive).
func (r *WidgetRegistry) Get(mdlName string) (*WidgetDefinition, bool) {
	def, ok := r.byMDLName[strings.ToUpper(mdlName)]
	return def, ok
}

// GetByWidgetID returns a widget definition by its full widget ID.
func (r *WidgetRegistry) GetByWidgetID(widgetID string) (*WidgetDefinition, bool) {
	def, ok := r.byWidgetID[widgetID]
	return def, ok
}

// All returns all registered definitions.
func (r *WidgetRegistry) All() []*WidgetDefinition {
	result := make([]*WidgetDefinition, 0, len(r.byMDLName))
	for _, def := range r.byMDLName {
		result = append(result, def)
	}
	return result
}

// Count returns the number of registered definitions.
func (r *WidgetRegistry) Count() int {
	return len(r.byMDLName)
}

// LoadUserDefinitions scans global and project-level directories for user-provided definitions.
// Project definitions override global ones with the same MDL name.
func (r *WidgetRegistry) LoadUserDefinitions(projectPath string) error {
	// 1. Global: ~/.mxcli/widgets/*.def.json
	homeDir, err := os.UserHomeDir()
	if err == nil {
		globalDir := filepath.Join(homeDir, ".mxcli", "widgets")
		if err := r.loadDefinitionsFromDir(globalDir); err != nil {
			return fmt.Errorf("global widgets: %w", err)
		}
	}

	// 2. Project: <projectDir>/.mxcli/widgets/*.def.json (overrides global)
	if projectPath != "" {
		projectDir := filepath.Dir(projectPath)
		localDir := filepath.Join(projectDir, ".mxcli", "widgets")
		if err := r.loadDefinitionsFromDir(localDir); err != nil {
			return fmt.Errorf("project widgets: %w", err)
		}
	}

	return nil
}

// loadDefinitionsFromDir loads all .def.json files from a directory.
// Returns nil if the directory doesn't exist; returns errors for malformed files.
func (r *WidgetRegistry) loadDefinitionsFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // directory doesn't exist or not readable — not an error
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".def.json") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read %s: %w", filePath, err)
		}

		var def WidgetDefinition
		if err := json.Unmarshal(data, &def); err != nil {
			return fmt.Errorf("parse %s: %w", filePath, err)
		}

		if def.WidgetID == "" || def.MDLName == "" {
			return fmt.Errorf("invalid definition %s: widgetId and mdlName are required", entry.Name())
		}

		r.byMDLName[strings.ToUpper(def.MDLName)] = &def
		r.byWidgetID[def.WidgetID] = &def
	}
	return nil
}
