# Reader Methods

Complete list of methods available on the reader returned by `modelsdk.Open()`.

## Metadata

```go
reader.Path()                    // string -- file path of the .mpr
reader.Version()                 // int -- MPR version (1 or 2)
reader.GetMendixVersion()        // string -- Mendix Studio Pro version (e.g., "11.6.0")
```

## Modules

```go
reader.ListModules()             // ([]*model.Module, error) -- all modules
reader.GetModule(id)             // (*model.Module, error) -- module by ID
reader.GetModuleByName(name)     // (*model.Module, error) -- module by name
```

## Domain Models

```go
reader.ListDomainModels()        // ([]*domainmodel.DomainModel, error) -- all domain models
reader.GetDomainModel(moduleID)  // (*domainmodel.DomainModel, error) -- domain model for a module
```

The `DomainModel` struct contains:

- `Entities` -- slice of `*domainmodel.Entity`, each with `Attributes`, `Indexes`, `ValidationRules`, `AccessRules`, and `EventHandlers`
- `Associations` -- slice of `*domainmodel.Association`

## Microflows and Nanoflows

```go
reader.ListMicroflows()          // ([]*microflows.Microflow, error) -- all microflows
reader.GetMicroflow(id)          // (*microflows.Microflow, error) -- microflow by ID
reader.ListNanoflows()           // ([]*microflows.Nanoflow, error) -- all nanoflows
reader.GetNanoflow(id)           // (*microflows.Nanoflow, error) -- nanoflow by ID
```

## Pages and Layouts

```go
reader.ListPages()               // ([]*pages.Page, error) -- all pages
reader.GetPage(id)               // (*pages.Page, error) -- page by ID
reader.ListLayouts()             // ([]*pages.Layout, error) -- all layouts
reader.GetLayout(id)             // (*pages.Layout, error) -- layout by ID
```

## Enumerations

```go
reader.ListEnumerations()        // ([]*model.Enumeration, error) -- all enumerations
```

Each enumeration contains:

- `Name` -- enumeration name
- `Values` -- slice of `*model.EnumerationValue` with `Name` and `Caption`

## Other Documents

```go
reader.ListConstants()           // ([]*model.Constant, error) -- all constants
reader.ListScheduledEvents()     // ([]*model.ScheduledEvent, error) -- all scheduled events
```

## Export

```go
reader.ExportJSON()              // ([]byte, error) -- entire model as JSON
```

## Example: Full Project Scan

```go
reader, _ := modelsdk.Open("/path/to/MyApp.mpr")
defer reader.Close()

fmt.Printf("Mendix version: %s\n", reader.GetMendixVersion())

modules, _ := reader.ListModules()
for _, m := range modules {
    fmt.Printf("Module: %s\n", m.Name)

    dm, _ := reader.GetDomainModel(m.ID)
    for _, entity := range dm.Entities {
        fmt.Printf("  Entity: %s (persistent: %v)\n", entity.Name, entity.Persistable)
        for _, attr := range entity.Attributes {
            fmt.Printf("    - %s: %s\n", attr.Name, attr.Type.GetTypeName())
        }
    }
}

microflows, _ := reader.ListMicroflows()
fmt.Printf("Total microflows: %d\n", len(microflows))

pages, _ := reader.ListPages()
fmt.Printf("Total pages: %d\n", len(pages))
```
