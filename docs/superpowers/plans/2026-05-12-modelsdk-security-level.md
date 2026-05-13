# modelsdk Security Level Write Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `SetProjectSecurityLevel` in `MprBackend` with a modelsdk-based decode→mutate→encode roundtrip that avoids the `_Transaction` update causing SQLITE_READONLY_DBMOVED (1544) on hard-linked MPR files.

**Architecture:** `MprBackend` gains a second writer field (`msdkWriter *modelsdkmpr.Writer`). `SetProjectSecurityLevel` reads raw BSON via the modelsdk reader, decodes to `*security.ProjectSecurity`, calls `SetSecurityLevel`, re-encodes, and writes back via `WriteTransaction` — no `updateTransactionID` is called. All other backend methods continue using the existing `sdk/mpr.Writer`.

**Tech Stack:** `modelsdk/mpr` (WriteTransaction), `modelsdk/codec` (Decoder, Encoder), `modelsdk/gen/security` (ProjectSecurity), `modernc.org/sqlite`, `go.mongodb.org/mongo-driver/bson`

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `mdl/backend/mpr/backend.go` | Modify | Add `msdkWriter` field; update `Connect`, `Disconnect`, `Wrap` |
| `mdl/backend/mpr/security_modelsdk.go` | Create | `setSecurityLevelViaModelsdk` helper — full decode→mutate→encode→write |
| `mdl/backend/mpr/security_modelsdk_test.go` | Create | Integration test: insert security unit, call backend method, read back |

---

### Task 1: Add `msdkWriter` field and wire lifecycle

**Files:**
- Modify: `mdl/backend/mpr/backend.go:8-21` (imports), `:33-37` (struct), `:48-54` (Wrap), `:60-69` (Connect), `:71-80` (Disconnect)

- [ ] **Step 1: Add import alias for modelsdk/mpr in backend.go**

Open `mdl/backend/mpr/backend.go`. The import block currently ends at line 21. Add one line:

```go
import (
    "github.com/mendixlabs/mxcli/mdl/backend"
    "github.com/mendixlabs/mxcli/mdl/linter"
    "github.com/mendixlabs/mxcli/mdl/types"
    "github.com/mendixlabs/mxcli/model"
    "github.com/mendixlabs/mxcli/sdk/agenteditor"
    "github.com/mendixlabs/mxcli/sdk/domainmodel"
    "github.com/mendixlabs/mxcli/sdk/javaactions"
    "github.com/mendixlabs/mxcli/sdk/microflows"
    "github.com/mendixlabs/mxcli/sdk/mpr"
    "github.com/mendixlabs/mxcli/sdk/pages"
    "github.com/mendixlabs/mxcli/sdk/security"
    "github.com/mendixlabs/mxcli/sdk/workflows"
    modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)
```

- [ ] **Step 2: Add `msdkWriter` to `MprBackend` struct**

Replace the struct definition at lines 33–37:

```go
type MprBackend struct {
    reader     *mpr.Reader
    writer     *mpr.Writer
    msdkWriter *modelsdkmpr.Writer
    path       string
}
```

- [ ] **Step 3: Update `Connect` to also open modelsdk writer**

Replace the `Connect` function at lines 60–69:

```go
func (b *MprBackend) Connect(path string) error {
    w, err := mpr.NewWriter(path)
    if err != nil {
        return err
    }
    mw, err := modelsdkmpr.NewWriter(path)
    if err != nil {
        _ = w.Close()
        return err
    }
    b.writer = w
    b.reader = w.Reader()
    b.msdkWriter = mw
    b.path = path
    return nil
}
```

- [ ] **Step 4: Update `Disconnect` to also close modelsdk writer**

Replace the `Disconnect` function at lines 71–80:

```go
func (b *MprBackend) Disconnect() error {
    if b.writer == nil {
        return nil
    }
    err := b.writer.Close()
    if b.msdkWriter != nil {
        if cerr := b.msdkWriter.Close(); cerr != nil && err == nil {
            err = cerr
        }
    }
    b.writer = nil
    b.reader = nil
    b.msdkWriter = nil
    b.path = ""
    return err
}
```

- [ ] **Step 5: Update `Wrap` to also open modelsdk writer**

Replace the `Wrap` function at lines 44–54:

