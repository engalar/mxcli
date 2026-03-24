// SPDX-License-Identifier: Apache-2.0

// Package mpr - Unit listing infrastructure for Reader.
package mpr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

// resolveModuleName walks the container hierarchy upward until it finds a module.
// This is necessary because in MPR v2 projects, documents live inside folders,
// so a document's direct ContainerID is a folder, not the module.
func resolveModuleName(containerID string, moduleMap map[string]string, containerParent map[string]string) string {
	current := containerID
	for range 20 {
		if name, ok := moduleMap[current]; ok {
			return name
		}
		parent, ok := containerParent[current]
		if !ok || parent == current {
			break
		}
		current = parent
	}
	return ""
}

// buildContainerParent builds a map of unit ID → parent container ID for hierarchy walking.
func (r *Reader) buildContainerParent() (map[string]string, error) {
	units, err := r.ListUnits()
	if err != nil {
		return nil, err
	}
	containerParent := make(map[string]string, len(units))
	for _, u := range units {
		containerParent[string(u.ID)] = string(u.ContainerID)
	}
	return containerParent, nil
}

// rawUnit holds raw unit data from the database.
type rawUnit struct {
	ID              string
	ContainerID     string
	ContainmentName string
	Type            string
	Contents        []byte
}

// listUnitsByType returns all units matching the given type prefix.
func (r *Reader) listUnitsByType(typePrefix string) ([]rawUnit, error) {
	if r.version == MPRVersionV2 {
		return r.listUnitsByTypeV2(typePrefix)
	}
	return r.listUnitsByTypeV1(typePrefix)
}

