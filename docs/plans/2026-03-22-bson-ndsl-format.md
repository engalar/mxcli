# BSON Normalized DSL (NDSL) Format Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `--format ndsl` flag to `mxcli bson dump` that renders any BSON object as a normalized, diffable text format suitable for LLM-assisted debugging of BSON serialization issues.

**Architecture:** Add a `Render(doc bson.D) string` function to the `bson/` package that converts `bson.D` to an alphabetically-sorted, UUID-normalized, array-marker-annotated text format. Wire it into `cmd_bson_dump.go` via a `--format` flag. Works for any object type (workflow, page, microflow, etc.).

**Tech Stack:** Go, `go.mongodb.org/mongo-driver/bson`, `//go:build debug` build tag

---

## Output Format Spec

```
Workflows$Workflow
  Documentation: ""
  DueDate: ""
  Excluded: false
  ExportLevel: "Hidden"
  Flow: Workflows$Flow
    Activities [marker=3]:
      - Workflows$StartWorkflowActivity
          Annotation: null
          BoundaryEvents [marker=2]: []
          Caption: "Start"
          Name: "startWorkflow1"
          PersistentId: <uuid>
  OnWorkflowEvent [marker=2]: []
  Parameter: Workflows$Parameter
    Entity: "WorkflowBaseline.Entity"
    Name: "WorkflowContext"
  WorkflowName: Microflows$StringTemplate
    Text: "Workflow"
```

**Rules:**
- Fields sorted alphabetically within each object
- `$ID` omitted entirely
- Binary UUID fields → `<uuid>`
- `null` values printed explicitly
- Arrays with int32 first element → `[marker=N]:` header, elements as `- TypeName` or values
- Empty arrays → `[marker=N]: []` on one line
- Nested objects with `$Type` → inline type name, fields indented below
- `$Type` is rendered as the object header, not as a field

---

## Task 1: Add `bson/render.go` with core NDSL renderer

**Files:**
- Create: `bson/render.go`
- Create: `bson/render_test.go`

**Step 1: Write failing test**

```go
// bson/render_test.go
//go:build debug

package bson

import (
    "testing"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

func TestRenderScalarFields(t *testing.T) {
    doc := bson.D{
        {Key: "$Type", Value: "Workflows$Workflow"},
        {Key: "Name", Value: "TestWf"},
        {Key: "Excluded", Value: false},
        {Key: "AdminPage", Value: nil},
    }
    got := Render(doc, 0)
    want := `Workflows$Workflow
  AdminPage: null
  Excluded: false
  Name: "TestWf"`
    if got != want {
        t.Errorf("got:\n%s\nwant:\n%s", got, want)
    }
}

func TestRenderUUIDNormalized(t *testing.T) {
    doc := bson.D{
        {Key: "$Type", Value: "Workflows$Flow"},
        {Key: "$ID", Value: primitive.Binary{Subtype: 3, Data: []byte("anything")}},
        {Key: "PersistentId", Value: primitive.Binary{Subtype: 3, Data: []byte("anything")}},
    }
    got := Render(doc, 0)
    want := `Workflows$Flow
  PersistentId: <uuid>`
    if got != want {
        t.Errorf("got:\n%s\nwant:\n%s", got, want)
    }
}

func TestRenderArrayWithMarker(t *testing.T) {
    doc := bson.D{
        {Key: "$Type", Value: "Workflows$Flow"},
        {Key: "Activities", Value: bson.A{int32(3), bson.D{
            {Key: "$Type", Value: "Workflows$EndWorkflowActivity"},
            {Key: "Name", Value: "end1"},
        }}},
    }
    got := Render(doc, 0)
    want := `Workflows$Flow
  Activities [marker=3]:
    - Workflows$EndWorkflowActivity
        Name: "end1"`
    if got != want {
        t.Errorf("got:\n%s\nwant:\n%s", got, want)
    }
}

func TestRenderEmptyArray(t *testing.T) {
    doc := bson.D{
        {Key: "$Type", Value: "Workflows$StartWorkflowActivity"},
        {Key: "BoundaryEvents", Value: bson.A{int32(2)}},
    }
    got := Render(doc, 0)
    want := `Workflows$StartWorkflowActivity
  BoundaryEvents [marker=2]: []`
    if got != want {
        t.Errorf("got:\n%s\nwant:\n%s", got, want)
    }
}
```

**Step 2: Run test to verify it fails**

```bash
go test -tags debug ./bson/ -run TestRender -v
```
Expected: FAIL with "undefined: Render"

**Step 3: Implement `bson/render.go`**

