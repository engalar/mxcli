# ISP 修复：PageBackend 拆分为 PageReader + PageWriter

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `PageBackend` 接口（19 方法，混合读写）拆分为 `PageReader`（只读，8 方法）和 `PageWriter`（只写，11 方法），让 lint 规则、catalog builder 等只读消费者能声明最小依赖，不再依赖包含 Create/Update/Delete/Move 的胖接口。

**Architecture:** `PageBackend = PageReader + PageWriter` 保留（向后兼容）。`FullBackend` 继续嵌入 `PageBackend` 不变。`LintReader` 中重复的 `ListPagesGen()` 等 3 个方法改为嵌入 `PageReader`（消除重复）。`CatalogReader` 中的 `ListPagesGen/ListLayoutsGen/ListSnippetsGen` 同样改为嵌入 `PageReader`。未来新增的只读 handler 可以声明接受 `PageReader` 而非完整 `FullBackend`，减少过度授权。没有运行时行为变更。

**Tech Stack:** Go 1.24，`mdl/backend` 包，`mdl/linter` 包，`mdl/catalog` 包。

---

## 影响文件概览

| 文件 | 操作 |
|------|------|
| `mdl/backend/page.go` | 重构：拆分为 `PageReader` + `PageWriter`；`PageBackend` 改为组合接口 |
| `mdl/linter/context.go` | 修改：`LintReader` 中 3 个重复方法改为 `backend.PageReader` 嵌入 |
| `mdl/catalog/builder.go` | 修改：`CatalogReader` 中 3 个重复方法改为 `backend.PageReader` 嵌入 |
| `mdl/backend/mock/mock_backend.go` | 确认：无变更（`MockBackend` 已满足 `PageBackend`，自动满足 `PageReader` + `PageWriter`） |
| `mdl/backend/mpr/backend.go` | 确认：无变更（`MprBackend` 已实现所有方法） |

---

## Task 1：拆分 PageBackend 接口

- [ ] **Step 1.1：先写一个编译期断言测试，确认拆分后 MprBackend 和 MockBackend 满足新接口**

新建 `mdl/backend/page_split_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0
package backend_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/mdl/backend/mpr"
)

// TestPageBackendSplit 是编译期断言——测试通过即证明接口满足正确。
func TestPageBackendSplit(t *testing.T) {
	var _ backend.PageReader = (*mpr.MprBackend)(nil)
	var _ backend.PageWriter = (*mpr.MprBackend)(nil)
	var _ backend.PageBackend = (*mpr.MprBackend)(nil)

	var _ backend.PageReader = (*mock.MockBackend)(nil)
	var _ backend.PageWriter = (*mock.MockBackend)(nil)
	var _ backend.PageBackend = (*mock.MockBackend)(nil)

	t.Log("PageReader + PageWriter + PageBackend all satisfied by MprBackend and MockBackend")
}
```

- [ ] **Step 1.2：运行确认测试当前会因 `PageReader`/`PageWriter` 不存在而编译失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/backend/... -run TestPageBackendSplit 2>&1 | head -10
```

预期：`undefined: backend.PageReader`。

- [ ] **Step 1.3：重写 `mdl/backend/page.go`**

将现有 19 个方法按只读/只写拆分：

```go
// SPDX-License-Identifier: Apache-2.0

package backend

