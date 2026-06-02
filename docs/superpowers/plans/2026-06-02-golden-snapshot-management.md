# Golden Snapshot 统一管理 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 统一 `testdata/helpdesk-*/describe-snapshot.mdl` 的基线管理工作流：多版本参数化、snapshot 独立重建 target、pre-commit hook 守卫 + mxcli check 验证、幂等性集成测试。

**Architecture:** 通过 `HELPDESK_VERSION` 环境变量参数化现有测试 helper，使同一测试函数覆盖 11.6.6 和 11.10.0 两个版本。新增 `make update-snapshots` 和 `make validate-snapshots` target 分离 MPR 重建与 snapshot 重建。新增 pre-commit hook `06-describe-snapshot-sync.sh` 守卫 snapshot 与 MPR 同步并运行 mxcli check。新增 `TestHelpdeskGolden_DescribeSnapshot_Idempotent` 验证从 clean MPR 执行 snapshot.mdl 后 re-describe 结果不变。

**Tech Stack:** Go testing, Makefile, POSIX sh, `mxcli check --references`，现有 `goldenfs.Open`（FUSE overlay），`runMDL`（已在 `bsoncompare_integration_test.go` 定义）。

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `internal/goldenfs/helpdesk_regression_test.go` | 参数化版本 helper；新增幂等性测试 |
| Modify | `Makefile` | 加 `HELPDESK_VERSIONS`；扩展 `update-helpdesk-golden`；新增 `update-snapshots`、`validate-snapshots` |
| Create | `.githooks/checks/06-describe-snapshot-sync.sh` | Hook：Rule A（MPR 无 snapshot）+ Rule B（snapshot mxcli check）|
| Create | `.githooks/sop/06-describe-snapshot-sync.md` | SOP 文档 |

---

## Task 1: 参数化测试 helper（HELPDESK_VERSION env var）

**Files:**
- Modify: `internal/goldenfs/helpdesk_regression_test.go:27-62`

- [ ] **Step 1: 在文件顶部 `var updateGolden` 声明之后，添加版本 helper**

在 `helpdesk_regression_test.go` 第 27 行 `var updateGolden = ...` 声明**之后**插入：

```go
// helpdeskVersion returns the Mendix version to use for golden tests.
// Controlled by the HELPDESK_VERSION env var; defaults to "11.6.6".
func helpdeskVersion() string {
	if v := os.Getenv("HELPDESK_VERSION"); v != "" {
		return v
	}
	return "11.6.6"
}
```

- [ ] **Step 2: 修改 `helpdeskBlankDir` 替换硬编码**

将：
```go
func helpdeskBlankDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), "testdata", "helpdesk-clean-11.6.6")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("testdata/helpdesk-clean-11.6.6 not found: %v", err)
	}
	return dir
}
```
替换为：
```go
func helpdeskBlankDir(t *testing.T) string {
	t.Helper()
	v := helpdeskVersion()
	dir := filepath.Join(repoRoot(t), "testdata", "helpdesk-clean-"+v)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("testdata/helpdesk-clean-%s not found: %v", v, err)
	}
	return dir
}
```

- [ ] **Step 3: 修改 `helpdeskBlankMPR`**

将：
```go
func helpdeskBlankMPR(t *testing.T) string {
	t.Helper()
	p := filepath.Join(helpdeskBlankDir(t), "minimal.mpr")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("testdata/helpdesk-clean-11.6.6/minimal.mpr not found: %v", err)
	}
	return p
}
```
替换为：
```go
func helpdeskBlankMPR(t *testing.T) string {
	t.Helper()
	p := filepath.Join(helpdeskBlankDir(t), "minimal.mpr")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("testdata/helpdesk-clean-%s/minimal.mpr not found: %v", helpdeskVersion(), err)
	}
	return p
}
```

- [ ] **Step 4: 修改 `helpdeskGoldenDir`**

将：
```go
func helpdeskGoldenDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "testdata", "helpdesk-golden-11.6.6")
}
```
替换为：
```go
func helpdeskGoldenDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "testdata", "helpdesk-golden-"+helpdeskVersion())
}
```

- [ ] **Step 5: 更新 `updateGolden` flag 描述（可选，避免误导）**

