# Image Collection CRUD + TUI 图片显示修复 设计文档

**日期**: 2026-06-10  
**状态**: 已批准

---

## 背景

两个独立问题：

1. **TUI Bug**：`describe image collection` 在终端预览中无法显示图片。根因：`extractImagePaths()` 搜索大写字符串 `"FROM FILE '"`，但 commit `f70a74158` 已将 MDL 输出改为小写关键字，导致路径提取失败。

2. **MDL CRUD 缺失**：现有命令只能整体创建/删除 image collection。无法单独对集合内的图片进行增删改查，使用体验不完整。

---

## 第一部分：TUI Bug 修复

**文件**：`cmd/mxcli/tui/image_render.go`

**改动**：在 `extractImagePaths()` 中，对每行做大小写不敏感搜索：

```go
// 修复前：
idx := strings.Index(line, "FROM FILE '")
// 修复后：
idx := strings.Index(strings.ToUpper(line), "FROM FILE '")
```

同样处理 `findImagePathAtClick()` 中的同一字符串（`miller.go:921`）。

**测试**：新增 L1 单元测试：`extractImagePaths` 对小写 `from file` 输出能正确提取路径。

---

## 第二部分：ALTER IMAGE COLLECTION

### 目标语法（方案 A）

```mdl
-- 添加一张或多张图片
ALTER IMAGE COLLECTION MyFirstModule.Icons
  ADD IMAGE logo FROM FILE './assets/logo.png',
  ADD IMAGE banner FROM FILE './assets/banner.svg';

-- 删除一张图片
ALTER IMAGE COLLECTION MyFirstModule.Icons DROP IMAGE logo;

-- 重命名一张图片（保持数据不变）
ALTER IMAGE COLLECTION MyFirstModule.Icons RENAME IMAGE logo TO logo_v2;

-- 替换一张图片的文件内容（保持名称）
ALTER IMAGE COLLECTION MyFirstModule.Icons SET IMAGE logo FROM FILE './assets/logo_new.png';

-- 移动整个集合到另一模块
ALTER IMAGE COLLECTION MyFirstModule.Icons MOVE TO OtherModule.Icons;

-- 导出一张图片到本地文件
ALTER IMAGE COLLECTION MyFirstModule.Icons EXPORT IMAGE logo TO FILE './out/logo.png';
```

### Grammar（`MDLParser.g4`）

在 `alterStatement` 中新增：

```antlr
| ALTER IMAGE COLLECTION qualifiedName alterImageCollectionAction+ SEMICOLON?
```

新规则 `alterImageCollectionAction`：

```antlr
alterImageCollectionAction
    : ADD IMAGE imageName FROM FILE_KW STRING_LITERAL
    | DROP IMAGE imageName
    | RENAME IMAGE imageName TO imageName
    | SET IMAGE imageName FROM FILE_KW STRING_LITERAL
    | MOVE TO qualifiedName
    | EXPORT IMAGE imageName TO FILE_KW STRING_LITERAL
    ;
```

所有关键字（ADD、DROP、RENAME、SET、MOVE、EXPORT、TO、FROM、FILE_KW）均为现有 token，无需新增。

### AST（`mdl/ast/ast_imagecollection.go`）

```go
// AlterImageCollectionStmt represents ALTER IMAGE COLLECTION Module.Name action+
type AlterImageCollectionStmt struct {
    Name    QualifiedName
    Actions []ImageCollectionAction
}

func (s *AlterImageCollectionStmt) isStatement() {}

type ImageCollectionAction interface{ isImageCollectionAction() }

type AddImageAction         struct{ ImageName, FilePath string }
type DropImageAction        struct{ ImageName string }
type RenameImageAction      struct{ From, To string }
type SetImageAction         struct{ ImageName, FilePath string }
type MoveImageCollectionAction struct{ Target QualifiedName }
type ExportImageAction      struct{ ImageName, FilePath string }

func (a *AddImageAction)          isImageCollectionAction() {}
func (a *DropImageAction)         isImageCollectionAction() {}
func (a *RenameImageAction)       isImageCollectionAction() {}
func (a *SetImageAction)          isImageCollectionAction() {}
func (a *MoveImageCollectionAction) isImageCollectionAction() {}
func (a *ExportImageAction)       isImageCollectionAction() {}
```

### Visitor（`mdl/visitor/visitor_imagecollection.go`）

新增 `ExitAlterImageCollectionStatement`，遍历 `alterImageCollectionAction` 子节点，对每种 alt 分支构建对应 action 并追加到 `AlterImageCollectionStmt.Actions`。

### Backend 接口（`mdl/backend/infrastructure.go`）

在 `ImageCollectionBackend` 接口中新增：

```go
MoveImageCollection(ic *types.ImageCollection) error
```

