# CREATE IMAGE COLLECTION Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `CREATE IMAGE COLLECTION Module.Name` MDL syntax to create empty Mendix image collections.

**Architecture:** Follow the same 7-layer pattern already established for `CREATE ENUMERATION`: Lexer token → Parser rule → AST node → Visitor handler → BSON writer → Executor command → dispatch registration. Phase 1 creates empty collections only (no image embedding); Phase 2 adds `IMAGE "name" FROM FILE 'path'` syntax.

**Tech Stack:** ANTLR4 (grammar), Go (parser/visitor/executor/writer), `go.mongodb.org/mongo-driver/bson` (serialization), `modernc.org/sqlite` (MPR storage)

---

## Background: How Image Collections Work

A Mendix image collection is stored as a single BSON unit in the MPR with type `Images$ImageCollection`. Its NDSL representation (from `mxcli bson dump --format ndsl`) looks like:

```
Images$ImageCollection
  Documentation: ""
  Excluded: false
  ExportLevel: "Hidden"
  Images [marker=3]: []
  Name: "Icons"
```

Each image inside is an `Images$Image` sub-document embedded in the `Images` array. The actual image bytes are stored inline as BSON binary — not as file paths.

The `model.ImageCollection` and `mpr.Reader.ListImageCollections()` already exist (`sdk/mpr/reader_types.go`). We only need to add the **writer** and the **MDL pipeline**.

---

## Task 1: Add BSON Writer for Image Collections

**Files:**
- Create: `sdk/mpr/writer_imagecollection.go`
- Read first: `sdk/mpr/writer_enumeration.go` (reference pattern), `sdk/mpr/reader_types.go` lines 220–261

**Step 1: Read the reference writer**

```bash
cat sdk/mpr/writer_enumeration.go
```

This shows the pattern: generate UUID if empty → serialize to BSON → call `w.insertUnit(...)`.

**Step 2: Write the failing test**

Create `sdk/mpr/writer_imagecollection_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0
package mpr_test

import (
    "testing"

    "github.com/mendixlabs/mxcli/sdk/mpr"
)

func TestCreateImageCollectionEmpty(t *testing.T) {
    w, cleanup := openTestWriter(t)
    defer cleanup()

    // Get MyFirstModule's ID
    modules, err := w.ListModules()
    if err != nil {
        t.Fatalf("ListModules: %v", err)
    }
    var moduleID string
    for _, m := range modules {
        if m.Name == "MyFirstModule" {
            moduleID = string(m.ID)
            break
        }
    }
    if moduleID == "" {
        t.Skip("MyFirstModule not found in test project")
    }

    ic := &mpr.ImageCollection{
        ContainerID: mpr.ID(moduleID),
        Name:        "TestIcons",
        ExportLevel: "Hidden",
    }

    if err := w.CreateImageCollection(ic); err != nil {
        t.Fatalf("CreateImageCollection: %v", err)
    }

    // Verify it can be read back
    collections, err := w.ListImageCollections()
    if err != nil {
        t.Fatalf("ListImageCollections after create: %v", err)
    }
    var found bool
    for _, c := range collections {
        if c.Name == "TestIcons" {
            found = true
            break
        }
    }
    if !found {
        t.Error("created image collection not found in ListImageCollections")
    }
}
```

**Step 3: Run test to verify it fails**

```bash
go test ./sdk/mpr/ -run TestCreateImageCollectionEmpty -v
```

Expected: FAIL — `w.CreateImageCollection undefined`

**Step 4: Create the writer file**

