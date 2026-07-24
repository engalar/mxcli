# Marketplace Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement `download`, `install`, `list`, and `update` subcommands for `mxcli marketplace`, and refactor the existing `search`/`info`/`versions` subcommands into a DDD-layered architecture.

**Architecture:** DDD layers: `domain/` (pure types + repository interfaces), `application/` (MarketplaceService orchestration), `infrastructure/` (REST client, downloader, installer, project reader). Cobra CLI layer (`cmd_marketplace.go`) calls service only.

**Tech Stack:** Go, Cobra, httptest, auth.ClientFor, modelsdk Reader (for project module queries), `mx module-import` subprocess, marketplace API at `marketplace-api.mendix.com` + `marketplace.mendix.com`.

## Global Constraints

- All existing tests must continue to pass
- `files.appstore.mendix.com` must NOT be added to `auth/scheme.go` — downloader handles CDN access via a plain HTTP client
- `marketplace.mendix.com` must be added to `auth/scheme.go` with `SchemePAT`
- Download must use atomic file write (temp file + rename)
- Install must NOT auto-update already-installed modules (safety lock per spec)
- All new types must mirror API JSON field names exactly
- Search cache stored at `~/.mxcli/marketplace-cache/catalog-{profile}.json` with 24h TTL
- All 7 subcommands must produce equivalent JSON output for `--json` flag

---
### Task 1: Refactor domain types to new layout

**Files:**
- Create: `internal/marketplace/domain/content.go`
- Create: `internal/marketplace/domain/repository.go`
- Delete (eventually): `internal/marketplace/types.go` (after Task 2 is done)
- Delete (eventually): `internal/marketplace/client.go` (after Task 2 is done)

**Interfaces:**
- Produces: `domain.Content`, `domain.Version`, `domain.Category`, `domain.ContentID`, `domain.VersionID`, `domain.InstalledModule`
- Produces: `domain.ContentRepository`, `domain.VersionRepository`, `domain.DownloadRepository`, `domain.ProjectRepository`

- [ ] **Step 1: Create `domain/content.go`**

Extract existing types from `internal/marketplace/types.go`, add new fields:

```go
package domain

import "time"

type ContentID int
type VersionID string

type Content struct {
    ContentID       ContentID  `json:"contentId"`
    Publisher       string     `json:"publisher"`
    Type            string     `json:"type"`
    Categories      []Category `json:"categories"`
    SupportCategory string     `json:"supportCategory"`
    LicenseURL      string     `json:"licenseUrl,omitempty"`
    IsPrivate       bool       `json:"isPrivate"`
    IsCompanyApproved bool     `json:"isCompanyApproved,omitempty"`
    LatestVersion   *Version   `json:"latestVersion,omitempty"`
}

type Category struct {
    Name string `json:"name"`
}

type Version struct {
    Name                      string    `json:"name"`
    VersionID                 string    `json:"versionId"`
    VersionNumber             string    `json:"versionNumber"`
    MinSupportedMendixVersion string    `json:"minSupportedMendixVersion"`
    PublicationDate           time.Time `json:"publicationDate"`
    ReleaseNotes              string    `json:"releaseNotes,omitempty"`
    VersionType               string    `json:"versionType,omitempty"`
    DownloadURL               string    `json:"downloadUrl,omitempty"`
}

type InstalledModule struct {
    Name            string
    ModuleID        string
    AppStoreGuid    string
    AppStoreVersion string
}
```

- [ ] **Step 2: Create `domain/repository.go`**

```go
package domain

import "context"

type ContentRepository interface {
    Search(ctx context.Context, query string, limit int) ([]*Content, error)
    Get(ctx context.Context, id ContentID) (*Content, error)
}

type VersionRepository interface {
    GetVersions(ctx context.Context, id ContentID) ([]*Version, error)
}

type DownloadRepository interface {
    DownloadVersion(ctx context.Context, version *Version, destPath string) error
}

type ProjectRepository interface {
    ListInstalledModules(projectPath string) ([]InstalledModule, error)
    InstallModule(ctx context.Context, mpkPath, projectPath string) error
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/marketplace/...`
Expected: compiles (currently there is no `internal/marketplace/domain/` package yet, but the new files should compile on their own since they have no external imports.)

---
### Task 2: Refactor REST client to infrastructure/api_client.go

**Files:**
- Create: `internal/marketplace/infrastructure/api_client.go`
- Create: `internal/marketplace/infrastructure/api_client_test.go`
- Modify: `internal/marketplace/domain/repository.go` (already done in Task 1)
- Delete: `internal/marketplace/client.go`
- Delete: `internal/marketplace/client_test.go`
- Delete: `internal/marketplace/types.go`

**Interfaces:**
- Consumes: `domain.ContentRepository`, `domain.VersionRepository`
- Produces: `infrastructure.APIClient` implementing both interfaces

- [ ] **Step 1: Write the failing test**

Create `internal/marketplace/infrastructure/api_client_test.go`:

