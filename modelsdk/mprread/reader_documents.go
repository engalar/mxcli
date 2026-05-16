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
	genBE "github.com/mendixlabs/mxcli/modelsdk/gen/businessevents"
	genConst "github.com/mendixlabs/mxcli/modelsdk/gen/constants"
	genDBC "github.com/mendixlabs/mxcli/modelsdk/gen/databaseconnector"
	genDT "github.com/mendixlabs/mxcli/modelsdk/gen/datatransformers"
	genEnum "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
	genExpMap "github.com/mendixlabs/mxcli/modelsdk/gen/exportmappings"
	genImg "github.com/mendixlabs/mxcli/modelsdk/gen/images"
	genImpMap "github.com/mendixlabs/mxcli/modelsdk/gen/importmappings"
	genJson "github.com/mendixlabs/mxcli/modelsdk/gen/jsonstructures"
	genNav "github.com/mendixlabs/mxcli/modelsdk/gen/navigation"
	genODataPub "github.com/mendixlabs/mxcli/modelsdk/gen/odatapublish"
	genProj "github.com/mendixlabs/mxcli/modelsdk/gen/projects"
	genRest "github.com/mendixlabs/mxcli/modelsdk/gen/rest"
	genSched "github.com/mendixlabs/mxcli/modelsdk/gen/scheduledevents"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	genSet "github.com/mendixlabs/mxcli/modelsdk/gen/settings"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
)

// decodeOne fetches and decodes a single unit by ID into the requested gen type.
// Returns nil if the bytes cannot be read or do not decode into T.
func decodeOne[T element.Element](r *mmpr.Reader, unitID string) (T, error) {
	var zero T
	raw, err := r.GetRawUnitBytes(unitID)
	if err != nil {
		return zero, fmt.Errorf("read unit %s: %w", unitID, err)
	}
	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, err := dec.Decode(bson.Raw(raw))
	if err != nil {
		return zero, fmt.Errorf("decode unit %s: %w", unitID, err)
	}
	typed, ok := elem.(T)
	if !ok {
		return zero, fmt.Errorf("unit %s decoded as %T, want %T", unitID, elem, zero)
	}
	return typed, nil
}

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

// ---------------------------------------------------------------------------
// Web services & related document types
// ---------------------------------------------------------------------------

// ListBusinessEventServices decodes every BusinessEvents$BusinessEventService unit.
func ListBusinessEventServices(r *mmpr.Reader) ([]*genBE.BusinessEventService, error) {
	return ListUnitsByType[*genBE.BusinessEventService](r, "BusinessEvents$BusinessEventService")
}

// ListDatabaseConnections decodes every DatabaseConnector$DatabaseConnection unit.
func ListDatabaseConnections(r *mmpr.Reader) ([]*genDBC.DatabaseConnection, error) {
	return ListUnitsByType[*genDBC.DatabaseConnection](r, "DatabaseConnector$DatabaseConnection")
}

// ListDataTransformers decodes every DataTransformers$DataTransformer unit.
func ListDataTransformers(r *mmpr.Reader) ([]*genDT.DataTransformer, error) {
	return ListUnitsByType[*genDT.DataTransformer](r, "DataTransformers$DataTransformer")
}

// ListImageCollections decodes every Images$ImageCollection unit.
func ListImageCollections(r *mmpr.Reader) ([]*genImg.ImageCollection, error) {
	return ListUnitsByType[*genImg.ImageCollection](r, "Images$ImageCollection")
}

// ListConsumedODataServices decodes every Rest$ConsumedODataService unit.
//
// Both ConsumedOData and PublishedRest services live under the `Rest` BSON
// namespace; only PublishedOData uses the legacy `ODataPublish$...Service2`
// type name (see ListPublishedODataServices).
func ListConsumedODataServices(r *mmpr.Reader) ([]*genRest.ConsumedODataService, error) {
	return ListUnitsByType[*genRest.ConsumedODataService](r, "Rest$ConsumedODataService")
}

