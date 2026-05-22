# HelpDesk Regression Baseline — BSON + MDL Roundtrip Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 执行 `helpdesk-app.mdl` 两次（代码修改前后），在 BSON 层和 MDL describe 层对比两次执行结果，任何字段级退化都会使 CI 失败。

**Architecture:** 利用已有的 `internal/goldenfs`（FUSE copy-on-write overlay）+ `internal/bsoncompare`（字段级 BSON 对比）。空白 MPR（A = `testdata/expr-checker/minimal.mpr`）通过 FUSE 挂载，执行 helpdesk-app.mdl 后得到 B2（内存中）；同时维护已 commit 的 B1 golden（`testdata/helpdesk-golden/`）。测试比较 B1 vs B2：空 diff = 无退化。

**Tech Stack:** Go 1.22+、`internal/bsoncompare`（已实现）、`internal/goldenfs`（已实现，Linux FUSE）、`go-difflib`（已有依赖）、`//go:build linux && integration` tag

---

## 关键约束

- **A（空白基准）**：`testdata/expr-checker/minimal.mpr`（已存在，v2 格式，Mendix 11.6.6）
- **B1 golden**：`testdata/helpdesk-golden/`（新目录，首次 make 后 commit）—— 是 FUSE mount 目录的完整拷贝（含 `minimal.mpr` + `mprcontents/`）
- **B2**：每次测试从 A 通过 FUSE 执行 helpdesk-app.mdl 得到，在内存中
- **比较**：`bsoncompare.Compare(B1/minimal.mpr, snap.MountDir()/minimal.mpr)` → 期望空 diff
- **goldenfs.Commit() 不能用**：它写回 baseDir（A），会污染 expr-checker；必须用 `copyDir` 写到独立的 `helpdesk-golden/`
- **bsoncompare.AssertEqual / Compare** 的路径参数是 `.mpr` 文件路径（不是目录）
- `runMDL(t, mprPath, script)` 已在 `bsoncompare_integration_test.go` 中定义，本包内直接复用

---

## 文件清单

| 操作 | 路径 | 说明 |
|------|------|------|
| 新建 | `internal/goldenfs/helpdesk_regression_test.go` | 所有测试 + 辅助函数 |
| 新建 | `testdata/helpdesk-golden/` | B1 golden 目录（Task 3 生成后 commit） |
| 修改 | `Makefile` | 新增两个 target |

不需要新的生产代码；所有基础设施已就绪。

---

## Task 1：写辅助函数 + 目录定位（编译验证）

**Files:**
- Create: `internal/goldenfs/helpdesk_regression_test.go`

- [ ] **Step 1.1：写文件骨架和辅助函数**

```go
// internal/goldenfs/helpdesk_regression_test.go

//go:build linux && integration

package goldenfs

import (
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden: go test ... -update-golden -run TestHelpdeskGolden_Update
var updateGolden = flag.Bool("update-golden", false,
	"overwrite testdata/helpdesk-golden/ with the current MDL execution result")

// helpdeskBlankMPR returns the path to the blank base MPR (A).
// Uses expr-checker/minimal.mpr which is already committed and v2-format.
func helpdeskBlankMPR(t *testing.T) string {
	t.Helper()
	p := filepath.Join(repoRoot(t), "testdata", "expr-checker", "minimal.mpr")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("testdata/expr-checker/minimal.mpr not found: %v", err)
	}
	return p
}

// helpdeskBlankDir returns the directory containing the blank base MPR.
func helpdeskBlankDir(t *testing.T) string {
	t.Helper()
	return filepath.Dir(helpdeskBlankMPR(t))
}

// helpdeskGoldenDir returns the path to the committed B1 golden directory.
func helpdeskGoldenDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "testdata", "helpdesk-golden")
}

// helpdeskGoldenMPR returns the MPR path inside the golden directory.
func helpdeskGoldenMPR(t *testing.T) string {
	t.Helper()
	return filepath.Join(helpdeskGoldenDir(t), "minimal.mpr")
}

// helpdeskMDLScript reads mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
// and returns its content as a string.
func helpdeskMDLScript(t *testing.T) string {
	t.Helper()
	p := filepath.Join(repoRoot(t), "mdl-examples", "use-cases", "helpdesk", "helpdesk-app.mdl")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read helpdesk-app.mdl: %v", err)
	}
	return string(data)
}

// copyDir copies src directory tree to dst, creating dst if needed.
// Overwrites existing files in dst.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(dstPath, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, data, 0o644)
	})
}
```

