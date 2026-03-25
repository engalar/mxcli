# Mendix Version Compatibility

Supported Mendix Studio Pro versions and MPR format mapping.

## Supported Versions

mxcli supports Mendix Studio Pro versions **8.x through 11.x**.

| Studio Pro Version | MPR Format | Status |
|-------------------|------------|--------|
| 8.x | v1 | Supported |
| 9.x | v1 | Supported |
| 10.0 -- 10.17 | v1 | Supported |
| 10.18+ | v2 | Supported |
| 11.x | v2 | Supported (primary development target) |

> **Note:** Development and testing is primarily done against **Mendix 11.6**. Other versions are supported but may have untested edge cases.

## MPR Format Versions

### v1 (Mendix < 10.18)

- Single `.mpr` SQLite database file
- All documents stored as BSON blobs in the `UnitContents` table
- Self-contained -- one file holds the entire project

### v2 (Mendix >= 10.18)

- `.mpr` SQLite file for metadata only
- `mprcontents/` folder with individual `.mxunit` files for each document
- Better suited for Git version control (smaller, per-document diffs)

The library auto-detects the format. No configuration is needed.

## Widget Template Versions

Pluggable widget templates are versioned by Mendix release. The embedded templates cover:

| Mendix Version | Template Set |
|---------------|--------------|
| 10.6.0 | DataGrid2, ComboBox, Gallery, and others |
| 11.6.0 | Updated templates for current widgets |

When creating pages with pluggable widgets, the library selects the template matching the project's Mendix version, falling back to the nearest earlier version if an exact match is not available.

## Feature Availability by Version

| Feature | Minimum Version | Notes |
|---------|----------------|-------|
| Domain models | 8.x | Full support |
| Microflows | 8.x | 60+ activity types |
| Nanoflows | 8.x | Client-side flows |
| Pages | 8.x | 50+ widget types |
| Pluggable widgets | 9.x | Requires widget templates |
| Workflows | 10.x | User tasks, decisions, parallel splits |
| Business events | 10.x | Event service definitions |
| View entities | 10.x | OQL-backed entities |
| MPR v2 format | 10.18 | Per-document file storage |
| Calculated attributes | 10.x | CALCULATED BY microflow |

## MxBuild Compatibility

The `mx` validation tool must match the project's Mendix version:

```bash
# Auto-download the correct MxBuild version
mxcli setup mxbuild -p app.mpr

# Check the project
~/.mxcli/mxbuild/*/modeler/mx check app.mpr
```

MxBuild is downloaded on demand and cached in `~/.mxcli/mxbuild/{version}/`.

## Platform Support

mxcli runs on:

| Platform | Architecture |
|----------|-------------|
| Linux | amd64, arm64 |
| macOS | amd64, arm64 (Apple Silicon) |
| Windows | amd64, arm64 |

No CGO or C compiler is required -- the binary is fully statically linked using pure Go dependencies.