// ListPublishedODataServices decodes every ODataPublish$PublishedODataService2
// unit. The trailing `2` is part of the BSON $Type, matching the v2 schema
// generation; the Go type drops it from the documentation but keeps it in the
// struct name (PublishedODataService2).
func ListPublishedODataServices(r *mmpr.Reader) ([]*genODataPub.PublishedODataService2, error) {
	return ListUnitsByType[*genODataPub.PublishedODataService2](r, "ODataPublish$PublishedODataService2")
}

// ListConsumedRestServices decodes every Rest$ConsumedRestService unit.
func ListConsumedRestServices(r *mmpr.Reader) ([]*genRest.ConsumedRestService, error) {
	return ListUnitsByType[*genRest.ConsumedRestService](r, "Rest$ConsumedRestService")
}

// ListPublishedRestServices decodes every Rest$PublishedRestService unit.
func ListPublishedRestServices(r *mmpr.Reader) ([]*genRest.PublishedRestService, error) {
	return ListUnitsByType[*genRest.PublishedRestService](r, "Rest$PublishedRestService")
}

// ---------------------------------------------------------------------------
// Navigation
// ---------------------------------------------------------------------------

// ListNavigationDocuments decodes every Navigation$NavigationDocument unit.
func ListNavigationDocuments(r *mmpr.Reader) ([]*genNav.NavigationDocument, error) {
	return ListUnitsByType[*genNav.NavigationDocument](r, "Navigation$NavigationDocument")
}

// GetNavigation returns the first NavigationDocument in the project, which is
// the singleton navigation root that every Mendix project carries.
func GetNavigation(r *mmpr.Reader) (*genNav.NavigationDocument, error) {
	docs, err := ListNavigationDocuments(r)
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("navigation document not found")
	}
	return docs[0], nil
}

// ---------------------------------------------------------------------------
// Settings
// ---------------------------------------------------------------------------

// GetProjectSettings returns the project-level Settings$ProjectSettings doc.
// There is at most one per project.
func GetProjectSettings(r *mmpr.Reader) (*genSet.ProjectSettings, error) {
	settings, err := ListUnitsByType[*genSet.ProjectSettings](r, "Settings$ProjectSettings")
	if err != nil {
		return nil, err
	}
	if len(settings) == 0 {
		return nil, fmt.Errorf("project settings not found")
	}
	return settings[0], nil
}

// ListModuleSettings decodes every Projects$ModuleSettings unit. ModuleSettings
// lives in the `projects` gen package even though its BSON $Type is namespaced
// under `Projects` — the gen layer preserves the historical type name.
func ListModuleSettings(r *mmpr.Reader) ([]*genProj.ModuleSettings, error) {
	return ListUnitsByType[*genProj.ModuleSettings](r, "Projects$ModuleSettings")
}

// GetModuleSettings returns the ModuleSettings whose container is the given
// module ID. Uses the raw unit ContainerID for matching because the decoded
// element drops container linkage by default.
func GetModuleSettings(r *mmpr.Reader, moduleID model.ID) (*genProj.ModuleSettings, error) {
	refs, err := r.ListUnitsByType("Projects$ModuleSettings")
	if err != nil {
		return nil, err
	}
	for _, ref := range refs {
		if ref.ContainerID != string(moduleID) {
			continue
		}
		return decodeOne[*genProj.ModuleSettings](r, ref.ID)
	}
	return nil, fmt.Errorf("module settings not found for module: %s", moduleID)
}

// ---------------------------------------------------------------------------
// Security
// ---------------------------------------------------------------------------

// GetProjectSecurity returns the project-level Security$ProjectSecurity doc.
func GetProjectSecurity(r *mmpr.Reader) (*genSec.ProjectSecurity, error) {
	secs, err := ListUnitsByType[*genSec.ProjectSecurity](r, "Security$ProjectSecurity")
	if err != nil {
		return nil, err
	}
	if len(secs) == 0 {
		return nil, fmt.Errorf("project security not found")
	}
	return secs[0], nil
}

// ListModuleSecurity decodes every Security$ModuleSecurity unit.
func ListModuleSecurity(r *mmpr.Reader) ([]*genSec.ModuleSecurity, error) {
	return ListUnitsByType[*genSec.ModuleSecurity](r, "Security$ModuleSecurity")
}