- [ ] **Step 1.2：编译验证（只构建，不运行）**

```bash
CGO_ENABLED=0 go build -tags "linux integration" ./internal/goldenfs/...
```

期望：无报错

- [ ] **Step 1.3：commit**

```bash
git add internal/goldenfs/helpdesk_regression_test.go
git commit -m "test(goldenfs): add helpdesk regression test skeleton and helpers"
```

---

## Task 2：写 update-golden 函数（生成 B1）

**Files:**
- Modify: `internal/goldenfs/helpdesk_regression_test.go`

这个函数不是真正的"测试"——它是一个只在 `-update-golden` flag 下运行的 golden 生成器。

- [ ] **Step 2.1：在 Task 1 的文件末尾追加 update 函数**

```go
// TestHelpdeskGolden_Update 生成或更新 testdata/helpdesk-golden/。
// 只在 -update-golden flag 存在时有效；否则直接 Skip。
// 运行方式：
//   go test ./internal/goldenfs/ -tags linux,integration \
//          -run TestHelpdeskGolden_Update -update-golden -v
func TestHelpdeskGolden_Update(t *testing.T) {
	if !*updateGolden {
		t.Skip("pass -update-golden to regenerate testdata/helpdesk-golden/")
	}

	blankDir := helpdeskBlankDir(t)
	blankMPRName := "minimal.mpr"
	goldenDir := helpdeskGoldenDir(t)
	script := helpdeskMDLScript(t)

	// Open FUSE overlay on top of blank A.
	snap, err := Open(blankDir)
	if err != nil {
		t.Fatalf("goldenfs.Open: %v", err)
	}
	defer func() {
		if err := snap.Close(); err != nil {
			t.Logf("snap.Close: %v", err)
		}
	}()

	mountMPR := filepath.Join(snap.MountDir(), blankMPRName)

	// Execute helpdesk-app.mdl — produces B2 in the FUSE overlay.
	runMDL(t, mountMPR, script)

	// Copy entire FUSE mount (A + dirty layer = B2) to testdata/helpdesk-golden/.
	// NOTE: we do NOT call snap.Commit() — that would write back to blankDir (A).
	if err := os.RemoveAll(goldenDir); err != nil {
		t.Fatalf("remove old golden: %v", err)
	}
	if err := copyDir(snap.MountDir(), goldenDir); err != nil {
		t.Fatalf("copy to golden: %v", err)
	}

	t.Logf("Golden updated: %s", goldenDir)
	t.Logf("Next step: git add testdata/helpdesk-golden/ && git commit -m 'update: helpdesk golden baseline'")

	// Do NOT snap.Rollback() here — we already copied, and we want the FUSE
	// overlay to stay consistent through snap.Close().
}
```

- [ ] **Step 2.2：编译验证**

```bash
CGO_ENABLED=0 go build -tags "linux integration" ./internal/goldenfs/...
```

期望：无报错

- [ ] **Step 2.3：commit**

```bash
git add internal/goldenfs/helpdesk_regression_test.go
git commit -m "test(goldenfs): add TestHelpdeskGolden_Update — generates B1 golden"
```

---

## Task 3：运行 update-golden，生成并 commit B1

这一步在本地执行，产物是 `testdata/helpdesk-golden/` 目录。

- [ ] **Step 3.1：运行 update-golden**

```bash
CGO_ENABLED=0 go test ./internal/goldenfs/ \
  -tags linux,integration \
  -run TestHelpdeskGolden_Update \
  -update-golden \
  -v \
  -timeout 5m
```

期望输出（末尾）：
```
=== RUN   TestHelpdeskGolden_Update
    helpdesk_regression_test.go:XX: Golden updated: .../testdata/helpdesk-golden
    helpdesk_regression_test.go:XX: Next step: git add testdata/helpdesk-golden/ && ...
--- PASS: TestHelpdeskGolden_Update (Xs)
```

如果有 MDL 执行错误，先修复 `helpdesk-app.mdl` 中对应的语法（记住未实现特性已经用注释占位，不会失败）。

- [ ] **Step 3.2：验证目录已生成**

```bash
ls testdata/helpdesk-golden/
# 期望：minimal.mpr  mprcontents/  以及其他项目文件
ls testdata/helpdesk-golden/mprcontents/ | wc -l
# 期望：> 50（expr-checker 本身约 50+ units，helpdesk 新增约 30-40 units）
```

