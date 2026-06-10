# Image Collection CRUD + TUI 图片显示修复 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 TUI describe image collection 无法显示图片的 bug，并为 MDL 添加完整的 ALTER IMAGE COLLECTION 语法，支持对集合内图片的 ADD/DROP/RENAME/SET/MOVE/EXPORT 操作。

**Architecture:** TUI bug 是单文件一行修复（大小写不敏感搜索）。ALTER IMAGE COLLECTION 走全栈：Grammar → AST → Visitor → Backend interface/MPR/Mock → Executor → 注册 → 测试。所有 ADD/DROP/RENAME/SET 动作复用现有 `UpdateImageCollection`；MOVE 需新增 `MoveImageCollection` backend 方法；EXPORT 是纯 OS 文件写入，不走 backend。

**Tech Stack:** Go, ANTLR4, charmbracelet/bubbletea, modernc.org/sqlite

---

## 变更文件清单

| 文件 | 操作 |
|------|------|
| `cmd/mxcli/tui/image_render.go` | 修改：大小写不敏感搜索（bug fix） |
| `cmd/mxcli/tui/miller.go` | 修改：同上（findImagePathAtClick） |
| `cmd/mxcli/tui/preview_test.go` | 修改：新增 extractImagePaths 回归测试 |
| `mdl/grammar/MDLParser.g4` | 修改：新增 alterImageCollectionAction 规则 |
| `mdl/grammar/parser/` | 重新生成：`make grammar` |
| `mdl/ast/ast_imagecollection.go` | 修改：新增 AlterImageCollectionStmt + 6 个 action 类型 |
| `mdl/visitor/visitor_imagecollection.go` | 修改：新增 ExitAlterImageCollectionStatement |
| `mdl/backend/infrastructure.go` | 修改：ImageBackend 接口新增 MoveImageCollection |
| `mdl/backend/mpr/backend.go` | 修改：MoveImageCollection 路由方法 |
| `mdl/backend/mpr/update_services.go` | 修改：moveImageCollectionViaModelsdk 实现 |
| `mdl/backend/mock/mock_backend.go` | 修改：新增 MoveImageCollectionFunc 字段 |
| `mdl/backend/mock/mock_workflow.go` | 修改：新增 MoveImageCollection 方法 |
| `mdl/executor/cmd_imagecollections.go` | 修改：新增 execAlterImageCollection |
| `mdl/executor/register_stubs.go` | 修改：注册 AlterImageCollectionStmt |
| `mdl/executor/stmt_summary.go` | 修改：新增 case *ast.AlterImageCollectionStmt |
| `mdl/executor/validate.go` | 修改：新增 case *ast.AlterImageCollectionStmt |
| `mdl/executor/imagecollections_mock_test.go` | 修改：新增全套 ALTER 测试 |
| `mdl-examples/image_collection_alter.mdl` | 新建：MDL 示例脚本 |

---

## Task 1: TUI Bug Fix — 大小写不敏感路径提取

**Files:**
- Modify: `cmd/mxcli/tui/image_render.go:120-136`
- Modify: `cmd/mxcli/tui/miller.go:910-939`
- Modify: `cmd/mxcli/tui/preview_test.go`

- [ ] **Step 1: 写失败测试**

在 `cmd/mxcli/tui/preview_test.go` 末尾追加：

```go
func TestExtractImagePaths_LowercaseKeywords(t *testing.T) {
	// MDL output uses lowercase keywords since commit f70a74158
	output := `create or modify image collection MyModule.Icons (
    image logo from file '/tmp/mxcli-preview/MyModule.Icons/logo.png',
    image banner from file '/tmp/mxcli-preview/MyModule.Icons/banner.svg'
);`
	paths := extractImagePaths(output)
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
	if paths[0] != "/tmp/mxcli-preview/MyModule.Icons/logo.png" {
		t.Errorf("paths[0] = %q, want /tmp/mxcli-preview/MyModule.Icons/logo.png", paths[0])
	}
	if paths[1] != "/tmp/mxcli-preview/MyModule.Icons/banner.svg" {
		t.Errorf("paths[1] = %q, want /tmp/mxcli-preview/MyModule.Icons/banner.svg", paths[1])
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./cmd/mxcli/tui/ -run TestExtractImagePaths_LowercaseKeywords -v
```

期望：`FAIL — got 0 paths`

- [ ] **Step 3: 修复 extractImagePaths（image_render.go）**

将 `image_render.go:120-136` 替换为：

```go
func extractImagePaths(output string) []string {
	var paths []string
	const marker = "FROM FILE '"
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		idx := strings.Index(strings.ToUpper(line), marker)
		if idx == -1 {
			continue
		}
		rest := line[idx+len(marker):]
		end := strings.Index(rest, "'")
		if end == -1 {
			continue
		}
		paths = append(paths, rest[:end])
	}
	return paths
}
```

- [ ] **Step 4: 修复 findImagePathAtClick（miller.go:921-925）**

将 `miller.go:920-925` 中的两处 `"FROM FILE '"` 搜索改为大小写不敏感：

```go
// 原代码：
plain := stripAnsi(contentLines[srcIdx])
i := strings.Index(plain, "FROM FILE '")
if i == -1 {
    continue
}
rest := plain[i+len("FROM FILE '"):]

// 改为：
plain := stripAnsi(contentLines[srcIdx])
const markerFIC = "FROM FILE '"
i := strings.Index(strings.ToUpper(plain), markerFIC)
if i == -1 {
    continue
}
rest := plain[i+len(markerFIC):]
```

