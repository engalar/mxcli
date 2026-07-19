// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/modelsdk/widgets/mpk"
)

// DiscoveredWidget holds metadata for a widget found in widgets/*.mpk but not
// yet loaded into the registry (no embedded template or .def.json extracted).
type DiscoveredWidget struct {
	WidgetID    string // e.g. "com.mendix.widget.web.combobox.Combobox"
	Name        string // display name from widget XML, e.g. "Combo box"
	Description string // description from widget XML
}

// WidgetRegistry holds loaded widget definitions keyed by uppercase MDL name.
type WidgetRegistry struct {
	byMDLName       map[string]*WidgetDefinition // keyed by uppercase MDLName
	byWidgetID      map[string]*WidgetDefinition // keyed by widgetId
	knownOperations map[string]bool              // operations accepted during validation
	projectDir      string                       // project root for MPK fallback
	mpkNameMap      map[string]string            // uppercase MDLName → widgetID (pre-scan, legacy)
	mpkDiscovered   map[string]*DiscoveredWidget // uppercase MDLName → full discovery info
	mxGraph         *mxgraph.Graph               // mxgraph index for fast widget lookup
}

// defaultKnownOperations is the set of operation names supported by the widget engine.
var defaultKnownOperations = map[string]bool{
	"attribute":        true,
	"association":      true,
	"primitive":        true,
	"selection":        true,
	"expression":       true,
	"datasource":       true,
	"widgets":          true,
	"texttemplate":     true,
	"action":           true,
	"attributeObjects": true,
}

// knownOperations is the active set used for validation, initialized from
// defaultKnownOperations and now stored per-registry to avoid global mutable state.

