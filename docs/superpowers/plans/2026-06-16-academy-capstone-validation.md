# Academy Capstone E2E Validation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 创建一个本地 Bash 脚本，端到端验证 `academy/zh/capstone-helpdesk/参考实现/` 的 8 个 MDL 参考文件，运行 Studio Pro 级别的 BSON 校验，并构建+启动应用供人工验证。

**Architecture:** 三个文件协作：`scripts/mx-path/main.go` 通过 `docker.CachedMxPath()` 解析 mx 二进制路径；`scripts/validate-academy-capstone.sh` 编排全流程（创建项目、批量 exec、mx check、local build、local run）；Makefile target 提供入口。所有 mxcli 调用通过 `go run`，无需安装已编译的二进制。

**Tech Stack:** Bash (`#!/usr/bin/env bash`, `set -euo pipefail`)、Go（`go run`）、`scripts/lib/mx-check.sh`（已有）、`cmd/mxcli/docker.CachedMxPath`、`cmd/mxcli-local`（build/run 子命令）。

**Spec:** `docs/superpowers/specs/2026-06-16-academy-capstone-validation-design.md`

---

## File Map

| 文件 | 操作 | 职责 |
|------|------|------|
| `scripts/mx-path/main.go` | 新建 | 通过 `docker.CachedMxPath(version)` 输出 mx 二进制绝对路径 |
| `scripts/validate-academy-capstone.sh` | 新建 | 主验证脚本：new → exec → mx check → local build → local run |
| `Makefile` | 修改（追加 target + .PHONY） | 提供 `make validate-academy-capstone` 入口 |

> `.gitignore` 已包含 `HelpDeskE2E/`（第 87 行），无需修改。

---

## Task 1: Go helper — mx 二进制路径解析

**Files:**
- Create: `scripts/mx-path/main.go`

- [ ] **Step 1: 创建文件**

```go
// scripts/mx-path/main.go
// 通过 docker.CachedMxPath() 解析本机缓存的 mx 二进制路径。
// 用法：go run ./scripts/mx-path/main.go <version>
// 成功时向 stdout 输出绝对路径（无换行），失败时 exit 1/2。
package main

import (
	"fmt"
	"os"

	"github.com/mendixlabs/mxcli/cmd/mxcli/docker"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/mx-path/main.go <version>")
		os.Exit(2)
	}
	version := os.Args[1]
	path := docker.CachedMxPath(version)
	if path == "" {
		fmt.Fprintf(os.Stderr,
			"ERROR: mx %s not cached\n  Run: go run ./cmd/mxcli setup mxbuild --version %s\n",
			version, version)
		os.Exit(1)
	}
	fmt.Print(path)
}
```

- [ ] **Step 2: 验证编译无错误**

```bash
cd D:/gh/mxcli
go vet ./scripts/mx-path/
```

期望输出：无输出（无错误）。

- [ ] **Step 3: 验证参数校验路径**

```bash
go run ./scripts/mx-path/main.go 2>&1 || true
```

期望输出：
```
usage: go run ./scripts/mx-path/main.go <version>
```

- [ ] **Step 4: 验证版本未缓存时的错误提示**

```bash
go run ./scripts/mx-path/main.go 0.0.0 2>&1 || true
```

期望输出：
```
ERROR: mx 0.0.0 not cached
  Run: go run ./cmd/mxcli setup mxbuild --version 0.0.0
```

- [ ] **Step 5: 验证已缓存版本返回路径（需本机有 11.6.6）**

```bash
go run ./scripts/mx-path/main.go 11.6.6
```

期望输出（示例，路径因平台而异）：
```
/home/user/.mxcli/mxbuild/11.6.6/modeler/mx
```

若本机无 11.6.6，跳过此步，继续 Task 2。

- [ ] **Step 6: 提交**

```bash
git add scripts/mx-path/main.go
git commit -m "feat(academy): add mx-path go helper for environment-aware mx binary discovery"
```

---

## Task 2: 主验证脚本

**Files:**
- Create: `scripts/validate-academy-capstone.sh`

- [ ] **Step 1: 创建脚本文件**