- [ ] **Step 5: 运行测试，确认通过**

```bash
go test ./cmd/mxcli/tui/ -run TestExtractImagePaths_LowercaseKeywords -v
```

期望：`PASS`

- [ ] **Step 6: 运行 TUI 全量测试**

```bash
go test ./cmd/mxcli/tui/ -v 2>&1 | tail -20
```

期望：所有测试 PASS，无新失败。

- [ ] **Step 7: Commit**

```bash
git add cmd/mxcli/tui/image_render.go cmd/mxcli/tui/miller.go cmd/mxcli/tui/preview_test.go
git commit -m "fix(tui): case-insensitive FROM FILE matching in extractImagePaths and findImagePathAtClick"
```

---

## Task 2: Grammar — 新增 alterImageCollectionAction 规则

**Files:**
- Modify: `mdl/grammar/MDLParser.g4:133-147`（alterStatement 规则）
- Modify: `mdl/grammar/parser/`（`make grammar` 重新生成）

- [ ] **Step 1: 编辑 MDLParser.g4**

在 `alterStatement` 规则（第 133-147 行）中，在 `| alterModuleJarDepStatement` **之前**插入新行：

```antlr
alterStatement
    : ALTER ENTITY qualifiedName alterEntityAction+
    | ALTER ASSOCIATION qualifiedName alterAssociationAction+
    | ALTER ENUMERATION qualifiedName alterEnumerationAction+
    | ALTER NOTEBOOK qualifiedName alterNotebookAction+
    | ALTER ODATA CLIENT qualifiedName SET odataAlterAssignment (COMMA odataAlterAssignment)*
    | ALTER ODATA SERVICE qualifiedName SET odataAlterAssignment (COMMA odataAlterAssignment)*
    | ALTER STYLING ON (PAGE | SNIPPET) qualifiedName WIDGET IDENTIFIER alterStylingAction+
    | ALTER SETTINGS alterSettingsClause
    | ALTER PAGE qualifiedName LBRACE alterPageOperation+ RBRACE
    | ALTER SNIPPET qualifiedName LBRACE alterPageOperation+ RBRACE
    | ALTER WORKFLOW qualifiedName alterWorkflowAction+ SEMICOLON?
    | ALTER PUBLISHED REST SERVICE qualifiedName alterPublishedRestServiceAction (COMMA? alterPublishedRestServiceAction)*
    | ALTER IMAGE COLLECTION qualifiedName alterImageCollectionAction (COMMA alterImageCollectionAction)* SEMICOLON?
    | alterModuleJarDepStatement
    ;
```

在文件中找一个合适位置（如 `alterPublishedRestServiceAction` 规则之后）添加新规则：

```antlr
alterImageCollectionAction
    : ADD IMAGE imageName FROM FILE_KW STRING_LITERAL   // ADD IMAGE logo FROM FILE 'path'
    | DROP IMAGE imageName                              // DROP IMAGE logo
    | RENAME IMAGE imageName TO imageName               // RENAME IMAGE logo TO logo_v2
    | SET IMAGE imageName FROM FILE_KW STRING_LITERAL  // SET IMAGE logo FROM FILE 'path'
    | MOVE TO qualifiedName                             // MOVE TO OtherModule.Icons
    | EXPORT IMAGE imageName TO FILE_KW STRING_LITERAL // EXPORT IMAGE logo TO FILE 'path'
    ;
```

- [ ] **Step 2: 重新生成 parser**

```bash
make grammar
```

期望：`mdl/grammar/parser/` 下的文件更新，无报错。

- [ ] **Step 3: 验证编译通过**

```bash
go build ./mdl/...
```

期望：编译成功（此时 visitor 还没有实现新回调，但 base_listener 有默认空实现，不影响编译）。

- [ ] **Step 4: Commit**

```bash
git add mdl/grammar/MDLParser.g4 mdl/grammar/parser/
git commit -m "feat(grammar): add alterImageCollectionAction rule"
```

---

## Task 3: AST — 新增 AlterImageCollectionStmt 和 action 类型

**Files:**
- Modify: `mdl/ast/ast_imagecollection.go`

- [ ] **Step 1: 写失败测试（编译检查）**

在 `mdl/executor/imagecollections_mock_test.go` 末尾临时追加以下测试（用于驱动 AST 编译）：

```go
func TestAlterImageCollectionStmt_Compile(t *testing.T) {
	stmt := &ast.AlterImageCollectionStmt{
		Name: ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{
			&ast.AddImageAction{ImageName: "logo", FilePath: "./logo.png"},
			&ast.DropImageAction{ImageName: "logo"},
			&ast.RenameImageAction{From: "logo", To: "logo_v2"},
			&ast.SetImageAction{ImageName: "logo", FilePath: "./logo_new.png"},
			&ast.MoveImageCollectionAction{Target: ast.QualifiedName{Module: "Other", Name: "Icons"}},
			&ast.ExportImageAction{ImageName: "logo", FilePath: "./out/logo.png"},
		},
	}
	_ = stmt
}
```

- [ ] **Step 2: 运行测试，确认编译失败**

```bash
go build ./mdl/... 2>&1 | head -10
```

期望：`ast.AlterImageCollectionStmt undefined`

- [ ] **Step 3: 扩展 ast_imagecollection.go**

