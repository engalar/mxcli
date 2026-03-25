# Open / OpenForWriting

The two entry points for accessing a Mendix project from Go code.

## Read-Only Access

Use `modelsdk.Open()` when you only need to inspect the project:

```go
import modelsdk "github.com/mendixlabs/mxcli"

reader, err := modelsdk.Open("/path/to/MyApp.mpr")
if err != nil {
    log.Fatal(err)
}
defer reader.Close()

// Query the project
modules, _ := reader.ListModules()
dm, _ := reader.GetDomainModel(modules[0].ID)
```

The reader provides all query methods (listing modules, entities, microflows, pages, etc.) but cannot modify the project. The `.mpr` file is opened with a read-only SQLite connection.

## Read-Write Access

Use `modelsdk.OpenForWriting()` when you need to create or modify elements:

```go
import modelsdk "github.com/mendixlabs/mxcli"

writer, err := modelsdk.OpenForWriting("/path/to/MyApp.mpr")
if err != nil {
    log.Fatal(err)
}
defer writer.Close()

// Access read methods through the embedded reader
reader := writer.Reader()
modules, _ := reader.ListModules()

// Create new elements
entity := modelsdk.NewEntity("Customer")
writer.CreateEntity(domainModelID, entity)
```

The writer opens the `.mpr` file with a read-write SQLite connection. **Always back up your `.mpr` file before modifying it** -- the library writes directly to the database.

## Format Auto-Detection

Both functions automatically detect the MPR format version:

| Version | Mendix | Storage |
|---------|--------|---------|
| **v1** | < 10.18 | Single `.mpr` SQLite file with BSON blobs |
| **v2** | >= 10.18 | `.mpr` metadata + `mprcontents/` folder with `.mxunit` files |

No configuration is needed -- the library inspects the file structure and selects the appropriate reader/writer implementation.

## Error Handling

Both functions return an error if:

- The file path does not exist
- The file is not a valid SQLite database
- The file is not a recognized MPR format
- (For `OpenForWriting`) the file is read-only or locked by another process

```go
reader, err := modelsdk.Open("/path/to/project.mpr")
if err != nil {
    // Handle: file not found, not a valid MPR, etc.
    log.Fatalf("Failed to open project: %v", err)
}
```

## Pure Go -- No CGO Required

The library uses `modernc.org/sqlite`, a pure Go SQLite implementation. No C compiler, CGO, or system SQLite library is needed. This simplifies cross-compilation and deployment.