```go
// SPDX-License-Identifier: Apache-2.0

package mpr

import (
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateImageCollection creates a new empty image collection unit in the MPR.
func (w *Writer) CreateImageCollection(ic *ImageCollection) error {
    if ic.ID == "" {
        ic.ID = ID(generateUUID())
    }
    if ic.ExportLevel == "" {
        ic.ExportLevel = "Hidden"
    }

    contents, err := w.serializeImageCollection(ic)
    if err != nil {
        return err
    }

    return w.insertUnit(string(ic.ID), string(ic.ContainerID),
        "Documents", "Images$ImageCollection", contents)
}

// DeleteImageCollection deletes an image collection by ID.
func (w *Writer) DeleteImageCollection(id ID) error {
    return w.deleteUnit(string(id))
}

func (w *Writer) serializeImageCollection(ic *ImageCollection) ([]byte, error) {
    // Images array always starts with the array marker int32(3)
    images := bson.A{int32(3)}
    for _, img := range ic.Images {
        if img.ID == "" {
            img.ID = ID(generateUUID())
        }
        images = append(images, bson.D{
            {Key: "$ID", Value: idToBsonBinary(string(img.ID))},
            {Key: "$Type", Value: "Images$Image"},
            {Key: "Image", Value: primitive.Binary{Subtype: 0, Data: img.Data}},
            {Key: "ImageFormat", Value: img.Format},
            {Key: "Name", Value: img.Name},
        })
    }

    doc := bson.D{
        {Key: "$ID", Value: idToBsonBinary(string(ic.ID))},
        {Key: "$Type", Value: "Images$ImageCollection"},
        {Key: "Documentation", Value: ic.Documentation},
        {Key: "Excluded", Value: false},
        {Key: "ExportLevel", Value: ic.ExportLevel},
        {Key: "Images", Value: images},
        {Key: "Name", Value: ic.Name},
    }

    return bson.Marshal(doc)
}
```

**Note:** `ImageCollection` and `Image` types are defined in `sdk/mpr/reader_types.go`. The `Image` struct currently only has `ID` and `Name` fields — you will need to add `Data []byte` and `Format string` fields to support Phase 2. For Phase 1 (empty collections), `ic.Images` will be empty so those fields won't be accessed.

**Step 5: Extend the Image struct in reader_types.go**

Modify `sdk/mpr/reader_types.go` lines 228–232:

```go
// Image represents an image in a collection.
type Image struct {
    ID     ID     `json:"id"`
    Name   string `json:"name"`
    Data   []byte `json:"data,omitempty"`   // raw image bytes
    Format string `json:"format,omitempty"` // "Png", "Svg", "Gif", "Jpeg", "Bmp"
}
```

**Step 6: Run test to verify it passes**

```bash
go test ./sdk/mpr/ -run TestCreateImageCollectionEmpty -v
```

Expected: PASS

**Step 7: Commit**

```bash
git add sdk/mpr/writer_imagecollection.go sdk/mpr/writer_imagecollection_test.go sdk/mpr/reader_types.go
git commit -m "feat(mpr): add CreateImageCollection BSON writer"
```

---

## Task 2: Add COLLECTION Lexer Token and Grammar Rule

**Files:**
- Modify: `mdl/grammar/MDLLexer.g4`
- Modify: `mdl/grammar/MDLParser.g4`

After modifying grammar, always regenerate: `make grammar`

**Step 1: Add COLLECTION token to the lexer**

In `mdl/grammar/MDLLexer.g4`, find the block containing `IMAGE`, `STATICIMAGE`, `DYNAMICIMAGE` (around line 311). Add `COLLECTION` nearby (alphabetical order is not required; group with related keywords):

```antlr
COLLECTION: C O L L E C T I O N;
```

You can add it right after line 311 (`IMAGE: I M A G E;`).

**Step 2: Add COLLECTION to the parser's keyword-as-identifier list**

Search for `commonNameKeyword` in `MDLParser.g4`. It's a rule that lists all keywords that can also be used as identifiers. Add `COLLECTION` to this list so it doesn't break existing identifiers that might contain it.

Also add `COLLECTION` to the `qualifiedNameKeyword` rule if it exists.

**Step 3: Add createImageCollectionStatement rule to the parser**

In `MDLParser.g4`, find the `createStatement` rule (around line 80). It lists all valid `| createXxxStatement` alternatives. Add:

```antlr
| createImageCollectionStatement
```

Then add the rule definition near the ENUMERATION section (around line 725):

```antlr
// =============================================================================
// IMAGE COLLECTION CREATION
// =============================================================================

createImageCollectionStatement
    : IMAGE COLLECTION qualifiedName imageCollectionOptions?
    ;

imageCollectionOptions
    : imageCollectionOption+
    ;

imageCollectionOption
    : EXPORT LEVEL STRING_LITERAL   // e.g. EXPORT LEVEL 'Public'
    | COMMENT STRING_LITERAL
    ;
```

**Step 4: Regenerate the ANTLR parser**

```bash
make grammar
```