func copyOps(src map[string]bool) map[string]bool {
	dst := make(map[string]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// NewWidgetRegistry creates a registry pre-loaded with embedded definitions.
// Uses the default set of known operations for validation.
func NewWidgetRegistry() (*WidgetRegistry, error) {
	return NewWidgetRegistryWithOps(nil)
}

// NewWidgetRegistryWithOps creates a registry pre-loaded with embedded definitions,
// extending the default known operations with extraOps for validation.
// This allows user-defined widgets to declare custom operations that would otherwise
// fail validation. Pass nil for the default set.
func NewWidgetRegistryWithOps(extraOps map[string]bool) (*WidgetRegistry, error) {
	ops := copyOps(defaultKnownOperations)
	for op := range extraOps {
		ops[op] = true
	}

	reg := &WidgetRegistry{
		byMDLName:       make(map[string]*WidgetDefinition),
		byWidgetID:      make(map[string]*WidgetDefinition),
		knownOperations: ops,
		mpkNameMap:      make(map[string]string),
		mpkDiscovered:   make(map[string]*DiscoveredWidget),
	}

	for _, def := range embeddedDefinitions() {
		def.MDLName = strings.ToLower(def.MDLName)
		for i := range def.ChildSlots {
			def.ChildSlots[i].MDLContainer = strings.ToLower(def.ChildSlots[i].MDLContainer)
		}
		if err := reg.validateDefinitionOperations(def, def.WidgetID); err != nil {
			return nil, err
		}
		reg.byMDLName[strings.ToUpper(def.MDLName)] = def
		reg.byWidgetID[def.WidgetID] = def
	}

	return reg, nil
}

// Get returns a widget definition by MDL name (case-insensitive).
func (r *WidgetRegistry) Get(mdlName string) (*WidgetDefinition, bool) {
	name := strings.ToUpper(mdlName)
	if def, ok := r.byMDLName[name]; ok {
		return def, ok
	}
	if r.projectDir == "" {
		return nil, false
	}
	widgetID, ok := r.mpkNameMap[name]
	if !ok {
		return nil, false
	}
	def, err := r.deriveFromMPK(widgetID)
	if err != nil {
		log.Printf("warning: MPK fallback for %s: %v", name, err)
		return nil, false
	}
	if def == nil {
		return nil, false
	}
	r.byMDLName[strings.ToUpper(def.MDLName)] = def
	r.byWidgetID[def.WidgetID] = def
	return def, true
}

// GetByWidgetID returns a widget definition by its full widget ID.
func (r *WidgetRegistry) GetByWidgetID(widgetID string) (*WidgetDefinition, bool) {
	if def, ok := r.byWidgetID[widgetID]; ok {
		return def, ok
	}

	// Fast path: check mxgraph index (populated from .mxcli/graph.gob snapshot)
	if r.mxGraph != nil {
		nodes := r.mxGraph.FindNodes("Widget", map[string]any{"WidgetID": widgetID})
		if len(nodes) > 0 {
			def := widgetDefinitionFromNode(nodes[0])
			if def != nil {
				r.byWidgetID[widgetID] = def
				r.byMDLName[strings.ToUpper(def.MDLName)] = def
				return def, true
			}
		}
	}

	if r.projectDir == "" {
		return nil, false
	}
	def, err := r.deriveFromMPK(widgetID)
	if err != nil {
		log.Printf("warning: MPK fallback for widget ID %s: %v", widgetID, err)
		return nil, false
	}
	if def == nil {
		return nil, false
	}
	r.byMDLName[strings.ToUpper(def.MDLName)] = def
	r.byWidgetID[def.WidgetID] = def
	return def, true
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

// MPKDiscovered returns widgets found in the project's widgets/*.mpk files that
// are not yet loaded into the registry (no embedded definition, no .def.json).
// Call SetProjectDir first to populate this map.
// The map key is the uppercase MDL name derived from the widget ID.
func (r *WidgetRegistry) MPKDiscovered() map[string]*DiscoveredWidget {
	result := make(map[string]*DiscoveredWidget, len(r.mpkDiscovered))
	for k, v := range r.mpkDiscovered {
		result[k] = v
	}
	return result
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
	} else {
		log.Printf("warning: cannot determine home directory for user widget definitions: %v", err)
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
		if os.IsNotExist(err) {
			return nil
		}
		log.Printf("warning: cannot read widget definitions from %s: %v", dir, err)
		return nil
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".def.json") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return mdlerrors.NewBackend(fmt.Sprintf("read %s", filePath), err)
		}

		var def WidgetDefinition
		if err := json.Unmarshal(data, &def); err != nil {
			return mdlerrors.NewBackend(fmt.Sprintf("parse %s", filePath), err)
		}

		if def.WidgetID == "" || def.MDLName == "" {
			return mdlerrors.NewValidationf("invalid definition %s: widgetId and mdlName are required", entry.Name())
		}

		if err := r.validateDefinitionOperations(&def, entry.Name()); err != nil {
			return err
		}

		def.MDLName = strings.ToLower(def.MDLName)
		for i := range def.ChildSlots {
			def.ChildSlots[i].MDLContainer = strings.ToLower(def.ChildSlots[i].MDLContainer)
		}
		upperName := strings.ToUpper(def.MDLName)
		if existing, ok := r.byMDLName[upperName]; ok {
			// Skip user skeleton definitions (no mappings/modes) when built-in has mappings
			if len(def.PropertyMappings) == 0 && len(def.Modes) == 0 &&
				(len(existing.PropertyMappings) > 0 || len(existing.Modes) > 0) {
				log.Printf("info: skipping user skeleton %q — built-in %s has mappings", entry.Name(), def.MDLName)
				continue
			}
			log.Printf("info: user definition %q overrides built-in %s (widgetId: %s → %s)",
				entry.Name(), def.MDLName, existing.WidgetID, def.WidgetID)
		}
		r.byMDLName[upperName] = &def
		r.byWidgetID[def.WidgetID] = &def
	}
	return nil
}