```go
//go:build debug

package bson

import (
    "fmt"
    "sort"
    "strings"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

// Render converts a bson.D document to Normalized DSL text.
// indent is the base indentation level (0 for top-level).
func Render(doc bson.D, indent int) string {
    var sb strings.Builder
    renderDoc(&sb, doc, indent)
    return strings.TrimRight(sb.String(), "\n")
}

func renderDoc(sb *strings.Builder, doc bson.D, indent int) {
    pad := strings.Repeat("  ", indent)

    // Extract $Type for header
    typeName := ""
    for _, e := range doc {
        if e.Key == "$Type" {
            typeName, _ = e.Value.(string)
            break
        }
    }
    if typeName != "" {
        sb.WriteString(pad + typeName + "\n")
    }

    // Collect non-structural fields, sort alphabetically
    type field struct {
        key string
        val any
    }
    var fields []field
    for _, e := range doc {
        if e.Key == "$ID" || e.Key == "$Type" {
            continue
        }
        fields = append(fields, field{e.Key, e.Value})
    }
    sort.Slice(fields, func(i, j int) bool {
        return fields[i].key < fields[j].key
    })

    for _, f := range fields {
        renderField(sb, f.key, f.val, indent+1)
    }
}

func renderField(sb *strings.Builder, key string, val any, indent int) {
    pad := strings.Repeat("  ", indent)

    switch v := val.(type) {
    case nil:
        fmt.Fprintf(sb, "%s%s: null\n", pad, key)

    case primitive.Binary:
        // UUID binary fields
        fmt.Fprintf(sb, "%s%s: <uuid>\n", pad, key)

    case bson.D:
        // Nested object with $Type
        typeName := ""
        for _, e := range v {
            if e.Key == "$Type" {
                typeName, _ = e.Value.(string)
                break
            }
        }
        if typeName != "" {
            fmt.Fprintf(sb, "%s%s: %s\n", pad, key, typeName)
            renderDoc(sb, v, indent+1)
        } else {
            fmt.Fprintf(sb, "%s%s:\n", pad, key)
            renderDoc(sb, v, indent+1)
        }

    case bson.A:
        renderArray(sb, key, v, indent)

    case string:
        fmt.Fprintf(sb, "%s%s: %q\n", pad, key, v)

    case bool:
        fmt.Fprintf(sb, "%s%s: %v\n", pad, key, v)

    default:
        fmt.Fprintf(sb, "%s%s: %v\n", pad, key, v)
    }
}

func renderArray(sb *strings.Builder, key string, arr bson.A, indent int) {
    pad := strings.Repeat("  ", indent)

    // Check for array marker (first element is int32)
    markerStr := ""
    startIdx := 0
    if len(arr) > 0 {
        if marker, ok := arr[0].(int32); ok {
            markerStr = fmt.Sprintf(" [marker=%d]", marker)
            startIdx = 1
        }
    }

    elements := arr[startIdx:]
    if len(elements) == 0 {
        fmt.Fprintf(sb, "%s%s%s: []\n", pad, key, markerStr)
        return
    }

    fmt.Fprintf(sb, "%s%s%s:\n", pad, key, markerStr)
    for _, elem := range elements {
        renderArrayElement(sb, elem, indent+1)
    }
}

func renderArrayElement(sb *strings.Builder, elem any, indent int) {
    pad := strings.Repeat("  ", indent)

    switch v := elem.(type) {
    case bson.D:
        typeName := ""
        for _, e := range v {
            if e.Key == "$Type" {
                typeName, _ = e.Value.(string)
                break
            }
        }
        if typeName != "" {
            fmt.Fprintf(sb, "%s- %s\n", pad, typeName)
            // Render fields of this element at indent+1
            type field struct {
                key string
                val any
            }
            var fields []field
            for _, e := range v {
                if e.Key == "$ID" || e.Key == "$Type" {
                    continue
                }
                fields = append(fields, field{e.Key, e.Value})
            }
            sort.Slice(fields, func(i, j int) bool {
                return fields[i].key < fields[j].key
            })
            for _, f := range fields {
                renderField(sb, f.key, f.val, indent+1)
            }
        } else {
            fmt.Fprintf(sb, "%s-\n", pad)
            renderDoc(sb, v, indent+1)
        }

    case string:
        fmt.Fprintf(sb, "%s- %q\n", pad, v)

    default:
        fmt.Fprintf(sb, "%s- %v\n", pad, elem)
    }
}
```

**Step 4: Run tests**