将第 27-28 行：
```go
var updateGolden = flag.Bool("update-golden", false,
	"overwrite testdata/helpdesk-golden-11.6.6/ with the current MDL execution result")
```
替换为：
```go
var updateGolden = flag.Bool("update-golden", false,
	"overwrite testdata/helpdesk-golden-<HELPDESK_VERSION>/ with the current MDL execution result")
```

- [ ] **Step 6: 验证编译**

```bash
CGO_ENABLED=0 go build ./internal/goldenfs/...
```
Expected: 无错误（测试文件有 build tag，只检查非测试文件）

```bash
CGO_ENABLED=0 go vet ./internal/goldenfs/ 2>&1 | grep -v 'helpdesk_regression_test'
```
Expected: 无 vet 错误

- [ ] **Step 7: 验证 11.6.6 回归测试仍通过**

```bash
CGO_ENABLED=0 go test ./internal/goldenfs/ \
  -tags linux,integration \
  -run TestHelpdeskGolden_DescribeSnapshot \
  -v -timeout 3m 2>&1 | tail -5
```
Expected: `--- PASS: TestHelpdeskGolden_DescribeSnapshot`

- [ ] **Step 8: Commit**

```bash
git add internal/goldenfs/helpdesk_regression_test.go
git commit -m "refactor(goldenfs): parameterize helpdesk version via HELPDESK_VERSION env var"
```

---

## Task 2: 新增幂等性集成测试

**Files:**
- Modify: `internal/goldet.go` (末尾追加)

- [ ] **Step 1: 在文件末尾追加 `TestHelpdeskGolden_DescribeSnapshot_Idempotent`**

注意：`runMDL(t, mprPath, script string)` 已定义在同包的 `bsoncompare_integration_test.go`，可直接调用。

```go
// TestHelpdeskGolden_DescribeSnapshot_Idempotent verifies that describe-snapshot.mdl
// is self-consistent: executing it on the clean MPR and re-describing must produce
// the same output as the original snapshot.
//
// This catches lossy describe output that drops information the builder writes.
//
// Run: go test ./internal/goldenfs/ -tags linux,integration -run TestHelpdeskGolden_DescribeSnapshot_Idempotent -v
func TestHelpdeskGolden_DescribeSnapshot_Idempotent(t *testing.T) {
	snapshotPath := filepath.Join(helpdeskGoldenDir(t), "describe-snapshot.mdl")
	snapshotBytes, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("describe-snapshot.mdl not found at %s — run: make update-snapshots", snapshotPath)
	}

	snap, err := Open(helpdeskBlankDir(t))
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

	// Execute describe-snapshot.mdl on the clean MPR using create-or-modify semantics.
	// runMDL is defined in bsoncompare_integration_test.go (same package).
	runMDL(t, mountMPR, string(snapshotBytes))

	// Re-describe: this must produce the same output as the original snapshot.
	got := describeMDLParseable(t, mountMPR)

	if string(snapshotBytes) == got {
		return
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(snapshotBytes)),
		B:        difflib.SplitLines(got),
		FromFile: "describe-snapshot.mdl (expected)",
		ToFile:   "re-describe after execute on clean MPR (actual)",
		Context:  5,
	}
	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	t.Errorf("describe-snapshot.mdl is not idempotent — re-describe after executing on clean MPR differs:\n%s", text)
}
```

- [ ] **Step 2: 运行新测试（期望通过或定位差异）**

```bash
CGO_ENABLED=0 go test ./internal/goldenfs/ \
  -tags linux,integration \
  -run TestHelpdeskGolden_DescribeSnapshot_Idempotent \
  -v -timeout 5m 2>&1 | tail -20
```

Expected: `--- PASS` 或者有 diff 输出（显示哪些语句不幂等）。如果有 diff，记录下来——这是已知的 IR 覆盖缺口，不在本次任务修复范围内；测试本身可以通过意味着幂等性已满足。

- [ ] **Step 3: Commit**

```bash
git add internal/goldenfs/helpdesk_regression_test.go
git commit -m "test(goldenfs): add TestHelpdeskGolden_DescribeSnapshot_Idempotent"
```

---

## Task 3: 扩展 Makefile

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: 在 `Makefile` 第 21 行（`BINARY_NAME = mxcli` 附近）之前添加版本变量**

在 `BINARY_NAME = mxcli` 行**之前**插入：

```makefile
HELPDESK_VERSIONS := 11.6.6 11.10.0
```

- [ ] **Step 2: 扩展 `update-helpdesk-golden` target**