// validateDefinitionOperations checks that all operation names in a definition
// are recognized by the known operations set, and validates source/operation
// compatibility and mapping order dependencies.
func (r *WidgetRegistry) validateDefinitionOperations(def *WidgetDefinition, source string) error {
	if err := r.validateMappings(def.PropertyMappings, source, ""); err != nil {
		return err
	}
	for _, s := range def.ChildSlots {
		if !r.knownOperations[s.Operation] {
			return mdlerrors.NewValidationf("%s: unknown operation %q in childSlots for key %q", source, s.Operation, s.PropertyKey)
		}
	}
	for _, mode := range def.Modes {
		ctx := fmt.Sprintf("mode %q ", mode.Name)
		if err := r.validateMappings(mode.PropertyMappings, source, ctx); err != nil {
			return err
		}
		for _, s := range mode.ChildSlots {
			if !r.knownOperations[s.Operation] {
				return mdlerrors.NewValidationf("%s: unknown operation %q in %schildSlots for key %q", source, s.Operation, ctx, s.PropertyKey)
			}
		}
	}
	return nil
}

// sourceOperationCompatible checks that a mapping's Source and Operation are compatible.
var incompatibleSourceOps = map[string]map[string]bool{
	"Attribute":      {"association": true, "datasource": true},
	"Attributes":     {"association": true, "datasource": true, "attribute": true},
	"FirstAttribute": {"association": true, "datasource": true, "attributeObjects": true},
	"Association":    {"attribute": true, "datasource": true},
	"DataSource":     {"attribute": true, "association": true},
}

// validateMappings validates a slice of property mappings for operation existence,
// source/operation compatibility, and mapping order (Association requires prior DataSource).
func (r *WidgetRegistry) validateMappings(mappings []PropertyMapping, source, modeCtx string) error {
	hasDataSource := false
	for _, m := range mappings {
		if !r.knownOperations[m.Operation] {
			return mdlerrors.NewValidationf("%s: unknown operation %q in %spropertyMappings for key %q", source, m.Operation, modeCtx, m.PropertyKey)
		}
		// Check source/operation compatibility
		if incompatible, ok := incompatibleSourceOps[m.Source]; ok {
			if incompatible[m.Operation] {
				return mdlerrors.NewValidationf("%s: incompatible source %q with operation %q in %spropertyMappings for key %q",
					source, m.Source, m.Operation, modeCtx, m.PropertyKey)
			}
		}
		// Track DataSource ordering
		if m.Source == "DataSource" {
			hasDataSource = true
		}
		// Association depends on entityContext set by a prior DataSource mapping
		if m.Source == "Association" && !hasDataSource {
			return mdlerrors.NewValidationf("%s: %spropertyMappings key %q uses source 'Association' before any 'DataSource' mapping — entityContext will be stale",
				source, modeCtx, m.PropertyKey)
		}
	}
	return nil
}

// SetProjectDir enables real-time MPK fallback for this registry.
func (r *WidgetRegistry) SetProjectDir(projectDir string) error {
	r.projectDir = projectDir
	r.mpkNameMap = make(map[string]string)
	r.mpkDiscovered = make(map[string]*DiscoveredWidget)
	return r.preScanWidgets(projectDir)
}

// SetMxGraph injects an mxgraph index for fast widget definition lookup.
// Called by initPluggableEngine when a graph snapshot is available.
func (r *WidgetRegistry) SetMxGraph(g *mxgraph.Graph) {
	r.mxGraph = g
}

func (r *WidgetRegistry) preScanWidgets(projectDir string) error {
	widgetsDir := filepath.Join(projectDir, "widgets")
	matches, err := filepath.Glob(filepath.Join(widgetsDir, "*.mpk"))
	if err != nil {
		return fmt.Errorf("scan widgets dir: %w", err)
	}
	for _, mpkPath := range matches {
		defs, err := mpk.ParseAll(mpkPath)
		if err != nil {
			log.Printf("warning: widget pre-scan skipping %s: %v", filepath.Base(mpkPath), err)
			continue
		}
		for _, d := range defs {
			name := strings.ToUpper(lastIDSegment(d.ID))
			if _, exists := r.byMDLName[name]; exists {
				continue // builtin or user-override wins
			}
			r.mpkNameMap[name] = d.ID
			r.mpkDiscovered[name] = &DiscoveredWidget{
				WidgetID:    d.ID,
				Name:        d.Name,
				Description: d.Description,
			}
		}
	}
	return nil
}