Expected: no errors, files in `mdl/grammar/parser/` are updated.

**Step 5: Verify the generated parser compiles**

```bash
go build ./mdl/...
```

Expected: no errors

**Step 6: Commit**

```bash
git add mdl/grammar/MDLLexer.g4 mdl/grammar/MDLParser.g4 mdl/grammar/parser/
git commit -m "feat(grammar): add COLLECTION token and createImageCollectionStatement rule"
```

---

## Task 3: Add AST Node

**Files:**
- Create: `mdl/ast/ast_imagecollection.go`

**Step 1: Write failing test**

In `mdl/executor/` there are round-trip tests. Write a parse-only test in a new file `mdl/ast/ast_imagecollection_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0
package ast_test

import (
    "testing"

    "github.com/mendixlabs/mxcli/mdl/visitor"
)

func TestParseCreateImageCollection(t *testing.T) {
    stmts, err := visitor.Parse("CREATE IMAGE COLLECTION MyModule.Icons;")
    if err != nil {
        t.Fatalf("parse error: %v", err)
    }
    if len(stmts) != 1 {
        t.Fatalf("expected 1 statement, got %d", len(stmts))
    }
    // Import the ast package and type-assert
    // (will fail to compile until AST node + visitor are added)
    _ = stmts[0]
}
```

**Step 2: Run to confirm compile failure**

```bash
go test ./mdl/ast/ -run TestParseCreateImageCollection -v
```

Expected: compile error or panic (visitor not wired yet)

**Step 3: Create the AST node**

Create `mdl/ast/ast_imagecollection.go`:

```go
// SPDX-License-Identifier: Apache-2.0

package ast

// CreateImageCollectionStmt represents:
//
//	CREATE IMAGE COLLECTION Module.Name [EXPORT LEVEL 'Public'] [COMMENT '...']
type CreateImageCollectionStmt struct {
    Name        QualifiedName
    ExportLevel string // "Hidden" (default) or "Public"
    Comment     string
}

func (s *CreateImageCollectionStmt) isStatement() {}

// DropImageCollectionStmt represents: DROP IMAGE COLLECTION Module.Name
type DropImageCollectionStmt struct {
    Name QualifiedName
}

func (s *DropImageCollectionStmt) isStatement() {}
```

**Step 4: Commit**

```bash
git add mdl/ast/ast_imagecollection.go mdl/ast/ast_imagecollection_test.go
git commit -m "feat(ast): add CreateImageCollectionStmt and DropImageCollectionStmt"
```

---

## Task 4: Add Visitor Handler

**Files:**
- Create: `mdl/visitor/visitor_imagecollection.go`
- Read first: `mdl/visitor/visitor_enumeration.go` (reference pattern)

The ANTLR visitor calls `ExitXxxStatement` methods when it finishes parsing a rule. The method name is derived from the rule name: `createImageCollectionStatement` → `ExitCreateImageCollectionStatement`.

**Step 1: Check the generated context type name**

After `make grammar`, look at `mdl/grammar/parser/MDLParser.go` to find the context struct for image collection:

```bash
grep -n "ImageCollection" mdl/grammar/parser/MDLParser.go | head -10
```

This will tell you the exact context type (e.g. `CreateImageCollectionStatementContext`).

**Step 2: Create the visitor file**

```go
// SPDX-License-Identifier: Apache-2.0

package visitor

import (
    "github.com/mendixlabs/mxcli/mdl/ast"
    "github.com/mendixlabs/mxcli/mdl/grammar/parser"
)

// ExitCreateImageCollectionStatement builds a CreateImageCollectionStmt from the parse tree.
func (b *Builder) ExitCreateImageCollectionStatement(ctx *parser.CreateImageCollectionStatementContext) {
    stmt := &ast.CreateImageCollectionStmt{
        Name:        buildQualifiedName(ctx.QualifiedName()),
        ExportLevel: "Hidden", // default
    }

    if opts := ctx.ImageCollectionOptions(); opts != nil {
        optsCtx := opts.(*parser.ImageCollectionOptionsContext)
        for _, opt := range optsCtx.AllImageCollectionOption() {
            optCtx := opt.(*parser.ImageCollectionOptionContext)
            if optCtx.EXPORT() != nil && optCtx.LEVEL() != nil && optCtx.STRING_LITERAL() != nil {
                stmt.ExportLevel = unquoteString(optCtx.STRING_LITERAL().GetText())
            }
            if optCtx.COMMENT() != nil && optCtx.STRING_LITERAL() != nil {
                stmt.Comment = unquoteString(optCtx.STRING_LITERAL().GetText())
            }
        }
    }

    b.statements = append(b.statements, stmt)
}
```