```go
package infrastructure

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/mendixlabs/mxcli/internal/marketplace/domain"
)

func TestSearch_PassesQueryAndLimit(t *testing.T) {
    var gotPath, gotQuery string
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        gotPath = r.URL.Path
        gotQuery = r.URL.RawQuery
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"items":[{"contentId":2888,"publisher":"Mendix","type":"Module","latestVersion":{"name":"Database Connector","versionId":"aaaa","versionNumber":"3.1.0","minSupportedMendixVersion":"9.0.0","publicationDate":"2025-06-01T00:00:00Z"}}]}`))
    }))
    t.Cleanup(ts.Close)

    client := NewAPIClient(ts.Client(), ts.URL)
    results, err := client.Search(context.Background(), "database", 10)
    if err != nil {
        t.Fatal(err)
    }
    if gotPath != "/v1/content" {
        t.Errorf("path: got %q, want /v1/content", gotPath)
    }
    if len(results) == 0 {
        t.Error("expected results")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/marketplace/infrastructure/ -run TestSearch_PassesQueryAndLimit -v`
Expected: FAIL (package doesn't exist yet)

- [ ] **Step 3: Write minimal implementation**

Create `internal/marketplace/infrastructure/api_client.go`:

```go
package infrastructure

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strconv"
    "strings"

    "github.com/mendixlabs/mxcli/internal/marketplace/domain"
)

const searchFetchLimit = 200

type APIClient struct {
    httpClient *http.Client
    baseURL    string
}

func NewAPIClient(httpClient *http.Client, baseURL string) *APIClient {
    return &APIClient{httpClient: httpClient, baseURL: baseURL}
}

func (c *APIClient) Search(ctx context.Context, query string, limit int) ([]*domain.Content, error) {
    q := url.Values{}
    fetchLimit := limit
    if query != "" {
        q.Set("search", query)
        fetchLimit = searchFetchLimit
    }
    if fetchLimit > 0 {
        q.Set("limit", strconv.Itoa(fetchLimit))
    }
    path := "/v1/content"
    if len(q) > 0 {
        path += "?" + q.Encode()
    }
    var result struct {
        Items []*domain.Content `json:"items"`
    }
    if err := c.get(ctx, path, &result); err != nil {
        return nil, err
    }
    if query != "" {
        matched := filterItems(result.Items, query)
        if limit > 0 && len(matched) > limit {
            matched = matched[:limit]
        }
        return matched, nil
    }
    return result.Items, nil
}

func filterItems(items []*domain.Content, query string) []*domain.Content {
    q := strings.ToLower(query)
    var out []*domain.Content
    for _, item := range items {
        name := ""
        if item.LatestVersion != nil {
            name = strings.ToLower(item.LatestVersion.Name)
        }
        if strings.Contains(name, q) || strings.Contains(strings.ToLower(item.Publisher), q) {
            out = append(out, item)
        }
    }
    return out
}

func (c *APIClient) Get(ctx context.Context, id domain.ContentID) (*domain.Content, error) {
    var out domain.Content
    if err := c.get(ctx, fmt.Sprintf("/v1/content/%d", id), &out); err != nil {
        return nil, err
    }
    return &out, nil
}

func (c *APIClient) GetVersions(ctx context.Context, id domain.ContentID) ([]*domain.Version, error) {
    var result struct {
        Items []*domain.Version `json:"items"`
    }
    if err := c.get(ctx, fmt.Sprintf("/v1/content/%d/versions", id), &result); err != nil {
        return nil, err
    }
    return result.Items, nil
}

func (c *APIClient) get(ctx context.Context, path string, dst any) error {
    req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
    if err != nil {
        return err
    }
    req.Header.Set("Accept", "application/json")
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("marketplace %s: %w", path, err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
        return fmt.Errorf("marketplace %s: HTTP %d: %s", path, resp.StatusCode, string(body))
    }
    return json.NewDecoder(resp.Body).Decode(dst)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/marketplace/infrastructure/ -run TestSearch_PassesQueryAndLimit -v`
Expected: PASS

- [ ] **Step 5: Port all existing client_test.go test cases**

Copy all test functions from `internal/marketplace/client_test.go` into `api_client_test.go`:
- `TestSearch_ClientSideFiltering`
- `TestSearch_ClientSideFiltering_NoMatch`
- `TestSearch_ClientSideFiltering_LimitApplied`
- `TestSearch_PublisherFiltering`
- `TestSearch_NoQueryOrLimit`
- `TestGet_ParsesContentDetail`
- `TestVersions_ParsesList`
- `TestGet_HTTPErrorIsReported`
- `TestGet_InvalidJSONReported`
- `TestNew_UsesDefaultBaseURL`

Add the downloadUrl field assertions to version tests (check that `v.DownloadURL` is populated).

- [ ] **Step 6: Run all tests**

Run: `go test ./internal/marketplace/infrastructure/ -v`
Expected: All PASS

- [ ] **Step 7: Remove old files and commit**

```bash
git rm internal/marketplace/types.go internal/marketplace/client.go internal/marketplace/client_test.go
git add internal/marketplace/domain/ internal/marketplace/infrastructure/
git commit -m "refactor(marketplace): migrate to DDD layers (domain + infrastructure)"
```

---
### Task 3: Add marketplace.mendix.com to auth scheme

**Files:**
- Modify: `internal/auth/scheme.go`

**Interfaces:**
- Produces: `marketplace.mendix.com` recognized by `auth.ClientFor`

- [ ] **Step 1: Add the host**

In `internal/auth/scheme.go`, add to `hostSchemes`:

```go
"marketplace.mendix.com": SchemePAT,   // download endpoint
```

- [ ] **Step 2: Run auth tests**

Run: `go test ./internal/auth/ -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/auth/scheme.go
git commit -m "feat(auth): add marketplace.mendix.com to known hosts"
```

---
### Task 4: Implement Downloader

**Files:**
- Create: `internal/marketplace/infrastructure/downloader.go`
- Create: `internal/marketplace/infrastructure/downloader_test.go`

**Interfaces:**
- Consumes: `domain.DownloadRepository`
- Produces: `downloader.DownloadVersion()` — downloads .mpk via 303→CDN

- [ ] **Step 1: Write the failing test**

```go
package infrastructure

import (
    "context"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "testing"
    "github.com/mendixlabs/mxcli/internal/marketplace/domain"
)

func TestDownloader_FollowsRedirectToCDN(t *testing.T) {
    // Mock marketplace.mendix.com: returns 303 → CDN URL
    // Mock files.appstore.mendix.com: returns .mpk content
    var cdnCalled bool
    cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        cdnCalled = true
        w.Header().Set("Content-Type", "application/octet-stream")
        w.Write([]byte("fake-mpk-content"))
    }))
    t.Cleanup(cdn.Close)

    api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Location", cdn.URL+"/module.mpk")
        w.WriteHeader(http.StatusSeeOther)
    }))
    t.Cleanup(api.Close)

    d := NewDownloader(http.DefaultClient, "test-token")
    d.baseURL = api.URL // test hook

    dest := filepath.Join(t.TempDir(), "module.mpk")
    err := d.DownloadVersion(context.Background(), &domain.Version{
        DownloadURL: api.URL + "/download",
    }, dest)
    if err != nil {
        t.Fatal(err)
    }
    if !cdnCalled {
        t.Error("CDN was never called")
    }
    data, _ := os.ReadFile(dest)
    if string(data) != "fake-mpk-content" {
        t.Errorf("got %q, want %q", string(data), "fake-mpk-content")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/marketplace/infrastructure/ -run TestDownloader_FollowsRedirectToCDN -v`
Expected: FAIL (package doesn't exist yet)

- [ ] **Step 3: Write minimal implementation**

```go
package infrastructure

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"

    "github.com/mendixlabs/mxcli/internal/marketplace/domain"
)

