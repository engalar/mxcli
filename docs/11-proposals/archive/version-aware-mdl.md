# Proposal: Version-Aware MDL and BSON Serialization

## Problem Statement

The Mendix metamodel evolves across versions. The reflection data in `reference/mendixmodellib/reflection-data/` demonstrates that BSON document structures can change between Mendix versions:

- **Field names** may change (storageName evolution)
- **New properties** are added in newer versions
- **Properties are deprecated/removed** in newer versions
- **Default values** may change
- **Type definitions** may evolve (e.g., enum values added)
- **Reference types** may change (BY_ID vs BY_NAME)

Currently, the SDK uses a single serialization approach that may not work correctly across all Mendix versions.

## Goals

1. **Read any version**: Parse BSON from any supported Mendix version
2. **Write target version**: Serialize BSON compatible with a specific target version
3. **MDL compatibility**: MDL syntax adapts to version capabilities
4. **Clear error messages**: Warn when using features not available in target version
5. **Maintainable**: Minimize code duplication across version handlers

## Scope

| Mendix Version | Support Level |
|----------------|---------------|
| 6.x | Read-only (legacy) |
| 7.x | Read-only (legacy) |
| 8.x | Read-only (legacy) |
| 9.x | Read + Write |
| 10.x | Read + Write (primary) |
| 11.x | Read + Write |

## Architecture Options

### Option A: Version-Specific Serializers (Recommended)

Create separate serializer implementations per major version, with shared base logic.

```
sdk/mpr/
├── reader.go           # Version-agnostic reading (auto-detect)
├── writer.go           # Version-aware writing (dispatches to version)
├── version/
│   ├── detector.go     # Detect MPR version from metadata
│   ├── v9/
│   │   └── serializer.go
│   ├── v10/
│   │   └── serializer.go
│   └── v11/
│       └── serializer.go
└── schema/
    ├── loader.go       # Load reflection data at runtime
    └── registry.go     # Version -> Schema mapping
```

**Pros:**
- Clear separation of version-specific logic
- Easy to add new versions
- Can optimize for each version's quirks

**Cons:**
- Some code duplication
- Must maintain multiple serializers

### Option B: Schema-Driven Serialization

Use the reflection data JSON files at runtime to drive serialization.

```go
type SchemaRegistry struct {
    schemas map[string]*VersionSchema  // "10.18.0" -> schema
}

type VersionSchema struct {
    Structures map[string]*StructureDefinition
    StorageNames map[string]string
}

func (s *Serializer) Serialize(element Element, version string) ([]byte, error) {
    schema := s.registry.GetSchema(version)
    return s.serializeWithSchema(element, schema)
}
```

**Pros:**
- Single serialization codebase
- Schema changes automatically picked up
- Accurate to actual Mendix behavior

**Cons:**
- More complex implementation
- Runtime schema loading overhead
- Must handle schema interpretation edge cases

### Option C: Hybrid Approach (Recommended)

Combine both approaches:
1. Use reflection data to **validate** and **inform** serialization
2. Use version-specific code for **critical differences**
3. Share common serialization logic via composition

```go
// Base serializer with common logic
type BaseSerializer struct {
    schema *VersionSchema
}

// Version-specific overrides
type V10Serializer struct {
    BaseSerializer
}

func (s *V10Serializer) SerializeValidationRule(vr *ValidationRule) bson.D {
    // V10+ uses BY_NAME_REFERENCE for Attribute
    return bson.D{
        {Key: "$ID", Value: s.serializeID(vr.ID)},
        {Key: "$Type", Value: "DomainModels$ValidationRule"},
        {Key: "Attribute", Value: s.qualifiedName(vr)},  // String
        // ...
    }
}
```

## Proposed Implementation

### Phase 1: Version Detection

Add version detection to MPR reading:

```go
type ProjectVersion struct {
    MendixVersion   string  // e.g., "10.18.0"
    MajorVersion    int     // e.g., 10
    MPRFormatVersion int    // 1 or 2
}

func (r *Reader) GetProjectVersion() (*ProjectVersion, error) {
    // Read from _ProjectVersion or metadata table
}
```

### Phase 2: Schema Registry

Load and cache reflection data:

```go
//go:embed reflection-data/*.json
var reflectionData embed.FS

type SchemaRegistry struct {
    schemas sync.Map  // version string -> *VersionSchema
}

func (r *SchemaRegistry) GetSchema(version string) (*VersionSchema, error) {
    // Find closest matching schema version
    // Load and parse JSON
    // Cache result
}
```

### Phase 3: Version-Aware Writer

```go
type WriterOptions struct {
    TargetVersion string  // e.g., "10.18.0" or "10" for latest 10.x
}

func NewWriter(path string, opts WriterOptions) (*Writer, error) {
    version := detectOrUseTarget(path, opts.TargetVersion)
    serializer := NewSerializer(version)
    return &Writer{
        reader:     reader,
        serializer: serializer,
        version:    version,
    }
}
```

### Phase 4: MDL Version Awareness