```bash
#!/usr/bin/env bash
# scripts/validate-academy-capstone.sh
# End-to-end validation of academy/zh/capstone-helpdesk reference implementation.
#
# Flow:
#   1. Create fresh HelpDeskE2E project via mxcli new
#   2. Batch-exec all 8 capstone MDL files
#   3. Run mx check (Studio Pro-level BSON validation, baseline=0)
#   4. Build PAD package via mxcli-local build
#   5. Start runtime for human validation (blocks until Ctrl+C)
#
# Overrides:
#   MXCLI=path/to/mxcli         use compiled binary instead of go run ./cmd/mxcli
#   MXCLI_LOCAL=path/to/local   use compiled binary instead of go run ./cmd/mxcli-local
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
. "$SCRIPT_DIR/lib/mx-check.sh"

MX_VERSION="11.6.6"
PROJECT_NAME="HelpDeskE2E"
MPR="$REPO_ROOT/$PROJECT_NAME/$PROJECT_NAME.mpr"
CAPSTONE_DIR="$REPO_ROOT/academy/zh/capstone-helpdesk/参考实现"
BASELINE=""

# ── helpers ────────────────────────────────────────────────────────────────

# daemon-routed commands: new, exec, setup mxbuild
run_mxcli() {
    if [ -n "${MXCLI:-}" ]; then
        "$MXCLI" "$@"
    else
        (cd "$REPO_ROOT" && go run ./cmd/mxcli "$@")
    fi
}

# local runtime commands: build, run — bypasses launcher entirely
run_mxcli_local() {
    if [ -n "${MXCLI_LOCAL:-}" ]; then
        "$MXCLI_LOCAL" "$@"
    else
        (cd "$REPO_ROOT" && go run ./cmd/mxcli-local "$@")
    fi
}

cleanup() {
    [ -n "$BASELINE" ] && rm -f "$BASELINE"
    echo
    echo "Project preserved at: $REPO_ROOT/$PROJECT_NAME"
}
trap cleanup EXIT

# ── step 1: fresh project ──────────────────────────────────────────────────

echo "=== validate-academy-capstone ==="
echo "  creating project $PROJECT_NAME ($MX_VERSION)..."
rm -rf "$REPO_ROOT/$PROJECT_NAME"
(cd "$REPO_ROOT" && run_mxcli new "$PROJECT_NAME" --version "$MX_VERSION")

# ── step 2: batch exec ────────────────────────────────────────────────────

echo "  exec 8 MDL files..."
run_mxcli exec \
    "$CAPSTONE_DIR/01-domain.mdl" \
    "$CAPSTONE_DIR/02-microflows.mdl" \
    "$CAPSTONE_DIR/03-nanoflows.mdl" \
    "$CAPSTONE_DIR/04-pages.mdl" \
    "$CAPSTONE_DIR/05-security.mdl" \
    "$CAPSTONE_DIR/06-kb.mdl" \
    "$CAPSTONE_DIR/07-escalation.mdl" \
    "$CAPSTONE_DIR/99-seed-data.mdl" \
    -p "$MPR"

# ── step 3: mx check ──────────────────────────────────────────────────────

echo "  mx check..."
MX_BIN=$(cd "$REPO_ROOT" && go run ./scripts/mx-path/main.go "$MX_VERSION")
BASELINE=$(mktemp)
echo 0 > "$BASELINE"
mx_check_against_baseline "$MPR" "$BASELINE" "$MX_BIN"

# ── step 4: PAD build ─────────────────────────────────────────────────────

echo "  building..."
run_mxcli_local build -p "$MPR"
echo "  build complete."

# ── step 5: human handoff ─────────────────────────────────────────────────

echo
echo "=== Human validation ==="
echo "  URL:      http://localhost:8080"
echo "  Customer: demo_customer@helpdesk.test / Demo12345678"
echo "  Agent:    demo_agent@helpdesk.test    / Demo12345678"
echo "  Manager:  demo_manager@helpdesk.test  / Demo12345678"
echo
echo "Starting runtime — Ctrl+C to stop."
run_mxcli_local run -p "$MPR" --admin-password Admin1234
```

