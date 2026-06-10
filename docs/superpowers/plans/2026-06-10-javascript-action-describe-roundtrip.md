# JavaScript Action Describe Round-Trip Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `DESCRIBE JAVASCRIPT ACTION` output a user-friendly, round-trippable `create or modify javascript action` MDL statement with `{imports $$ $$ extra $$ $$ code $$ $$}` block, folder path, and PLATFORM clause.

**Architecture:** Add `createJavaScriptActionStatement` grammar rule (reusing Java action's parameter/return-type sub-rules), wire AST → visitor → executor (file-write + Platform BSON update), then update the describe path to emit the new format. The `{}` wrapper groups the three code sections; `source` path is omitted (always derivable as `javascriptsource/<lowercase-module>/actions/<Name>.js`).

**Tech Stack:** ANTLR4 (grammar), Go (executor/backend/AST/visitor), `modernc.org/sqlite` (MPR v2 via modelsdk writer)

---

## File Map

| Action | File | What changes |
|--------|------|-------------|
| Modify | `mdl/grammar/MDLLexer.g4` | Add `PLATFORM` token |
| Modify | `mdl/grammar/domains/MDLSettings.g4` | Add `PLATFORM` to keyword rule |
| Modify | `mdl/grammar/domains/MDLMicroflow.g4` | Add `createJavaScriptActionStatement` + `jsActionBodyBlock` |
| Modify | `mdl/grammar/MDLParser.g4` | Add dispatcher entry |
| Regen  | `mdl/grammar/parser/` | `make grammar` output |
| Modify | `mdl/ast/ast_javaaction.go` | Add `CreateJavaScriptActionStmt` |
| Modify | `mdl/visitor/visitor_javaaction.go` | Add `ExitCreateJavaScriptActionStatement` |
| Modify | `mdl/executor/cmd_javaactions_v2.go` | Add `writeJavaScriptActionSource`, `execCreateJavaScriptAction`, update `describeJavaScriptActionGen` |
| Modify | `mdl/executor/cmd_javaactions_source_test.go` | Add `TestWriteJavaScriptActionSource_*` |
| Modify | `mdl/backend/java.go` | Add `UpdateJavaScriptActionGen` to `JavaBackend` interface |
| Modify | `mdl/backend/mpr/repos/javaactions.go` | Add `Update` to `javaScriptActionRepo` |
| Modify | `mdl/backend/mpr/backend.go` | Implement `UpdateJavaScriptActionGen` |
| Modify | `mdl/backend/mock/mock_backend.go` | Add `UpdateJavaScriptActionGenFunc` field |
| Modify | `mdl/backend/mock/mock_java.go` | Add mock impl for `UpdateJavaScriptActionGen` |
| Modify | `mdl/executor/register_stubs.go` | Register `CreateJavaScriptActionStmt` handler |
| Modify | `mdl/executor/stmt_summary.go` | Add summary case |
| Modify | `mdl/executor/validate_duplicates.go` | Add duplicate-check case |
| Modify | `mdl/executor/roundtrip_js_action_test.go` | Update describe assertions |
| Modify | `mdl/executor/roundtrip_helpers_test.go` | Copy `javascriptsource` in `copyRoundtripProject` |
| Create | `mdl-examples/doctype-tests/javascript_action.mdl` | Working MDL example |

---

### Task 1: Grammar — PLATFORM token + createJavaScriptActionStatement

**Files:**
- Modify: `mdl/grammar/MDLLexer.g4`
- Modify: `mdl/grammar/domains/MDLSettings.g4`
- Modify: `mdl/grammar/domains/MDLMicroflow.g4`
- Modify: `mdl/grammar/MDLParser.g4`

- [ ] **Step 1: Add PLATFORM token to lexer**

In `mdl/grammar/MDLLexer.g4`, find the block with `WEB: W E B;` (line 195) and add immediately after it:

```antlr
PLATFORM: P L A T F O R M;
```

- [ ] **Step 2: Add PLATFORM to the keyword rule**

In `mdl/grammar/domains/MDLSettings.g4`, find the keyword rule section containing:
```antlr
    | NOTHING | EXPRESSION | JAVASCRIPT
```

Add `| PLATFORM` on that same line (or the next), so it reads:
```antlr
    | NOTHING | EXPRESSION | JAVASCRIPT | PLATFORM
```

- [ ] **Step 3: Add grammar rule to MDLMicroflow.g4**

In `mdl/grammar/domains/MDLMicroflow.g4`, add the following **after** the existing `createJavaActionStatement` rule (after line 49):

```antlr
createJavaScriptActionStatement
    : JAVASCRIPT ACTION qualifiedName
      LPAREN javaActionParameterList? RPAREN
      javaActionReturnType?
      (PLATFORM STRING_LITERAL)?
      (FOLDER STRING_LITERAL)?
      (LBRACE jsActionBodyBlock* RBRACE)?
      SEMICOLON?
    ;

jsActionBodyBlock
    : IDENTIFIER DOLLAR_STRING
    ;
```

Note: `javaActionParameterList`, `javaActionReturnType`, `LBRACE`, `RBRACE`, `FOLDER`, `STRING_LITERAL` are all already defined — no new tokens needed beyond `PLATFORM`.

- [ ] **Step 4: Add dispatcher entry to MDLParser.g4**

In `mdl/grammar/MDLParser.g4`, find the line:
```antlr
      | createJavaActionStatement
```

Add the new rule immediately after:
```antlr
      | createJavaScriptActionStatement
```

- [ ] **Step 5: Regenerate the ANTLR parser**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02 && make grammar
```

Expected: no errors; files in `mdl/grammar/parser/` updated (check `git diff --stat mdl/grammar/parser/`).

- [ ] **Step 6: Verify the grammar compiles**

```bash
go build ./mdl/... 2>&1
```

Expected: no output (clean build).

- [ ] **Step 7: Commit**

```bash
git add mdl/grammar/MDLLexer.g4 mdl/grammar/domains/MDLSettings.g4 \
    mdl/grammar/domains/MDLMicroflow.g4 mdl/grammar/MDLParser.g4 \
    mdl/grammar/parser/
git commit -m "feat(grammar): add PLATFORM token + createJavaScriptActionStatement rule"
```

---

### Task 2: AST node + Visitor

**Files:**
- Modify: `mdl/ast/ast_javaaction.go`
- Modify: `mdl/visitor/visitor_javaaction.go`
- Modify: `mdl/visitor/visitor_javaaction_test.go` (existing or new test)

- [ ] **Step 1: Write a failing parse test**

In `mdl/visitor/visitor_javaaction_test.go`, add:

```go
func TestParse_CreateJavaScriptActionStatement(t *testing.T) {
    src := `create or modify javascript action conversationalui.MyAction(
        textAreaName: String not null
        buttonName: String not null
    )
      returns Nothing
      PLATFORM 'All'
      folder 'ConversationalUI/Actions'
    {
    imports $$
    import { Big } from "big.js";
    $$
    extra $$
    function helper() {}
    $$
    code $$
    return true;
    $$
    };`

    stmts, errs := Build(src)
    if len(errs) > 0 {
        t.Fatalf("parse errors: %v", errs)
    }
    if len(stmts) != 1 {
        t.Fatalf("expected 1 statement, got %d", len(stmts))
    }
    stmt, ok := stmts[0].(*ast.CreateJavaScriptActionStmt)
    if !ok {
        t.Fatalf("expected *ast.CreateJavaScriptActionStmt, got %T", stmts[0])
    }
    if stmt.Name.Module != "conversationalui" || stmt.Name.Name != "MyAction" {
        t.Errorf("Name = %+v", stmt.Name)
    }
    if len(stmt.Parameters) != 2 {
        t.Errorf("len(Parameters) = %d, want 2", len(stmt.Parameters))
    }
    if stmt.Platform != "All" {
        t.Errorf("Platform = %q, want 'All'", stmt.Platform)
    }
    if stmt.Folder != "ConversationalUI/Actions" {
        t.Errorf("Folder = %q", stmt.Folder)
    }
    if !strings.Contains(stmt.Imports, `import { Big }`) {
        t.Errorf("Imports = %q", stmt.Imports)
    }
    if !strings.Contains(stmt.ExtraCode, "helper") {
        t.Errorf("ExtraCode = %q", stmt.ExtraCode)
    }
    if !strings.Contains(stmt.UserCode, "return true") {
        t.Errorf("UserCode = %q", stmt.UserCode)
    }
    if !stmt.CreateOrModify {
        t.Error("CreateOrModify should be true for 'create or modify'")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./mdl/visitor/ -run TestParse_CreateJavaScriptActionStatement -v 2>&1
```

Expected: FAIL — `*ast.CreateJavaScriptActionStmt` undefined.

- [ ] **Step 3: Add the AST node**

In `mdl/ast/ast_javaaction.go`, add after the `DropJavaActionStmt` type:

```go
// CreateJavaScriptActionStmt represents:
//
//	CREATE OR MODIFY JAVASCRIPT ACTION Module.Name(params)
//	  RETURNS type
//	  PLATFORM 'All'
//	  FOLDER 'path'
//	{
//	  imports $$ ... $$
//	  extra $$ ... $$
//	  code $$ ... $$
//	};
type CreateJavaScriptActionStmt struct {
	Name           QualifiedName
	Parameters     []JavaActionParam // reuses JavaActionParam (same grammar sub-rule)
	ReturnType     DataType
	Platform       string // "Web", "Native", "All" — empty treated as "All"
	Folder         string // Mendix project folder path, e.g. "ConversationalUI/Actions"
	Imports        string // raw content from imports $$ ... $$
	ExtraCode      string // raw content from extra $$ ... $$
	UserCode       string // raw content from code $$ ... $$
	Documentation  string // from /** ... */ doc comment
	CreateOrModify bool
}

func (s *CreateJavaScriptActionStmt) isStatement() {}
```

- [ ] **Step 4: Add the visitor**

In `mdl/visitor/visitor_javaaction.go`, add at the end of the file:

```go
// ExitCreateJavaScriptActionStatement handles CREATE [OR MODIFY] JAVASCRIPT ACTION statements.
func (b *Builder) ExitCreateJavaScriptActionStatement(ctx *parser.CreateJavaScriptActionStatementContext) {
	stmt := &ast.CreateJavaScriptActionStmt{}

	if qn := ctx.QualifiedName(); qn != nil {
		stmt.Name = buildQualifiedName(qn)
	}

	if paramList := ctx.JavaActionParameterList(); paramList != nil {
		for _, paramCtx := range paramList.AllJavaActionParameter() {
			param := ast.JavaActionParam{}
			if pn := paramCtx.ParameterName(); pn != nil {
				param.Name = parameterNameText(pn)
			}
			if dt := paramCtx.DataType(); dt != nil {
				param.Type = buildDataType(dt)
			}
			if paramCtx.NOT_NULL() != nil {
				param.IsRequired = true
			}
			stmt.Parameters = append(stmt.Parameters, param)
		}
	}

	if retType := ctx.JavaActionReturnType(); retType != nil {
		if dt := retType.DataType(); dt != nil {
			stmt.ReturnType = buildJavaActionReturnType(dt)
		}
	}

	// PLATFORM 'Web' | 'Native' | 'All'
	allStrings := ctx.AllSTRING_LITERAL()
	strIdx := 0
	if ctx.PLATFORM() != nil && strIdx < len(allStrings) {
		stmt.Platform = unquoteString(allStrings[strIdx].GetText())
		strIdx++
	}
	// FOLDER 'path'
	if ctx.FOLDER() != nil && strIdx < len(allStrings) {
		stmt.Folder = unquoteString(allStrings[strIdx].GetText())
		strIdx++
	}

	// Body blocks: imports / extra / code
	for _, block := range ctx.AllJsActionBodyBlock() {
		raw := block.DOLLAR_STRING().GetText()
		content := strings.TrimSpace(raw[2 : len(raw)-2])
		switch strings.ToLower(block.IDENTIFIER().GetText()) {
		case "imports":
			stmt.Imports = content
		case "extra":
			stmt.ExtraCode = content
		case "code":
			stmt.UserCode = content
		}
	}

	// Doc comment and OR MODIFY from parent createStatement
	if parent, ok := ctx.GetParent().(*parser.CreateStatementContext); ok {
		if docComment := parent.DocComment(); docComment != nil {
			stmt.Documentation = extractDocComment(docComment.GetText())
		}
		if parent.OR() != nil && (parent.MODIFY() != nil || parent.REPLACE() != nil) {
			stmt.CreateOrModify = true
		}
	}
	if stmt.Documentation == "" {
		if stmtCtx := findParentStatement(ctx); stmtCtx != nil {
			if docCtx := stmtCtx.DocComment(); docCtx != nil {
				stmt.Documentation = extractDocComment(docCtx.GetText())
			}
		}
	}

	b.statements = append(b.statements, stmt)
}
```

Note: `PLATFORM()` and `FOLDER()` methods are available on the context because these are grammar tokens. If the generated context names differ (check `mdl/grammar/parser/` after Task 1), adjust accordingly — the ANTLR-generated context names match the token names exactly.

- [ ] **Step 5: Run test to verify it passes**

```bash
go test ./mdl/visitor/ -run TestParse_CreateJavaScriptActionStatement -v 2>&1
```

Expected: PASS.

- [ ] **Step 6: Run full visitor tests**

```bash
go test ./mdl/visitor/ -timeout 60s -count=1 2>&1 | tail -5
```

Expected: `ok github.com/mendixlabs/mxcli/mdl/visitor`

- [ ] **Step 7: Commit**

```bash
git add mdl/ast/ast_javaaction.go mdl/visitor/visitor_javaaction.go mdl/visitor/visitor_javaaction_test.go
git commit -m "feat(ast/visitor): add CreateJavaScriptActionStmt + visitor"
```

---

### Task 3: JS file writer (TDD)

**Files:**
- Modify: `mdl/executor/cmd_javaactions_source_test.go`
- Modify: `mdl/executor/cmd_javaactions_v2.go`

- [ ] **Step 1: Write failing tests for writeJavaScriptActionSource**

Add to `mdl/executor/cmd_javaactions_source_test.go`:

```go
func TestWriteJavaScriptActionSource_CreatesFileWithAllSections(t *testing.T) {
	tmpDir := t.TempDir()
	mprPath := filepath.Join(tmpDir, "test.mpr")
	if err := os.WriteFile(mprPath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	imports := "import { Big } from \"big.js\";\nimport \"mx-global\";"
	extraCode := "function helper() { return 42; }"
	userCode := "return helper();"

	if err := writeJavaScriptActionSource(mprPath, "MyModule", "MyAction", imports, extraCode, userCode); err != nil {
		t.Fatalf("writeJavaScriptActionSource: %v", err)
	}

	jsPath := filepath.Join(tmpDir, "javascriptsource", "mymodule", "actions", "MyAction.js")
	content, err := os.ReadFile(jsPath)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	src := string(content)

	// Verify Mendix header
	if !strings.Contains(src, "This file was generated by Mendix Studio Pro") {
		t.Error("missing Mendix header")
	}
	// Verify imports are present (outside function body)
	if !strings.Contains(src, `import { Big } from "big.js"`) {
		t.Error("missing imports")
	}
	if !strings.Contains(src, `import "mx-global"`) {
		t.Error("missing mx-global import")
	}
	// Verify extra code section
	if !strings.Contains(src, "// BEGIN EXTRA CODE") || !strings.Contains(src, "// END EXTRA CODE") {
		t.Error("missing EXTRA CODE markers")
	}
	if !strings.Contains(src, "function helper()") {
		t.Error("missing extra code content")
	}
	// Verify user code section with correct markers
	if !strings.Contains(src, "\t// BEGIN USER CODE") {
		t.Error("missing BEGIN USER CODE marker (with tab)")
	}
	if !strings.Contains(src, "\t// END USER CODE") {
		t.Error("missing END USER CODE marker (with tab)")
	}
	if !strings.Contains(src, "return helper();") {
		t.Error("missing user code content")
	}
	// Verify function signature uses action name
	if !strings.Contains(src, "export async function MyAction(") {
		t.Error("missing function signature")
	}
}

func TestWriteJavaScriptActionSource_EmptySectionsOmitted(t *testing.T) {
	tmpDir := t.TempDir()
	mprPath := filepath.Join(tmpDir, "test.mpr")
	if err := os.WriteFile(mprPath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	if err := writeJavaScriptActionSource(mprPath, "Mod", "Act", "", "", "throw new Error('not implemented');"); err != nil {
		t.Fatalf("writeJavaScriptActionSource: %v", err)
	}

	jsPath := filepath.Join(tmpDir, "javascriptsource", "mod", "actions", "Act.js")
	content, err := os.ReadFile(jsPath)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	src := string(content)

	// EXTRA CODE markers should still be present (Mendix Studio Pro always writes them)
	if !strings.Contains(src, "// BEGIN EXTRA CODE") {
		t.Error("EXTRA CODE section should always be present")
	}
	// No imports beyond the default big.js
	if strings.Contains(src, "import \"mx-global\"") {
		t.Error("mx-global import should not be present when imports is empty")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./mdl/executor/ -run "TestWriteJavaScriptActionSource" -v 2>&1
```

Expected: FAIL — `writeJavaScriptActionSource` undefined.

- [ ] **Step 3: Implement writeJavaScriptActionSource**

In `mdl/executor/cmd_javaactions_v2.go`, add after the `readJavaScriptActionSource` function (after line ~119):

```go
// writeJavaScriptActionSource writes or overwrites the JS action source file at
// javascriptsource/<lowercase-module>/actions/<ActionName>.js.
// The file is always regenerated from scratch using Mendix Studio Pro's standard
// template format with BEGIN/END markers.
func writeJavaScriptActionSource(mprPath, moduleName, actionName, imports, extraCode, userCode string) error {
	if mprPath == "" {
		return fmt.Errorf("writeJavaScriptActionSource: mprPath is empty")
	}
	projectRoot := filepath.Dir(mprPath)
	modLower := strings.ToLower(moduleName)
	dir := filepath.Join(projectRoot, "javascriptsource", modLower, "actions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create js action dir: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("// This file was generated by Mendix Studio Pro.\n")
	sb.WriteString("//\n")
	sb.WriteString("// WARNING: Only the following code will be retained when actions are regenerated:\n")
	sb.WriteString("// - the import list\n")
	sb.WriteString("// - the code between BEGIN USER CODE and END USER CODE\n")
	sb.WriteString("// - the code between BEGIN EXTRA CODE and END EXTRA CODE\n")
	sb.WriteString("// Other code you write will be lost the next time you deploy the project.\n")
	sb.WriteString("import { Big } from \"big.js\";\n")
	if imports != "" {
		for _, line := range strings.Split(imports, "\n") {
			line = strings.TrimSpace(line)
			// Skip the default big.js import if already written above.
			if line == "" || line == `import { Big } from "big.js";` {
				continue
			}
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")
	sb.WriteString("// BEGIN EXTRA CODE\n")
	if extraCode != "" {
		sb.WriteString(extraCode)
		if !strings.HasSuffix(extraCode, "\n") {
			sb.WriteString("\n")
		}
	}
	sb.WriteString("// END EXTRA CODE\n")
	sb.WriteString("\n")
	sb.WriteString("export async function ")
	sb.WriteString(actionName)
	sb.WriteString("() {\n")
	sb.WriteString("\t// BEGIN USER CODE\n")
	if userCode != "" {
		for _, line := range strings.Split(userCode, "\n") {
			sb.WriteString("\t")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\t// END USER CODE\n")
	sb.WriteString("}\n")

	jsPath := filepath.Join(dir, actionName+".js")
	return os.WriteFile(jsPath, []byte(sb.String()), 0644)
}
```

Note: The function signature in the written file uses `()` without parameters since the parameter list comes from BSON — the JS wrapper is regenerated by Mendix on next deploy. The user code section is the only part that matters for the content.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./mdl/executor/ -run "TestWriteJavaScriptActionSource" -v 2>&1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/cmd_javaactions_v2.go mdl/executor/cmd_javaactions_source_test.go
git commit -m "feat(executor): add writeJavaScriptActionSource file writer"
```

---

### Task 4: Backend — UpdateJavaScriptActionGen

**Files:**
- Modify: `mdl/backend/java.go`
- Modify: `mdl/backend/mpr/repos/javaactions.go`
- Modify: `mdl/backend/mpr/backend.go`
- Modify: `mdl/backend/mock/mock_backend.go`
- Modify: `mdl/backend/mock/mock_java.go`

- [ ] **Step 1: Add UpdateJavaScriptActionGen to the JavaBackend interface**

In `mdl/backend/java.go`, add to the `JavaBackend` interface after `ReadJavaScriptActionByNameGen`:

```go
UpdateJavaScriptActionGen(jsa *genJSA.JavaScriptAction) error
```

- [ ] **Step 2: Add Update method to javaScriptActionRepo**

In `mdl/backend/mpr/repos/javaactions.go`, change the `javaScriptActionRepo` struct and constructor to add write support:

Change the struct (currently has `w`, `r`, `dec`):
```go
type javaScriptActionRepo struct {
	w    *mmpr.Writer
	r    *mmpr.Reader
	dec  *decoder
	enc  *encoder  // add
	sink writeSink // add
}
```

Change `NewJavaScriptActionRepository`:
```go
func NewJavaScriptActionRepository(w *mmpr.Writer) repos.JavaScriptActionRepository {
	return &javaScriptActionRepo{
		w:    w,
		r:    w.ConcreteReader(),
		dec:  newDecoder(),
		enc:  newEncoder(),       // add
		sink: newWriterSink(w),   // add
	}
}
```

Add `Update` method after the `ListAll` method:
```go
func (r *javaScriptActionRepo) Update(jsa *genJSA.JavaScriptAction) error {
	if jsa == nil {
		return fmt.Errorf("javaScriptActionRepo.Update: nil JavaScriptAction")
	}
	contents, err := r.enc.Encode(jsa)
	if err != nil {
		return err
	}
	return r.sink.UpdateRawUnit(string(jsa.ID()), contents)
}
```

Also add the compile-time interface check at the end of the javaScriptAction section:
```go
var _ repos.JavaScriptActionRepository = (*javaScriptActionRepo)(nil)
```

Check the `repos.JavaScriptActionRepository` interface in `mdl/repos/` to verify `Update` is declared there. If it is NOT, add `Update(jsa *genJSA.JavaScriptAction) error` to the interface in `mdl/repos/java_actions.go` (or wherever the interface is defined).

- [ ] **Step 3: Implement UpdateJavaScriptActionGen in MprBackend**

In `mdl/backend/mpr/backend.go`, add after `ReadJavaScriptActionByNameGen`:

```go
func (b *MprBackend) UpdateJavaScriptActionGen(jsa *genJSA.JavaScriptAction) error {
	if jsa == nil {
		return fmt.Errorf("UpdateJavaScriptActionGen: nil JavaScriptAction")
	}
	w, ok := b.concreteWriter()
	if !ok {
		return fmt.Errorf("UpdateJavaScriptActionGen: no modelsdk writer")
	}
	return mprrepos.NewJavaScriptActionRepository(w).Update(jsa)
}
```

- [ ] **Step 4: Add mock stub**

In `mdl/backend/mock/mock_backend.go`, find the block with `UpdateJavaActionGenFunc` and add:
```go
UpdateJavaScriptActionGenFunc func(jsa *genJSA.JavaScriptAction) error
```

In `mdl/backend/mock/mock_java.go`, add after the `ReadJavaScriptActionByNameGen` mock:
```go
func (m *MockBackend) UpdateJavaScriptActionGen(jsa *genJSA.JavaScriptAction) error {
	if m.UpdateJavaScriptActionGenFunc != nil {
		return m.UpdateJavaScriptActionGenFunc(jsa)
	}
	return fmt.Errorf("MockBackend.UpdateJavaScriptActionGen not configured")
}
```

- [ ] **Step 5: Verify it compiles**

```bash
go build ./mdl/... 2>&1
```

Expected: no output.

- [ ] **Step 6: Run backend tests**

```bash
go test ./mdl/backend/... -timeout 120s -count=1 2>&1 | tail -10
```

Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add mdl/backend/java.go mdl/backend/mpr/repos/javaactions.go \
    mdl/backend/mpr/backend.go mdl/backend/mock/mock_backend.go \
    mdl/backend/mock/mock_java.go
git commit -m "feat(backend): add UpdateJavaScriptActionGen write path"
```

---

### Task 5: Executor handler + registration

**Files:**
- Modify: `mdl/executor/cmd_javaactions_v2.go`
- Modify: `mdl/executor/register_stubs.go`
- Modify: `mdl/executor/stmt_summary.go`
- Modify: `mdl/executor/validate_duplicates.go`

- [ ] **Step 1: Write a failing executor test**

In `mdl/executor/cmd_javaactions_v2_test.go` (or a new file `cmd_javascript_actions_v2_test.go`), add:

```go
func TestExecCreateJavaScriptAction_NotFoundError(t *testing.T) {
	// When the JS action does not exist in BSON, executor must return a clear error.
	mock := &backend.MockBackend{}
	mock.ListJavaScriptActionsGenFunc = func() ([]*genJSA.JavaScriptAction, error) {
		return nil, nil // empty — action not in BSON
	}
	ctx := newTestExecContext(mock)

	stmt := &ast.CreateJavaScriptActionStmt{
		Name:           ast.QualifiedName{Module: "MyModule", Name: "MyAction"},
		CreateOrModify: true,
		UserCode:       "return true;",
	}
	err := execCreateJavaScriptAction(ctx, stmt)
	if err == nil {
		t.Fatal("expected error when JS action not found in BSON")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want 'not found'", err.Error())
	}
}
```

Note: `newTestExecContext` is a helper that already exists in the test package — find and reuse it. Look for existing test helpers in `cmd_javaactions_v2_test.go`.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./mdl/executor/ -run "TestExecCreateJavaScriptAction_NotFoundError" -v 2>&1
```

Expected: FAIL — `execCreateJavaScriptAction` undefined.

- [ ] **Step 3: Implement execCreateJavaScriptAction**

In `mdl/executor/cmd_javaactions_v2.go`, add a new section after `execDropJavaActionGen`:

```go
// ─────────────────────────────────────────────────────────────────────
// A6 — execCreateJavaScriptAction
// ─────────────────────────────────────────────────────────────────────

// execCreateJavaScriptAction handles CREATE OR MODIFY JAVASCRIPT ACTION.
// Phase 1: updates the JS source file and Platform BSON field.
// The action must already exist in BSON (must be created in Studio Pro first).
func execCreateJavaScriptAction(ctx *ExecContext, s *ast.CreateJavaScriptActionStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	// Find existing JS action — it must already exist in BSON.
	pairs, err := listJavaScriptActionsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list javascript actions", err)
	}
	var existingJSA *genJSA.JavaScriptAction
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(modelIDFromElementID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && p.Elem.Name() == s.Name.Name {
			existingJSA = p.Elem
			break
		}
	}
	if existingJSA == nil {
		return mdlerrors.NewNotFoundf(
			"javascript action %s.%s not found in project — create it in Mendix Studio Pro first",
			s.Name.Module, s.Name.Name,
		)
	}

	// Update Platform BSON field if provided.
	if s.Platform != "" {
		existingJSA.SetPlatform(s.Platform)
		if err := ctx.Backend.UpdateJavaScriptActionGen(existingJSA); err != nil {
			return mdlerrors.NewBackend("update javascript action platform", err)
		}
	}

	// Write JS source file when any code section is present.
	if s.Imports != "" || s.ExtraCode != "" || s.UserCode != "" {
		if err := writeJavaScriptActionSource(ctx.MprPath, s.Name.Module, s.Name.Name,
			s.Imports, s.ExtraCode, s.UserCode); err != nil {
			return mdlerrors.NewBackend("write javascript source file", err)
		}
		invalidateJavaScriptActionsCache(ctx)
		fmt.Fprintf(ctx.Output, "updated javascript action %s.%s\n", s.Name.Module, s.Name.Name)
	}
	return nil
}
```

Note: `mdlerrors.NewNotFoundf` may not exist — check `mdl/errors/`. If only `NewNotFound(kind, name string)` exists, use:
```go
return mdlerrors.NewNotFound("javascript action", s.Name.Module+"."+s.Name.Name)
```
with an additional hint line `fmt.Fprintf(ctx.Output, "hint: create the action in Mendix Studio Pro first\n")` before the return.

- [ ] **Step 4: Register the handler**

In `mdl/executor/register_stubs.go`, add after the `CreateJavaActionStmt` registration:

```go
r.Register(&ast.CreateJavaScriptActionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
    return execCreateJavaScriptAction(ctx, stmt.(*ast.CreateJavaScriptActionStmt))
})
```

- [ ] **Step 5: Add stmt_summary case**

In `mdl/executor/stmt_summary.go`, add after the Java action cases:

```go
case *ast.CreateJavaScriptActionStmt:
    return fmt.Sprintf("create javascript action %s", s.Name)
```

- [ ] **Step 6: Add validate_duplicates case**

In `mdl/executor/validate_duplicates.go`, find the switch that handles `*ast.CreateJavaActionStmt` and add parallel case:

```go
case *ast.CreateJavaScriptActionStmt:
    if s.CreateOrModify {
        return // OR MODIFY is idempotent — no duplicate check needed
    }
    key := "javascriptaction:" + s.Name.String()
    if seen[key] {
        return fmt.Errorf("duplicate create javascript action: %s", s.Name)
    }
    seen[key] = true
```

Check the exact structure of `validate_duplicates.go` — the pattern may differ. Match what `CreateJavaActionStmt` does.

- [ ] **Step 7: Run failing test to verify it now passes**

```bash
go test ./mdl/executor/ -run "TestExecCreateJavaScriptAction_NotFoundError" -v 2>&1
```

Expected: PASS.

- [ ] **Step 8: Run full executor tests**

```bash
go test ./mdl/executor/ -timeout 120s -count=1 2>&1 | tail -5
```

Expected: `ok github.com/mendixlabs/mxcli/mdl/executor`

- [ ] **Step 9: Commit**

```bash
git add mdl/executor/cmd_javaactions_v2.go mdl/executor/register_stubs.go \
    mdl/executor/stmt_summary.go mdl/executor/validate_duplicates.go
git commit -m "feat(executor): add execCreateJavaScriptAction handler (Phase 1: file + platform)"
```

---

### Task 6: Update describeJavaScriptActionGen to new format

**Files:**
- Modify: `mdl/executor/cmd_javaactions_v2.go`
- Modify: `mdl/executor/roundtrip_js_action_test.go`
- Modify: `mdl/executor/roundtrip_helpers_test.go`

- [ ] **Step 1: Update TestRoundtrip_JavaScriptAction_Describe assertions**

In `mdl/executor/roundtrip_js_action_test.go`, update the test to check for new format:

```go
func TestRoundtrip_JavaScriptAction_Describe(t *testing.T) {
    env := setupRoundtripEnv(t)
    defer env.teardown()

    mdl, err := env.describeMDL("describe javascript action FeedbackModule.JS_isStrictMode")
    if err != nil {
        t.Fatalf("describe javascript action FeedbackModule.JS_isStrictMode: %v", err)
    }

    if strings.TrimSpace(mdl) == "" {
        t.Fatal("expected non-empty output from describe javascript action")
    }
    if !strings.Contains(mdl, "create or modify javascript action ") {
        t.Errorf("expected 'create or modify javascript action' in output, got:\n%s", mdl)
    }
    if !strings.Contains(mdl, "JS_isStrictMode") {
        t.Errorf("expected action name 'JS_isStrictMode' in output, got:\n%s", mdl)
    }
    if !strings.Contains(mdl, "PLATFORM") {
        t.Errorf("expected PLATFORM clause in output, got:\n%s", mdl)
    }
    if !strings.Contains(mdl, "{") || !strings.Contains(mdl, "}") {
        t.Errorf("expected { } body block in output, got:\n%s", mdl)
    }
}
```

- [ ] **Step 2: Run test to verify it fails (wrong verb + missing blocks)**

```bash
go test ./mdl/executor/ -run "TestRoundtrip_JavaScriptAction_Describe" -v -tags integration 2>&1 | head -30
```

Expected: FAIL — output has old format.

- [ ] **Step 3: Rewrite describeJavaScriptActionGen**

Replace the entire `describeJavaScriptActionGen` function in `mdl/executor/cmd_javaactions_v2.go`:

```go
func describeJavaScriptActionGen(ctx *ExecContext, name ast.QualifiedName) error {
	if ctx == nil || ctx.JavaScriptActions == nil {
		return mdlerrors.NewNotFound("javascript action", name.Module+"."+name.Name)
	}
	qn := name.Module + "." + name.Name

	// Use the container-aware listing to get both the element and its folder path.
	pairs, err := listJavaScriptActionsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list javascript actions", err)
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	var jsa *genJSA.JavaScriptAction
	var folderPath string
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(modelIDFromElementID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName+"."+p.Elem.Name() == qn {
			jsa = p.Elem
			folderPath = h.BuildFolderPath(modelIDFromElementID(p.ContainerID))
			break
		}
	}
	if jsa == nil {
		return mdlerrors.NewNotFound("javascript action", qn)
	}

	var sb strings.Builder

	// Doc comment
	doc := strings.ReplaceAll(jsa.Documentation(), "\r\n", "\n")
	doc = strings.ReplaceAll(doc, "\r", "\n")
	if doc != "" {
		sb.WriteString("/**\n")
		for _, line := range strings.Split(doc, "\n") {
			sb.WriteString(" * ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sb.WriteString(" */\n")
	}

	sb.WriteString("create or modify javascript action ")
	sb.WriteString(qn)

	// Type parameters
	typeParams := jsa.ActionTypeParametersItems()
	if len(typeParams) > 0 {
		names := make([]string, 0, len(typeParams))
		for _, tp := range typeParams {
			if typed, ok := tp.(*genJA.TypeParameter); ok && typed.Name() != "" {
				names = append(names, typed.Name())
			} else if tn := genJA.ReadBSONString(tp, "Name"); tn != "" {
				names = append(names, tn)
			}
		}
		if len(names) > 0 {
			sb.WriteString("<")
			sb.WriteString(strings.Join(names, ", "))
			sb.WriteString(">")
		}
	}
	sb.WriteString("(")

	params := jsa.ActionParametersItems()
	hasDescriptions := false
	for _, p := range params {
		if pp, ok := p.(*genJSA.JavaScriptActionParameter); ok && pp.Description() != "" {
			hasDescriptions = true
			break
		}
	}
	wrote := 0
	for _, p := range params {
		pp, ok := p.(*genJSA.JavaScriptActionParameter)
		if !ok {
			continue
		}
		if wrote > 0 {
			sb.WriteString(", ")
		}
		if hasDescriptions {
			sb.WriteString("\n    ")
		}
		sb.WriteString(pp.Name())
		sb.WriteString(": ")
		sb.WriteString(formatJavaActionTypeGen(pp.ActionParameterType(), typeParams))
		if pp.IsRequired() {
			sb.WriteString(" not null")
		}
		if pp.Description() != "" {
			pd := strings.ReplaceAll(pp.Description(), "\r\n", "\n")
			pd = strings.ReplaceAll(pd, "\r", "\n")
			firstLine, _, _ := strings.Cut(pd, "\n")
			sb.WriteString("  -- ")
			sb.WriteString(firstLine)
		}
		wrote++
	}
	if hasDescriptions {
		sb.WriteString("\n")
	}
	sb.WriteString(")")

	if rt := jsa.ActionReturnType(); rt != nil {
		sb.WriteString("\n  returns ")
		sb.WriteString(formatJavaActionReturnTypeGen(rt, typeParams))
	}

	platform := jsa.Platform()
	if platform == "" {
		platform = "All"
	}
	sb.WriteString("\n  PLATFORM '")
	sb.WriteString(platform)
	sb.WriteString("'")

	if folderPath != "" {
		sb.WriteString("\n  folder '")
		sb.WriteString(folderPath)
		sb.WriteString("'")
	}

	// EXPOSED AS clause
	if mai := jsa.ModelerActionInfo(); mai != nil {
		caption := genJA.ReadBSONString(mai, "Caption")
		category := genJA.ReadBSONString(mai, "Category")
		if caption != "" {
			sb.WriteString("\n  exposed as '")
			sb.WriteString(caption)
			sb.WriteString("' in '")
			sb.WriteString(category)
			sb.WriteString("'")
		}
	}

	// Code body block { imports $$ $$ extra $$ $$ code $$ $$ }
	userCode, extraCode, _ := readJavaScriptActionSource(ctx.MprPath, name.Module, name.Name)

	// Read imports from the JS file (all import lines in the file)
	importsStr := ""
	if ctx.MprPath != "" {
		projectRoot := filepath.Dir(ctx.MprPath)
		modLower := strings.ToLower(name.Module)
		jsPath := filepath.Join(projectRoot, "javascriptsource", modLower, "actions", name.Name+".js")
		if content, err := os.ReadFile(jsPath); err == nil {
			var importLines []string
			for _, line := range strings.Split(string(content), "\n") {
				t := strings.TrimSpace(line)
				if strings.HasPrefix(t, "import ") {
					importLines = append(importLines, t)
				}
			}
			importsStr = strings.Join(importLines, "\n")
		}
	}

	sb.WriteString("\n{")
	if importsStr != "" {
		sb.WriteString("\nimports $$\n")
		sb.WriteString(importsStr)
		sb.WriteString("\n$$")
	}
	if extraCode != "" {
		sb.WriteString("\nextra $$\n")
		sb.WriteString(extraCode)
		sb.WriteString("\n$$")
	}
	if userCode != "" {
		sb.WriteString("\ncode $$\n")
		sb.WriteString(userCode)
		sb.WriteString("\n$$")
	}
	sb.WriteString("\n}")

	sb.WriteString(";")
	fmt.Fprintln(ctx.Output, sb.String())

	if el := jsa.ExportLevel(); el != "" && el != "Hidden" {
		fmt.Fprintf(ctx.Output, "-- export level: %s\n", el)
	}
	if jsa.Excluded() {
		fmt.Fprintln(ctx.Output, "-- EXCLUDED: true")
	}
	if rn := jsa.ActionDefaultReturnName(); rn != "" {
		fmt.Fprintf(ctx.Output, "-- return NAME: '%s'\n", rn)
	}
	return nil
}
```

- [ ] **Step 4: Update copyRoundtripProject to include javascriptsource**

In `mdl/executor/roundtrip_helpers_test.go`, find `copyRoundtripProject` and change the `sub` slice:

```go
for _, sub := range []string{"mprcontents", "widgets", "javascriptsource"} {
```

(Was: `[]string{"mprcontents", "widgets"}`)

- [ ] **Step 5: Run the failing test**

```bash
go test ./mdl/executor/ -run "TestRoundtrip_JavaScriptAction_Describe" -v -tags integration 2>&1 | head -30
```

Expected: PASS (or skip if roundtrip project has no `javascriptsource` — that's OK).

- [ ] **Step 6: Run full executor tests**

```bash
go test ./mdl/executor/ -timeout 120s -count=1 2>&1 | tail -5
```

Expected: `ok github.com/mendixlabs/mxcli/mdl/executor`

- [ ] **Step 7: Commit**

```bash
git add mdl/executor/cmd_javaactions_v2.go mdl/executor/roundtrip_js_action_test.go \
    mdl/executor/roundtrip_helpers_test.go
git commit -m "feat(executor): update describeJavaScriptActionGen to new {} block format with folder + PLATFORM"
```

---

### Task 7: MDL example + full test sweep

**Files:**
- Create: `mdl-examples/doctype-tests/javascript_action.mdl`

- [ ] **Step 1: Create the MDL example file**

Create `mdl-examples/doctype-tests/javascript_action.mdl`:

```mdl
-- DESCRIBE JAVASCRIPT ACTION
-- Verifies the describe output format (read-only, no write applied).
describe javascript action FeedbackModule.JS_isStrictMode;
```

Note: This is intentionally minimal — a `create or modify javascript action` example can't be run without a project that already has the action. Document this in a comment.

- [ ] **Step 2: Run full test suite**

```bash
go test ./... -timeout 180s -count=1 2>&1 | grep -E "FAIL|ok" | tail -20
```

Expected: all packages pass (`ok ...`), no `FAIL`.

- [ ] **Step 3: Run `make build`**

```bash
make build 2>&1 | tail -10
```

Expected: clean build, no errors.

- [ ] **Step 4: Commit**

```bash
git add mdl-examples/doctype-tests/javascript_action.mdl
git commit -m "test: add javascript_action MDL example"
```

---

## Self-Review Checklist

After completing all tasks, verify:

- [ ] `go test ./... -count=1` passes clean
- [ ] `make build` succeeds
- [ ] `mxcli check` on a `create or modify javascript action` statement parses without errors
- [ ] `DESCRIBE JAVASCRIPT ACTION` output contains `create or modify`, `PLATFORM '...'`, `folder '...'`, `{`, `}`
- [ ] The output is parseable back through `mxcli check` (round-trip syntactic check)
- [ ] No `-- source:` comments remain in describe output
- [ ] No `-- EXTRA CODE:` comments remain in describe output
- [ ] No `-- PLATFORM:` trailing comments remain in describe output