在 `mdl/ast/ast_imagecollection.go` 的 `DropImageCollectionStmt` 之后追加：

```go
// AlterImageCollectionStmt represents ALTER IMAGE COLLECTION Module.Name action [, action...]
type AlterImageCollectionStmt struct {
	Name    QualifiedName
	Actions []ImageCollectionAction
}

func (s *AlterImageCollectionStmt) isStatement() {}

// ImageCollectionAction is the interface for all ALTER IMAGE COLLECTION sub-actions.
type ImageCollectionAction interface{ isImageCollectionAction() }

// AddImageAction: ADD IMAGE name FROM FILE 'path'
type AddImageAction struct {
	ImageName string
	FilePath  string
}

// DropImageAction: DROP IMAGE name
type DropImageAction struct {
	ImageName string
}

// RenameImageAction: RENAME IMAGE oldName TO newName
type RenameImageAction struct {
	From string
	To   string
}

// SetImageAction: SET IMAGE name FROM FILE 'path'
type SetImageAction struct {
	ImageName string
	FilePath  string
}

// MoveImageCollectionAction: MOVE TO Module.Name
type MoveImageCollectionAction struct {
	Target QualifiedName
}

// ExportImageAction: EXPORT IMAGE name TO FILE 'path'
type ExportImageAction struct {
	ImageName string
	FilePath  string
}

func (a *AddImageAction)             isImageCollectionAction() {}
func (a *DropImageAction)            isImageCollectionAction() {}
func (a *RenameImageAction)          isImageCollectionAction() {}
func (a *SetImageAction)             isImageCollectionAction() {}
func (a *MoveImageCollectionAction)  isImageCollectionAction() {}
func (a *ExportImageAction)          isImageCollectionAction() {}
```

- [ ] **Step 4: 运行测试，确认编译通过**

```bash
go build ./mdl/...
```

期望：编译成功。

- [ ] **Step 5: Commit**

```bash
git add mdl/ast/ast_imagecollection.go mdl/executor/imagecollections_mock_test.go
git commit -m "feat(ast): add AlterImageCollectionStmt and ImageCollectionAction types"
```

---

## Task 4: Visitor — 解析 ALTER IMAGE COLLECTION 语句

**Files:**
- Modify: `mdl/visitor/visitor_imagecollection.go`

- [ ] **Step 1: 写 visitor 测试**

在 `mdl/visitor/visitor_imagecollection_test.go` 末尾追加：

```go
func TestAlterImageCollection_Add(t *testing.T) {
	stmts := parse(t, `ALTER IMAGE COLLECTION MyModule.Icons ADD IMAGE logo FROM FILE './logo.png';`)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}
	s, ok := stmts[0].(*ast.AlterImageCollectionStmt)
	if !ok {
		t.Fatalf("expected *ast.AlterImageCollectionStmt, got %T", stmts[0])
	}
	if s.Name.Module != "MyModule" || s.Name.Name != "Icons" {
		t.Errorf("unexpected name: %v", s.Name)
	}
	if len(s.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(s.Actions))
	}
	add, ok := s.Actions[0].(*ast.AddImageAction)
	if !ok {
		t.Fatalf("expected *ast.AddImageAction, got %T", s.Actions[0])
	}
	if add.ImageName != "logo" {
		t.Errorf("ImageName = %q, want logo", add.ImageName)
	}
	if add.FilePath != "./logo.png" {
		t.Errorf("FilePath = %q, want ./logo.png", add.FilePath)
	}
}

func TestAlterImageCollection_MultipleActions(t *testing.T) {
	src := `ALTER IMAGE COLLECTION Mod.Icons
		ADD IMAGE logo FROM FILE './logo.png',
		DROP IMAGE banner,
		RENAME IMAGE old TO new_name,
		SET IMAGE logo FROM FILE './logo2.png',
		MOVE TO Other.Icons,
		EXPORT IMAGE logo TO FILE './out.png';`
	stmts := parse(t, src)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}
	s := stmts[0].(*ast.AlterImageCollectionStmt)
	if len(s.Actions) != 6 {
		t.Fatalf("expected 6 actions, got %d: %v", len(s.Actions), s.Actions)
	}
	if _, ok := s.Actions[0].(*ast.AddImageAction); !ok {
		t.Errorf("actions[0]: want AddImageAction, got %T", s.Actions[0])
	}
	if _, ok := s.Actions[1].(*ast.DropImageAction); !ok {
		t.Errorf("actions[1]: want DropImageAction, got %T", s.Actions[1])
	}
	if r, ok := s.Actions[2].(*ast.RenameImageAction); !ok || r.From != "old" || r.To != "new_name" {
		t.Errorf("actions[2]: want RenameImageAction{old,new_name}, got %T %v", s.Actions[2], s.Actions[2])
	}
	if _, ok := s.Actions[3].(*ast.SetImageAction); !ok {
		t.Errorf("actions[3]: want SetImageAction, got %T", s.Actions[3])
	}
	mv, ok := s.Actions[4].(*ast.MoveImageCollectionAction)
	if !ok || mv.Target.Module != "Other" || mv.Target.Name != "Icons" {
		t.Errorf("actions[4]: want MoveImageCollectionAction{Other.Icons}, got %T %v", s.Actions[4], s.Actions[4])
	}
	if _, ok := s.Actions[5].(*ast.ExportImageAction); !ok {
		t.Errorf("actions[5]: want ExportImageAction, got %T", s.Actions[5])
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

```bash
go test ./mdl/visitor/ -run "TestAlterImageCollection" -v
```

期望：`FAIL — 解析后 AlterImageCollectionStmt 为 nil 或 actions 数量为 0`

- [ ] **Step 3: 实现 ExitAlterImageCollectionStatement**

在 `mdl/visitor/visitor_imagecollection.go` 末尾追加（需要导入 `"github.com/mendixlabs/mxcli/mdl/grammar/parser"`）：

```go
// ExitAlterImageCollectionStatement handles ALTER IMAGE COLLECTION Module.Name action+ statements.
func (b *Builder) ExitAlterImageCollectionStatement(ctx *parser.AlterStatementContext) {
	if ctx.IMAGE() == nil || ctx.COLLECTION() == nil {
		return
	}
	qn := ctx.QualifiedName()
	if qn == nil {
		return
	}
	stmt := &ast.AlterImageCollectionStmt{
		Name: buildQualifiedName(qn),
	}
	for _, rawAction := range ctx.AllAlterImageCollectionAction() {
		action := rawAction.(*parser.AlterImageCollectionActionContext)
		switch {
		case action.ADD() != nil:
			stmt.Actions = append(stmt.Actions, &ast.AddImageAction{
				ImageName: extractImageName(action.AllImageName()[0]),
				FilePath:  unquoteString(action.STRING_LITERAL().GetText()),
			})
		case action.DROP() != nil && action.MOVE() == nil:
			stmt.Actions = append(stmt.Actions, &ast.DropImageAction{
				ImageName: extractImageName(action.AllImageName()[0]),
			})
		case action.RENAME() != nil:
			names := action.AllImageName()
			stmt.Actions = append(stmt.Actions, &ast.RenameImageAction{
				From: extractImageName(names[0]),
				To:   extractImageName(names[1]),
			})
		case action.SET() != nil:
			stmt.Actions = append(stmt.Actions, &ast.SetImageAction{
				ImageName: extractImageName(action.AllImageName()[0]),
				FilePath:  unquoteString(action.STRING_LITERAL().GetText()),
			})
		case action.MOVE() != nil:
			stmt.Actions = append(stmt.Actions, &ast.MoveImageCollectionAction{
				Target: buildQualifiedName(action.QualifiedName()),
			})
		case action.EXPORT() != nil:
			stmt.Actions = append(stmt.Actions, &ast.ExportImageAction{
				ImageName: extractImageName(action.AllImageName()[0]),
				FilePath:  unquoteString(action.STRING_LITERAL().GetText()),
			})
		}
	}
	b.statements = append(b.statements, stmt)
}