import (
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// PageReader provides read-only access to page, layout, and snippet documents.
// Satisfied by any backend; also embedded in LintReader and CatalogReader
// to eliminate duplicate method declarations.
type PageReader interface {
	ListPagesGen() ([]*genPg.Page, error)
	GetPageGen(id model.ID) (*genPg.Page, error)

	ListLayoutsGen() ([]*genPg.Layout, error)
	GetLayoutGen(id model.ID) (*genPg.Layout, error)

	ListSnippetsGen() ([]*genPg.Snippet, error)
	GetSnippetGen(id model.ID) (*genPg.Snippet, error)

	// GetPageContainerUUID resolves the parent container UUID (folder or module ID)
	// of a Page unit. Gen objects do not carry container IDs; this helper bridges
	// Page-level lint rules and other consumers that need to build qualified names.
	GetPageContainerUUID(id model.ID) (model.ID, error)
}

// PageWriter provides write access to page, layout, and snippet documents.
// Only executors and migration commands should depend on this interface.
type PageWriter interface {
	CreatePageGen(parentUUID, containmentName string, page *genPg.Page) error
	UpdatePageGen(page *genPg.Page) error
	DeletePageGen(id model.ID) error
	MovePageGen(id, containerID model.ID) error

	CreateLayoutGen(parentUUID, containmentName string, layout *genPg.Layout) error
	UpdateLayoutGen(layout *genPg.Layout) error
	DeleteLayoutGen(id model.ID) error
	MoveLayoutGen(id, containerID model.ID) error

	CreateSnippetGen(parentUUID, containmentName string, snippet *genPg.Snippet) error
	UpdateSnippetGen(snippet *genPg.Snippet) error
	DeleteSnippetGen(id model.ID) error
	MoveSnippetGen(id, containerID model.ID) error
}

// PageBackend composes PageReader and PageWriter, providing the full
// page/layout/snippet surface. FullBackend embeds this interface.
// Consumers that only read pages should declare PageReader; consumers
// that only write should declare PageWriter.
type PageBackend interface {
	PageReader
	PageWriter
}
```

- [ ] **Step 1.4：运行编译确认 `mdl/backend` 包正常**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/backend/...
```

预期：无错误（`FullBackend` 嵌入了 `PageBackend`，不受影响）。

- [ ] **Step 1.5：运行测试确认 MprBackend 和 MockBackend 满足新接口**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/backend/... -run TestPageBackendSplit -v
```

预期：`PASS`。

---

## Task 2：更新 LintReader 消除重复方法

- [ ] **Step 2.1：查看 `mdl/linter/context.go` 中重复的页面读方法**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
grep -n "ListPagesGen\|ListLayoutsGen\|ListSnippetsGen\|GetPageContainerUUID" mdl/linter/context.go
```

预期输出：这 3 个（或 4 个）方法的声明行。

- [ ] **Step 2.2：修改 `LintReader` 接口，改为嵌入 `backend.PageReader`**

找到 `LintReader` 中关于页面的重复声明，例如：
```go
type LintReader interface {
    GetMicroflowGen(id model.ID) (*genMf.Microflow, error)
    GetProjectSecurityGen() (*genSec.ProjectSecurity, error)
    GetNavigation() (*types.NavigationDocument, error)
    ListPagesGen() ([]*genPg.Page, error)       // ← 重复
    ListLayoutsGen() ([]*genPg.Layout, error)   // ← 重复
    ListSnippetsGen() ([]*genPg.Snippet, error) // ← 重复
    // ...
}
```

替换为嵌入：
```go
type LintReader interface {
    GetMicroflowGen(id model.ID) (*genMf.Microflow, error)
    GetProjectSecurityGen() (*genSec.ProjectSecurity, error)
    GetNavigation() (*types.NavigationDocument, error)
    backend.PageReader // embeds List+Get for pages, layouts, snippets
    // ...
}
```

在文件 import 块加入：
```go
"github.com/mendixlabs/mxcli/mdl/backend"
```

- [ ] **Step 2.3：编译 linter 包确认**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/linter/...
```

预期：无错误。若有 `import cycle`，说明 `linter` 包已经 import 了 `backend`（应该已有）。

---

## Task 3：更新 CatalogReader 消除重复方法

- [ ] **Step 3.1：查看 `mdl/catalog/builder.go` 中重复的页面读方法**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
grep -n "ListPagesGen\|ListLayoutsGen\|ListSnippetsGen" mdl/catalog/builder.go
```

预期：`CatalogReader` 接口定义中有这 3 行。

- [ ] **Step 3.2：修改 `CatalogReader` 接口**

找到（约行 54-56）：
```go
ListPagesGen() ([]*genPg.Page, error)
ListLayoutsGen() ([]*genPg.Layout, error)
ListSnippetsGen() ([]*genPg.Snippet, error)
```

替换为单行嵌入：
```go
backend.PageReader // ListPagesGen, GetPageGen, ListLayoutsGen, GetLayoutGen, ListSnippetsGen, GetSnippetGen, GetPageContainerUUID
```

在 import 块加入（若未有）：
```go
"github.com/mendixlabs/mxcli/mdl/backend"
```

> **注意**：`backend.PageReader` 嵌入后，`CatalogReader` 会多出 `GetPageGen`、`GetLayoutGen`、`GetSnippetGen`、`GetPageContainerUUID` 这 4 个方法。确认 `MprBackend` 实现了这 4 个方法（它是 `PageReader` 的实现，必然满足）：
>
> ```bash
> grep -n "func.*MprBackend.*GetPageGen\|func.*MprBackend.*GetLayoutGen\|func.*MprBackend.*GetSnippetGen\|func.*MprBackend.*GetPageContainerUUID" mdl/backend/mpr/backend.go
> ```

- [ ] **Step 3.3：编译 catalog 包确认**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/catalog/...
```

预期：无错误。

---

## Task 4：运行全量测试并提交

- [ ] **Step 4.1：全量编译确认**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./...
```

预期：无错误。

- [ ] **Step 4.2：运行相关包测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/backend/... ./mdl/linter/... ./mdl/catalog/... -count=1 2>&1 | tail -15
```

预期：全部 `ok`，无 FAIL。

- [ ] **Step 4.3：确认无其他直接声明了这 3 个方法的 interface（防止遗漏）**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
grep -rn "ListPagesGen\(\) \(\[\]\*genPg" . --include="*.go" | grep -v "_test.go\|mock_page.go\|backend.go\|mpr/backend.go"
```

预期：只有 `mdl/backend/page.go` 一行（`PageReader` 定义）。若其他文件有，同步更改为 `backend.PageReader`。

- [ ] **Step 4.4：commit**

```bash
git add mdl/backend/page.go mdl/backend/page_split_test.go mdl/linter/context.go mdl/catalog/builder.go
git commit -m "$(cat <<'EOF'
refactor(backend): split PageBackend into PageReader + PageWriter (ISP)

PageBackend mixed 19 read and write methods, forcing read-only consumers
(linter, catalog) to depend on create/update/delete/move operations they
never call (ISP violation).

Introduce PageReader (7 read methods) and PageWriter (12 write methods).
PageBackend = PageReader + PageWriter for backward compatibility.
FullBackend is unchanged.

LintReader and CatalogReader drop their duplicate ListPages/Layouts/Snippets
declarations and embed backend.PageReader instead, eliminating 6 lines of
copy-pasted interface surface.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## 自检 Checklist

- [ ] `go build ./...` 无错误
- [ ] `go test ./mdl/backend/... ./mdl/linter/... ./mdl/catalog/...` 无 FAIL
- [ ] `grep -n "PageReader\|PageWriter" mdl/backend/page.go` 输出接口定义行
- [ ] `grep "backend.PageReader" mdl/linter/context.go` 输出嵌入行
- [ ] `grep "backend.PageReader" mdl/catalog/builder.go` 输出嵌入行
- [ ] `grep -rn "ListPagesGen() \(\[\]\*genPg" . --include="*.go" | grep -v "mock_page\|backend.go\|mpr/backend"` 仅剩 `page.go` 一行

## 后续扩展（范围外）

拆分完成后，新增的只读 handler 可以这样声明，进一步减少过度授权：
```go
// 只读 handler — 明确声明只需要读权限
func listPagesInLintContext(r backend.PageReader, module string) error { ... }
```
这是后续 handler 签名迁移的基础，超出本次计划范围。