type Downloader struct {
    httpClient *http.Client
    token      string
    baseURL    string // test hook; empty = use download URL as-is
}

func NewDownloader(httpClient *http.Client, token string) *Downloader {
    return &Downloader{httpClient: httpClient, token: token}
}

func (d *Downloader) DownloadVersion(ctx context.Context, version *domain.Version, destPath string) error {
    dlURL := version.DownloadURL
    if d.baseURL != "" {
        dlURL = d.baseURL + "/download"
    }

    // Step 1: request with auth, don't follow redirect
    req, err := http.NewRequestWithContext(ctx, "GET", dlURL, nil)
    if err != nil {
        return fmt.Errorf("download request: %w", err)
    }
    req.Header.Set("Authorization", "MxToken "+d.token)

    client := &http.Client{
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            return http.ErrUseLastResponse
        },
    }
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("download auth: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusSeeOther && resp.StatusCode != http.StatusFound {
        return fmt.Errorf("download expected 303 redirect, got %d", resp.StatusCode)
    }

    cdnURL := resp.Header.Get("Location")
    if cdnURL == "" {
        return fmt.Errorf("download: no Location header in 303 response")
    }

    // Step 2: download from CDN (public, no auth)
    cdnResp, err := d.httpClient.Get(cdnURL)
    if err != nil {
        return fmt.Errorf("cdn download: %w", err)
    }
    defer cdnResp.Body.Close()

    if cdnResp.StatusCode != http.StatusOK {
        return fmt.Errorf("cdn download: HTTP %d", cdnResp.StatusCode)
    }

    // Step 3: atomic write to temp file
    if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
        return fmt.Errorf("create dir: %w", err)
    }
    tmpPath := destPath + ".tmp"
    f, err := os.Create(tmpPath)
    if err != nil {
        return fmt.Errorf("create temp: %w", err)
    }
    if _, err := io.Copy(f, cdnResp.Body); err != nil {
        f.Close()
        os.Remove(tmpPath)
        return fmt.Errorf("write temp: %w", err)
    }
    if err := f.Close(); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("close temp: %w", err)
    }
    if err := os.Rename(tmpPath, destPath); err != nil {
        os.Remove(tmpPath)
        return fmt.Errorf("rename: %w", err)
    }
    return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/marketplace/infrastructure/ -run TestDownloader_FollowsRedirectToCDN -v`
Expected: PASS

- [ ] **Step 5: Add tests for error cases**

Add to `downloader_test.go`:
- `TestDownloader_NoRedirectReturnsError` — mock returns 200 instead of 303
- `TestDownloader_CDNFailureReturnsError` — CDN returns 500

- [ ] **Step 6: Run all tests**

Run: `go test ./internal/marketplace/infrastructure/ -v`
Expected: All PASS

- [ ] **Step 7: Commit**

```bash
git add internal/marketplace/infrastructure/downloader.go internal/marketplace/infrastructure/downloader_test.go
git commit -m "feat(marketplace): implement downloader with 303→CDN redirect"
```

---
### Task 5: Implement Installer (mx module-import wrapper)

**Files:**
- Create: `internal/marketplace/infrastructure/installer.go`
- Create: `internal/marketplace/infrastructure/installer_test.go`

**Interfaces:**
- Consumes: `domain.ProjectRepository`
- Produces: `installer.InstallModule()` — runs `mx module-import`

- [ ] **Step 1: Write the failing test**

```go
package infrastructure

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "github.com/mendixlabs/mxcli/internal/marketplace/domain"
)