// extractImageName returns the text of an imageName rule, stripping optional quotes.
func extractImageName(ctx parser.IImageNameContext) string {
	if ctx == nil {
		return ""
	}
	name := ctx.GetText()
	if len(name) >= 2 && (name[0] == '"' || name[0] == '`') {
		return name[1 : len(name)-1]
	}
	return name
}
```

**注意**：ANTLR 生成的监听器是通过 `ExitAlterStatement` 分发的（参见 `visitor_alter.go`）。实际检查该文件如何路由到各 alter 动作，确保 `ExitAlterStatement` 会调用 `ExitAlterImageCollectionStatement`。如果分发方式不同，在 `visitor_alter.go` 中添加对应分支。

- [ ] **Step 4: 检查 visitor_alter.go 分发机制**

```bash
cat mdl/visitor/visitor_alter.go
```

如果 `ExitAlterStatement` 是通过 ANTLR 规则 alt 标签自动分发，则无需改动。如果是手动 switch，需要在该文件的 `ExitAlterStatement` 中添加：

```go
case ctx.IMAGE() != nil && ctx.COLLECTION() != nil:
    b.ExitAlterImageCollectionStatement(ctx)
```

- [ ] **Step 5: 运行测试，确认通过**

```bash
go test ./mdl/visitor/ -run "TestAlterImageCollection" -v
```

期望：`PASS`

- [ ] **Step 6: Commit**

```bash
git add mdl/visitor/visitor_imagecollection.go mdl/visitor/visitor_alter.go mdl/visitor/visitor_imagecollection_test.go
git commit -m "feat(visitor): parse ALTER IMAGE COLLECTION into AlterImageCollectionStmt"
```

---

## Task 5: Backend — MoveImageCollection 接口 + MPR 实现 + Mock Stub

**Files:**
- Modify: `mdl/backend/infrastructure.go:92-98`
- Modify: `mdl/backend/mpr/update_services.go`
- Modify: `mdl/backend/mpr/backend.go`
- Modify: `mdl/backend/mock/mock_backend.go`
- Modify: `mdl/backend/mock/mock_workflow.go`

- [ ] **Step 1: 更新 ImageBackend 接口**

在 `mdl/backend/infrastructure.go` 的 `ImageBackend` 接口中新增一行：

```go
// ImageBackend provides image collection operations.
type ImageBackend interface {
    ListImageCollections() ([]*types.ImageCollection, error)
    CreateImageCollection(ic *types.ImageCollection) error
    UpdateImageCollection(ic *types.ImageCollection) error
    DeleteImageCollection(id string) error
    MoveImageCollection(ic *types.ImageCollection) error // 新增
}
```

- [ ] **Step 2: 确认编译失败（接口未实现）**

```bash
go build ./mdl/... 2>&1 | grep "does not implement"
```

期望：`MprBackend does not implement ImageBackend (missing MoveImageCollection method)`

- [ ] **Step 3: 实现 moveImageCollectionViaModelsdk**

在 `mdl/backend/mpr/update_services.go` 末尾追加：

```go
func (b *MprBackend) moveImageCollectionViaModelsdk(ic *types.ImageCollection) error {
    if b.msdkWriter == nil {
        return fmt.Errorf("modelsdk writer not initialized")
    }
    return b.msdkWriter.UpdateUnitContainer(string(ic.ID), string(ic.ContainerID))
}
```

- [ ] **Step 4: 在 backend.go 添加 MoveImageCollection 路由**

在 `mdl/backend/mpr/backend.go` 中 `DeleteImageCollection` 之后添加：

```go
func (b *MprBackend) MoveImageCollection(ic *types.ImageCollection) error {
    return b.moveImageCollectionViaModelsdk(ic)
}
```

- [ ] **Step 5: 在 mock_backend.go 新增 Func 字段**

在 `mdl/backend/mock/mock_backend.go` 的 `DeleteImageCollectionFunc` 行之后添加：

```go
MoveImageCollectionFunc func(ic *types.ImageCollection) error
```

- [ ] **Step 6: 在 mock_workflow.go 新增方法**

在 `DeleteImageCollection` 方法之后追加：

```go
func (m *MockBackend) MoveImageCollection(ic *types.ImageCollection) error {
    if m.MoveImageCollectionFunc != nil {
        return m.MoveImageCollectionFunc(ic)
    }
    return fmt.Errorf("MockBackend.MoveImageCollection not configured")
}
```

- [ ] **Step 7: 确认编译通过**

```bash
go build ./mdl/...
```

期望：编译成功，`var _ backend.ImageBackend = (*MprBackend)(nil)` 检查通过（如 backend.go 中有此行）。

- [ ] **Step 8: Commit**

```bash
git add mdl/backend/infrastructure.go mdl/backend/mpr/update_services.go mdl/backend/mpr/backend.go \
        mdl/backend/mock/mock_backend.go mdl/backend/mock/mock_workflow.go