- [ ] **Step 2: 设置可执行权限**

```bash
chmod +x scripts/validate-academy-capstone.sh
```

- [ ] **Step 3: 语法检查**

```bash
bash -n scripts/validate-academy-capstone.sh
```

期望输出：无输出（无语法错误）。

- [ ] **Step 4: shellcheck 静态分析（若已安装）**

```bash
shellcheck scripts/validate-academy-capstone.sh || true
```

常见警告可忽略：SC2317（trap 内的 unreachable code 误报）。若 shellcheck 未安装，跳过此步。

- [ ] **Step 5: 提交**

```bash
git add scripts/validate-academy-capstone.sh
git commit -m "feat(academy): add validate-academy-capstone e2e validation script"
```

---

## Task 3: Makefile target

**Files:**
- Modify: `Makefile`（在 `validate-snapshots` 附近追加；更新 `.PHONY`）

- [ ] **Step 1: 在 `.PHONY` 行追加新 target 名**

找到 Makefile 第 59 行的 `.PHONY` 声明，在末尾追加 `validate-academy-capstone`：

```makefile
.PHONY: build mdlrun build-local install-local build-debug release release-launcher release-daemon release-local-bins clean test _test-inner test-mdl report report-bench report-reset-baseline bench-baseline grammar sync-skills sync-commands sync-lint-rules sync-changelog sync-examples sync-all docs documentation docs-site docs-serve source-tree sbom sbom-report lint lint-go fmt vet update-helpdesk-golden test-helpdesk-regression setup install install-daemon test-section-check update-snapshots validate-snapshots validate-academy-capstone
```

- [ ] **Step 2: 在 `validate-snapshots` target 之后追加新 target**

在 `validate-snapshots` block 结束后（约第 465 行），插入：

```makefile
## validate-academy-capstone: full e2e validation of academy/zh capstone reference implementation
validate-academy-capstone:
	@./scripts/validate-academy-capstone.sh
```

- [ ] **Step 3: 验证 target 可识别（dry-run）**

```bash
make -n validate-academy-capstone
```

期望输出：
```
./scripts/validate-academy-capstone.sh
```

- [ ] **Step 4: 提交**

```bash
git add Makefile
git commit -m "feat(academy): add validate-academy-capstone Makefile target"
```

---

## Self-Review

### Spec coverage

| Spec 要求 | 对应 Task |
|-----------|-----------|
| `mxcli new` 创建全新 HelpDeskE2E 项目 | Task 2 step 1（`rm -rf` + `run_mxcli new`）|
| 批量 exec 8 个 MDL 文件 | Task 2 step 1（`run_mxcli exec ... 01..99`）|
| `mx_check_against_baseline` baseline=0 | Task 2 step 1（`BASELINE=$(mktemp); echo 0 > "$BASELINE"`）|
| `go run ./scripts/mx-path/main.go` 解析 MX_BIN | Task 1 + Task 2 step 1 |
| `run_mxcli_local build` | Task 2 step 1 |
| `run_mxcli_local run --admin-password` | Task 2 step 1 |
| `MXCLI`/`MXCLI_LOCAL` env var 覆盖 | Task 2 step 1（两个 helper 函数）|
| 失败保留 HelpDeskE2E 目录 | Task 2 step 1（`trap cleanup EXIT`，不含 `rm -rf`）|
| `.gitignore` `/HelpDeskE2E/` | 已存在（第 87 行），无需 task |
| Makefile target，无 build 依赖 | Task 3 |

### Placeholder scan

无 TBD / TODO / "类似 Task N" / 未定义引用。

### Type consistency

- `run_mxcli` / `run_mxcli_local`：Task 2 中定义并使用，名称一致。
- `docker.CachedMxPath`：Task 1 中使用，包路径 `github.com/mendixlabs/mxcli/cmd/mxcli/docker` 与源码一致。
- `mx_check_against_baseline`：来自已有的 `scripts/lib/mx-check.sh`，签名 `(mpr, baseline_file, mx_bin)` 与 Task 2 调用一致。