- [ ] **Step 3.3：用 mx check 验证 B1 golden 的 BSON 有效性**

```bash
~/.mxcli/mxbuild/11.6.6/modeler/mx check \
  testdata/helpdesk-golden/minimal.mpr 2>&1 \
  | grep -i "StorageLoadException\|TypeCacheUnknown\|InvalidOperation\|CE0066\|CE0463"
```

期望：**空输出**（没有任何这些错误）

如果有错误：根据错误类型修复 `helpdesk-app.mdl`，重新运行 Step 3.1。

- [ ] **Step 3.4：commit golden**

```bash
git add testdata/helpdesk-golden/
git commit -m "test(golden): add helpdesk baseline golden (B1)

Generated by TestHelpdeskGolden_Update from testdata/expr-checker/minimal.mpr
+ mdl-examples/use-cases/helpdesk/helpdesk-app.mdl.
mx check passes with zero StorageLoadException / TypeCacheUnknown errors."
```

---

## Task 4：写 BSON 层回归测试（主测试）

**Files:**
- Modify: `internal/goldenfs/helpdesk_regression_test.go`

- [ ] **Step 4.1：在文件末尾追加 BSON 回归测试**

```go
// TestHelpdeskGolden_Regression_BSON 是主回归测试。
// 从空白 A（expr-checker）出发，通过 FUSE 执行 helpdesk-app.mdl 得到 B2，
// 然后与 B1 golden 做字段级 BSON 对比。
// 任何字段缺失、字段值变化、单元增减都会导致测试失败。
//
// 运行：go test ./internal/goldenfs/ -tags linux,integration -run TestHelpdeskGolden_Regression_BSON -v
func TestHelpdeskGolden_Regression_BSON(t *testing.T) {
	goldenMPR := helpdeskGoldenMPR(t)
	if _, err := os.Stat(goldenMPR); err != nil {
		t.Fatalf("B1 golden not found at %s — run: make update-helpdesk-golden", goldenMPR)
	}

	blankDir := helpdeskBlankDir(t)
	script := helpdeskMDLScript(t)

	snap, err := Open(blankDir)
	if err != nil {
		t.Fatalf("goldenfs.Open: %v", err)
	}
	defer func() {
		snap.Rollback()
		if err := snap.Close(); err != nil {
			t.Logf("snap.Close: %v", err)
		}
	}()

	mountMPR := filepath.Join(snap.MountDir(), "minimal.mpr")

	// Execute helpdesk-app.mdl → B2 in FUSE.
	runMDL(t, mountMPR, script)

	// BSON comparison: B1 (golden) vs B2 (current run).
	// Expected: empty diff — no structural regressions.
	bsoncompare.AssertEqual(t,
		goldenMPR,  // A = B1 golden
		mountMPR,   // B = B2 current
		bsoncompare.DefaultOptions(),
		bsoncompare.ExpectNoOtherChanges(),
	)
}
```

- [ ] **Step 4.2：补充 import**

在文件顶部的 import 块中添加 bsoncompare：

```go
import (
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)
```

- [ ] **Step 4.3：编译验证**

```bash
CGO_ENABLED=0 go build -tags "linux integration" ./internal/goldenfs/...
```

期望：无报错

- [ ] **Step 4.4：运行回归测试，期望通过**

```bash
CGO_ENABLED=0 go test ./internal/goldenfs/ \
  -tags linux,integration \
  -run TestHelpdeskGolden_Regression_BSON \
  -v -timeout 10m
```

期望输出：
```
=== RUN   TestHelpdeskGolden_Regression_BSON
--- PASS: TestHelpdeskGolden_Regression_BSON (Xs)
```

如果有 diff 输出（`[CHANGED]` / `[ADDED]` / `[REMOVED]`），说明 MDL 执行结果与 golden 不一致。有两种情况：
1. **预期外的差异**：修复 executor 代码中的 BSON 写入 bug
2. **预期内的差异**（helpdesk-app.mdl 刚被更新）：重新运行 `make update-helpdesk-golden`

- [ ] **Step 4.5：commit**

```bash
git add internal/goldenfs/helpdesk_regression_test.go
git commit -m "test(goldenfs): add TestHelpdeskGolden_Regression_BSON — BSON layer"
```

---

## Task 5：写 describe-MDL 层回归测试（第二层）