```bash
go test -tags debug ./bson/ -run TestRender -v
```
Expected: all 4 tests PASS

**Step 5: Commit**

```bash
git add bson/render.go bson/render_test.go
git commit -m "feat(bson): add NDSL normalized DSL renderer for LLM debugging"
```

---

## Task 2: Wire `--format ndsl` into `bson dump`

**Files:**
- Modify: `cmd/mxcli/cmd_bson_dump.go`

**Step 1: Check current dump output path**

Read `cmd_bson_dump.go` to find where JSON is printed (the `json.MarshalIndent` call).

**Step 2: Add `--format` flag and NDSL branch**

In `init()`, add:
```go
dumpCmd.Flags().String("format", "json", "Output format: json, ndsl")
```

In the dump handler, after fetching the raw unit bytes, add:
```go
format, _ := cmd.Flags().GetString("format")
if format == "ndsl" {
    var doc bson.D
    if err := bson.Unmarshal(unit.Contents, &doc); err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing BSON: %v\n", err)
        os.Exit(1)
    }
    fmt.Println(bsondebug.Render(doc, 0))
    return
}
```

Do this for each output path (single object, list, compare).

**Step 3: Manual smoke test**

```bash
go build -tags debug -o /tmp/mxcli-debug ./cmd/mxcli/
/tmp/mxcli-debug bson dump \
  -p /mnt/data_sdd/gh/mxproj-GenAIDemo/App.mpr \
  --type workflow \
  --object "WorkflowBaseline.Sub_Workflow" \
  --format ndsl
```

Expected: readable NDSL output with `[marker=N]`, `<uuid>`, sorted fields, no raw JSON.

**Step 4: Commit**

```bash
git add cmd/mxcli/cmd_bson_dump.go
git commit -m "feat(bson): add --format ndsl flag to bson dump command"
```

---

## Task 3: Wire NDSL into `bson compare` for side-by-side diff

**Files:**
- Modify: `cmd/mxcli/cmd_bson_compare.go`

**Step 1: Add `--format ndsl` to compare**

When `--format ndsl`, instead of current diff output, print:
```
=== LEFT: WorkflowBaseline.Workflow ===
<ndsl of left>

=== RIGHT: WorkflowBaseline.Sub_Workflow ===
<ndsl of right>
```

So an LLM can `diff` the two blocks directly.

**Step 2: Implement**

```go
format, _ := cmd.Flags().GetString("format")
if format == "ndsl" {
    var leftDoc, rightDoc bson.D
    bson.Unmarshal(leftUnit.Contents, &leftDoc)
    bson.Unmarshal(rightUnit.Contents, &rightDoc)
    fmt.Printf("=== LEFT: %s ===\n%s\n\n=== RIGHT: %s ===\n%s\n",
        leftName, bsondebug.Render(leftDoc, 0),
        rightName, bsondebug.Render(rightDoc, 0))
    return
}
```

Also add `--format` flag in `init()`:
```go
bsonCompareCmd.Flags().String("format", "diff", "Output format: diff, ndsl")
```

**Step 3: Manual smoke test**

```bash
/tmp/mxcli-debug bson compare \
  -p /mnt/data_sdd/gh/mxproj-GenAIDemo/App.mpr \
  --type workflow \
  --format ndsl \
  "WorkflowBaseline.Workflow" "WorkflowBaseline.Sub_Workflow"
```

Expected: two NDSL blocks printed, easy to diff manually or paste to LLM.

**Step 4: Commit**

```bash
git add cmd/mxcli/cmd_bson_compare.go
git commit -m "feat(bson): add --format ndsl to bson compare for LLM-friendly side-by-side output"
```

---

## Task 4: Handle `bson.M` vs `bson.D` in compare (fix array marker loss)

**Context:** Current `compare` uses `bson.Unmarshal` into `bson.M` (map). Maps lose array markers. For NDSL rendering we use `bson.D`. The diff engine in `compare.go` also uses `bson.M`. This task adds a `bson.D`-aware path.

This is a **stretch task** — only do it if time allows. The NDSL render path (Task 3) already uses `bson.D` correctly.

---

## Testing Summary

After all tasks:

```bash
# Unit tests
go test -tags debug ./bson/ -v

# Smoke: dump as NDSL
/tmp/mxcli-debug bson dump -p App.mpr --type workflow --object "WorkflowBaseline.Workflow" --format ndsl

# Smoke: compare as NDSL (for LLM diff)
/tmp/mxcli-debug bson compare -p App.mpr -p2 generated.mpr --type workflow --format ndsl "WorkflowBaseline.Workflow"
```