找到（约第 303 行）：
```makefile
update-helpdesk-golden:
	CGO_ENABLED=0 go test ./internal/goldenfs/ \
		-tags linux,integration \
		-run TestHelpdeskGolden_Update \
		-update-golden \
		-v -timeout 10m
```

替换为：
```makefile
update-helpdesk-golden:
	@for v in $(HELPDESK_VERSIONS); do \
	  echo "=== Rebuilding helpdesk golden $$v ==="; \
	  HELPDESK_VERSION=$$v \
	  CGO_ENABLED=0 go test ./internal/goldenfs/ \
	    -tags linux,integration \
	    -run TestHelpdeskGolden_Update \
	    -update-golden \
	    -v -timeout 10m || exit 1; \
	done
	$(MAKE) update-snapshots
```

- [ ] **Step 3: 在 `update-helpdesk-golden` 之后添加 `update-snapshots` target**

```makefile
# Rebuild describe-snapshot.mdl for all helpdesk versions from their existing golden MPRs.
# Faster than update-helpdesk-golden (~5s vs ~10m). Run after describe logic changes.
# Validates each snapshot with: mxcli check snapshot.mdl -p minimal.mpr --references
update-snapshots: build
	@for v in $(HELPDESK_VERSIONS); do \
	  echo "=== Updating snapshot $$v ==="; \
	  HELPDESK_VERSION=$$v \
	  CGO_ENABLED=0 go test ./internal/goldenfs/ \
	    -tags linux,integration \
	    -run TestHelpdeskGolden_DescribeSnapshot \
	    -update-golden \
	    -v -timeout 3m || exit 1; \
	  echo "  mxcli check describe-snapshot.mdl ($$v)..."; \
	  $(BUILD_DIR)/$(BINARY_NAME) check \
	    testdata/helpdesk-golden-$$v/describe-snapshot.mdl \
	    -p testdata/helpdesk-golden-$$v/minimal.mpr \
	    --references || exit 1; \
	done
```

- [ ] **Step 4: 在 `update-snapshots` 之后添加 `validate-snapshots` target**

```makefile
# Validate describe-snapshot.mdl for all versions without rebuilding.
# Runs mxcli check + idempotency integration test. Suitable for CI.
validate-snapshots: build
	@for v in $(HELPDESK_VERSIONS); do \
	  echo "=== Validating snapshot $$v ==="; \
	  $(BUILD_DIR)/$(BINARY_NAME) check \
	    testdata/helpdesk-golden-$$v/describe-snapshot.mdl \
	    -p testdata/helpdesk-golden-$$v/minimal.mpr \
	    --references || exit 1; \
	  echo "  idempotency test ($$v)..."; \
	  HELPDESK_VERSION=$$v \
	  CGO_ENABLED=0 go test ./internal/goldenfs/ \
	    -tags linux,integration \
	    -run TestHelpdeskGolden_DescribeSnapshot_Idempotent \
	    -v -timeout 5m || exit 1; \
	done
```

- [ ] **Step 5: 更新 `.PHONY` 行，加入新 target**

找到 `.PHONY:` 行（约第 54 行），在末尾追加 `update-snapshots validate-snapshots`。

- [ ] **Step 6: 验证 Makefile 语法**

```bash
make -n update-snapshots 2>&1 | head -20
make -n validate-snapshots 2>&1 | head -20
```
Expected: 打印预期执行命令，无语法错误。

- [ ] **Step 7: 实际运行 `make update-snapshots`（仅 11.6.6，验证）**

```bash
HELPDESK_VERSIONS=11.6.6 make update-snapshots 2>&1 | tail -10
```
Expected: snapshot 重建成功，mxcli check PASS。

- [ ] **Step 8: Commit**

```bash
git add Makefile
git commit -m "feat(make): add HELPDESK_VERSIONS, update-snapshots, validate-snapshots targets"
```

---

## Task 4: 新增 Hook 06 + SOP

**Files:**
- Create: `.githooks/checks/06-describe-snapshot-sync.sh`
- Create: `.githooks/sop/06-describe-snapshot-sync.md`

- [ ] **Step 1: 创建 hook 文件**

创建 `.githooks/checks/06-describe-snapshot-sync.sh`：

