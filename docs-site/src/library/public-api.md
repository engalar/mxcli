# Public API

The public API surface is defined in `modelsdk.go` at the package root. It provides convenience functions for opening projects and constructing model elements.

## Entry Points

```go
// Read-only access
reader, err := modelsdk.Open("/path/to/project.mpr")
defer reader.Close()

// Read-write access
writer, err := modelsdk.OpenForWriting("/path/to/project.mpr")
defer writer.Close()
```

## Core Types

| Type | Description |
|------|-------------|
| `modelsdk.ID` | Unique identifier for model elements (UUID) |
| `modelsdk.Module` | Represents a Mendix module |
| `modelsdk.Project` | Represents a Mendix project |
| `modelsdk.DomainModel` | Contains entities and associations |
| `modelsdk.Entity` | An entity in the domain model |
| `modelsdk.Attribute` | An attribute of an entity |
| `modelsdk.Association` | A relationship between entities |
| `modelsdk.Microflow` | A microflow (server-side logic) |
| `modelsdk.Nanoflow` | A nanoflow (client-side logic) |
| `modelsdk.Page` | A page in the UI |
| `modelsdk.Layout` | A page layout template |

## Package Structure

```
github.com/mendixlabs/mxcli/
├── modelsdk.go          # Main package with convenience functions
├── model/               # Core model types (ID, Module, Project, etc.)
├── api/                 # High-level fluent API (builders)
│   ├── api.go           # ModelAPI entry point
│   ├── domainmodels.go  # EntityBuilder, AssociationBuilder
│   ├── enumerations.go  # EnumerationBuilder
│   ├── microflows.go    # MicroflowBuilder
│   ├── pages.go         # PageBuilder, widget builders
│   └── modules.go       # ModulesAPI
├── sdk/
│   ├── domainmodel/     # Domain model types (Entity, Attribute, Association)
│   ├── microflows/      # Microflow and Nanoflow types
│   ├── pages/           # Page, Layout, and Widget types
│   └── mpr/             # MPR file reader and writer
└── examples/            # Example applications
```

## Reader API

The reader provides read-only access to all project elements:

```go
reader, _ := modelsdk.Open("path/to/project.mpr")
defer reader.Close()

// Metadata
reader.Path()                    // Get file path
reader.Version()                 // Get MPR version (1 or 2)
reader.GetMendixVersion()        // Get Mendix Studio Pro version

// Modules
reader.ListModules()             // List all modules
reader.GetModule(id)             // Get module by ID
reader.GetModuleByName(name)     // Get module by name

// Domain Models
reader.ListDomainModels()        // List all domain models
reader.GetDomainModel(moduleID)  // Get domain model for module

// Microflows & Nanoflows
reader.ListMicroflows()          // List all microflows
reader.GetMicroflow(id)          // Get microflow by ID
reader.ListNanoflows()           // List all nanoflows
reader.GetNanoflow(id)           // Get nanoflow by ID

// Pages & Layouts
reader.ListPages()               // List all pages
reader.GetPage(id)               // Get page by ID
reader.ListLayouts()             // List all layouts
reader.GetLayout(id)             // Get layout by ID

// Other
reader.ListEnumerations()        // List all enumerations
reader.ListConstants()           // List all constants
reader.ListScheduledEvents()     // List all scheduled events
reader.ExportJSON()              // Export entire model as JSON
```

## Writer API

The writer provides read-write access. Use `writer.Reader()` for read operations:

```go
writer, _ := modelsdk.OpenForWriting("path/to/project.mpr")
defer writer.Close()

reader := writer.Reader()

// Module operations
writer.CreateModule(module)
writer.UpdateModule(module)
writer.DeleteModule(id)

// Entity operations
writer.CreateEntity(domainModelID, entity)
writer.UpdateEntity(domainModelID, entity)
writer.DeleteEntity(domainModelID, entityID)

// Attribute operations
writer.AddAttribute(domainModelID, entityID, attribute)

// Association operations
writer.CreateAssociation(domainModelID, association)
writer.DeleteAssociation(domainModelID, associationID)

// Microflow operations
writer.CreateMicroflow(microflow)
writer.UpdateMicroflow(microflow)
writer.DeleteMicroflow(id)

// Page operations
writer.CreatePage(page)
writer.UpdatePage(page)
writer.DeletePage(id)

// Other
writer.CreateEnumeration(enumeration)
writer.CreateConstant(constant)
```

## Constructor Functions

Helper functions for creating model elements with proper defaults:

```go
// Attributes
modelsdk.NewStringAttribute(name, length)
modelsdk.NewIntegerAttribute(name)
modelsdk.NewDecimalAttribute(name)
modelsdk.NewBooleanAttribute(name)
modelsdk.NewDateTimeAttribute(name, localize)
modelsdk.NewEnumerationAttribute(name, enumID)

// Entities
modelsdk.NewEntity(name)                 // Persistable
modelsdk.NewNonPersistableEntity(name)   // Non-persistable

// Associations
modelsdk.NewAssociation(name, parentID, childID)      // 1:N
modelsdk.NewReferenceSetAssociation(name, p, c)       // M:N

// Flows
modelsdk.NewMicroflow(name)
modelsdk.NewNanoflow(name)

// Pages
modelsdk.NewPage(name)

// IDs
modelsdk.GenerateID()
```