其他操作（ADD/DROP/RENAME/SET）复用现有 `UpdateImageCollection`；EXPORT 是纯 OS 文件写入，不需要新 backend 方法。

### Backend MPR 实现（`mdl/backend/mpr/`）

```go
// update_services.go 或 backend.go
func (b *MprBackend) MoveImageCollection(ic *types.ImageCollection) error {
    if b.msdkWriter == nil {
        return fmt.Errorf("modelsdk writer not initialized")
    }
    return b.msdkWriter.UpdateUnitContainer(string(ic.ID), string(ic.ContainerID))
}
```

复用 `MoveEnumeration`/`MoveConstant` 的同一模式。

### Mock（`mdl/backend/mock/`）

新增：

```go
MoveImageCollectionFunc func(ic *types.ImageCollection) error
```

默认返回 `"MockBackend.MoveImageCollection not configured"` 错误。

### Executor（`mdl/executor/cmd_imagecollections.go`）

新增 `execAlterImageCollection(ctx, stmt)` 函数：

| 动作 | 逻辑 |
|------|------|
| `AddImageAction` | 读文件 → `ic.Images = append(ic.Images, Image{...})` → `UpdateImageCollection` |
| `DropImageAction` | 过滤掉同名 image → `UpdateImageCollection` |
| `RenameImageAction` | 找到旧名 image，改 `Name` 字段 → `UpdateImageCollection` |
| `SetImageAction` | 找到同名 image，替换 `Data`/`Format` → `UpdateImageCollection` |
| `MoveImageCollectionAction` | 解析目标 module ID → 更新 `ic.ContainerID` → `MoveImageCollection` |
| `ExportImageAction` | 找到 image.Data → `os.WriteFile(filePath, data, 0o644)` |

注意：多个动作在同一 ALTER 语句中时，先读一次集合，依次应用所有动作，最后调用一次 `UpdateImageCollection`（批量 ADD 场景下避免多次写入）。

### 错误处理

| 场景 | 错误 |
|------|------|
| 集合不存在 | `mdlerrors.NewNotFound("image collection", name)` |
| 动作目标 image 不存在 | `mdlerrors.NewNotFound("image", imageName)` |
| RENAME 目标名已存在 | `mdlerrors.NewAlreadyExists("image", toName)` |
| 文件读取失败 | `mdlerrors.NewBackend("read image file", err)` |
| 文件写入失败（EXPORT） | `mdlerrors.NewBackend("write image file", err)` |
| 目标 module 不存在（MOVE） | `mdlerrors.NewNotFound("module", targetModule)` |
| 未连接写 | `mdlerrors.NewNotConnected()` |

### 注册（`mdl/executor/register_stubs.go`）

```go
r.Register(&ast.AlterImageCollectionStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
    return execAlterImageCollection(ctx, stmt.(*ast.AlterImageCollectionStmt))
})
```

---

## 测试计划

| 测试 | 类型 |
|------|------|
| `extractImagePaths` 小写 `from file` 能提取路径 | L1 单元 |
| `findImagePathAtClick` 小写 `from file` 能映射路径 | L1 单元 |
| ADD IMAGE 正常路径 | L3 happy |
| DROP IMAGE 正常路径 | L3 happy |
| RENAME IMAGE 正常路径 | L3 happy |
| SET IMAGE 正常路径 | L3 happy |
| MOVE TO 正常路径 | L3 happy |
| EXPORT IMAGE 正常路径 | L3 happy |
| 集合不存在 → 报错 | L3 error |
| RENAME 目标名已存在 → 报错 | L3 error |
| DROP 不存在的 image → 报错 | L3 error |
| 未连接写 → 报错 | L3 error |

MDL 示例脚本添加到 `mdl-examples/`。

---

## 变更文件清单

```
cmd/mxcli/tui/image_render.go        # TUI bug fix
cmd/mxcli/tui/miller.go              # TUI bug fix（findImagePathAtClick）
mdl/grammar/MDLParser.g4             # 新增 alterImageCollectionAction 规则
mdl/grammar/parser/                  # make grammar 重新生成
mdl/ast/ast_imagecollection.go       # AlterImageCollectionStmt + 6 个 action 类型
mdl/visitor/visitor_imagecollection.go # ExitAlterImageCollectionStatement
mdl/backend/infrastructure.go        # MoveImageCollection 接口方法
mdl/backend/mpr/backend.go           # MoveImageCollection 路由
mdl/backend/mpr/update_services.go   # moveImageCollectionViaModelsdk
mdl/backend/mock/                    # MoveImageCollectionFunc stub
mdl/executor/cmd_imagecollections.go # execAlterImageCollection
mdl/executor/register_stubs.go       # 注册新语句
mdl/executor/imagecollections_mock_test.go # 新测试
cmd/mxcli/tui/image_render_test.go   # TUI bug 回归测试（若存在）
mdl-examples/                        # MDL 示例脚本
```