**Note:** The exact context method names (`ImageCollectionOptions()`, `AllImageCollectionOption()`, etc.) depend on what ANTLR generates from your grammar rules. Verify by checking the generated `mdl/grammar/parser/MDLParser.go` after `make grammar`. Adjust if needed.

**Step 3: Verify it compiles**

```bash
go build ./mdl/...
```

**Step 4: Run the earlier parse test**

```bash
go test ./mdl/ast/ -run TestParseCreateImageCollection -v
```

Expected: PASS (or skip if the test needs updating to import ast and type-assert properly)

**Step 5: Commit**

```bash
git add mdl/visitor/visitor_imagecollection.go
git commit -m "feat(visitor): wire ExitCreateImageCollectionStatement"
```

---

## Task 5: Add Executor Command

**Files:**
- Create: `mdl/executor/cmd_imagecollections.go`
- Modify: `mdl/executor/executor.go` (add dispatch case)
- Modify: `mdl/executor/stmt_summary.go` (add summary case)
- Modify: `mdl/executor/validate.go` (add validate case)
- Read first: `mdl/executor/cmd_enumerations.go` lines 1–80 (reference)

**Step 1: Write failing test**

Create `mdl/executor/imagecollection_test.go`:

```go
// SPDX-License-Identifier: Apache-2.0
package executor_test

import (
    "strings"
    "testing"
)

func TestExecCreateImageCollection(t *testing.T) {
    env := newTestEnv(t) // opens test MPR in write mode
    defer env.Close()

    output, err := env.Exec("CREATE IMAGE COLLECTION MyFirstModule.TestIcons;")
    if err != nil {
        t.Fatalf("exec error: %v", err)
    }
    if !strings.Contains(output, "Created image collection") {
        t.Errorf("expected success message, got: %q", output)
    }
}

func TestExecCreateImageCollectionDuplicate(t *testing.T) {
    env := newTestEnv(t)
    defer env.Close()

    if _, err := env.Exec("CREATE IMAGE COLLECTION MyFirstModule.TestIcons;"); err != nil {
        t.Fatalf("first create: %v", err)
    }

    _, err := env.Exec("CREATE IMAGE COLLECTION MyFirstModule.TestIcons;")
    if err == nil {
        t.Error("expected error for duplicate, got nil")
    }
}
```

**Step 2: Run to confirm failure**

```bash
go test ./mdl/executor/ -run TestExecCreateImageCollection -v
```

Expected: FAIL — `execCreateImageCollection` doesn't exist yet

**Step 3: Create the executor file**

