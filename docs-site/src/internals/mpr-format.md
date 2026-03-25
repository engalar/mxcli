# MPR File Format

Mendix projects are stored in `.mpr` files, which are SQLite databases containing BSON-encoded model elements. This page provides an overview of the format; see [v1 vs v2](./mpr-v1-v2.md) for version-specific details.

## Structure

An MPR file is a standard SQLite database with two key tables:

### Unit Table

The `Unit` table stores document metadata:

| Column | Description |
|--------|-------------|
| `UnitID` | Binary UUID identifying the document |
| `ContainerID` | Parent module UUID |
| `ContainmentName` | Relationship name (e.g., `documents`) |
| `UnitType` | Fully qualified type name |
| `Name` | Document name |

### UnitContents Table (v1 only)

In MPR v1, the `UnitContents` table stores the actual BSON document content:

| Column | Description |
|--------|-------------|
| `UnitID` | Binary UUID matching the Unit table |
| `Contents` | BSON blob containing the full document |

In MPR v2, document contents are stored as individual files in the `mprcontents/` folder instead.

## Unit Types

Every document in a Mendix project has a unit type:

| UnitType | Document Type |
|----------|---------------|
| `DomainModels$DomainModel` | Domain model (entities, associations) |
| `DomainModels$ViewEntitySourceDocument` | OQL query for VIEW entities |
| `Microflows$Microflow` | Microflow definition |
| `Microflows$Nanoflow` | Nanoflow definition |
| `Pages$Page` | Page definition |
| `Pages$Layout` | Layout definition |
| `Pages$Snippet` | Snippet definition |
| `Pages$BuildingBlock` | Building block definition |
| `Enumerations$Enumeration` | Enumeration definition |
| `JavaActions$JavaAction` | Java action definition |
| `Security$ProjectSecurity` | Project security settings |
| `Security$ModuleSecurity` | Module security settings |
| `Navigation$NavigationDocument` | Navigation profile |
| `Settings$ProjectSettings` | Project settings |
| `BusinessEvents$BusinessEventService` | Business event service |

## BSON Document Structure

Every BSON document contains at minimum:

| Field | Type | Description |
|-------|------|-------------|
| `$ID` | Binary (UUID) | Unique identifier for this element |
| `$Type` | String | Fully qualified type name (using **storageName**, not qualifiedName) |

### ID Format

IDs are stored as BSON Binary subtype 0 (generic) containing UUID bytes:

```json
{
  "$ID": {
    "Subtype": 0,
    "Data": "base64-encoded-uuid"
  }
}
```

### Array Convention

Arrays in Mendix BSON have an integer count as the first element:

```json
{
  "Attributes": [
    3,              // Count/type prefix
    { ... },        // First attribute
    { ... }         // Second attribute
  ]
}
```

The prefix value (typically `2` or `3`) indicates the array type. When writing arrays, you must include this prefix. When parsing, skip the first element.

### Reference Types

| Reference Type | Storage Format | Example |
|---------------|----------------|---------|
| `BY_ID_REFERENCE` | Binary UUID | Index `AttributePointer` |
| `BY_NAME_REFERENCE` | Qualified name string | ValidationRule `Attribute` |
| `PART` | Embedded BSON object | Child objects serialized inline |

Using the wrong reference format causes Studio Pro to fail loading the model. Always check the metamodel reflection data to determine which format each property uses.

## Storage Names vs Qualified Names

The `$Type` field in BSON must use the **storageName**, not the **qualifiedName**. These are often identical but not always:

| qualifiedName (SDK) | storageName (BSON $Type) |
|---------------------|--------------------------|
| `DomainModels$Entity` | `DomainModels$EntityImpl` |
| `DomainModels$Index` | `DomainModels$EntityIndex` |

Using the wrong name causes `TypeCacheUnknownTypeException` when opening in Studio Pro. See [Storage Names](./storage-names.md) for more details.
