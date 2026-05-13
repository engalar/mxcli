# Design: modelsdk Security Level Write (Option C)

**Date**: 2026-05-12
**Branch**: feature/expression-checker
**Status**: Approved

## Problem

`alter project security level production` fails with:

```
Error: failed to set security level: attempt to write a readonly database (1544)
```

Root cause: `sdk/mpr.Writer.updateUnit()` (v2 path) calls `updateTransactionID()`, which executes `UPDATE _Transaction SET LastTransactionID = ?`. On MPR files with multiple hard links, SQLite returns SQLITE_READONLY_DBMOVED (1544) for this specific table.

## Goal

Implement `SetProjectSecurityLevel` using the full modelsdk decode→mutate→encode roundtrip so the write path never touches `_Transaction`.

## Architecture

### Data flow

```
alter project security level production
  ↓ executor (unchanged)
MprBackend.SetProjectSecurityLevel(unitID, "Security$SecurityLevel_Production")
  ↓
modelsdk/mpr.Reader.GetRawUnitBytes(unitID)           → []byte  (raw BSON)
codec.NewDecoder(DefaultRegistry).Decode(bson.Raw)    → element.Element
cast to *msdksecurity.ProjectSecurity
ps.SetSecurityLevel(level)                            → marks dirty in property.Enum
codec.Encoder{}.Encode(ps)                            → []byte  (new BSON)
modelsdk/mpr.WriteTransaction.WriteUnit(unitID, bytes)
WriteTransaction.Commit()                             → rename .tmp → .mxunit, tx.Commit()
```

### Why this avoids 1544

`modelsdk/mpr.WriteTransaction.WriteUnit()` does only:
1. `os.WriteFile` to a `.tmp` file (deferred rename on Commit)
2. `tx.Exec("UPDATE Unit SET ContentsHash = ?")` inside a SQL transaction

It does **not** call `updateTransactionID()` and never touches `_Transaction`. The SQL transaction also means the ContentsHash update is atomic with the file rename.

### modelsdk components used

| Component | Location | Role |
|-----------|----------|------|
| `Writer.BeginWriteTransaction()` | `modelsdk/mpr/writer_core.go` | Opens SQL tx + deferred file writes |
| `WriteTransaction.WriteUnit()` | `modelsdk/mpr/writer_core.go` | Writes `.mxunit` + updates ContentsHash |
| `WriteTransaction.Commit()` | `modelsdk/mpr/writer_core.go` | Renames `.tmp` files, commits SQL tx |
| `Reader.GetRawUnitBytes()` | `modelsdk/mpr/reader.go:249` | Reads raw BSON from mprcontents |
| `codec.NewDecoder(DefaultRegistry)` | `modelsdk/codec/decoder.go` | Decodes BSON → typed element |
| `codec.Encoder{}.Encode(elem)` | `modelsdk/codec/encoder.go` | Dirty-aware BSON serialization |
| `security.ProjectSecurity` | `modelsdk/gen/security/types.go:334` | Typed struct with `SetSecurityLevel()` |
| `DefaultRegistry` init | `modelsdk/gen/security/types.go:960` | Auto-registers `Security$ProjectSecurity` |

## Files Changed

| File | Change |
|------|--------|
| `mdl/backend/mpr/backend.go` | Add `msdkWriter *modelsdkmpr.Writer` field; update `Connect`, `Disconnect`, `Wrap` |
| `mdl/backend/mpr/backend_security.go` | Replace `SetProjectSecurityLevel` body with modelsdk path |

**Unchanged**: executor, grammar, AST, visitor, all other backend methods, tests.

## Implementation Detail

### `MprBackend` struct addition

```go
import (
    modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type MprBackend struct {
    reader     *mpr.Reader
    writer     *mpr.Writer
    msdkWriter *modelsdkmpr.Writer  // NEW: for modelsdk-based writes
    path       string
}
```

### Connect / Disconnect / Wrap

`Connect`: after opening `sdk/mpr.Writer`, also open `modelsdkmpr.NewWriter(path)`.
`Disconnect`: also close `msdkWriter`.
`Wrap`: open a `modelsdkmpr.NewWriter(path)` from the same path.

Two SQLite connections to the same file are safe under `journal_mode=delete`; SQLite serializes writes via file locking.

### SetProjectSecurityLevel (new body)

```go
import (
    msdksecurity "github.com/mendixlabs/mxcli/modelsdk/gen/security"
    "github.com/mendixlabs/mxcli/modelsdk/codec"
    "go.mongodb.org/mongo-driver/bson"
)

func (b *MprBackend) SetProjectSecurityLevel(unitID model.ID, level string) error {
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
        return fmt.Errorf("unexpected type %T for security unit", elem)
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

## Error Handling

- `GetRawUnitBytes` failure → propagate with context
- Decode type mismatch → explicit error (guards against wrong unit ID being passed)
- `WriteUnit` failure → explicit Rollback before returning
- `Commit` failure → WriteTransaction.Commit already cleans up temp files on SQL tx failure

## Testing

Existing `sdk/mpr/writer_security_test.go` tests remain. Add one test in `mdl/backend/mpr/` that:
1. Opens a v2 test MPR
2. Calls `SetProjectSecurityLevel` via the backend
3. Reopens and asserts `SecurityLevel` changed

## Out of Scope

- Migrating other security write methods (AddUserRole, etc.) to modelsdk — that is a future step
- Fixing the `_Transaction` write for other operations
- Breaking the hard link on the test MPR file (user-side fix: `cp MPR MPR.new && mv MPR.new MPR`)
