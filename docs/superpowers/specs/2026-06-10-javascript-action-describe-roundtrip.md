# JavaScript Action Describe Round-Trip Design

**Date**: 2026-06-10  
**Status**: Approved

## Problem

`DESCRIBE JAVASCRIPT ACTION` 输出存在三个问题：

1. **Extra code 以注释形式输出**（`-- EXTRA CODE: ...`），不可解析，无法 round-trip
2. **Imports 完全缺失**，无法复原 JS 文件完整内容
3. **文件夹路径缺失**，用户不知道该 action 在 Mendix 项目树中的位置

## Solution: Three-Section Block Format

采用与 Java action 一致的三段式结构，并用 `{}` 包裹代码段，移除可推导的 `source` 路径。

### Output Format

```mdl
/**
 * Doc comment from BSON Documentation field
 */
create or modify javascript action conversationalui.Textarea_ExecuteButtonAction(
    textAreaName: String not null  -- Name of the textarea on the page
    buttonName: String not null    -- Name of the button on the page
    submitOnEnter: Boolean not null
    submitOnShiftEnter: Boolean not null
)
  returns Nothing
  PLATFORM All
  folder 'ConversationalUI/Actions'
{
imports $$
import { Big } from "big.js";
import "mx-global";
$$
extra $$
function clickButton(buttonName) {
    const buttonElement = document.querySelector('button[class*="' + buttonName + '"]');
    if (buttonElement === null) {
        throw new Error("Button with that name could not be found.")
    }
    buttonElement.click();
}
$$
code $$
try {
    if (!textAreaName || textAreaName.trim().length === 0) {
        throw new Error("TextAreaName is required.")
    }
    // ...user code...
}
catch (err) {
    console.error(err)
}
$$
};
```

**Key decisions**:
- **No `source` clause** — path is always deterministic: `javascriptsource/<lowercase-module>/actions/<Name>.js`
- **`{}` wraps** imports/extra/code sections — visually groups the file body
- **`folder 'path'`** — Mendix project hierarchy path (from `h.BuildFolderPath()`)
- **`PLATFORM`** — new lexer token; value is `Web`, `Native`, or `All`
- **`create or modify`** — idempotent verb enabling round-trip apply
- Sections are optional; omit when empty (no `imports $$$$`, no `extra $$$$`)

### Round-Trip Semantics

When `create or modify javascript action` is applied:
1. **JS source file** is written/updated at `javascriptsource/<lowercase-module>/actions/<Name>.js`:
   - Regenerates the file header and function scaffold
   - Inserts `imports` content as ES module imports
   - Inserts `extra` content between `// BEGIN EXTRA CODE` / `// END EXTRA CODE`
   - Inserts `code` content between `// BEGIN USER CODE` / `// END USER CODE`
2. **BSON platform field** updated if `PLATFORM` clause is present
3. **Error** if the action does not exist in BSON — must be created first in Studio Pro

Parameter structural changes (add/remove/rename/type-change) are **not** applied by the executor — the parameter list in the MDL is informational when re-applying. This is a known limitation of Phase 1.

## Components to Implement

### 1. Lexer — add `PLATFORM` token

File: `mdl/grammar/MDLLexer.g4`

```antlr
PLATFORM: P L A T F O R M;
```

Add to `keyword` rule in `MDLSettings.g4` to avoid IDENTIFIER conflicts.

### 2. Grammar — add `createJavaScriptActionStatement`

File: `mdl/grammar/domains/MDLMicroflow.g4`

```antlr
createJavaScriptActionStatement
    : (CREATE OR MODIFY | CREATE OR REPLACE | CREATE) JAVASCRIPT ACTION qualifiedName
      LPAREN javaActionParameterList? RPAREN
      javaActionReturnType?
      (PLATFORM IDENTIFIER)?
      (FOLDER STRING_LITERAL)?
      (LBRACE jsActionBodyBlock* RBRACE)?
      SEMICOLON?
    ;

jsActionBodyBlock
    : IDENTIFIER DOLLAR_STRING   // imports $$ ... $$  |  extra $$ ... $$  |  code $$ ... $$
    ;
```

Add `| createJavaScriptActionStatement` to the main statement dispatcher in `MDLParser.g4`.

Run `make grammar` to regenerate parser.

### 3. AST — `CreateJavaScriptActionStmt`

File: `mdl/ast/ast_javaaction.go`

```go
type CreateJavaScriptActionStmt struct {
    Name           QualifiedName
    Parameters     []JavaActionParam  // reuse existing type
    ReturnType     DataType
    Platform       string   // "Web", "Native", "All" — empty = All
    Folder         string   // Mendix project folder path
    Imports        string   // raw content from imports $$ ... $$
    ExtraCode      string   // raw content from extra $$ ... $$
    UserCode       string   // raw content from code $$ ... $$
    Documentation  string   // from /** ... */ doc comment
    CreateOrModify bool
}
func (s *CreateJavaScriptActionStmt) isStatement() {}
```

### 4. Visitor — bridge parse tree → AST

File: `mdl/visitor/visitor_javaaction.go` (new file or append to existing)

Map `CreateJavaScriptActionStatementContext` → `CreateJavaScriptActionStmt`. Reuse `visitJavaActionParameterList()` and `visitDataType()` helpers already used by Java action visitor.

### 5. Executor handler

File: `mdl/executor/cmd_javaactions_v2.go` (new section A6)

```
execCreateJavaScriptAction(ctx, stmt):
  1. Look up JS action by QN via listJavaScriptActionsWithContainerGen
  2. If not found: return error "JS action not found; create it in Studio Pro first"
  3. Update BSON (Platform, parameters, return type) via backend write
  4. Build JS file content and write to javascriptsource/<mod>/actions/<Name>.js
```

Register in `executor_dispatch.go`.

### 6. Describe output update

File: `mdl/executor/cmd_javaactions_v2.go` (`describeJavaScriptActionGen`)

- Change verb to `create or modify javascript action`
- Add `PLATFORM` clause
- Add `folder 'path'` clause (needs container lookup via `listJavaScriptActionsWithContainerGen`)
- Output `{ imports $$ $$ extra $$ $$ code $$ $$ }` block
- Remove `-- source:`, `-- EXTRA CODE:`, `-- PLATFORM:` comments

### 7. JS file writer helper

New function `writeJavaScriptActionSource(mprPath, moduleName, actionName, imports, extraCode, userCode string) error`

Reconstructs the full JS file with Mendix Studio Pro header and BEGIN/END markers.

## Test Coverage

- **Unit test** `TestReadJavaScriptActionSource_*` (already passing after prior fix)
- **Unit test** `TestWriteJavaScriptActionSource` — verifies the file writer produces correct BEGIN/END markers
- **Unit test** `TestDescribeJavaScriptAction_Format` — verifies describe output contains `folder`, `imports`, `extra`, `code` blocks
- **Integration test** `TestRoundtrip_JavaScriptAction_Describe` — update to assert new format fields
- **MDL example** in `mdl-examples/doctype-tests/javascript_action.mdl`

## Out of Scope

- Creating a brand-new JS action (requires Studio Pro to create BSON first)
- Modifying BSON parameter structure (parameter add/remove/rename) in the round-trip handler — Phase 1 updates the JS file only; BSON metadata update is Phase 2