```go
func Wrap(writer *mpr.Writer, path string) *MprBackend {
    mw, _ := modelsdkmpr.NewWriter(path) // best-effort; nil if path unavailable
    return &MprBackend{
        reader:     writer.Reader(),
        writer:     writer,
        msdkWriter: mw,
        path:       path,
    }
}
```

- [ ] **Step 6: Verify it compiles**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/backend/mpr/...
```

Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add mdl/backend/mpr/backend.go
git commit -m "feat(backend): add msdkWriter field to MprBackend lifecycle"
```

---

### Task 2: Implement `setSecurityLevelViaModelsdk` and wire it in

**Files:**
- Create: `mdl/backend/mpr/security_modelsdk.go`
- Modify: `mdl/backend/mpr/backend.go:348-349`

- [ ] **Step 1: Create `security_modelsdk.go`**

Create the file `mdl/backend/mpr/security_modelsdk.go` with this exact content:

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
    "fmt"

    "go.mongodb.org/mongo-driver/bson"

    "github.com/mendixlabs/mxcli/model"
    "github.com/mendixlabs/mxcli/modelsdk/codec"
    msdksecurity "github.com/mendixlabs/mxcli/modelsdk/gen/security"
)

// setSecurityLevelViaModelsdk writes the SecurityLevel field using the modelsdk
// decode→mutate→encode roundtrip. This avoids the sdk/mpr updateTransactionID()
// call that triggers SQLITE_READONLY_DBMOVED (1544) on hard-linked MPR files.
func (b *MprBackend) setSecurityLevelViaModelsdk(unitID model.ID, level string) error {
    rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
    if err != nil {
        return fmt.Errorf("read security unit: %w", err)
    }

    elem, err := codec.NewDecoder(codec.DefaultRegistry).Decode(bson.Raw(rawBytes))
    if err != nil {
        return fmt.Errorf("decode security unit: %w", err)
    }

    ps, ok := elem.(*msdksecurity.ProjectSecurity)
    if !ok {
        return fmt.Errorf("unexpected type %T for security unit (want *security.ProjectSecurity)", elem)
    }

    ps.SetSecurityLevel(level)

    newBytes, err := (&codec.Encoder{}).Encode(ps)
    if err != nil {
        return fmt.Errorf("encode security unit: %w", err)
    }

    wtx, err := b.msdkWriter.BeginWriteTransaction()
    if err != nil {
        return fmt.Errorf("begin write transaction: %w", err)
    }
    if err := wtx.WriteUnit(string(unitID), newBytes); err != nil {
        _ = wtx.Rollback()
        return fmt.Errorf("write security unit: %w", err)
    }
    return wtx.Commit()
}
```

- [ ] **Step 2: Wire `SetProjectSecurityLevel` to use the new helper**

In `backend.go` at lines 348–349, replace the one-liner:

```go
// Before:
func (b *MprBackend) SetProjectSecurityLevel(unitID model.ID, level string) error {
    return b.writer.SetProjectSecurityLevel(unitID, level)
}

// After:
func (b *MprBackend) SetProjectSecurityLevel(unitID model.ID, level string) error {
    return b.setSecurityLevelViaModelsdk(unitID, level)
}
```

- [ ] **Step 3: Verify it compiles**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/backend/mpr/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add mdl/backend/mpr/security_modelsdk.go mdl/backend/mpr/backend.go
git commit -m "feat(backend): implement SetProjectSecurityLevel via modelsdk roundtrip"
```

---

### Task 3: Integration test

**Files:**
- Create: `mdl/backend/mpr/security_modelsdk_test.go`

The test creates a real v1 SQLite MPR file (temp dir), inserts a minimal `Security$ProjectSecurity` unit, connects `MprBackend`, calls `SetProjectSecurityLevel`, then reads back and asserts the field changed.

- [ ] **Step 1: Write the failing test**