git commit -m "feat(backend): add MoveImageCollection to ImageBackend interface and MprBackend"
```

---

## Task 6: Executor — execAlterImageCollection

**Files:**
- Modify: `mdl/executor/cmd_imagecollections.go`

- [ ] **Step 1: 追加 execAlterImageCollection 函数**

在 `mdl/executor/cmd_imagecollections.go` 末尾追加：

```go
// execAlterImageCollection handles ALTER IMAGE COLLECTION statements.
// Multiple actions are applied to a single read of the collection and flushed in one UpdateImageCollection call
// (except MOVE and EXPORT which are separate operations).
func execAlterImageCollection(ctx *ExecContext, s *ast.AlterImageCollectionStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnected()
	}

	ic := findImageCollection(ctx, s.Name.Module, s.Name.Name)
	if ic == nil {
		return mdlerrors.NewNotFound("image collection", s.Name.String())
	}

	dirty := false // track if UpdateImageCollection is needed

	for _, rawAction := range s.Actions {
		switch action := rawAction.(type) {

		case *ast.AddImageAction:
			data, format, err := readImageFile(action.FilePath)
			if err != nil {
				return err
			}
			ic.Images = append(ic.Images, types.Image{
				Name:   action.ImageName,
				Data:   data,
				Format: format,
			})
			dirty = true
			fmt.Fprintf(ctx.Output, "Added image %q to %s\n", action.ImageName, s.Name)

		case *ast.DropImageAction:
			idx := findImageIndex(ic, action.ImageName)
			if idx < 0 {
				return mdlerrors.NewNotFound("image", action.ImageName)
			}
			ic.Images = append(ic.Images[:idx], ic.Images[idx+1:]...)
			dirty = true
			fmt.Fprintf(ctx.Output, "Dropped image %q from %s\n", action.ImageName, s.Name)

		case *ast.RenameImageAction:
			idx := findImageIndex(ic, action.From)
			if idx < 0 {
				return mdlerrors.NewNotFound("image", action.From)
			}
			if findImageIndex(ic, action.To) >= 0 {
				return mdlerrors.NewAlreadyExists("image", action.To)
			}
			ic.Images[idx].Name = action.To
			dirty = true
			fmt.Fprintf(ctx.Output, "Renamed image %q to %q in %s\n", action.From, action.To, s.Name)

		case *ast.SetImageAction:
			idx := findImageIndex(ic, action.ImageName)
			if idx < 0 {
				return mdlerrors.NewNotFound("image", action.ImageName)
			}
			data, format, err := readImageFile(action.FilePath)
			if err != nil {
				return err
			}
			ic.Images[idx].Data = data
			ic.Images[idx].Format = format
			dirty = true
			fmt.Fprintf(ctx.Output, "Updated image %q in %s\n", action.ImageName, s.Name)

		case *ast.MoveImageCollectionAction:
			// Flush pending changes before move
			if dirty {
				if err := ctx.Backend.UpdateImageCollection(ic); err != nil {
					return mdlerrors.NewBackend("update image collection before move", err)
				}
				dirty = false
			}
			targetMod, err := findModule(ctx, action.Target.Module)
			if err != nil {
				return mdlerrors.NewNotFound("module", action.Target.Module)
			}
			ic.ContainerID = targetMod.ID
			if err := ctx.Backend.MoveImageCollection(ic); err != nil {
				return mdlerrors.NewBackend("move image collection", err)
			}
			invalidateHierarchy(ctx)
			fmt.Fprintf(ctx.Output, "Moved image collection %s to module %s\n", s.Name, action.Target.Module)

		case *ast.ExportImageAction:
			idx := findImageIndex(ic, action.ImageName)
			if idx < 0 {
				return mdlerrors.NewNotFound("image", action.ImageName)
			}
			filePath := action.FilePath
			if !filepath.IsAbs(filePath) {
				cwd, err := os.Getwd()
				if err != nil {
					return mdlerrors.NewBackend("get working directory", err)
				}
				filePath = filepath.Join(cwd, filePath)
			}
			if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
				return mdlerrors.NewBackend("create output directory", err)
			}
			if err := os.WriteFile(filePath, ic.Images[idx].Data, 0o644); err != nil {
				return mdlerrors.NewBackend(fmt.Sprintf("write image file %q", filePath), err)
			}
			fmt.Fprintf(ctx.Output, "Exported image %q to %s\n", action.ImageName, filePath)
		}
	}

	// Flush pending mutations
	if dirty {
		if err := ctx.Backend.UpdateImageCollection(ic); err != nil {
			return mdlerrors.NewBackend("update image collection", err)
		}
	}

	return nil
}