```sh
#!/bin/sh
# Guard: describe-snapshot.mdl must move together with MPR mprcontents/;
#        staged describe-snapshot.mdl must pass mxcli check --references.
#
# Rule A — MPR mprcontents/ staged without describe-snapshot.mdl → block.
# Rule B — describe-snapshot.mdl staged → run mxcli check --references.
#
# Versions checked: 11.6.6 11.10.0
# mxcli binary: bin/mxcli (skipped if not built)

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"
MXCLI="${REPO_ROOT}/bin/mxcli"

if [ ! -x "$MXCLI" ]; then
    exit 0  # mxcli not built — skip (normal for non-describe changes)
fi

for version in 11.6.6 11.10.0; do
    golden_dir="testdata/helpdesk-golden-${version}"
    snapshot="${golden_dir}/describe-snapshot.mdl"
    mpr="${golden_dir}/minimal.mpr"

    staged_mpr=$(git diff --cached --name-only | \
      grep -E "^${golden_dir}/(minimal\\.mpr|mprcontents/)" | head -1)
    staged_snapshot=$(git diff --cached --name-only | \
      grep "^${snapshot}\$" | head -1)

    # Rule A: MPR staged without corresponding describe-snapshot.mdl.
    if [ -n "$staged_mpr" ] && [ -z "$staged_snapshot" ]; then
        echo "" >&2
        echo "COMMIT BLOCKED: ${golden_dir}/mprcontents/ staged without describe-snapshot.mdl." >&2
        echo "" >&2
        echo "  Run: make update-snapshots" >&2
        echo "  Then: git add ${snapshot}" >&2
        echo "" >&2
        echo "SOP: .githooks/sop/06-describe-snapshot-sync.md" >&2
        echo "CONTEXT: TRIGGER_FILE=${staged_mpr}" >&2
        exit 1
    fi

    # Rule B: describe-snapshot.mdl staged → validate with mxcli check.
    if [ -n "$staged_snapshot" ]; then
        if [ ! -f "$mpr" ]; then
            continue  # MPR not present locally — skip check
        fi
        echo "pre-commit: mxcli check describe-snapshot.mdl (Mendix ${version})..." >&2
        check_output=$("$MXCLI" check "$snapshot" -p "$mpr" --references 2>&1)
        check_rc=$?
        if [ $check_rc -ne 0 ]; then
            echo "" >&2
            echo "COMMIT BLOCKED: describe-snapshot.mdl failed mxcli check (Mendix ${version})." >&2
            echo "" >&2
            echo "$check_output" >&2
            echo "" >&2
            echo "  Fix the describe output, then: make update-snapshots" >&2
            echo "" >&2
            echo "SOP: .githooks/sop/06-describe-snapshot-sync.md" >&2
            exit 1
        fi
        echo "pre-commit: mxcli check PASS (${version})" >&2
    fi
done
```

- [ ] **Step 2: 设置 hook 可执行权限**

```bash
chmod +x .githooks/checks/06-describe-snapshot-sync.sh
```

- [ ] **Step 3: 创建 SOP 文档**

创建 `.githooks/sop/06-describe-snapshot-sync.md`：

```markdown
# SOP: 06-describe-snapshot-sync

## Trigger A: MPR staged without describe-snapshot.mdl

**Message:** `COMMIT BLOCKED: testdata/helpdesk-golden-X/mprcontents/ staged without describe-snapshot.mdl`

**Steps:**
1. Run: `make update-snapshots`
2. Stage the updated snapshot: `git add testdata/helpdesk-golden-X/describe-snapshot.mdl`
3. Re-attempt commit

**Why:** The MPR and its describe snapshot must always move together. A stale
snapshot means CI tests (TestHelpdeskGolden_DescribeSnapshot) will fail.

---

## Trigger B: describe-snapshot.mdl fails mxcli check

**Message:** `COMMIT BLOCKED: describe-snapshot.mdl failed mxcli check`

**Steps:**
1. Run manually to see the full error:
   ```bash
   bin/mxcli check testdata/helpdesk-golden-X/describe-snapshot.mdl \
     -p testdata/helpdesk-golden-X/minimal.mpr --references
   ```
2. Identify the failing statement (entity ref, page param syntax, etc.)
3. Fix the describe logic in `mdl/executor/` or `mdl/backend/mpr/`
4. Rebuild and re-validate: `make update-snapshots`
5. Stage and re-commit

**Common causes:**
- Renamed entity/attribute not updated in describe output
- Page parameter syntax changed in MDL grammar
- New reserved keyword conflicting with widget/entity name in snapshot

---

## Idempotency failure (make validate-snapshots)

**Test:** `TestHelpdeskGolden_DescribeSnapshot_Idempotent`

**Meaning:** Executing `describe-snapshot.mdl` on the clean MPR and re-describing
produces different output. The snapshot is not a complete, self-contained description.

**Steps:**
1. Run to see the diff:
   ```bash
   HELPDESK_VERSION=11.6.6 go test ./internal/goldenfs/ \
     -tags linux,integration \
     -run TestHelpdeskGolden_DescribeSnapshot_Idempotent -v
   ```
2. Identify which statements change on re-describe (typically lossy IR describe output)
3. Fix the relevant describe path in `mdl/executor/cmd_pages_describe.go` or
   `mdl/executor/cmd_pages_model_to_mdl.go`
4. Run `make update-snapshots` and re-validate

**Known gaps (as of 2026-06-02):** PageModel IR does not fully cover button actions,
checkbox, radiobuttons, and pluggable widget Object trees. These fall back to the
legacy describe path which produces correct output. See `pageModelHasLossyWidget`
in `mdl/executor/cmd_pages_create_v3.go`.
```