func TestInstaller_RunsMxModuleImport(t *testing.T) {
    // Create a fake "mx" binary
    binDir := t.TempDir()
    mxPath := filepath.Join(binDir, "mx")
    script := "#!/bin/sh\necho 'import ok'"
    if err := os.WriteFile(mxPath, []byte(script), 0755); err != nil {
        t.Fatal(err)
    }

    inst := NewInstaller(mxPath)
    mpkPath := filepath.Join(t.TempDir(), "test.mpk")
    os.WriteFile(mpkPath, []byte("fake"), 0644)

    err := inst.InstallModule(context.Background(), mpkPath, "/tmp/test.mpr")
    if err != nil {
        t.Fatal(err)
    }
}

func TestInstaller_NotFoundReturnsError(t *testing.T) {
    inst := NewInstaller("/nonexistent/mx")
    err := inst.InstallModule(context.Background(), "test.mpk", "/tmp/test.mpr")
    if err == nil {
        t.Fatal("expected error for missing mx binary")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/marketplace/infrastructure/ -run TestInstaller -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

```go
package infrastructure

import (
    "context"
    "fmt"
    "os/exec"
    "github.com/mendixlabs/mxcli/internal/marketplace/domain"
)

type Installer struct {
    mxPath string
}

func NewInstaller(mxPath string) *Installer {
    return &Installer{mxPath: mxPath}
}

func (inst *Installer) InstallModule(ctx context.Context, mpkPath, projectPath string) error {
    cmd := exec.CommandContext(ctx, inst.mxPath, "module-import", mpkPath, projectPath)
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("mx module-import failed: %w\n%s", err, string(out))
    }
    return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/marketplace/infrastructure/ -run TestInstaller -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/marketplace/infrastructure/installer.go internal/marketplace/infrastructure/installer_test.go
git commit -m "feat(marketplace): implement installer (mx module-import wrapper)"
```

---
### Task 6: Implement ProjectReader (list installed modules)

**Files:**
- Create: `internal/marketplace/infrastructure/project.go`
- Create: `internal/marketplace/infrastructure/project_test.go`

**Interfaces:**
- Consumes: `model.Module` fields (`FromAppStore`, `AppStoreGuid`, `AppStoreVersion`)
- Produces: `ProjectReader.ListInstalledModules()` → []InstalledModule

- [ ] **Step 1: Write the failing test**

This test uses `mdl/backend/mpr` test fixtures to open a project. Since the project reader needs the backend, we'll use a mock approach:

```go
package infrastructure

import (
    "testing"
    "github.com/mendixlabs/mxcli/internal/marketplace/domain"
    "github.com/mendixlabs/mxcli/model"
)

func TestProjectReader_FiltersAppStoreModules(t *testing.T) {
    // Mock backend: return 3 modules, 2 from app store
    fakeModules := []*model.Module{
        {Name: "MyModule", FromAppStore: false},
        {Name: "DatabaseConnector", FromAppStore: true, AppStoreGuid: "2888", AppStoreVersion: "7.0.2"},
        {Name: "CommunityCommons", FromAppStore: true, AppStoreGuid: "170", AppStoreVersion: "11.5.0"},
    }

    pr := &ProjectReader{listModules: func() ([]*model.Module, error) { return fakeModules, nil }}
    result, err := pr.ListInstalledModules("")
    if err != nil {
        t.Fatal(err)
    }
    if len(result) != 2 {
        t.Fatalf("expected 2 modules, got %d", len(result))
    }
    if result[0].Name != "DatabaseConnector" {
        t.Errorf("first: %s", result[0].Name)
    }
    if result[1].AppStoreVersion != "11.5.0" {
        t.Errorf("version: %s", result[1].AppStoreVersion)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/marketplace/infrastructure/ -run TestProjectReader -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

```go
package infrastructure

import (
    "github.com/mendixlabs/mxcli/internal/marketplace/domain"
    "github.com/mendixlabs/mxcli/model"
)

type ModuleLister interface {
    ListModules() ([]*model.Module, error)
}

type ProjectReader struct {
    lister ModuleLister
}

func NewProjectReader(lister ModuleLister) *ProjectReader {
    return &ProjectReader{lister: lister}
}

func (pr *ProjectReader) ListInstalledModules(projectPath string) ([]domain.InstalledModule, error) {
    // projectPath is unused when a lister is injected directly
    modules, err := pr.lister.ListModules()
    if err != nil {
        return nil, err
    }
    var out []domain.InstalledModule
    for _, m := range modules {
        if m.FromAppStore {
            out = append(out, domain.InstalledModule{
                Name:            m.Name,
                ModuleID:        string(m.ID),
                AppStoreGuid:    m.AppStoreGuid,
                AppStoreVersion: m.AppStoreVersion,
            })
        }
    }
    return out, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/marketplace/infrastructure/ -run TestProjectReader -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/marketplace/infrastructure/project.go internal/marketplace/infrastructure/project_test.go
git commit -m "feat(marketplace): implement project reader for installed modules"
```

---
### Task 7: Implement Cache

**Files:**
- Create: `internal/marketplace/infrastructure/cache.go`
- Include tests inline

**Interfaces:**
- Produces: `Cache` for download MPK files and search catalog

- [ ] **Step 1: Write implementation + tests**

`internal/marketplace/infrastructure/cache.go`:

```go
package infrastructure

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "time"
)

type Cache struct {
    baseDir string
}

type catalogCache struct {
    Timestamp time.Time `json:"timestamp"`
    Data      []byte    `json:"data"`
}

func NewCache(baseDir string) *Cache {
    return &Cache{baseDir: baseDir}
}

func (c *Cache) MPKPath(contentID int, version string) string {
    return filepath.Join(c.baseDir, fmt.Sprintf("%d", contentID), version, "module.mpk")
}

func (c *Cache) IsCached(contentID int, version string) bool {
    _, err := os.Stat(c.MPKPath(contentID, version))
    return err == nil
}

func (c *Cache) CatalogPath(profile string) string {
    return filepath.Join(c.baseDir, fmt.Sprintf("catalog-%s.json", profile))
}

func (c *Cache) ReadCatalog(profile string, maxAge time.Duration) ([]byte, bool) {
    path := c.CatalogPath(profile)
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, false
    }
    var entry catalogCache
    if err := json.Unmarshal(data, &entry); err != nil {
        return nil, false
    }
    if time.Since(entry.Timestamp) > maxAge {
        return nil, false
    }
    return entry.Data, true
}

func (c *Cache) WriteCatalog(profile string, data []byte) error {
    path := c.CatalogPath(profile)
    if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
        return err
    }
    entry := catalogCache{Timestamp: time.Now(), Data: data}
    raw, _ := json.Marshal(entry)
    return os.WriteFile(path, raw, 0600)
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/marketplace/infrastructure/ -v`
Expected: All PASS

- [ ] **Step 3: Commit**

```bash
git add internal/marketplace/infrastructure/cache.go
git commit -m "feat(marketplace): implement file cache for downloads and catalog"
```

---
### Task 8: Implement Application Service

**Files:**
- Create: `internal/marketplace/application/service.go`
- Create: `internal/marketplace/application/service_test.go`

**Interfaces:**
- Consumes: All 4 domain repository interfaces, Cache
- Produces: `MarketplaceService` with 7 use case methods

- [ ] **Step 1: Write the failing test**

```go
package application

import (
    "context"
    "errors"
    "testing"
    "github.com/mendixlabs/mxcli/internal/marketplace/domain"
)

type mockContentRepo struct {
    domain.ContentRepository
    searchFn func(ctx context.Context, query string, limit int) ([]*domain.Content, error)
    getFn    func(ctx context.Context, id domain.ContentID) (*domain.Content, error)
}
func (m *mockContentRepo) Search(ctx context.Context, query string, limit int) ([]*domain.Content, error) {
    return m.searchFn(ctx, query, limit)
}
func (m *mockContentRepo) Get(ctx context.Context, id domain.ContentID) (*domain.Content, error) {
    return m.getFn(ctx, id)
}

func TestService_Search_DelegatesToRepo(t *testing.T) {
    repo := &mockContentRepo{
        searchFn: func(_ context.Context, q string, _ int) ([]*domain.Content, error) {
            if q != "database" {
                return nil, errors.New("wrong query")
            }
            return []*domain.Content{{ContentID: 2888, Publisher: "Mendix"}}, nil
        },
    }
    svc := NewService(repo, nil, nil, nil)
    results, err := svc.Search(context.Background(), "database", 10)
    if err != nil {
        t.Fatal(err)
    }
    if len(results) != 1 || results[0].ContentID != 2888 {
        t.Errorf("unexpected: %+v", results)
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/marketplace/application/ -v`
Expected: FAIL

- [ ] **Step 3: Write minimal implementation**

```go
package application

import (
    "context"
    "fmt"
    "sort"
    "time"

    "github.com/mendixlabs/mxcli/internal/marketplace/domain"
    "github.com/mendixlabs/mxcli/internal/marketplace/infrastructure"
)

type Service struct {
    contentRepo  domain.ContentRepository
    versionRepo  domain.VersionRepository
    downloadRepo domain.DownloadRepository
    projectRepo  domain.ProjectRepository
    cache        *infrastructure.Cache
}

func NewService(
    contentRepo domain.ContentRepository,
    versionRepo domain.VersionRepository,
    downloadRepo domain.DownloadRepository,
    projectRepo domain.ProjectRepository,
    cache *infrastructure.Cache,
) *Service {
    return &Service{
        contentRepo:  contentRepo,
        versionRepo:  versionRepo,
        downloadRepo: downloadRepo,
        projectRepo:  projectRepo,
        cache:        cache,
    }
}

func (s *Service) Search(ctx context.Context, query string, limit int) ([]*domain.Content, error) {
    return s.contentRepo.Search(ctx, query, limit)
}

func (s *Service) Get(ctx context.Context, id domain.ContentID) (*domain.Content, error) {
    return s.contentRepo.Get(ctx, id)
}

func (s *Service) GetVersions(ctx context.Context, id domain.ContentID) ([]*domain.Version, error) {
    return s.versionRepo.GetVersions(ctx, id)
}

func (s *Service) Download(ctx context.Context, id domain.ContentID, versionNumber, outputPath string) (string, error) {
    versions, err := s.versionRepo.GetVersions(ctx, id)
    if err != nil {
        return "", err
    }
    version := selectVersion(versions, versionNumber)
    if version == nil {
        if versionNumber != "" {
            return "", fmt.Errorf("version %q not found", versionNumber)
        }
        return "", fmt.Errorf("no versions available")
    }
    if s.cache != nil && outputPath == "" {
        outputPath = s.cache.MPKPath(int(id), version.VersionNumber)
        if s.cache.IsCached(int(id), version.VersionNumber) {
            return outputPath, nil
        }
    }
    if outputPath == "" {
        outputPath = fmt.Sprintf("%s_%s.mpk", version.Name, version.VersionNumber)
    }
    if err := s.downloadRepo.DownloadVersion(ctx, version, outputPath); err != nil {
        return "", err
    }
    return outputPath, nil
}

func (s *Service) Install(ctx context.Context, id domain.ContentID, versionNumber, projectPath string) error {
    // 1. Verify module exists
    content, err := s.contentRepo.Get(ctx, id)
    if err != nil {
        return err
    }
    if content.Type != "Module" {
        return fmt.Errorf("content %d is type %q, not a Module", id, content.Type)
    }

    // 2. Check if already installed
    if s.projectRepo != nil {
        installed, _ := s.projectRepo.ListInstalledModules(projectPath)
        for _, m := range installed {
            if m.AppStoreGuid == fmt.Sprintf("%d", id) {
                return fmt.Errorf(
                    "module %q is already installed (version %s).\nTarget version: %s.\nIn-place module updates are not applied automatically. Update via Studio Pro.",
                    m.Name, m.AppStoreVersion, versionNumber,
                )
            }
        }
    }

    // 3. Download
    mpkPath, err := s.Download(ctx, id, versionNumber, "")
    if err != nil {
        return fmt.Errorf("download failed: %w", err)
    }

    // 4. Install
    if err := s.projectRepo.InstallModule(ctx, mpkPath, projectPath); err != nil {
        return fmt.Errorf("install failed: %w", err)
    }
    return nil
}

func (s *Service) ListInstalled(ctx context.Context, projectPath string) ([]domain.InstalledModule, error) {
    return s.projectRepo.ListInstalledModules(projectPath)
}

type UpdateResult struct {
    ModuleName      string
    InstalledVersion string
    LatestVersion   string
    Status          string // "up-to-date", "update-available", "error"
    Error           string
}

func (s *Service) Update(ctx context.Context, id domain.ContentID, projectPath string) (*UpdateResult, error) {
    installed, err := s.projectRepo.ListInstalledModules(projectPath)
    if err != nil {
        return nil, err
    }
    targetID := fmt.Sprintf("%d", id)
    for _, m := range installed {
        if m.AppStoreGuid == targetID {
            content, err := s.contentRepo.Get(ctx, id)
            if err != nil {
                return &UpdateResult{ModuleName: m.Name, Status: "error", Error: err.Error()}, nil
            }
            if content.LatestVersion == nil {
                return &UpdateResult{ModuleName: m.Name, Status: "error", Error: "no version info"}, nil
            }
            latest := content.LatestVersion.VersionNumber
            if latest == m.AppStoreVersion {
                return &UpdateResult{ModuleName: m.Name, InstalledVersion: m.AppStoreVersion, LatestVersion: latest, Status: "up-to-date"}, nil
            }
            return &UpdateResult{
                ModuleName: m.Name, InstalledVersion: m.AppStoreVersion,
                LatestVersion: latest, Status: "update-available",
            }, nil
        }
    }
    return nil, fmt.Errorf("module with content ID %d not found in project", id)
}

func (s *Service) UpdateAll(ctx context.Context, projectPath string) ([]UpdateResult, error) {
    installed, err := s.projectRepo.ListInstalledModules(projectPath)
    if err != nil {
        return nil, err
    }
    var results []UpdateResult
    for _, m := range installed {
        id := parseInt(m.AppStoreGuid)
        if id == 0 {
            continue
        }
        r, err := s.Update(ctx, domain.ContentID(id), projectPath)
        if err != nil {
            results = append(results, UpdateResult{ModuleName: m.Name, Status: "error", Error: err.Error()})
            continue
        }
        results = append(results, *r)
    }
    sort.Slice(results, func(i, j int) bool { return results[i].ModuleName < results[j].ModuleName })
    return results, nil
}

func selectVersion(versions []*domain.Version, versionNumber string) *domain.Version {
    if versionNumber != "" {
        for _, v := range versions {
            if v.VersionNumber == versionNumber {
                return v
            }
        }
        return nil
    }
    // Return the newest version
    var newest *domain.Version
    for _, v := range versions {
        if newest == nil || v.PublicationDate.After(newest.PublicationDate) {
            newest = v
        }
    }
    return newest
}

func parseInt(s string) int {
    var n int
    for _, r := range s {
        if r < '0' || r > '9' {
            return 0
        }
        n = n*10 + int(r-'0')
    }
    return n
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/marketplace/application/ -v`
Expected: PASS

- [ ] **Step 5: Add more service tests**

Add tests for:
- `TestService_Download_SelectsLatestVersion`
- `TestService_Download_SelectsSpecificVersion`
- `TestService_Install_BlocksAlreadyInstalled`
- `TestService_ListInstalled`
- `TestService_Update_ReportsUpToDate`
- `TestService_UpdateAll`

- [ ] **Step 6: Run all tests**

Run: `go test ./internal/marketplace/... -v`
Expected: All PASS

- [ ] **Step 7: Commit**

```bash
git add internal/marketplace/application/
git commit -m "feat(marketplace): implement application service layer"
```

---
### Task 9: Refactor CLI command (cmd_marketplace.go)

**Files:**
- Modify: `cmd/mxcli/cmd_marketplace.go`
- Modify: `cmd/mxcli/cmd_marketplace_test.go`

**Interfaces:**
- Consumes: `application.Service`
- Produces: 7 Cobra subcommands with table/JSON output

- [ ] **Step 1: Refactor `cmd_marketplace.go`**

Replace the existing code with:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"
    "strings"
    "text/tabwriter"
    "time"

    "github.com/mendixlabs/mxcli/internal/auth"
    "github.com/mendixlabs/mxcli/internal/marketplace/application"
    "github.com/mendixlabs/mxcli/internal/marketplace/domain"
    "github.com/mendixlabs/mxcli/internal/marketplace/infrastructure"
    "github.com/spf13/cobra"
)

var marketplaceCmd = &cobra.Command{
    Use:   "marketplace",
    Short: "Browse and install from the Mendix Marketplace",
    Long: `Browse published modules, widgets, and themes in the Mendix Marketplace.
Download and install them into your project.

Requires a Personal Access Token (PAT). Run 'mxcli auth login' first.`,
}

var (
    marketplaceSearchCmd   = &cobra.Command{Use: "search <query>", Short: "Search marketplace content by keyword", Example: `  mxcli marketplace search "database connector"`, Args: cobra.ExactArgs(1), RunE: runMarketplaceSearch}
    marketplaceInfoCmd     = &cobra.Command{Use: "info <content-id>", Short: "Show details of a marketplace item", Example: `  mxcli marketplace info 170`, Args: cobra.ExactArgs(1), RunE: runMarketplaceInfo}
    marketplaceVersionsCmd = &cobra.Command{Use: "versions <content-id>", Short: "List available versions", Example: `  mxcli marketplace versions 2888`, Args: cobra.ExactArgs(1), RunE: runMarketplaceVersions}
    marketplaceDownloadCmd = &cobra.Command{Use: "download <content-id>", Short: "Download a .mpk file", Example: `  mxcli marketplace download 2888`, Args: cobra.ExactArgs(1), RunE: runMarketplaceDownload}
    marketplaceInstallCmd  = &cobra.Command{Use: "install <content-id>", Short: "Download and install into a project", Example: `  mxcli marketplace install 2888 -p app.mpr`, Args: cobra.ExactArgs(1), RunE: runMarketplaceInstall}
    marketplaceListCmd     = &cobra.Command{Use: "list", Short: "List installed marketplace modules", Example: `  mxcli marketplace list -p app.mpr`, RunE: runMarketplaceList}
    marketplaceUpdateCmd   = &cobra.Command{Use: "update [content-id]", Short: "Check for module updates", Example: `  mxcli marketplace update -p app.mpr`, Args: cobra.MaximumNArgs(1), RunE: runMarketplaceUpdate}
)

func init() {
    // Search flags
    marketplaceSearchCmd.Flags().IntP("limit", "n", 20, "max results")
    marketplaceSearchCmd.Flags().String("profile", auth.ProfileDefault, "credential profile")
    marketplaceSearchCmd.Flags().Bool("json", false, "emit JSON")
    marketplaceSearchCmd.Flags().Bool("refresh", false, "bypass search cache")

    // Info flags
    marketplaceInfoCmd.Flags().String("profile", auth.ProfileDefault, "credential profile")
    marketplaceInfoCmd.Flags().Bool("json", false, "emit JSON")

    // Versions flags
    marketplaceVersionsCmd.Flags().String("profile", auth.ProfileDefault, "credential profile")
    marketplaceVersionsCmd.Flags().Bool("json", false, "emit JSON")
    marketplaceVersionsCmd.Flags().String("min-mendix", "", "filter by min Mendix version")

    // Download flags
    marketplaceDownloadCmd.Flags().String("version", "", "specific version to download")
    marketplaceDownloadCmd.Flags().StringP("output", "o", "", "output path")
    marketplaceDownloadCmd.Flags().String("profile", auth.ProfileDefault, "credential profile")

    // Install flags
    marketplaceInstallCmd.Flags().String("version", "", "specific version to install")
    marketplaceInstallCmd.Flags().String("profile", auth.ProfileDefault, "credential profile")

    // List flags
    marketplaceListCmd.Flags().Bool("json", false, "emit JSON")
    marketplaceListCmd.Flags().String("profile", auth.ProfileDefault, "credential profile")

    // Update flags
    marketplaceUpdateCmd.Flags().String("profile", auth.ProfileDefault, "credential profile")

    marketplaceCmd.AddCommand(marketplaceSearchCmd)
    marketplaceCmd.AddCommand(marketplaceInfoCmd)
    marketplaceCmd.AddCommand(marketplaceVersionsCmd)
    marketplaceCmd.AddCommand(marketplaceDownloadCmd)
    marketplaceCmd.AddCommand(marketplaceInstallCmd)
    marketplaceCmd.AddCommand(marketplaceListCmd)
    marketplaceCmd.AddCommand(marketplaceUpdateCmd)
}

// buildMarketplaceService constructs the full service chain from CLI flags.
func buildMarketplaceService(ctx context.Context, cmd *cobra.Command) (*application.Service, error) {
    profile, _ := cmd.Flags().GetString("profile")
    httpClient, err := auth.ClientFor(ctx, profile)
    if err != nil {
        return nil, fmt.Errorf("%w\nhint: run 'mxcli auth login'", err)
    }
    apiClient := infrastructure.NewAPIClient(httpClient, "https://marketplace-api.mendix.com")
    cred, _ := auth.Resolve(ctx, profile)
    downloader := infrastructure.NewDownloader(http.DefaultClient, cred.Token)
    cacheDir, _ := cacheDirForMarketplace()
    cache := infrastructure.NewCache(cacheDir)

    var projectRepo domain.ProjectRepository
    if p, err := cmd.Flags().GetString("project"); err == nil && p != "" {
        // Only build project reader when -p is provided
    }

    return application.NewService(apiClient, apiClient, downloader, projectRepo, cache), nil
}

func cacheDirForMarketplace() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(home, ".mxcli", "marketplace-cache"), nil
}
```

(Full implementation continues in the actual code — due to plan size constraints, the remaining runner functions follow the pattern: parse args → parse ContentID → build service → call service method → render output.)

- [ ] **Step 2: Write runner functions**

Implement 7 runner functions (`runMarketplaceSearch`, `runMarketplaceInfo`, `runMarketplaceVersions`, `runMarketplaceDownload`, `runMarketplaceInstall`, `runMarketplaceList`, `runMarketplaceUpdate`). Each follows the pattern:

```go
func runMarketplaceSearch(cmd *cobra.Command, args []string) error {
    query := args[0]
    limit, _ := cmd.Flags().GetInt("limit")
    asJSON, _ := cmd.Flags().GetBool("json")
    refresh, _ := cmd.Flags().GetBool("refresh")

    svc, err := buildMarketplaceService(cmd.Context(), cmd)
    if err != nil {
        return err
    }
    results, err := svc.Search(cmd.Context(), query, limit)
    if err != nil {
        return err
    }
    if asJSON {
        return emitJSON(cmd, results)
    }
    return renderContentTable(cmd, results)
}
```

- [ ] **Step 3: Keep existing render functions**

Preserve from existing code:
- `renderContentTable` — table with ID/TYPE/PUBLISHER/SUPPORT/LATEST/NAME
- `renderContentDetail` — key-value detail display
- `renderVersionsTable` — table with VERSION/MIN MENDIX/PUBLISHED/NAME
- `emitJSON` — JSON output helper
- `compareSemverLike` — version comparison
- `filterVersionsByMinMendix` — version filtering
- `parseContentID` — string to int parser

Add new render functions:
- `renderInstalledModulesTable` — installed module list
- `renderUpdateResultsTable` — update status table

- [ ] **Step 4: Refactor `cmd_marketplace_test.go`**

Update the test factory to use the new service construction. The existing `marketplaceClientFactory` pattern should be updated to a `marketplaceServiceFactory`:

```go
var marketplaceServiceFactory func(context.Context, *cobra.Command) (*application.Service, error)

func buildMarketplaceService(ctx context.Context, cmd *cobra.Command) (*application.Service, error) {
    if marketplaceServiceFactory != nil {
        return marketplaceServiceFactory(ctx, cmd)
    }
    // ... production path
}
```

Update all tests to inject mock services with `httptest.Server`. The existing test patterns (runMarketplace helper, sample responses) should be preserved and extended.

- [ ] **Step 5: Run tests**

Run: `go test ./cmd/mxcli/ -run TestMarketplace -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/mxcli/cmd_marketplace.go cmd/mxcli/cmd_marketplace_test.go
git commit -m "feat(marketplace): add download/install/list/update commands, refactor CLI layer"
```

---
### Task 10: Integration test — verify with real API

**Files:**
- Modify: `scripts/auth-discovery-spike.sh` (add download test)

- [ ] **Step 1: Build the CLI**

```bash
go build -o bin/mxcli ./cmd/mxcli
```

- [ ] **Step 2: Verify existing search still works**

```bash
./bin/mxcli marketplace search "database" --json
```
Expected: JSON output with Database Connector result

- [ ] **Step 3: Verify info**

```bash
./bin/mxcli marketplace info 2888
```
Expected: detail output with contentId, publisher, etc.

- [ ] **Step 4: Verify versions includes downloadUrl**

```bash
./bin/mxcli marketplace versions 170 --json | grep downloadUrl
```
Expected: downloadUrl present in output

- [ ] **Step 5: Verify download**

```bash
./bin/mxcli marketplace download 170 -o /tmp/test_comcommons.mpk
```
Expected: file downloaded, check `ls -la /tmp/test_comcommons.mpk`

- [ ] **Step 6: Run full test suite**

```bash
make test
```
Expected: All PASS

---
### Task 11: Write design doc and finalize

**Files:**
- Create: `docs/superpowers/specs/2026-07-24-marketplace-redesign-design.md`

- [ ] **Step 1: Write design doc**

Copy the approved design from the brainstorming session into `docs/superpowers/specs/2026-07-24-marketplace-redesign-design.md`.

- [ ] **Step 2: Commit all remaining changes**

```bash
git add docs/superpowers/specs/
git commit -m "docs: marketplace redesign design doc"
```