```sql
-- Specify target version in MDL
SET VERSION '10.18';

-- Or detect from connected project
CONNECT LOCAL './app.mpr';  -- Auto-detects version

-- Version-specific features show warnings
CREATE PERSISTENT ENTITY Module.Entity (
    -- NewFeature only available in 11.x
    NewField: SomeNewType  -- Warning: SomeNewType requires Mendix 11+
);
```

### Phase 5: Version Compatibility Checking

```go
type CompatibilityChecker struct {
    targetVersion *ProjectVersion
    schema        *VersionSchema
}

func (c *CompatibilityChecker) CheckEntity(e *Entity) []Warning {
    var warnings []Warning

    // Check if entity type is supported
    if e.Source == "SomeNewSource" && c.targetVersion.MajorVersion < 11 {
        warnings = append(warnings, Warning{
            Message: "Entity source 'SomeNewSource' requires Mendix 11+",
            Element: e,
        })
    }

    return warnings
}
```

## Key Version Differences to Handle

Based on reflection data analysis:

### Mendix 10.18+ (MPR v2)

- New file format: `mprcontents/` folder
- Content stored in separate `.mxunit` files
- Hash-based integrity checking

### Property Changes Across Versions

| Structure | Property | Versions | Notes |
|-----------|----------|----------|-------|
| `DomainModels$Association` | `storageFormat` | 11.0+ only | New in Mendix 11 |

### New Structure Types by Version

**10.0.0** (vs 9.24.0): 20 new structures

**10.18.0** (vs 10.0.0): 101 new structures, including:
- `DomainModels$OqlViewEntitySource` - View entity support
- `DomainModels$OqlViewAssociationSource` - View association support
- `DomainModels$OqlViewValue` - OQL view values
- `DomainModels$ViewEntitySource` - Base view entity source

**11.0.0** (vs 10.18.0): 25 new structures, including:
- `DomainModels$ViewEntitySourceDocument` - View entity documents

**11.6.0** (vs 11.0.0): 63 new structures

### Version Feature Matrix

| Feature | 9.x | 10.0-10.17 | 10.18+ | 11.x |
|---------|-----|------------|--------|------|
| MPR v1 format | Yes | Yes | No | No |
| MPR v2 format | No | No | Yes | Yes |
| View entities | Limited | Limited | Yes | Yes |
| Association storageFormat | No | No | No | Yes |
| Workflows | Yes | Yes | Yes | Yes |
| Business Events | Limited | Yes | Yes | Yes |

### Reference Type Changes

Some properties may change between BY_ID_REFERENCE and BY_NAME_REFERENCE across versions.

## Migration Path

### For Existing Code

1. Current serialization becomes `V10Serializer`
2. Add version detection to reader
3. Writer defaults to detected version
4. Add `--target-version` flag to mxcli

### For Users

```bash
# Auto-detect version from project
mxcli -p ./app.mpr

# Specify target version explicitly
mxcli -p ./app.mpr --target-version 10.18

# MDL version specification
mxcli -c "SET VERSION '10.18'; CREATE MODULE Foo;"
```

## Testing Strategy

1. **Version-specific test projects**: Maintain .mpr files for each major version
2. **Round-trip tests**: Read -> Modify -> Write -> Read for each version
3. **Studio Pro validation**: Open modified projects in corresponding Studio Pro versions
4. **Schema comparison tests**: Verify serialization matches reflection data expectations

## File Structure After Implementation

```
sdk/mpr/
├── reader.go
├── writer.go
├── version/
│   ├── detect.go
│   ├── schema.go
│   ├── registry.go
│   ├── base_serializer.go
│   ├── v9_serializer.go
│   ├── v10_serializer.go
│   └── v11_serializer.go
└── compat/
    ├── checker.go
    └── warnings.go

reference/mendixmodellib/reflection-data/
├── *.json                    # Existing reflection data
└── README.md                 # Documentation of schema format

docs/05-mdl-specification/
├── ...
└── 20-version-compatibility.md  # User-facing version docs
```

## Open Questions

1. **Granularity**: Should we support minor versions (10.18.0) or just major (10.x)?
2. **Fallback behavior**: What to do when target version lacks a feature?
3. **Reflection data updates**: How to keep reflection data current with new Mendix releases?
4. **Breaking changes**: How to handle breaking changes in MDL syntax across versions?

## Recommended Next Steps

1. **Analyze reflection data**: Script to compare structures across versions and identify differences
2. **Prototype version detection**: Implement basic version reading from MPR
3. **Create version compatibility matrix**: Document which features work in which versions
4. **Implement schema loader**: Load and parse reflection data at runtime
5. **Refactor serializers**: Extract current code into V10 serializer, add version dispatch

## Timeline Estimate

| Phase | Effort | Dependencies |
|-------|--------|--------------|
| Version detection | 1-2 days | None |
| Schema registry | 2-3 days | Version detection |
| Version-aware writer | 3-5 days | Schema registry |
| MDL version syntax | 2-3 days | Version-aware writer |
| Compatibility checking | 2-3 days | Schema registry |
| Testing & validation | 3-5 days | All above |

**Total: 2-3 weeks**

## References

- `reference/mendixmodellib/reflection-data/` - Metamodel definitions per version
- `docs/05-mdl-specification/10-bson-mapping.md` - Current BSON mapping docs
- TypeScript Model SDK - Reference implementation for version handling