- [ ] **Step 4: 验证 hook 被自动加入 pre-commit 流程**

`.githooks/pre-commit` 已有 `for check in .../checks/*.sh; do sh "$check" || exit 1; done`，所以 `06-` 前缀会被自动 pick up。验证：

```bash
ls -1 .githooks/checks/*.sh
```
Expected: `06-describe-snapshot-sync.sh` 在列表中（排在 `05-mx-check-golden.sh` 之后）。

- [ ] **Step 5: 测试 Rule A（暂存 MPR 但不暂存 snapshot 应被阻塞）**

```bash
# 模拟：暂存一个 mprcontents 文件但不暂存 snapshot
touch testdata/helpdesk-golden-11.6.6/mprcontents/test-dummy.tmp
git add testdata/helpdesk-golden-11.6.6/mprcontents/test-dummy.tmp
sh .githooks/checks/06-describe-snapshot-sync.sh; echo "exit: $?"
git restore --staged testdata/helpdesk-golden-11.6.6/mprcontents/test-dummy.tmp
rm testdata/helpdesk-golden-11.6.6/mprcontents/test-dummy.tmp
```
Expected: hook 输出 `COMMIT BLOCKED: ... without describe-snapshot.mdl`，exit code 1。

- [ ] **Step 6: 测试 Rule B（暂存 snapshot 且 mxcli check PASS）**

```bash
git add testdata/helpdesk-golden-11.6.6/describe-snapshot.mdl
sh .githooks/checks/06-describe-snapshot-sync.sh; echo "exit: $?"
git restore --staged testdata/helpdesk-golden-11.6.6/describe-snapshot.mdl
```
Expected: hook 输出 `mxcli check PASS (11.6.6)`，exit code 0。

- [ ] **Step 7: Commit**

```bash
git add .githooks/checks/06-describe-snapshot-sync.sh \
        .githooks/sop/06-describe-snapshot-sync.md
git commit -m "feat(hooks): add 06-describe-snapshot-sync (co-staging guard + mxcli check)"
```

---

## Self-Review Notes

**Spec coverage check:**
- `HELPDESK_VERSIONS` → Task 3 ✓
- `update-helpdesk-golden` 扩展 → Task 3 ✓
- `update-snapshots` → Task 3 ✓
- `validate-snapshots` → Task 3 ✓
- 测试参数化（`HELPDESK_VERSION`）→ Task 1 ✓
- `TestHelpdeskGolden_DescribeSnapshot_Idempotent` → Task 2 ✓
- Hook Rule A（MPR 无 snapshot 阻塞）→ Task 4 ✓
- Hook Rule B（snapshot mxcli check）→ Task 4 ✓
- SOP 文档 → Task 4 ✓

**Type consistency:** `helpdeskVersion()` 函数在 Task 1 定义，Task 3/4 的 Makefile 通过 env var 传入（一致）。`runMDL` 已在同包存在（Task 2 直接调用）。

**Known issue:** `TestHelpdeskGolden_DescribeSnapshot_Idempotent` 可能在当前 PageModel IR 覆盖不完整时有 diff（lossy widget 的 describe 输出差异）。这是已知的 IR Stage 2 债务，不阻塞本次 PR 合并——测试失败时记录差异即可，SOP 中有说明。