```go
// SPDX-License-Identifier: Apache-2.0

// Package executor - Image collection commands (CREATE/DROP IMAGE COLLECTION)
package executor

import (
    "fmt"

    "github.com/mendixlabs/mxcli/mdl/ast"
    "github.com/mendixlabs/mxcli/sdk/mpr"
)

// execCreateImageCollection handles CREATE IMAGE COLLECTION statements.
func (e *Executor) execCreateImageCollection(s *ast.CreateImageCollectionStmt) error {
    if e.reader == nil {
        return fmt.Errorf("not connected to a project")
    }

    module, err := e.findModule(s.Name.Module)
    if err != nil {
        return err
    }

    // Check for duplicate
    if existing := e.findImageCollection(s.Name.Module, s.Name.Name); existing != nil {
        return fmt.Errorf("image collection already exists: %s.%s", s.Name.Module, s.Name.Name)
    }

    ic := &mpr.ImageCollection{
        ContainerID: module.ID,
        Name:        s.Name.Name,
        ExportLevel: s.ExportLevel,
    }

    if err := e.writer.CreateImageCollection(ic); err != nil {
        return fmt.Errorf("failed to create image collection: %w", err)
    }

    e.invalidateHierarchy()

    fmt.Fprintf(e.output, "Created image collection: %s\n", s.Name)
    return nil
}

// execDropImageCollection handles DROP IMAGE COLLECTION statements.
func (e *Executor) execDropImageCollection(s *ast.DropImageCollectionStmt) error {
    if e.reader == nil {
        return fmt.Errorf("not connected to a project")
    }

    existing := e.findImageCollection(s.Name.Module, s.Name.Name)
    if existing == nil {
        return fmt.Errorf("image collection not found: %s.%s", s.Name.Module, s.Name.Name)
    }

    if err := e.writer.DeleteImageCollection(existing.ID); err != nil {
        return fmt.Errorf("failed to delete image collection: %w", err)
    }

    e.invalidateHierarchy()

    fmt.Fprintf(e.output, "Dropped image collection: %s\n", s.Name)
    return nil
}

// findImageCollection finds an image collection by module and name.
func (e *Executor) findImageCollection(moduleName, name string) *mpr.ImageCollection {
    collections, err := e.reader.ListImageCollections()
    if err != nil {
        return nil
    }

    h, err := e.getHierarchy()
    if err != nil {
        return nil
    }

    for _, ic := range collections {
        modID := h.FindModuleID(ic.ContainerID)
        modName := h.GetModuleName(modID)
        if ic.Name == name && modName == moduleName {
            return ic
        }
    }
    return nil
}
```

**Step 4: Add dispatch cases to executor.go**

Find the `switch s := stmt.(type)` block in `mdl/executor/executor.go` around line 176. Add:

```go
case *ast.CreateImageCollectionStmt:
    return e.execCreateImageCollection(s)
case *ast.DropImageCollectionStmt:
    return e.execDropImageCollection(s)
```

**Step 5: Add to stmt_summary.go**

In `mdl/executor/stmt_summary.go`, find the switch block and add after the Enumeration section:

```go
// Image Collection
case *ast.CreateImageCollectionStmt:
    return fmt.Sprintf("CREATE IMAGE COLLECTION %s", s.Name)
case *ast.DropImageCollectionStmt:
    return fmt.Sprintf("DROP IMAGE COLLECTION %s", s.Name)
```

**Step 6: Add to validate.go**