func (r *WidgetRegistry) deriveFromMPK(widgetID string) (*WidgetDefinition, error) {
	mpkPath, err := mpk.FindMPK(r.projectDir, widgetID)
	if err != nil {
		return nil, fmt.Errorf("find mpk for %s: %w", widgetID, err)
	}
	if mpkPath == "" {
		return nil, nil
	}
	mpkDef, err := mpk.ParseMPKForWidget(mpkPath, widgetID)
	if err != nil {
		return nil, fmt.Errorf("parse mpk for %s: %w", widgetID, err)
	}
	if mpkDef == nil {
		return nil, nil
	}
	return buildDefinitionFromMPK(mpkDef), nil
}

// lastIDSegment returns the last dot-separated segment of a widget ID, lowercased.
func lastIDSegment(widgetID string) string {
	parts := strings.Split(widgetID, ".")
	return strings.ToLower(parts[len(parts)-1])
}

// widgetDefinitionFromNode converts an mxgraph Widget node back to a
// WidgetDefinition. Only populates fields that the index stores — the
// caller gets a minimal definition sufficient for engine lookups.
func widgetDefinitionFromNode(n *mxgraph.Node) *WidgetDefinition {
	widgetID, _ := n.Props["WidgetID"].(string)
	mdlName, _ := n.Props["MDLName"].(string)
	widgetKind, _ := n.Props["WidgetKind"].(string)
	if widgetID == "" {
		return nil
	}
	return &WidgetDefinition{
		WidgetID:        widgetID,
		MDLName:         strings.ToLower(mdlName),
		WidgetKind:      widgetKind,
		TemplateFile:    strings.ToLower(mdlName) + ".json",
		DefaultEditable: "Always",
	}
}

func buildDefinitionFromMPK(mpkDef *mpk.WidgetDefinition) *WidgetDefinition {
	mdlName := lastIDSegment(mpkDef.ID)
	widgetKind := "custom"
	if mpkDef.IsPluggable {
		widgetKind = "pluggable"
	}
	def := &WidgetDefinition{
		WidgetID:        mpkDef.ID,
		MDLName:         mdlName,
		WidgetKind:      widgetKind,
		TemplateFile:    mdlName + ".json",
		DefaultEditable: "Always",
	}

	var assocMappings []PropertyMapping
	for _, p := range mpkDef.Properties {
		switch p.Type {
		case "widgets":
			container := strings.ToUpper(p.Key)
			if p.Key == "content" {
				container = "TEMPLATE"
			}
			def.ChildSlots = append(def.ChildSlots, ChildSlotMapping{
				PropertyKey:  p.Key,
				MDLContainer: strings.ToLower(container),
				Operation:    "widgets",
			})
		case "datasource":
			def.PropertyMappings = append(def.PropertyMappings, PropertyMapping{
				PropertyKey: p.Key,
				Source:      "DataSource",
				Operation:   "datasource",
			})
		case "attribute":
			def.PropertyMappings = append(def.PropertyMappings, PropertyMapping{
				PropertyKey: p.Key,
				Source:      "Attribute",
				Operation:   "attribute",
			})
		case "association":
			assocMappings = append(assocMappings, PropertyMapping{
				PropertyKey: p.Key,
				Source:      "Association",
				Operation:   "association",
			})
		case "selection":
			def.PropertyMappings = append(def.PropertyMappings, PropertyMapping{
				PropertyKey: p.Key,
				Source:      "Selection",
				Operation:   "selection",
				Default:     p.DefaultValue,
			})
		case "boolean", "integer", "decimal", "string", "enumeration":
			m := PropertyMapping{
				PropertyKey: p.Key,
				Operation:   "primitive",
			}
			if p.DefaultValue != "" {
				m.Value = p.DefaultValue
			}
			def.PropertyMappings = append(def.PropertyMappings, m)
		}
	}
	def.PropertyMappings = append(def.PropertyMappings, assocMappings...)
	return def
}

// ---------------------------------------------------------------------------
// Built-in registry singleton
// ---------------------------------------------------------------------------

var (
	builtinOnce     sync.Once
	builtinRegistry *WidgetRegistry
)