Create `mdl/backend/mpr/security_modelsdk_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
    "database/sql"
    "path/filepath"
    "testing"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"

    "github.com/mendixlabs/mxcli/model"
    _ "modernc.org/sqlite"
)

// makeSecurityTestMPR creates a minimal v1 MPR SQLite file in a temp dir
// and inserts one Security$ProjectSecurity unit with SecurityLevel = Off.
// Returns the file path and the unit ID.
func makeSecurityTestMPR(t *testing.T) (mprPath string, unitID model.ID) {
    t.Helper()
    dir := t.TempDir()
    mprPath = filepath.Join(dir, "test.mpr")

    db, err := sql.Open("sqlite", mprPath)
    if err != nil {
        t.Fatalf("open sqlite: %v", err)
    }
    defer db.Close()

    // Minimal schema required by both sdk/mpr and modelsdk/mpr readers.
    if _, err := db.Exec(`
        CREATE TABLE _MetaData (
            _FormatVersion INTEGER,
            _ProductVersion TEXT,
            _BuildVersion TEXT,
            _SchemaHash TEXT
        );
        INSERT INTO _MetaData VALUES (1, '10.18.0', '10.18.0.0', 'testhash');

        CREATE TABLE _Transaction (LastTransactionID TEXT);
        INSERT INTO _Transaction VALUES ('00000000-0000-0000-0000-000000000000');

        CREATE TABLE Unit (
            UnitID BLOB PRIMARY KEY NOT NULL,
            ContainerID BLOB,
            ContainmentName TEXT,
            TreeConflict LONG,
            ContentsHash TEXT,
            ContentsConflicts TEXT,
            Contents BLOB
        );
    `); err != nil {
        t.Fatalf("create schema: %v", err)
    }

    // Use a deterministic UUID for the security unit.
    unitIDStr := "11111111-1111-1111-1111-111111111111"
    unitID = model.ID(unitIDStr)

    // Build a minimal Security$ProjectSecurity BSON document.
    // Mendix stores UUIDs as Binary subtype 0 with a byte-swap on the first 3 groups.
    idBlob := mustUUIDBlob(unitIDStr)
    secDoc := bson.D{
        {Key: "$Type", Value: "Security$ProjectSecurity"},
        {Key: "$ID", Value: primitive.Binary{Subtype: 0x00, Data: idBlob}},
        {Key: "SecurityLevel", Value: "Security$SecurityLevel_Off"},
        {Key: "CheckSecurity", Value: true},
        {Key: "EnableDemoUsers", Value: false},
    }
    secBytes, err := bson.Marshal(secDoc)
    if err != nil {
        t.Fatalf("marshal security BSON: %v", err)
    }

    if _, err := db.Exec(
        `INSERT INTO Unit (UnitID, ContainerID, ContainmentName, TreeConflict, ContentsHash, ContentsConflicts, Contents)
         VALUES (?, ?, ?, 0, '', '', ?)`,
        idBlob,
        make([]byte, 16), // dummy container
        "ProjectSecurity",
        secBytes,
    ); err != nil {
        t.Fatalf("insert security unit: %v", err)
    }

    return mprPath, unitID
}

// mustUUIDBlob converts a standard UUID string to the 16-byte Mendix GUID blob
// (little-endian groups 1-3, big-endian groups 4-5).
func mustUUIDBlob(uuid string) []byte {
    // Strip dashes and decode hex.
    hex := ""
    for _, ch := range uuid {
        if ch != '-' {
            hex += string(ch)
        }
    }
    blob := make([]byte, 16)
    for i := 0; i < 16; i++ {
        var b byte
        fmt.Sscanf(hex[i*2:i*2+2], "%02x", &b)
        blob[i] = b
    }
    // Mendix GUID byte-swap: groups 1 (4 bytes), 2 (2 bytes), 3 (2 bytes) are LE.
    blob[0], blob[1], blob[2], blob[3] = blob[3], blob[2], blob[1], blob[0]
    blob[4], blob[5] = blob[5], blob[4]
    blob[6], blob[7] = blob[7], blob[6]
    return blob
}

func TestSetProjectSecurityLevel_ViaModelsdk(t *testing.T) {
    mprPath, unitID := makeSecurityTestMPR(t)

    b := New()
    if err := b.Connect(mprPath); err != nil {
        t.Fatalf("Connect: %v", err)
    }
    defer b.Disconnect()

    const wantLevel = "Security$SecurityLevel_Production"
    if err := b.SetProjectSecurityLevel(unitID, wantLevel); err != nil {
        t.Fatalf("SetProjectSecurityLevel: %v", err)
    }

    // Read back via modelsdk to verify the field persisted.
    rawBytes, err := b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))
    if err != nil {
        t.Fatalf("GetRawUnitBytes after write: %v", err)
    }
    var doc bson.D
    if err := bson.Unmarshal(rawBytes, &doc); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    got := ""
    for _, e := range doc {
        if e.Key == "SecurityLevel" {
            got, _ = e.Value.(string)
        }
    }
    if got != wantLevel {
        t.Errorf("SecurityLevel = %q, want %q", got, wantLevel)
    }
}
```

