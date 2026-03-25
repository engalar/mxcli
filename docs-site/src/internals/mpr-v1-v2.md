# v1 (SQLite) vs v2 (mprcontents/)

Mendix uses two MPR file format versions. The library auto-detects and handles both.

## MPR v1 (Mendix < 10.18)

A single `.mpr` SQLite database file containing all model data.

### Storage

- `Unit` table: document metadata (name, type, container)
- `UnitContents` table: BSON document blobs

### Characteristics

- Self-contained single file
- Larger file size (entire project in one database)
- Binary diffs make Git versioning difficult
- All documents read/written through SQLite queries

### Reading

```go
// Auto-detected by modelsdk.Open()
reader, _ := modelsdk.Open("/path/to/project.mpr")
// reader.Version() returns 1
```

## MPR v2 (Mendix >= 10.18)

An `.mpr` metadata file plus a `mprcontents/` folder with individual document files.

### Storage

- `.mpr` file: SQLite with `Unit` table (metadata only, no `UnitContents`)
- `mprcontents/` folder: individual `.mxunit` files containing BSON

Each `.mxunit` file is named by the document's UUID and contains the raw BSON for that document.

### Directory Layout

```
project/
├── app.mpr                    # SQLite metadata
└── mprcontents/
    ├── 2a3b4c5d-...           # Domain model BSON
    ├── 6e7f8a9b-...           # Microflow BSON
    ├── c0d1e2f3-...           # Page BSON
    └── ...
```

### Characteristics

- Better for Git versioning -- individual files change independently
- Smaller diffs when modifying a single document
- Parallel read access to different documents
- Requires both the `.mpr` file and the `mprcontents/` folder

### Reading

```go
// Auto-detected by modelsdk.Open()
reader, _ := modelsdk.Open("/path/to/project.mpr")
// reader.Version() returns 2
```

## Format Detection

The library detects the format by checking whether the `UnitContents` table exists in the SQLite database:

| Condition | Format |
|-----------|--------|
| `UnitContents` table exists and has rows | v1 |
| `UnitContents` table missing or empty, `mprcontents/` folder exists | v2 |

This detection is automatic -- callers of `Open()` and `OpenForWriting()` do not need to specify the format.

## Comparison

| Feature | v1 | v2 |
|---------|----|----|
| Mendix version | < 10.18 | >= 10.18 |
| File structure | Single `.mpr` | `.mpr` + `mprcontents/` |
| Document storage | SQLite `UnitContents` table | Individual `.mxunit` files |
| Git friendliness | Poor (binary diffs) | Good (per-document files) |
| File size | Larger single file | Distributed across files |
| Read performance | Single DB query | File I/O per document |
| Write granularity | Full DB transaction | Per-file write |

## Writing Behavior

When writing with `OpenForWriting()`:

- **v1**: Documents are written as BSON blobs into the `UnitContents` table within a SQLite transaction
- **v2**: Documents are written as individual `.mxunit` files in the `mprcontents/` folder; the `Unit` table metadata is updated in SQLite

The writer handles both formats transparently.
