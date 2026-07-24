# Marketplace Command Redesign

## Context

The `mxcli marketplace` command provides CLI access to the Mendix Marketplace for browsing, downloading, and installing modules and widgets. A previous spike (2026-04-16) found that the marketplace API at `marketplace-api.mendix.com` did not expose `.mpk` download URLs, blocking install. A re-probe on 2026-07-24 confirms the API **now returns `downloadUrl`** on every version object, unblocking the full pipeline.

### Verified API State

| Endpoint | Auth | Status |
|----------|------|--------|
| `GET /v1/content?search=` | MxToken PAT | ✅ Returns filtered content list |
| `GET /v1/content/{id}` | MxToken PAT | ✅ Single content detail |
| `GET /v1/content/{id}/versions` | MxToken PAT | ✅ Version list with `downloadUrl`, `versionType` |
| `GET marketplace.mendix.com/v1/versions/{versionId}/download` | MxToken PAT | ✅ 303 redirect to CDN |

### New fields confirmed

- `downloadUrl` on Version — `https://marketplace.mendix.com/v1/versions/{versionId}/download`
- `versionType` on Version — `"Regular"`
- `isCompanyApproved` on Content — boolean
- `downloadUrl` on `Content.latestVersion` — convenience field

## Subcommands

| # | Command | Description |
|---|---------|-------------|
| 1 | `search <query>` | Search marketplace content |
| 2 | `info <id>` | Show item details |
| 3 | `versions <id>` | List versions with optional min-mendix filter |
| 4 | `download <id>` | Download .mpk with atomic write |
| 5 | `install <id> -p app.mpr` | Download + mx module-import |
| 6 | `list -p app.mpr` | List installed marketplace modules |
| 7 | `update [<id>] -p app.mpr` | Check for updates (report only) |

## DDD Architecture

### Package Layout

```
internal/marketplace/
├── domain/                          # Pure domain — zero external imports
│   ├── content.go                   # Content aggregate, Version value object
│   └── repository.go                # 5 repository interfaces
│
├── application/                     # Orchestration — one public struct
│   └── service.go                   # MarketplaceService (9 methods)
│
├── infrastructure/                  # IO — HTTP, process, filesystem
│   ├── api_client.go               # REST client (implements ContentRepo + VersionRepo)
│   ├── api_client_test.go
│   ├── downloader.go               # 303→CDN redirect, atomic write
│   ├── downloader_test.go
│   ├── installer.go                # mx module-import wrapper
│   ├── installer_test.go
│   ├── project.go                  # List installed modules from MPR
│   ├── project_test.go
│   └── cache.go                    # Download + search catalog cache

cmd/mxcli/
├── cmd_marketplace.go               # Cobra commands — call Service, render output
└── cmd_marketplace_test.go          # CLI integration tests (httptest + mock)
```

### Repository Interfaces

```go
type ContentRepository interface {
    Search(ctx, query, limit) ([]Content, error)
    Get(ctx, id ContentID) (*Content, error)
}
type VersionRepository interface {
    GetVersions(ctx, id ContentID) ([]Version, error)
}
type DownloadRepository interface {
    DownloadVersion(ctx, version, destPath) error
}
type InstalledModuleLister interface {
    ListInstalledModules(projectPath) ([]InstalledModule, error)
}
type ModuleInstaller interface {
    InstallModule(ctx, mpkPath, projectPath) error
}
```

### Download Flow

1. Service gets versions → finds target version
2. Extracts version.DownloadURL
3. Downloader: GET with MxToken auth, 303 redirect intercepted (not followed automatically)
4. Extract CDN URL from Location header
5. Plain HTTP client GETs CDN URL → stream to temp file → rename to dest

### Install Safety Lock

When a module is already installed (same `AppStoreGuid` detected via `ListInstalledModules`), `install` reports and stops. In-place module updates are not applied automatically because they can discard local edits and change persistent entity IDs.

## Key Design Decisions

- `files.appstore.mendix.com` is NOT added to `auth/scheme.go` — downloader uses a separate plain HTTP client for CDN access
- `marketplace.mendix.com` added to `auth/scheme.go` with `SchemePAT`
- Download uses atomic file write (temp file + rename)
- Search cached at `~/.mxcli/marketplace-cache/catalog-{profile}.json` with 24h TTL
- All 7 subcommands support `--json` flag

## Test Coverage

| Package | Tests | Coverage |
|---------|-------|----------|
| domain/ | 0 (pure types) | N/A |
| application/ | 13 | All use cases covered |
| infrastructure/ | 22 | API client, cache, downloader, installer, project reader |
| cmd/mxcli/ (marketplace) | 12 | All CLI commands |
| auth/ | 26 | Auth scheme for marketplace.mendix.com verified |