**Files:**
- Modify: `internal/goldenfs/helpdesk_regression_test.go`

这一层独立于 FUSE/bsoncompare，直接对比两个 MPR 的 `describe` 输出文本。信号质量更低（不覆盖 describe 未输出的字段），但人类可读，方便定位问题。

- [ ] **Step 5.1：追加 describe 辅助函数**

```go
// describeMDL 对 mprPath 执行一组 describe/show 命令，返回合并的文本输出。
// 覆盖 helpdesk-app.mdl 中创建的主要文档类型。
func describeMDL(t *testing.T, mprPath string) string {
	t.Helper()
	var buf strings.Builder
	e := executor.New(&buf)
	e.SetQuiet(true)
	e.SetBackendFactory(func() backend.FullBackend { return mprbackend.New() })
	defer func() {
		if err := e.Close(); err != nil {
			t.Logf("executor close: %v", err)
		}
	}()

	script := fmt.Sprintf(`connect local '%s';
-- Domain model
describe entity HelpDesk.Ticket;
describe entity HelpDesk.Customer;
describe entity HelpDesk.Agent;
describe entity HelpDesk.EscalationRequest;
describe entity KnowledgeBase.Article;
describe entity KnowledgeBase.Category;
-- Enumerations
describe enumeration HelpDesk.TicketStatus;
describe enumeration HelpDesk.TicketPriority;
describe enumeration KnowledgeBase.ArticleStatus;
-- Microflows (status machine)
describe microflow HelpDesk.ACT_Ticket_Submit;
describe microflow HelpDesk.ACT_Ticket_Resolve;
describe microflow HelpDesk.ACT_Ticket_Reopen;
describe microflow KnowledgeBase.ACT_Article_Publish;
-- Security
show module roles in HelpDesk;
show module roles in KnowledgeBase;
show user roles;
`, mprPath)

	prog, errs := visitor.Build(script)
	if len(errs) > 0 {
		t.Fatalf("describeMDL parse: %v", errs)
	}
	if err := e.ExecuteProgram(prog); err != nil {
		t.Logf("describeMDL output so far:\n%s", buf.String())
		t.Fatalf("describeMDL exec: %v", err)
	}
	return buf.String()
}
```

- [ ] **Step 5.2：追加第二层测试**

```go
// TestHelpdeskGolden_Regression_DescribeMDL は describe 出力の文字列比較层。
// B1 golden の describe 出力と B2 の describe 出力を diff する。
// BSON 层が通っても describe が崩れる退化を捕捉する。
//
// 运行：go test ./internal/goldenfs/ -tags linux,integration -run TestHelpdeskGolden_Regression_DescribeMDL -v
func TestHelpdeskGolden_Regression_DescribeMDL(t *testing.T) {
	goldenMPR := helpdeskGoldenMPR(t)
	if _, err := os.Stat(goldenMPR); err != nil {
		t.Fatalf("B1 golden not found at %s — run: make update-helpdesk-golden", goldenMPR)
	}

	blankDir := helpdeskBlankDir(t)
	script := helpdeskMDLScript(t)

	snap, err := Open(blankDir)
	if err != nil {
		t.Fatalf("goldenfs.Open: %v", err)
	}
	defer func() {
		snap.Rollback()
		if err := snap.Close(); err != nil {
			t.Logf("snap.Close: %v", err)
		}
	}()

	mountMPR := filepath.Join(snap.MountDir(), "minimal.mpr")
	runMDL(t, mountMPR, script)

	// Describe outputs from B1 (golden) and B2 (current).
	b1Desc := describeMDL(t, goldenMPR)
	b2Desc := describeMDL(t, mountMPR)

	if b1Desc == b2Desc {
		return // perfect match
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(b1Desc),
		B:        difflib.SplitLines(b2Desc),
		FromFile: "golden (B1)",
		ToFile:   "current (B2)",
		Context:  3,
	}
	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	t.Errorf("describe MDL roundtrip mismatch:\n%s", text)
}
```

- [ ] **Step 5.3：补充 import**

```go
import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
	"github.com/mendixlabs/mxcli/mdl/backend"
	mprbackend "github.com/mendixlabs/mxcli/mdl/backend/mpr"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/pmezard/go-difflib/difflib"
)
```

- [ ] **Step 5.4：编译验证**

```bash
CGO_ENABLED=0 go build -tags "linux integration" ./internal/goldenfs/...
```

期望：无报错