In `mdl/executor/validate.go`, find the reference-validation switch. Add a no-op case for image collections (they don't reference other objects):

```go
case *ast.CreateImageCollectionStmt:
    // no cross-references to validate
case *ast.DropImageCollectionStmt:
    // no cross-references to validate
```

**Step 7: Run tests**

```bash
go test ./mdl/executor/ -run TestExecCreateImageCollection -v
```

Expected: PASS both tests

**Step 8: Commit**

```bash
git add mdl/executor/cmd_imagecollections.go mdl/executor/imagecollection_test.go \
        mdl/executor/executor.go mdl/executor/stmt_summary.go mdl/executor/validate.go
git commit -m "feat(executor): add CREATE/DROP IMAGE COLLECTION commands"
```

---

## Task 6: End-to-End Smoke Test

**Step 1: Build**

```bash
make build
```

Expected: `Built bin/mxcli bin/source_tree`

**Step 2: Syntax check only (no project)**

```bash
./bin/mxcli check /dev/stdin <<'EOF'
CREATE IMAGE COLLECTION MyModule.Icons;
EOF
```

Expected: `✓ Syntax OK` (or equivalent)

**Step 3: Create against real project**

```bash
./bin/mxcli -p /mnt/data_sdd/gh/mxproj-GenAIDemo/App.mpr -c \
    "CREATE IMAGE COLLECTION MyFirstModule.TestPlanIcons;"
```

Expected output: `Created image collection: MyFirstModule.TestPlanIcons`

**Step 4: Verify with BSON dump**

```bash
./bin/mxcli bson dump -p /mnt/data_sdd/gh/mxproj-GenAIDemo/App.mpr \
    --type imagecollection --object "MyFirstModule.TestPlanIcons" --format ndsl
```

Expected NDSL:
```
Images$ImageCollection
  Documentation: ""
  Excluded: false
  ExportLevel: "Hidden"
  Images [marker=3]: []
  Name: "TestPlanIcons"
```

**Step 5: Verify TUI preview works (NDSL mode)**

```bash
MXCLI_TUI_DEBUG=1 ./bin/mxcli tui /mnt/data_sdd/gh/mxproj-GenAIDemo/App.mpr
```

Navigate to `MyFirstModule > TestPlanIcons` and confirm NDSL preview shows without the old "Type not supported" error.

**Step 6: Commit**

```bash
git commit --allow-empty -m "test: e2e smoke test for CREATE IMAGE COLLECTION passes"
```

---

## Task 7: Add DROP IMAGE COLLECTION Grammar (needed for completeness)

**Files:**
- Modify: `mdl/grammar/MDLParser.g4` (add drop rule)

**Step 1: Add dropImageCollectionStatement rule**

In `MDLParser.g4`, find the `dropStatement` rule (search for `DROP ENUMERATION`). It looks like:

```antlr
dropStatement
    : DROP ENTITY qualifiedName
    | DROP ENUMERATION qualifiedName
    | ...
```

Add:

```antlr
| DROP IMAGE COLLECTION qualifiedName
```

**Step 2: Add visitor for drop**

In `mdl/visitor/visitor_imagecollection.go`, add:

```go
func (b *Builder) ExitDropImageCollectionStatement(ctx *parser.DropImageCollectionStatementContext) {
    b.statements = append(b.statements, &ast.DropImageCollectionStmt{
        Name: buildQualifiedName(ctx.QualifiedName()),
    })
}
```

**Step 3: Regenerate and build**

```bash
make grammar && go build ./mdl/...
```

**Step 4: Test DROP**

```bash
./bin/mxcli -p /mnt/data_sdd/gh/mxproj-GenAIDemo/App.mpr -c \
    "DROP IMAGE COLLECTION MyFirstModule.TestPlanIcons;"
```

Expected: `Dropped image collection: MyFirstModule.TestPlanIcons`

**Step 5: Commit**

```bash
git add mdl/grammar/MDLParser.g4 mdl/grammar/parser/ mdl/visitor/visitor_imagecollection.go
git commit -m "feat(grammar): add DROP IMAGE COLLECTION statement"
```

---

## Phase 2 (Future): CREATE IMAGE COLLECTION With Embedded Images

Phase 2 adds:

```sql
CREATE IMAGE COLLECTION MyModule.Icons (
    IMAGE "logo"  FROM FILE 'assets/logo.png',
    IMAGE "close" FROM FILE 'assets/close.svg'
);
```

Implementation additions needed:
- Grammar: extend `imageCollectionOptions` to include `IMAGE STRING_LITERAL FROM FILE STRING_LITERAL`
- AST: add `Images []ImageEntry` field to `CreateImageCollectionStmt` where `ImageEntry` holds `Name string` and `FilePath string`
- Executor: read file bytes at `FilePath` relative to the MDL script's directory, auto-detect format from extension (`.png`→`"Png"`, `.svg`→`"Svg"`, `.gif`→`"Gif"`, `.jpg`/`.jpeg`→`"Jpeg"`, `.bmp`→`"Bmp"`), populate `ic.Images`
- Writer: already supports embedded images in `serializeImageCollection` (via `img.Data` and `img.Format`)

---

## Key Reference Files

| Purpose | File |
|---------|------|
| Existing BSON reader for image collections | `sdk/mpr/reader_types.go` lines 220–261 |
| Existing BSON parser | `sdk/mpr/parser_misc.go` lines 444–479 |
| Reference writer pattern | `sdk/mpr/writer_enumeration.go` |
| Reference executor pattern | `mdl/executor/cmd_enumerations.go` lines 1–80 |
| Reference AST node | `mdl/ast/ast_enumeration.go` lines 30–39 |
| Reference visitor | `mdl/visitor/visitor_enumeration.go` lines 10–36 |
| Grammar: create rule pattern | `mdl/grammar/MDLParser.g4` lines 725–758 |
| Grammar: lexer token pattern | `mdl/grammar/MDLLexer.g4` lines 309–313 |
| Executor dispatch | `mdl/executor/executor.go` line 176 |
| Summary registration | `mdl/executor/stmt_summary.go` lines 54–59 |

## BSON Array Marker Note

The `Images` array in `serializeImageCollection` must start with `int32(3)` as the first element — this is Mendix's array marker for child-element arrays. **Do not** use `int32(2)` (which is for reference arrays). Verified from NDSL dumps of real project data.
