// SPDX-License-Identifier: Apache-2.0

// Package mpr - Unit listing infrastructure for Reader.
package mpr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/types"
)

// ResolveModuleName walks the container hierarchy upward until it finds a module.
// This is necessary because in MPR v2 projects, documents live inside folders,
// so a document's direct ContainerID is a folder, not the module.
func ResolveModuleName(containerID string, moduleMap map[string]string, containerParent map[string]string) string {
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

// BuildContainerParent builds a map of unit ID → parent container ID for hierarchy walking.
// Uses a direct SQL query (no file reads) for both v1 and v2 formats since
// ContainerID is always stored in the SQLite Unit table.
func (r *Reader) BuildContainerParent() (map[string]string, error) {
	rows, err := r.db.Query("SELECT UnitID, ContainerID FROM Unit")
	if err != nil {
		return nil, fmt.Errorf("BuildContainerParent: %w", err)
	}
	defer rows.Close()
	result := make(map[string]string)
	for rows.Next() {
		var unitID, containerID []byte
		if err := rows.Scan(&unitID, &containerID); err != nil {
			return nil, fmt.Errorf("BuildContainerParent scan: %w", err)
		}
		result[blobToUUID(unitID)] = blobToUUID(containerID)
	}
	return result, rows.Err()
}

// rawUnit holds raw unit data from the database.
type rawUnit struct {
	ID              string
	ContainerID     string
	ContainmentName string
	Type            string
	Contents        []byte
}

// UnitRef holds unit metadata returned by ListUnitsByType.
type UnitRef struct {
	ID          string
	ContainerID string
	Type        string // BSON $Type (e.g. "Microflows$Microflow")
	Contents    []byte
}

// ListUnitsByType returns all units matching the given BSON $Type prefix.
// This is the exported version for use by TreeWriter and other packages.
func (r *Reader) ListUnitsByType(typePrefix string) ([]UnitRef, error) {
	units, err := r.listUnitsByType(typePrefix)
	if err != nil {
		return nil, err
	}
	result := make([]UnitRef, len(units))
	for i, u := range units {
		result[i] = UnitRef{ID: u.ID, ContainerID: u.ContainerID, Type: u.Type, Contents: u.Contents}
	}
	return result, nil
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

	// Merge buffered script inserts so reads within EXECUTE SCRIPT see new units.
	for _, e := range r.scriptInserts {
		typeName := getTypeFromContents(e.Contents)
		if typePrefix == "" || strings.HasPrefix(typeName, typePrefix) {
			units = append(units, rawUnit{
				ID:              e.ID,
				ContainerID:     e.ContainerID,
				ContainmentName: e.ContainmentName,
				Type:            typeName,
				Contents:        e.Contents,
			})
		}
	}

	return units, nil
}

// listUnitsByTypeV2 handles MPR v2 format (contents in mprcontents folder).
// Uses caching to avoid reading every file for each query.
func (r *Reader) listUnitsByTypeV2(typePrefix string) ([]rawUnit, error) {
	if !r.unitCacheValid {
		if err := r.buildUnitCache(); err != nil {
			return nil, err
		}
	}

	// Filter by type using cache; use cached Contents directly.
	var units []rawUnit
	for _, cu := range r.unitCache {
		if typePrefix == "" || strings.HasPrefix(cu.Type, typePrefix) {
			contents := cu.Contents
			// Fall back to file read if contents were not cached (should not happen
			// after buildUnitCache, but kept for safety).
			if len(contents) == 0 {
				var err error
				contents, err = r.readMprContents(cu.ID)
				if err != nil {
					continue
				}
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
// On first build all files are read. On rebuild (after InvalidateCache),
// units whose ContentsHash matches the previous cache are reused without
// re-reading the file, so only modified units incur file I/O.
func (r *Reader) buildUnitCache() error {
	rows, err := r.db.Query(`
		SELECT UnitID, ContainerID, ContainmentName, ContentsHash
		FROM Unit
	`)
	if err != nil {
		return fmt.Errorf("failed to query units: %w", err)
	}
	defer rows.Close()

	// Index existing cache by unit ID for fast lookup during rebuild.
	prevByID := make(map[string]cachedUnit, len(r.unitCache))
	for _, cu := range r.unitCache {
		prevByID[cu.ID] = cu
	}

	newCache := make([]cachedUnit, 0, len(r.unitCache)+10)
	for rows.Next() {
		var unitID, containerID []byte
		var containmentName string
		var contentsHash string

		if err := rows.Scan(&unitID, &containerID, &containmentName, &contentsHash); err != nil {
			return fmt.Errorf("failed to scan unit row: %w", err)
		}

		unitUUID := blobToUUID(unitID)
		cu := cachedUnit{
			ID:              unitUUID,
			ContainerID:     blobToUUID(containerID),
			ContainmentName: containmentName,
		}

		// If the previous cache has this unit with the same ContentsHash,
		// reuse its Contents and Type without re-reading the file — UNLESS
		// the script overlay has fresher content (list-cache bug: writes go
		// to scriptOverlay, not SQLite, so SQLite hash never changes during
		// script execution, causing stale content to be reused).
		if prev, ok := prevByID[unitUUID]; ok && prev.ContentsHash == contentsHash && prev.ContentsHash != "" {
			if overlay, ok := r.scriptOverlay[unitUUID]; ok {
				cu.Type = getTypeFromContents(overlay)
				cu.Contents = overlay
				cu.ContentsHash = contentsHash // SQLite hash unchanged; overlay check on next rebuild
			} else {
				cu.Type = prev.Type
				cu.Contents = prev.Contents
				cu.ContentsHash = prev.ContentsHash
			}
		} else {
			// Hash changed or new unit: evict stale contentCache entry,
			// then read fresh from disk (populates cache with new data).
			if r.contentCache != nil {
				delete(r.contentCache, unitUUID)
			}
			contents, err := r.readMprContents(unitUUID)
			if err != nil {
				continue
			}
			cu.Type = getTypeFromContents(contents)
			cu.Contents = contents
			cu.ContentsHash = contentsHash
		}
		newCache = append(newCache, cu)
	}

	// Merge buffered script inserts so reads within EXECUTE SCRIPT see new units.
	for _, e := range r.scriptInserts {
		typeName := getTypeFromContents(e.Contents)
		newCache = append(newCache, cachedUnit{
			ID:              e.ID,
			ContainerID:     e.ContainerID,
			ContainmentName: e.ContainmentName,
			Type:            typeName,
			Contents:        e.Contents,
		})
	}

	r.unitCache = newCache
	r.unitCacheValid = true
	return nil
}

// InvalidateCache marks the unit metadata cache as invalid so the next
// ListUnitsByType / ListModules rebuilds it from SQLite + disk. The
// contentCache is NOT cleared — writers push updated content to it after
// each write, so subsequent reads hit memory instead of disk.
func (r *Reader) InvalidateCache() {
	r.unitCacheValid = false
}

// EnableContentCache activates the in-memory content cache for this reader.
// Call once after Connect in persistent daemon mode. The cache survives across
// requests; InvalidateCache empties it (but keeps caching active) on writes.
func (r *Reader) EnableContentCache() {
	if r.contentCache == nil {
		r.contentCache = make(map[string][]byte)
	}
}

// readMprContents reads content from the mprcontents folder for v2 format.
// The path is: mprcontents/XX/YY/UUID.mxunit where XX and YY are first two chars of UUID.
//
// When r.contentCache is non-nil (persistent daemon mode), the result is cached
// in memory so subsequent reads of the same unit skip the file I/O entirely.
// The cache is invalidated by InvalidateCache (called after every write).
func (r *Reader) readMprContents(unitUUID string) ([]byte, error) {
	// Script overlay: return buffered content immediately, skipping file/cache I/O.
	if b, ok := r.scriptOverlay[unitUUID]; ok {
		return b, nil
	}
	if len(unitUUID) < 4 {
		return nil, fmt.Errorf("invalid unit UUID: %s", unitUUID)
	}

	// Fast path: content cache hit.
	if r.contentCache != nil {
		if data, ok := r.contentCache[unitUUID]; ok {
			return data, nil
		}
	} else {
		r.contentCache = make(map[string][]byte)
	}

	// Build path: mprcontents/XX/YY/UUID.mxunit
	path := filepath.Join(
		r.contentsDir,
		unitUUID[0:2],
		unitUUID[2:4],
		unitUUID+".mxunit",
	)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Populate cache.
	r.contentCache[unitUUID] = data
	return data, nil
}

// getTypeFromContents extracts the $Type field from BSON contents.
// Uses bson.Raw.LookupErr for O(1) field extraction instead of unmarshalling
// the entire document into map[string]any.
func getTypeFromContents(contents []byte) string {
	if len(contents) == 0 {
		return ""
	}
	val, err := bson.Raw(contents).LookupErr("$Type")
	if err != nil {
		return ""
	}
	s, ok := val.StringValueOK()
	if !ok {
		return ""
	}
	return s
}

// RawUnitInfo contains information about a raw unit for BSON debugging.
// Aliased to mdl/types.RawUnitInfo so reader_raw.go methods and modelsdk/codec
// consumers share a single concrete struct.
type RawUnitInfo = types.RawUnitInfo
