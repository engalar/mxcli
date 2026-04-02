// SPDX-License-Identifier: Apache-2.0

// Package version provides Mendix project version detection and handling.
package version

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/mendixlabs/mxcli/sdk/versions"
)

// ProjectVersion contains version information for a Mendix project.
type ProjectVersion struct {
	// ProductVersion is the full Mendix version string (e.g., "10.18.0", "11.6.0")
	ProductVersion string

	// BuildVersion is the build version, usually same as ProductVersion
	BuildVersion string

	// FormatVersion is the MPR format version (1 for legacy, 2 for mprcontents)
	FormatVersion int

	// SchemaHash is the SHA256 hash of the metamodel schema
	SchemaHash string

	// MajorVersion is the major version number (e.g., 10, 11)
	MajorVersion int

	// MinorVersion is the minor version number (e.g., 18, 6)
	MinorVersion int

	// PatchVersion is the patch version number (e.g., 0, 1)
	PatchVersion int
}

// DefaultVersion returns the default version (11.6.0) used when detection fails.
func DefaultVersion() *ProjectVersion {
	return &ProjectVersion{
		ProductVersion: "11.6.0",
		BuildVersion:   "11.6.0",
		FormatVersion:  2,
		MajorVersion:   11,
		MinorVersion:   6,
		PatchVersion:   0,
	}
}

// DetectFromDB reads version information from the MPR database.
func DetectFromDB(db *sql.DB) (*ProjectVersion, error) {
	var formatVersion int
	var productVersion, buildVersion, schemaHash string

	// Try the old schema first (with _FormatVersion)
	row := db.QueryRow("SELECT _FormatVersion, _ProductVersion, _BuildVersion, _SchemaHash FROM _MetaData LIMIT 1")
	err := row.Scan(&formatVersion, &productVersion, &buildVersion, &schemaHash)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default if no metadata found
			return DefaultVersion(), nil
		}
		// Try new schema without _FormatVersion (Mendix 11.6.2+)
		row = db.QueryRow("SELECT _ProductVersion, _BuildVersion, _SchemaHash FROM _MetaData LIMIT 1")
		err = row.Scan(&productVersion, &buildVersion, &schemaHash)
		if err != nil {
			if err == sql.ErrNoRows {
				return DefaultVersion(), nil
			}
			return nil, fmt.Errorf("failed to read version metadata: %w", err)
		}
		// Default format version to 2 for newer schemas
		formatVersion = 2
	}

	pv := &ProjectVersion{
		ProductVersion: productVersion,
		BuildVersion:   buildVersion,
		FormatVersion:  formatVersion,
		SchemaHash:     schemaHash,
	}

	// Parse version components
	pv.MajorVersion, pv.MinorVersion, pv.PatchVersion = parseVersion(productVersion)

	return pv, nil
}

// parseVersion extracts major, minor, patch from a version string like "10.18.0"
func parseVersion(version string) (major, minor, patch int) {
	parts := strings.Split(version, ".")
	if len(parts) >= 1 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) >= 3 {
		patch, _ = strconv.Atoi(parts[2])
	}
	return
}

// String returns the product version string.
func (v *ProjectVersion) String() string {
	return v.ProductVersion
}

// IsMPRv2 returns true if the project uses MPR v2 format (mprcontents folder).
func (v *ProjectVersion) IsMPRv2() bool {
	return v.FormatVersion >= 2
}

// IsAtLeast returns true if this version is at least the specified major.minor version.
func (v *ProjectVersion) IsAtLeast(major, minor int) bool {
	if v.MajorVersion > major {
		return true
	}
	if v.MajorVersion == major && v.MinorVersion >= minor {
		return true
	}
	return false
}

// IsAtLeastFull returns true if this version is at least the specified major.minor.patch version.
func (v *ProjectVersion) IsAtLeastFull(major, minor, patch int) bool {
	if v.MajorVersion > major {
		return true
	}
	if v.MajorVersion == major && v.MinorVersion > minor {
		return true
	}
	if v.MajorVersion == major && v.MinorVersion == minor && v.PatchVersion >= patch {
		return true
	}
	return false
}

// SupportedVersionRange defines the range of Mendix versions supported for read/write.
var SupportedVersionRange = struct {
	MinMajor int
	MaxMajor int
}{
	MinMajor: 9,
	MaxMajor: 11,
}

// IsSupported returns true if this version is within the supported range for writing.
func (v *ProjectVersion) IsSupported() bool {
	return v.MajorVersion >= SupportedVersionRange.MinMajor &&
		v.MajorVersion <= SupportedVersionRange.MaxMajor
}

// SupportsFeature checks if a specific feature is available in this version.
// It first checks the YAML-based version registry, falling back to the
// hardcoded featureVersions map for features not yet in the registry.
func (v *ProjectVersion) SupportsFeature(feature Feature) bool {
	// Try the YAML registry first via the feature-to-registry mapping.
	if mapping, ok := featureRegistry[feature]; ok {
		reg, err := versions.Load()
		if err == nil {
			pv := versions.SemVer{Major: v.MajorVersion, Minor: v.MinorVersion, Patch: v.PatchVersion}
			return reg.IsAvailable(mapping.Area, mapping.Name, pv)
		}
	}

	// Fallback to hardcoded map.
	minVersion, ok := featureVersions[feature]
	if !ok {
		return false
	}
	return v.IsAtLeast(minVersion.Major, minVersion.Minor)
}

// Feature represents a Mendix feature that may or may not be available.
type Feature string

// Known features with version requirements
const (
	FeatureViewEntities       Feature = "ViewEntities"
	FeatureAssociationStorage Feature = "AssociationStorageFormat"
	FeatureMPRv2              Feature = "MPRv2Format"
	FeatureBusinessEvents     Feature = "BusinessEvents"
	FeatureWorkflows          Feature = "Workflows"
	FeaturePortableApp        Feature = "PortableApp"
)

// registryMapping maps a Feature constant to its area.name in the YAML registry.
type registryMapping struct {
	Area string
	Name string
}

// featureRegistry maps Feature constants to their YAML registry keys.
var featureRegistry = map[Feature]registryMapping{
	FeatureViewEntities:       {Area: "domain_model", Name: "view_entities"},
	FeatureAssociationStorage: {Area: "mpr_format", Name: "association_storage"},
	FeatureMPRv2:              {Area: "mpr_format", Name: "mpr_v2"},
	FeatureBusinessEvents:     {Area: "integration", Name: "business_events"},
	FeatureWorkflows:          {Area: "workflows", Name: "basic"},
	FeaturePortableApp:        {Area: "mpr_format", Name: "portable_app"},
}

// MinVersion represents a minimum version requirement.
type MinVersion struct {
	Major int
	Minor int
}

// featureVersions maps features to their minimum required versions.
// This is the fallback when the YAML registry is unavailable.
var featureVersions = map[Feature]MinVersion{
	FeatureViewEntities:       {Major: 10, Minor: 18},
	FeatureAssociationStorage: {Major: 11, Minor: 0},
	FeatureMPRv2:              {Major: 10, Minor: 18},
	FeatureBusinessEvents:     {Major: 10, Minor: 0},
	FeatureWorkflows:          {Major: 9, Minor: 0},
	FeaturePortableApp:        {Major: 11, Minor: 6},
}