- [ ] **Step 5.5：运行 describe 回归测试**

```bash
CGO_ENABLED=0 go test ./internal/goldenfs/ \
  -tags linux,integration \
  -run TestHelpdeskGolden_Regression_DescribeMDL \
  -v -timeout 10m
```

期望：
```
--- PASS: TestHelpdeskGolden_Regression_DescribeMDL (Xs)
```

如果失败，检查 diff 输出中是哪个 describe 命令的输出不一致，确认是 executor bug 还是 helpdesk-app.mdl 变更导致。

- [ ] **Step 5.6：commit**

```bash
git add internal/goldenfs/helpdesk_regression_test.go
git commit -m "test(goldenfs): add TestHelpdeskGolden_Regression_DescribeMDL — describe layer"
```

---

## Task 6：Makefile targets

**Files:**
- Modify: `Makefile`

- [ ] **Step 6.1：在 Makefile 的 `test-integration` target 附近添加两个新 target**

在 `test-integration:` 行之后插入：

```makefile
# Regenerate testdata/helpdesk-golden/ from helpdesk-app.mdl.
# Run after intentional changes to helpdesk-app.mdl; then commit the result.
update-helpdesk-golden:
	CGO_ENABLED=0 go test ./internal/goldenfs/ \
		-tags linux,integration \
		-run TestHelpdeskGolden_Update \
		-update-golden \
		-v -timeout 10m

# Run both helpdesk regression layers (BSON + describe MDL).
# Requires testdata/helpdesk-golden/ to exist (run update-helpdesk-golden first).
test-helpdesk-regression:
	CGO_ENABLED=0 go test ./internal/goldenfs/ \
		-tags linux,integration \
		-run 'TestHelpdeskGolden_Regression' \
		-v -timeout 15m
```

- [ ] **Step 6.2：同步 .PHONY**

在现有 `.PHONY` 行末尾追加：

```makefile
.PHONY: ... update-helpdesk-golden test-helpdesk-regression
```

- [ ] **Step 6.3：验证 make 目标可解析**

```bash
make -n update-helpdesk-golden
make -n test-helpdesk-regression
```

期望：打印命令但不执行（`-n` = dry run），无 make 语法报错

- [ ] **Step 6.4：端到端冒烟：通过 make 运行回归测试**

```bash
make test-helpdesk-regression
```

期望：两个测试均 PASS，exit 0

- [ ] **Step 6.5：commit**

```bash
git add Makefile
git commit -m "build: add update-helpdesk-golden and test-helpdesk-regression make targets"
```

---

## 自检（spec coverage）

| 需求 | 覆盖 task |
|------|----------|
| 从空白 A 执行 MDL 得到 B2 | Task 4（FUSE + runMDL） |
| B1 golden committed | Task 3 |
| BSON 层字段级对比 | Task 4（bsoncompare.AssertEqual） |
| describe MDL 层文本对比 | Task 5 |
| 更新 golden 的命令 | Task 3 + Task 6（make update-helpdesk-golden） |
| CI 可运行 | Task 6（make test-helpdesk-regression） |
| mx check 验证 golden 有效性 | Task 3 Step 3.3 |

---

## 回归触发流程（写给工程师的操作手册）

**日常 CI（不改 helpdesk-app.mdl）：**
```bash
make test-helpdesk-regression
# 两层均 PASS = 无退化
```

**意图修改 helpdesk-app.mdl 后：**
```bash
# 1. 修改 helpdesk-app.mdl
# 2. 刷新 golden
make update-helpdesk-golden
# 3. 验证 mx check 通过（golden 有效）
~/.mxcli/mxbuild/11.6.6/modeler/mx check testdata/helpdesk-golden/minimal.mpr \
  2>&1 | grep -i "StorageLoadException\|TypeCacheUnknown\|CE0066\|CE0463"
# 4. review diff（确认 golden 变化符合预期）
git diff testdata/helpdesk-golden/
# 5. commit
git add testdata/helpdesk-golden/ && git commit -m "update: helpdesk golden baseline"
# 6. 回归测试再次通过
make test-helpdesk-regression
```

**代码 BSON 写路径退化时的表现：**
- BSON 层：`[CHANGED] HelpDesk.Ticket — .ObjectCollection[...].Attribute[X].Type: "Date" → "String"`
- describe 层：unified diff 显示 `describe entity HelpDesk.Ticket` 输出不同
- 两层均会失败，提供互补的诊断信息