// listUnitsByTypeV1 handles MPR v1 format (contents in database).
func (r *Reader) listUnitsByTypeV1(typePrefix string) ([]rawUnit, error) {
	rows, err := r.db.Query(`
		SELECT UnitID, ContainerID, ContainmentName, Contents
		FROM Unit
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query units: %w", err)
	}
	defer rows.Close()

	var units []rawUnit
	for rows.Next() {
		var unitID, containerID []byte
		var containmentName string
		var contents []byte

		if err := rows.Scan(&unitID, &containerID, &containmentName, &contents); err != nil {
			return nil, fmt.Errorf("failed to scan unit row: %w", err)
		}

		typeName := getTypeFromContents(contents)
		if typePrefix == "" || strings.HasPrefix(typeName, typePrefix) {
			units = append(units, rawUnit{
				ID:              blobToUUID(unitID),
				ContainerID:     blobToUUID(containerID),
				ContainmentName: containmentName,
				Type:            typeName,
				Contents:        contents,
			})
		}
	}

	return units, nil
}

// listUnitsByTypeV2 handles MPR v2 format (contents in mprcontents folder).
// Uses caching to avoid reading every file for each query.
func (r *Reader) listUnitsByTypeV2(typePrefix string) ([]rawUnit, error) {
	// Build cache if not valid
	if !r.unitCacheValid {
		if err := r.buildUnitCache(); err != nil {
			return nil, err
		}
	}

	// Filter by type using cache, only read contents for matching units
	var units []rawUnit
	for _, cu := range r.unitCache {
		if typePrefix == "" || strings.HasPrefix(cu.Type, typePrefix) {
			// Read contents from mprcontents folder
			// Note: cu.ID is already in the correct swapped format from blobToUUID
			contents, err := r.readMprContents(cu.ID)
			if err != nil {
				// Skip units with missing content files
				continue
			}

			units = append(units, rawUnit{
				ID:              cu.ID,
				ContainerID:     cu.ContainerID,
				ContainmentName: cu.ContainmentName,
				Type:            cu.Type,
				Contents:        contents,
			})
		}
	}

	return units, nil
}

// buildUnitCache reads all unit metadata once and caches it.
func (r *Reader) buildUnitCache() error {
	rows, err := r.db.Query(`
		SELECT UnitID, ContainerID, ContainmentName
		FROM Unit
	`)
	if err != nil {
		return fmt.Errorf("failed to query units: %w", err)
	}
	defer rows.Close()

	r.unitCache = nil
	for rows.Next() {
		var unitID, containerID []byte
		var containmentName string

		if err := rows.Scan(&unitID, &containerID, &containmentName); err != nil {
			return fmt.Errorf("failed to scan unit row: %w", err)
		}

		// Convert UnitID to UUID string
		unitUUID := blobToUUID(unitID)

		// Read contents to get type (only done once during cache build)
		contents, err := r.readMprContents(unitUUID)
		if err != nil {
			// Skip units with missing content files
			continue
		}

		typeName := getTypeFromContents(contents)
		r.unitCache = append(r.unitCache, cachedUnit{
			ID:              blobToUUID(unitID),
			ContainerID:     blobToUUID(containerID),
			ContainmentName: containmentName,
			Type:            typeName,
		})
	}

	r.unitCacheValid = true
	return nil
}

// InvalidateCache marks the unit cache as invalid.
// Should be called after any write operation.
func (r *Reader) InvalidateCache() {
	r.unitCacheValid = false
}

// readMprContents reads content from the mprcontents folder for v2 format.
// The path is: mprcontents/XX/YY/UUID.mxunit where XX and YY are first two chars of UUID.
func (r *Reader) readMprContents(unitUUID string) ([]byte, error) {
	if len(unitUUID) < 4 {
		return nil, fmt.Errorf("invalid unit UUID: %s", unitUUID)
	}

	// Build path: mprcontents/XX/YY/UUID.mxunit
	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	// First two chars are positions 0-1, next two are positions 2-3
	path := filepath.Join(
		r.contentsDir,
		unitUUID[0:2],
		unitUUID[2:4],
		unitUUID+".mxunit",
	)

	return os.ReadFile(path)
}

// getTypeFromContents extracts the $Type field from BSON contents.
func getTypeFromContents(contents []byte) string {
	if len(contents) == 0 {
		return ""
	}

	var raw map[string]any
	if err := bson.Unmarshal(contents, &raw); err != nil {
		return ""
	}

	if typeName, ok := raw["$Type"].(string); ok {
		return typeName
	}
	return ""
}

// GetRawMicroflowByName returns the raw BSON contents for a microflow by qualified name.
// Used for debugging to compare serialized data.
func (r *Reader) GetRawMicroflowByName(qualifiedName string) ([]byte, error) {
	units, err := r.listUnitsByType("Microflows$Microflow")
	if err != nil {
		return nil, err
	}

	// Build a map of container ID to module name
	modules, err := r.ListModules()
	if err != nil {
		return nil, err
	}
	moduleMap := make(map[string]string)
	for _, m := range modules {
		moduleMap[string(m.ID)] = m.Name
	}

	for _, u := range units {
		// Parse just enough to get the qualified name
		var raw map[string]any
		if err := bson.Unmarshal(u.Contents, &raw); err != nil {
			continue
		}

		name, _ := raw["Name"].(string)
		// Get module name from container
		moduleName := moduleMap[u.ContainerID]
		if moduleName != "" && moduleName+"."+name == qualifiedName {
			return u.Contents, nil
		}
	}

	return nil, fmt.Errorf("microflow not found: %s", qualifiedName)
}

// RawUnitInfo contains information about a raw unit for BSON debugging.
type RawUnitInfo struct {
	ID            string
	QualifiedName string
	Type          string
	ModuleName    string
	Contents      []byte
}

// GetRawUnitByName returns the raw BSON contents for a unit by qualified name.
// Supported types: page, entity, microflow, nanoflow, enumeration, association, snippet.
// Used for debugging BSON serialization issues.
func (r *Reader) GetRawUnitByName(objectType, qualifiedName string) (*RawUnitInfo, error) {
	var typePrefix string
	switch strings.ToLower(objectType) {
	case "page":
		typePrefix = "Forms$Page"
	case "entity":
		typePrefix = "DomainModels$Entity"
	case "association":
		typePrefix = "DomainModels$Association"
	case "microflow":
		typePrefix = "Microflows$Microflow"
	case "nanoflow":
		typePrefix = "Microflows$Nanoflow"
	case "enumeration":
		typePrefix = "Enumerations$Enumeration"
	case "snippet":
		typePrefix = "Forms$Snippet"
	case "layout":
		typePrefix = "Forms$Layout"
	case "workflow":
		typePrefix = "Workflows$Workflow"
	case "imagecollection":
		typePrefix = "Images$ImageCollection"
	case "javaaction":
		typePrefix = "JavaActions$JavaAction"
	default:
		return nil, fmt.Errorf("unsupported object type: %s", objectType)
	}

	// For entities and associations, we need to search within domain models
	switch strings.ToLower(objectType) {
	case "entity":
		return r.getRawEntityByName(qualifiedName)
	case "association":
		return r.getRawAssociationByName(qualifiedName)
	}

	units, err := r.listUnitsByType(typePrefix)
	if err != nil {
		return nil, err
	}

	// Build module name map and container hierarchy for MPR v2 folder support.
	modules, err := r.ListModules()
	if err != nil {
		return nil, err
	}
	moduleMap := make(map[string]string)
	for _, m := range modules {
		moduleMap[string(m.ID)] = m.Name
	}
	containerParent, err := r.buildContainerParent()
	if err != nil {
		return nil, err
	}

	for _, u := range units {
		var raw map[string]any
		if err := bson.Unmarshal(u.Contents, &raw); err != nil {
			continue
		}

		name, _ := raw["Name"].(string)
		moduleName := resolveModuleName(u.ContainerID, moduleMap, containerParent)

		// Build full name, handling missing module
		var fullName string
		if moduleName != "" {
			fullName = moduleName + "." + name
		} else {
			fullName = name
		}

		if fullName == qualifiedName {
			return &RawUnitInfo{
				ID:            u.ID,
				QualifiedName: fullName,
				Type:          u.Type,
				ModuleName:    moduleName,
				Contents:      u.Contents,
			}, nil
		}
	}

	return nil, fmt.Errorf("%s not found: %s", objectType, qualifiedName)
}

// getRawEntityByName finds an entity within domain models.
func (r *Reader) getRawEntityByName(qualifiedName string) (*RawUnitInfo, error) {
	// Split qualified name
	parts := strings.Split(qualifiedName, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid entity name: %s (expected Module.Entity)", qualifiedName)
	}
	targetModule := parts[0]
	targetEntity := parts[1]

	// Get domain models
	units, err := r.listUnitsByType("DomainModels$DomainModel")
	if err != nil {
		return nil, err
	}

	// Build module name map
	modules, err := r.ListModules()
	if err != nil {
		return nil, err
	}
	moduleMap := make(map[string]string)
	for _, m := range modules {
		moduleMap[string(m.ID)] = m.Name
	}

	for _, u := range units {
		moduleName := moduleMap[u.ContainerID]
		if moduleName != targetModule {
			continue
		}

		// Parse domain model to find entity.
		// Unmarshal into bson.D so nested documents remain bson.D (not map[string]interface{}).
		var rawD bson.D
		if err := bson.Unmarshal(u.Contents, &rawD); err != nil {
			continue
		}

		var entitiesVal any
		for _, field := range rawD {
			if field.Key == "Entities" {
				entitiesVal = field.Value
				break
			}
		}

		entities, ok := entitiesVal.(bson.A)
		if !ok {
			continue
		}

		// Skip version marker (first element is int32 array type indicator)
		for i := 1; i < len(entities); i++ {
			entity, ok := entities[i].(bson.D)
			if !ok {
				continue
			}

			for _, field := range entity {
				if field.Key == "Name" {
					if name, ok := field.Value.(string); ok && name == targetEntity {
						// Found the entity - serialize it back to BSON
						entityBytes, err := bson.Marshal(entity)
						if err != nil {
							return nil, err
						}
						return &RawUnitInfo{
							ID:            u.ID,
							QualifiedName: qualifiedName,
							Type:          "DomainModels$Entity",
							ModuleName:    moduleName,
							Contents:      entityBytes,
						}, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("entity not found: %s", qualifiedName)
}

// getRawAssociationByName finds an association within domain models.
func (r *Reader) getRawAssociationByName(qualifiedName string) (*RawUnitInfo, error) {
	parts := strings.Split(qualifiedName, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid association name: %s (expected Module.AssociationName)", qualifiedName)
	}
	targetModule := parts[0]
	targetAssoc := parts[1]

	units, err := r.listUnitsByType("DomainModels$DomainModel")
	if err != nil {
		return nil, err
	}

	modules, err := r.ListModules()
	if err != nil {
		return nil, err
	}
	moduleMap := make(map[string]string)
	for _, m := range modules {
		moduleMap[string(m.ID)] = m.Name
	}

	for _, u := range units {
		moduleName := moduleMap[u.ContainerID]
		if moduleName != targetModule {
			continue
		}

		// Unmarshal into bson.D so nested documents remain bson.D (not map[string]interface{}).
		var rawD bson.D
		if err := bson.Unmarshal(u.Contents, &rawD); err != nil {
			continue
		}

		var assocsVal any
		for _, field := range rawD {
			if field.Key == "Associations" {
				assocsVal = field.Value
				break
			}
		}

		assocs, ok := assocsVal.(bson.A)
		if !ok {
			continue
		}

		// Skip version marker (first element is int32 array type indicator)
		for i := 1; i < len(assocs); i++ {
			assoc, ok := assocs[i].(bson.D)
			if !ok {
				continue
			}

			for _, field := range assoc {
				if field.Key == "Name" {
					if name, ok := field.Value.(string); ok && name == targetAssoc {
						assocBytes, err := bson.Marshal(assoc)
						if err != nil {
							return nil, err
						}
						return &RawUnitInfo{
							ID:            u.ID,
							QualifiedName: qualifiedName,
							Type:          "DomainModels$Association",
							ModuleName:    moduleName,
							Contents:      assocBytes,
						}, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("association not found: %s", qualifiedName)
}

// ListRawUnits returns all units of a given type for BSON debugging.
func (r *Reader) ListRawUnits(objectType string) ([]*RawUnitInfo, error) {
	var typePrefix string
	switch strings.ToLower(objectType) {
	case "page":
		typePrefix = "Forms$Page"
	case "microflow":
		typePrefix = "Microflows$Microflow"
	case "nanoflow":
		typePrefix = "Microflows$Nanoflow"
	case "enumeration":
		typePrefix = "Enumerations$Enumeration"
	case "snippet":
		typePrefix = "Forms$Snippet"
	case "layout":
		typePrefix = "Forms$Layout"
	case "workflow":
		typePrefix = "Workflows$Workflow"
	case "imagecollection":
		typePrefix = "Images$ImageCollection"
	case "":
		typePrefix = ""
	default:
		return nil, fmt.Errorf("unsupported object type: %s", objectType)
	}

	units, err := r.listUnitsByType(typePrefix)
	if err != nil {
		return nil, err
	}

	// Build module name map and container hierarchy for MPR v2 folder support.
	modules, err := r.ListModules()
	if err != nil {
		return nil, err
	}
	moduleMap := make(map[string]string)
	for _, m := range modules {
		moduleMap[string(m.ID)] = m.Name
	}
	containerParent, err := r.buildContainerParent()
	if err != nil {
		return nil, err
	}

	var result []*RawUnitInfo
	for _, u := range units {
		var raw map[string]any
		if err := bson.Unmarshal(u.Contents, &raw); err != nil {
			continue
		}

		name, _ := raw["Name"].(string)
		moduleName := resolveModuleName(u.ContainerID, moduleMap, containerParent)
		fullName := name
		if moduleName != "" {
			fullName = moduleName + "." + name
		}

		result = append(result, &RawUnitInfo{
			ID:            u.ID,
			QualifiedName: fullName,
			Type:          u.Type,
			ModuleName:    moduleName,
			Contents:      u.Contents,
		})
	}

	return result, nil
}