- [ ] **Step 2: Add missing `fmt` import**

The `mustUUIDBlob` helper uses `fmt.Sscanf`. Add `"fmt"` to the import block in the test file:

```go
import (
    "database/sql"
    "fmt"
    "path/filepath"
    "testing"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"

    "github.com/mendixlabs/mxcli/model"
    _ "modernc.org/sqlite"
)
```

- [ ] **Step 3: Run the test — expect it to pass**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/backend/mpr/... -run TestSetProjectSecurityLevel_ViaModelsdk -v
```

Expected output:
```
=== RUN   TestSetProjectSecurityLevel_ViaModelsdk
--- PASS: TestSetProjectSecurityLevel_ViaModelsdk (0.XXs)
PASS
```

If it fails, check:
- `Connect` error → the `_MetaData` table might be missing a required column. Check `modelsdk/mpr/version/version.go:55`.
- `decode` error → the `Security$ProjectSecurity` factory may not be registered. Ensure `modelsdk/gen/security` is imported somewhere in the test binary (it registers via `init()`). Add a blank import if needed: `_ "github.com/mendixlabs/mxcli/modelsdk/gen/security"`.
- Type cast error → `Security$ProjectSecurity` decoded as a different type. Print `elem.TypeName()` to debug.

- [ ] **Step 4: Run full backend test suite to catch regressions**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/backend/mpr/... -v 2>&1 | tail -30
```

Expected: all tests pass.

- [ ] **Step 5: Run full build and vet**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
make build && make vet
```

Expected: no errors.

- [ ] **Step 6: Smoke test against the real MPR (after user fixes hardlink)**

After the user runs:
```bash
cd /mnt/data_sdd/macnica/mendix-app
cp MacnicaApp.mpr MacnicaApp.mpr.new && mv MacnicaApp.mpr.new MacnicaApp.mpr
```

Run:
```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go run ./cmd/mxcli -p /mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr \
    -c "alter project security level production;"
go run ./cmd/mxcli -p /mnt/data_sdd/macnica/mendix-app/MacnicaApp.mpr \
    -c "show project security;"
```

Expected:
```
Set project security level to production
...
Security Level: Production
```

- [ ] **Step 7: Commit**

```bash
git add mdl/backend/mpr/security_modelsdk_test.go
git commit -m "test(backend): integration test for SetProjectSecurityLevel via modelsdk"
```

---

## Self-Review

**Spec coverage:**
- ✅ decode→mutate→encode roundtrip: Task 2 `setSecurityLevelViaModelsdk`
- ✅ `MprBackend` gains modelsdk writer: Task 1 struct + Connect/Disconnect/Wrap
- ✅ No `_Transaction` update: `WriteTransaction.WriteUnit` only touches `Unit.ContentsHash`
- ✅ Integration test: Task 3
- ✅ Other backend methods unchanged: only `SetProjectSecurityLevel` body changes

**Placeholder scan:** No TBD/TODO. All code is complete with real types and method names.

**Type consistency:**
- `b.msdkWriter` → `*modelsdkmpr.Writer` (Task 1 struct, Task 1 Connect, Task 2 security_modelsdk.go, Task 3 test)
- `b.msdkWriter.Reader().GetRawUnitBytes(string(unitID))` → Task 2 and Task 3 both use same call
- `codec.NewDecoder(codec.DefaultRegistry).Decode(bson.Raw(rawBytes))` → single call site in Task 2
- `*msdksecurity.ProjectSecurity` → type alias consistent across Task 2
- `wtx.Rollback()` / `wtx.Commit()` → `*modelsdkmpr.WriteTransaction`, consistent with modelsdk/mpr/writer_core.go API

**Note:** The blank import `_ "github.com/mendixlabs/mxcli/modelsdk/gen/security"` may be needed in the test if the `init()` registration doesn't trigger automatically. Task 3 Step 3 calls this out explicitly as a fallback.