// readImageFile reads an image file and returns (data, format, error).
func readImageFile(filePath string) ([]byte, string, error) {
	if !filepath.IsAbs(filePath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, "", mdlerrors.NewBackend("get working directory", err)
		}
		filePath = filepath.Join(cwd, filePath)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", mdlerrors.NewBackend(fmt.Sprintf("read image file %q", filePath), err)
	}
	return data, extToImageFormat(filepath.Ext(filePath)), nil
}

// findImageIndex returns the index of the image with the given name, or -1.
func findImageIndex(ic *types.ImageCollection, name string) int {
	for i, img := range ic.Images {
		if img.Name == name {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 2: 确认编译通过**

```bash
go build ./mdl/executor/
```

期望：编译成功。

- [ ] **Step 3: Commit**

```bash
git add mdl/executor/cmd_imagecollections.go
git commit -m "feat(executor): implement execAlterImageCollection with ADD/DROP/RENAME/SET/MOVE/EXPORT actions"
```

---

## Task 7: 注册 + stmt_summary + validate

**Files:**
- Modify: `mdl/executor/register_stubs.go`
- Modify: `mdl/executor/stmt_summary.go`
- Modify: `mdl/executor/validate.go`

- [ ] **Step 1: 注册新语句（register_stubs.go）**

在 `register_stubs.go` 中 `CreateImageCollectionStmt` 注册行之后添加：

```go
r.Register(&ast.AlterImageCollectionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
    return execAlterImageCollection(ctx, stmt.(*ast.AlterImageCollectionStmt))
})
```

- [ ] **Step 2: 更新 stmt_summary.go**

在 `stmt_summary.go` 的 `DropImageCollectionStmt` case 之后添加：

```go
case *ast.AlterImageCollectionStmt:
    return fmt.Sprintf("alter image collection %s", s.Name)
```

- [ ] **Step 3: 更新 validate.go**

在 `validate.go` 的 `DropImageCollectionStmt` case 之后添加（只验证模块存在性）：

```go
case *ast.AlterImageCollectionStmt:
    if s.Name.Module != "" && !sc.modules[s.Name.Module] {
        if _, err := findModule(ctx, s.Name.Module); err != nil {
            return mdlerrors.NewNotFound("module", s.Name.Module)
        }
    }
```

- [ ] **Step 4: 编译确认**

```bash
go build ./mdl/...
```

- [ ] **Step 5: Commit**

```bash
git add mdl/executor/register_stubs.go mdl/executor/stmt_summary.go mdl/executor/validate.go
git commit -m "feat(executor): register AlterImageCollectionStmt and update stmt_summary/validate"
```

---

## Task 8: 测试 — 全套 ALTER 动作的 Mock 测试

**Files:**
- Modify: `mdl/executor/imagecollections_mock_test.go`

移除 Task 3 中临时添加的 `TestAlterImageCollectionStmt_Compile` 测试，添加以下完整测试集：

- [ ] **Step 1: 写测试**

在 `imagecollections_mock_test.go` 末尾追加：

```go
// ── TestAlterImageCollection_NotConnected ────────────────────────────────────

func TestAlterImageCollection_NotConnected(t *testing.T) {
	mb := &mock.MockBackend{IsConnectedFunc: func() bool { return false }}
	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execAlterImageCollection(ctx, &ast.AlterImageCollectionStmt{
		Name: ast.QualifiedName{Module: "Mod", Name: "Icons"},
	})
	assertError(t, err)
}

// ── TestAlterImageCollection_CollectionNotFound ──────────────────────────────

func TestAlterImageCollection_CollectionNotFound(t *testing.T) {
	mod := mkModule("Mod")
	h := mkHierarchy(mod)
	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		IsWritableFunc:           func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return nil, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execAlterImageCollection(ctx, &ast.AlterImageCollectionStmt{
		Name:    ast.QualifiedName{Module: "Mod", Name: "NoSuch"},
		Actions: []ast.ImageCollectionAction{&ast.DropImageAction{ImageName: "logo"}},
	})
	assertError(t, err)
}

// ── TestAlterImageCollection_Drop ────────────────────────────────────────────

func TestAlterImageCollection_Drop(t *testing.T) {
	mod := mkModule("Mod")
	ic := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod.ID,
		Name:        "Icons",
		Images: []types.Image{
			{ID: nextID("img"), Name: "logo", Data: []byte("png"), Format: "Png"},
			{ID: nextID("img"), Name: "banner", Data: []byte("svg"), Format: "Svg"},
		},
	}
	h := mkHierarchy(mod)
	withContainer(h, ic.ContainerID, mod.ID)

	var updatedIC *types.ImageCollection
	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		IsWritableFunc:           func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
		UpdateImageCollectionFunc: func(u *types.ImageCollection) error {
			updatedIC = u
			return nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execAlterImageCollection(ctx, &ast.AlterImageCollectionStmt{
		Name:    ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{&ast.DropImageAction{ImageName: "logo"}},
	})
	assertNoError(t, err)
	if updatedIC == nil {
		t.Fatal("expected UpdateImageCollection to be called")
	}
	if len(updatedIC.Images) != 1 || updatedIC.Images[0].Name != "banner" {
		t.Errorf("expected only banner to remain, got %v", updatedIC.Images)
	}
	assertContainsStr(t, buf.String(), "Dropped image")
}

// ── TestAlterImageCollection_Drop_ImageNotFound ──────────────────────────────

func TestAlterImageCollection_Drop_ImageNotFound(t *testing.T) {
	mod := mkModule("Mod")
	ic := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod.ID,
		Name:        "Icons",
		Images:      []types.Image{{Name: "logo", Data: []byte("x"), Format: "Png"}},
	}
	h := mkHierarchy(mod)
	withContainer(h, ic.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		IsWritableFunc:           func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execAlterImageCollection(ctx, &ast.AlterImageCollectionStmt{
		Name:    ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{&ast.DropImageAction{ImageName: "nosuch"}},
	})
	assertError(t, err)
}

// ── TestAlterImageCollection_Rename ─────────────────────────────────────────

func TestAlterImageCollection_Rename(t *testing.T) {
	mod := mkModule("Mod")
	ic := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod.ID,
		Name:        "Icons",
		Images:      []types.Image{{Name: "logo", Data: []byte("x"), Format: "Png"}},
	}
	h := mkHierarchy(mod)
	withContainer(h, ic.ContainerID, mod.ID)

	var updatedIC *types.ImageCollection
	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		IsWritableFunc:           func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
		UpdateImageCollectionFunc: func(u *types.ImageCollection) error {
			updatedIC = u
			return nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execAlterImageCollection(ctx, &ast.AlterImageCollectionStmt{
		Name:    ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{&ast.RenameImageAction{From: "logo", To: "logo_v2"}},
	})
	assertNoError(t, err)
	if updatedIC == nil || updatedIC.Images[0].Name != "logo_v2" {
		t.Errorf("expected image renamed to logo_v2, got %v", updatedIC)
	}
}

// ── TestAlterImageCollection_Rename_TargetExists ─────────────────────────────

func TestAlterImageCollection_Rename_TargetExists(t *testing.T) {
	mod := mkModule("Mod")
	ic := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod.ID,
		Name:        "Icons",
		Images: []types.Image{
			{Name: "logo", Data: []byte("x"), Format: "Png"},
			{Name: "logo_v2", Data: []byte("y"), Format: "Png"},
		},
	}
	h := mkHierarchy(mod)
	withContainer(h, ic.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		IsWritableFunc:           func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execAlterImageCollection(ctx, &ast.AlterImageCollectionStmt{
		Name:    ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{&ast.RenameImageAction{From: "logo", To: "logo_v2"}},
	})
	assertError(t, err)
}

// ── TestAlterImageCollection_Move ────────────────────────────────────────────

func TestAlterImageCollection_Move(t *testing.T) {
	mod := mkModule("Mod")
	target := mkModule("Other")
	ic := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod.ID,
		Name:        "Icons",
	}
	h := mkHierarchy(mod, target)
	withContainer(h, ic.ContainerID, mod.ID)

	var movedIC *types.ImageCollection
	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		IsWritableFunc:           func() bool { return true },
		ListModulesFunc:          func() ([]*model.Module, error) { return []*model.Module{mod, target}, nil },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
		MoveImageCollectionFunc:  func(u *types.ImageCollection) error { movedIC = u; return nil },
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execAlterImageCollection(ctx, &ast.AlterImageCollectionStmt{
		Name: ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{
			&ast.MoveImageCollectionAction{Target: ast.QualifiedName{Module: "Other", Name: "Icons"}},
		},
	})
	assertNoError(t, err)
	if movedIC == nil || movedIC.ContainerID != target.ID {
		t.Errorf("expected collection moved to Other module, got %v", movedIC)
	}
	assertContainsStr(t, buf.String(), "Moved image collection")
}

// ── TestAlterImageCollection_Export ─────────────────────────────────────────

func TestAlterImageCollection_Export(t *testing.T) {
	mod := mkModule("Mod")
	imgData := []byte("\x89PNG\r\n\x1a\n") // minimal PNG header
	ic := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod.ID,
		Name:        "Icons",
		Images:      []types.Image{{Name: "logo", Data: imgData, Format: "Png"}},
	}
	h := mkHierarchy(mod)
	withContainer(h, ic.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		IsWritableFunc:           func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
	}
	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))

	outFile := t.TempDir() + "/logo.png"
	err := execAlterImageCollection(ctx, &ast.AlterImageCollectionStmt{
		Name:    ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{&ast.ExportImageAction{ImageName: "logo", FilePath: outFile}},
	})
	assertNoError(t, err)

	written, readErr := os.ReadFile(outFile)
	if readErr != nil {
		t.Fatalf("output file not written: %v", readErr)
	}
	if string(written) != string(imgData) {
		t.Errorf("exported data mismatch")
	}
}
```

- [ ] **Step 2: 运行测试，确认通过**

```bash
go test ./mdl/executor/ -run "TestAlterImageCollection" -v
```

期望：全部 PASS（除了 TestAlterImageCollection_Add，因为需要实际读文件，在 mock 环境中会报找不到文件——如果 AddImageAction 需要测试，请在 TempDir 创建测试文件，见下方补充）。

**补充 AddImageAction 测试**（需要真实临时文件）：

```go
func TestAlterImageCollection_Add(t *testing.T) {
	mod := mkModule("Mod")
	ic := &types.ImageCollection{
		BaseElement: model.BaseElement{ID: nextID("ic")},
		ContainerID: mod.ID,
		Name:        "Icons",
	}
	h := mkHierarchy(mod)
	withContainer(h, ic.ContainerID, mod.ID)

	// 创建临时 PNG 文件
	tmpFile := t.TempDir() + "/logo.png"
	if err := os.WriteFile(tmpFile, []byte("fakeimgdata"), 0o644); err != nil {
		t.Fatal(err)
	}

	var updatedIC *types.ImageCollection
	mb := &mock.MockBackend{
		IsConnectedFunc:          func() bool { return true },
		IsWritableFunc:           func() bool { return true },
		ListImageCollectionsFunc: func() ([]*types.ImageCollection, error) { return []*types.ImageCollection{ic}, nil },
		UpdateImageCollectionFunc: func(u *types.ImageCollection) error {
			updatedIC = u
			return nil
		},
	}
	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	err := execAlterImageCollection(ctx, &ast.AlterImageCollectionStmt{
		Name:    ast.QualifiedName{Module: "Mod", Name: "Icons"},
		Actions: []ast.ImageCollectionAction{&ast.AddImageAction{ImageName: "logo", FilePath: tmpFile}},
	})
	assertNoError(t, err)
	if updatedIC == nil || len(updatedIC.Images) != 1 || updatedIC.Images[0].Name != "logo" {
		t.Errorf("expected 1 image named logo, got %v", updatedIC)
	}
	assertContainsStr(t, buf.String(), "Added image")
}
```

- [ ] **Step 3: 运行完整 executor 测试套件**

```bash
go test ./mdl/executor/ -v 2>&1 | grep -E "FAIL|PASS|---"
```

期望：无新 FAIL。

- [ ] **Step 4: Commit**

```bash
git add mdl/executor/imagecollections_mock_test.go
git commit -m "test(executor): add full ALTER IMAGE COLLECTION mock test suite"
```

---

## Task 9: MDL 示例脚本

**Files:**
- Create: `mdl-examples/image_collection_alter.mdl`

- [ ] **Step 1: 写示例脚本**

```bash
cat > mdl-examples/image_collection_alter.mdl << 'EOF'
-- Image Collection Maintenance Examples
-- Run with: mxcli -p app.mpr -c "$(cat image_collection_alter.mdl)"

-- Create a collection (with images)
create or modify image collection MyFirstModule.Icons (
    image logo from file './assets/logo.png',
    image banner from file './assets/banner.svg'
);

-- Add a new image to an existing collection
alter image collection MyFirstModule.Icons
    add image spinner from file './assets/spinner.gif';

-- Drop an image
alter image collection MyFirstModule.Icons drop image banner;

-- Rename an image
alter image collection MyFirstModule.Icons rename image logo to logo_v2;

-- Replace an image's content (keep name)
alter image collection MyFirstModule.Icons set image logo_v2 from file './assets/logo_new.png';

-- Multiple actions in one statement
alter image collection MyFirstModule.Icons
    add image bg from file './assets/bg.png',
    drop image spinner;

-- Move the collection to another module
alter image collection MyFirstModule.Icons move to SharedModule.Icons;

-- Export a single image to a local file
alter image collection SharedModule.Icons export image logo_v2 to file './exports/logo_v2.png';

-- Drop the entire collection
-- drop image collection SharedModule.Icons;
EOF
```

- [ ] **Step 2: Syntax check (no project needed)**

```bash
./bin/mxcli check mdl-examples/image_collection_alter.mdl
```

期望：no syntax errors（build first if binary missing: `make build`）。

- [ ] **Step 3: Commit**

```bash
git add mdl-examples/image_collection_alter.mdl
git commit -m "docs(examples): add image_collection_alter.mdl with ALTER IMAGE COLLECTION examples"
```

---

## Task 10: 全量验证

- [ ] **Step 1: 运行全量测试**

```bash
make test 2>&1 | tail -30
```

期望：无 FAIL，覆盖率无明显下降。

- [ ] **Step 2: 运行 make report**

```bash
make report 2>&1 | tail -10
```

期望：无新增 FAIL。

- [ ] **Step 3: 构建 CLI**

```bash
make build
```

- [ ] **Step 4: 快速烟测（需要有测试项目）**

```bash
./bin/mxcli -p testdata/corpus-b/app.mpr -c "show image collections"
./bin/mxcli check mdl-examples/image_collection_alter.mdl
```

期望：正常输出，无崩溃。
