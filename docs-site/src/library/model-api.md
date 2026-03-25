# ModelAPI Entry Point

The `api` package provides a high-level fluent API inspired by the Mendix Web Extensibility Model API. It wraps the lower-level reader/writer with builder patterns for common operations.

## Creating the ModelAPI

```go
import (
    "github.com/mendixlabs/mxcli/api"
    "github.com/mendixlabs/mxcli/sdk/mpr"
)

writer, _ := mpr.OpenForWriting("/path/to/MyApp.mpr")
defer writer.Close()

modelAPI := api.New(writer)
```

## Setting the Module Context

Most operations require a module context. Set it before calling builders:

```go
module, _ := modelAPI.Modules.GetModule("MyModule")
modelAPI.SetModule(module)
```

All subsequent `Create*` calls will create elements in this module.

## Namespace Accessors

The `ModelAPI` struct provides access to domain-specific APIs through namespace accessors:

| Namespace | Accessor | Description |
|-----------|----------|-------------|
| **DomainModels** | `modelAPI.DomainModels` | Create/modify entities, attributes, associations |
| **Enumerations** | `modelAPI.Enumerations` | Create/modify enumerations and their values |
| **Microflows** | `modelAPI.Microflows` | Create microflows with parameters and return types |
| **Pages** | `modelAPI.Pages` | Create pages with widgets |
| **Modules** | `modelAPI.Modules` | List and retrieve modules |

## DomainModels Namespace

```go
// Create an entity
entity, _ := modelAPI.DomainModels.CreateEntity("Customer").
    Persistent().
    WithStringAttribute("Name", 200).
    Build()

// Create an association
assoc, _ := modelAPI.DomainModels.CreateAssociation("Customer_Orders").
    From("Customer").
    To("Order").
    OneToMany().
    Build()
```

## Enumerations Namespace

```go
enum, _ := modelAPI.Enumerations.CreateEnumeration("OrderStatus").
    WithValue("Draft", "Draft").
    WithValue("Active", "Active").
    WithValue("Closed", "Closed").
    Build()
```

## Microflows Namespace

```go
mf, _ := modelAPI.Microflows.CreateMicroflow("ACT_ProcessOrder").
    WithParameter("Order", "MyModule.Order").
    WithStringParameter("Note").
    ReturnsBoolean().
    Build()
```

## Pages Namespace

```go
page, _ := modelAPI.Pages.CreatePage("CustomerOverview").
    WithTitle("Customer Overview").
    Build()
```

## Modules Namespace

```go
// List all modules
modules, _ := modelAPI.Modules.ListModules()

// Get a specific module
module, _ := modelAPI.Modules.GetModule("MyModule")
```

## MDL to Fluent API Mapping

| MDL Statement | Fluent API Method |
|---------------|-------------------|
| `CREATE PERSISTENT ENTITY` | `DomainModels.CreateEntity().Persistent().Build()` |
| `CREATE NON-PERSISTENT ENTITY` | `DomainModels.CreateEntity().NonPersistent().Build()` |
| `CREATE ASSOCIATION` | `DomainModels.CreateAssociation().Build()` |
| `CREATE ENUMERATION` | `Enumerations.CreateEnumeration().Build()` |
| `CREATE MICROFLOW` | `Microflows.CreateMicroflow().Build()` |
| `CREATE PAGE` | `Pages.CreatePage().Build()` |

## When to Use the Fluent API vs Direct SDK

| Use Case | Recommended Approach |
|----------|---------------------|
| Simple CRUD operations | Fluent API (less boilerplate) |
| Complex microflow construction | Direct SDK types (more control) |
| Bulk operations | Direct writer methods (batch efficiency) |
| Integration testing | Fluent API (readable test code) |
