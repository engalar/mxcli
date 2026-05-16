// SPDX-License-Identifier: Apache-2.0

// Package mprread — gen-typed document reader functions.
//
// These functions live here (not on *mpr.Reader) because the codec → mpr import
// edge already exists for write/binary helpers, so the mpr package itself
// cannot import codec. mprread sits one level above both, depending on each.
//
// Each function follows the same shape: list units of a given BSON $Type
// prefix, decode via codec.DefaultRegistry (through ListUnitsByType[T]), and
// return the gen-typed pointers.
package mprread

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genConst "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
	genEnum "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
	genExpMap "github.com/mendixlabs/mxcli/modelsdk/gen/exportmappings"
	genImpMap "github.com/mendixlabs/mxcli/modelsdk/gen/importmappings"
	genJson "github.com/mendixlabs/mxcli/modelsdk/gen/jsonstructures"
	genSched "github.com/mendixlabs/mxcli/modelsdk/gen/scheduledevents"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// matchesQualified reports whether `qualified` (either a bare local name or a
// fully qualified Module.Name) targets an element whose local name is `local`.
// The plan-defined matching scheme accepts both forms uniformly across the
// GetXByQualifiedName helpers.
func matchesQualified(qualified, local string) bool {
	if local == "" {
		return false
	}
	return qualified == local || strings.HasSuffix(qualified, "."+local)
}

// ---------------------------------------------------------------------------
// Enumerations
// ---------------------------------------------------------------------------

// ListEnumerations decodes every Enumerations$Enumeration unit in the project.
func ListEnumerations(r *mmpr.Reader) ([]*genEnum.Enumeration, error) {
	return ListUnitsByType[*genEnum.Enumeration](r, "Enumerations$Enumeration")
}

// GetEnumeration retrieves a single enumeration by unit ID.
func GetEnumeration(r *mmpr.Reader, id model.ID) (*genEnum.Enumeration, error) {
	enums, err := ListEnumerations(r)
	if err != nil {
		return nil, err
	}
	for _, e := range enums {
		if element.ID(id) == e.ID() {
			return e, nil
		}
	}
	return nil, fmt.Errorf("enumeration not found: %s", id)
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

// ListConstants decodes every Constants$Constant unit in the project.
func ListConstants(r *mmpr.Reader) ([]*genConst.Constant, error) {
	return ListUnitsByType[*genConst.Constant](r, "Constants$Constant")
}

// GetConstant retrieves a single constant by unit ID.
func GetConstant(r *mmpr.Reader, id model.ID) (*genConst.Constant, error) {
	consts, err := ListConstants(r)
	if err != nil {
		return nil, err
	}
	for _, c := range consts {
		if element.ID(id) == c.ID() {
			return c, nil
		}
	}
	return nil, fmt.Errorf("constant not found: %s", id)
}

// ---------------------------------------------------------------------------
// Scheduled events
// ---------------------------------------------------------------------------

// ListScheduledEvents decodes every ScheduledEvents$ScheduledEvent unit.
func ListScheduledEvents(r *mmpr.Reader) ([]*genSched.ScheduledEvent, error) {
	return ListUnitsByType[*genSched.ScheduledEvent](r, "ScheduledEvents$ScheduledEvent")
}

// GetScheduledEvent retrieves a single scheduled event by unit ID.
func GetScheduledEvent(r *mmpr.Reader, id model.ID) (*genSched.ScheduledEvent, error) {
	events, err := ListScheduledEvents(r)
	if err != nil {
		return nil, err
	}
	for _, s := range events {
		if element.ID(id) == s.ID() {
			return s, nil
		}
	}
	return nil, fmt.Errorf("scheduled event not found: %s", id)
}

// ---------------------------------------------------------------------------
// Import mappings
// ---------------------------------------------------------------------------

// ListImportMappings decodes every ImportMappings$ImportMapping unit.
func ListImportMappings(r *mmpr.Reader) ([]*genImpMap.ImportMapping, error) {
	return ListUnitsByType[*genImpMap.ImportMapping](r, "ImportMappings$ImportMapping")
}

// GetImportMappingByQualifiedName retrieves an import mapping by its qualified
// name (Module.Name) or by its local name alone.
func GetImportMappingByQualifiedName(r *mmpr.Reader, qualifiedName string) (*genImpMap.ImportMapping, error) {
	all, err := ListImportMappings(r)
	if err != nil {
		return nil, err
	}
	for _, m := range all {
		if matchesQualified(qualifiedName, m.Name()) {
			return m, nil
		}
	}
	return nil, fmt.Errorf("import mapping not found: %s", qualifiedName)
}

// ---------------------------------------------------------------------------
// Export mappings
// ---------------------------------------------------------------------------

// ListExportMappings decodes every ExportMappings$ExportMapping unit.
func ListExportMappings(r *mmpr.Reader) ([]*genExpMap.ExportMapping, error) {
	return ListUnitsByType[*genExpMap.ExportMapping](r, "ExportMappings$ExportMapping")
}

// GetExportMappingByQualifiedName retrieves an export mapping by qualified name.
func GetExportMappingByQualifiedName(r *mmpr.Reader, qualifiedName string) (*genExpMap.ExportMapping, error) {
	all, err := ListExportMappings(r)
	if err != nil {
		return nil, err
	}
	for _, m := range all {
		if matchesQualified(qualifiedName, m.Name()) {
			return m, nil
		}
	}
	return nil, fmt.Errorf("export mapping not found: %s", qualifiedName)
}

// ---------------------------------------------------------------------------
// JSON structures
// ---------------------------------------------------------------------------

// ListJsonStructures decodes every JsonStructures$JsonStructure unit.
func ListJsonStructures(r *mmpr.Reader) ([]*genJson.JsonStructure, error) {
	return ListUnitsByType[*genJson.JsonStructure](r, "JsonStructures$JsonStructure")
}

// GetJsonStructureByQualifiedName retrieves a JSON structure by qualified name.
func GetJsonStructureByQualifiedName(r *mmpr.Reader, qualifiedName string) (*genJson.JsonStructure, error) {
	all, err := ListJsonStructures(r)
	if err != nil {
		return nil, err
	}
	for _, m := range all {
		if matchesQualified(qualifiedName, m.Name()) {
			return m, nil
		}
	}
	return nil, fmt.Errorf("json structure not found: %s", qualifiedName)
}