// BuiltinWidgetDef returns the WidgetDefinition for the given widget ID from the
// embedded built-in registry, or nil if not found. The registry is initialized
// once on first call and is safe for concurrent use. It is intended for use in
// describe/read contexts that have no access to the pageBuilder or project path.
// embeddedDefinitions returns the built-in widget definitions that were
// previously loaded from modelsdk/widgets/definitions/*.def.json.
func embeddedDefinitions() []*WidgetDefinition {
	return []*WidgetDefinition{
		{
			WidgetID: "com.mendix.widget.web.combobox.Combobox", MDLName: "COMBOBOX",
			TemplateFile: "combobox.json", DefaultEditable: "Always",
			Modes: []WidgetMode{
				{Name: "association", Condition: "hasDataSource", Description: "Association mode",
					PropertyMappings: []PropertyMapping{
						{PropertyKey: "optionsSourceType", Value: "association", Operation: "primitive"},
						{PropertyKey: "optionsSourceAssociationDataSource", Source: "DataSource", Operation: "datasource"},
						{PropertyKey: "attributeAssociation", Source: "Association", Operation: "association"},
						{PropertyKey: "optionsSourceAssociationCaptionAttribute", Source: "CaptionAttribute", Operation: "attribute"},
					}},
				{Name: "default", Description: "Enumeration mode",
					PropertyMappings: []PropertyMapping{
						{PropertyKey: "attributeEnumeration", Source: "Attribute", Operation: "attribute"},
					}},
			},
		},
		{
			WidgetID: "com.mendix.widget.web.gallery.Gallery", MDLName: "GALLERY",
			TemplateFile: "gallery.json", DefaultEditable: "Always",
			PropertyMappings: []PropertyMapping{
				{PropertyKey: "advanced", Value: "false", Operation: "primitive"},
				{PropertyKey: "datasource", Source: "DataSource", Operation: "datasource"},
				{PropertyKey: "itemSelection", Source: "Selection", Operation: "selection", Default: "Single"},
				{PropertyKey: "itemSelectionMode", Value: "clear", Operation: "primitive"},
				{PropertyKey: "desktopItems", Source: "DesktopColumns", Default: "1", Operation: "primitive"},
				{PropertyKey: "tabletItems", Source: "TabletColumns", Default: "1", Operation: "primitive"},
				{PropertyKey: "phoneItems", Source: "PhoneColumns", Default: "1", Operation: "primitive"},
				{PropertyKey: "pageSize", Value: "20", Operation: "primitive"},
				{PropertyKey: "pagination", Value: "buttons", Operation: "primitive"},
				{PropertyKey: "pagingPosition", Value: "below", Operation: "primitive"},
				{PropertyKey: "showEmptyPlaceholder", Value: "none", Operation: "primitive"},
				{PropertyKey: "onClickTrigger", Value: "single", Operation: "primitive"},
			},
			ChildSlots: []ChildSlotMapping{
				{PropertyKey: "content", MDLContainer: "TEMPLATE", Operation: "widgets"},
				{PropertyKey: "emptyPlaceholder", MDLContainer: "EMPTYPLACEHOLDER", Operation: "widgets"},
				{PropertyKey: "filtersPlaceholder", MDLContainer: "FILTER", Operation: "widgets"},
			},
		},
		{
			WidgetID: "com.mendix.widget.web.image.Image", MDLName: "IMAGE",
			TemplateFile: "image.json", DefaultEditable: "Always",
			PropertyMappings: []PropertyMapping{
				{PropertyKey: "datasource", Source: "ImageType", Default: "image", Operation: "primitive"},
				{PropertyKey: "imageUrl", Source: "ImageUrl", Operation: "texttemplate"},
				{PropertyKey: "alternativeText", Source: "AlternativeText", Operation: "texttemplate"},
				{PropertyKey: "onClick", Source: "OnClick", Operation: "action"},
				{PropertyKey: "onClickType", Source: "OnClickType", Default: "action", Operation: "primitive"},
				{PropertyKey: "widthUnit", Source: "WidthUnit", Default: "auto", Operation: "primitive"},
				{PropertyKey: "width", Source: "Width", Default: "100", Operation: "primitive"},
				{PropertyKey: "heightUnit", Source: "HeightUnit", Default: "auto", Operation: "primitive"},
				{PropertyKey: "height", Source: "Height", Default: "100", Operation: "primitive"},
				{PropertyKey: "iconSize", Source: "IconSize", Default: "14", Operation: "primitive"},
				{PropertyKey: "displayAs", Source: "DisplayAs", Default: "fullImage", Operation: "primitive"},
				{PropertyKey: "responsive", Source: "Responsive", Default: "true", Operation: "primitive"},
				{PropertyKey: "isBackgroundImage", Source: "IsBackgroundImage", Default: "false", Operation: "primitive"},
			},
			ChildSlots: []ChildSlotMapping{
				{PropertyKey: "children", MDLContainer: "CONTENT", Operation: "widgets"},
			},
		},
		{
			WidgetID: "com.mendix.widget.web.barcodescanner.BarcodeScanner", MDLName: "BARCODESCANNER",
			TemplateFile: "barcodescanner.json", DefaultEditable: "Always",
			PropertyMappings: []PropertyMapping{
				{PropertyKey: "datasource", Source: "Attribute", Operation: "attribute"},
			},
		},
		{
			WidgetID: "com.mendix.widget.web.datagridtextfilter.DatagridTextFilter", MDLName: "TEXTFILTER",
			TemplateFile: "datagrid-text-filter.json", DefaultEditable: "Always",
			PropertyMappings: []PropertyMapping{
				{PropertyKey: "attrChoice", Value: "linked", Operation: "primitive"},
				{PropertyKey: "attributes", Source: "Attributes", Operation: "attributeObjects"},
				{PropertyKey: "defaultFilter", Source: "FilterType", Operation: "primitive"},
			},
		},
		{
			WidgetID: "com.mendix.widget.web.datagridnumberfilter.DatagridNumberFilter", MDLName: "NUMBERFILTER",
			TemplateFile: "datagrid-number-filter.json", DefaultEditable: "Always",
			PropertyMappings: []PropertyMapping{
				{PropertyKey: "attrChoice", Value: "linked", Operation: "primitive"},
				{PropertyKey: "attributes", Source: "Attributes", Operation: "attributeObjects"},
				{PropertyKey: "defaultFilter", Source: "FilterType", Operation: "primitive"},
			},
		},
		{
			WidgetID: "com.mendix.widget.web.datagriddatefilter.DatagridDateFilter", MDLName: "DATEFILTER",
			TemplateFile: "datagrid-date-filter.json", DefaultEditable: "Always",
			PropertyMappings: []PropertyMapping{
				{PropertyKey: "attrChoice", Value: "linked", Operation: "primitive"},
				{PropertyKey: "attributes", Source: "Attributes", Operation: "attributeObjects"},
				{PropertyKey: "defaultFilter", Source: "FilterType", Operation: "primitive"},
			},
		},
		{
			WidgetID: "com.mendix.widget.web.datagriddropdownfilter.DatagridDropdownFilter", MDLName: "DROPDOWNFILTER",
			TemplateFile: "datagrid-dropdown-filter.json", DefaultEditable: "Always",
			PropertyMappings: []PropertyMapping{
				{PropertyKey: "attrChoice", Value: "custom", Operation: "primitive"},
				{PropertyKey: "attr", Source: "FirstAttribute", Operation: "attribute"},
			},
		},
		{
			WidgetID: "com.mendix.widget.web.dropdownsort.DropdownSort", MDLName: "DROPDOWNSORT",
			TemplateFile: "dropdownsort.json", DefaultEditable: "Always",
		},
	}
}

func BuiltinWidgetDef(widgetID string) *WidgetDefinition {
	builtinOnce.Do(func() {
		reg, err := NewWidgetRegistry()
		if err != nil {
			log.Printf("warning: failed to load built-in widget registry: %v", err)
			builtinRegistry = &WidgetRegistry{
				byWidgetID: make(map[string]*WidgetDefinition),
			}
			return
		}
		builtinRegistry = reg
	})
	if def, ok := builtinRegistry.GetByWidgetID(widgetID); ok {
		return def
	}
	return nil
}
