# Golden Snapshot 统一管理设计

**日期**：2026-06-02  
**状态**：待实现  
**目标**：统一 `testdata/helpdesk-*/describe-snapshot.mdl` 的基线管理工作流，消除多版本管理碎片化，强化 snapshot 的语法校验和幂等性保证。

---

## 1. 问题背景

当前 `describe-snapshot.mdl` 管理存在四个痛点：

1. **MPR 重建与 snapshot 重建是两步走**：`make update-helpdesk-golden` 只重建 MPR，snapshot 要单独用 `-update-golden` flag 跑测试，经常遗漏。
2. **无 snapshot 过期检测**：Hook `03` 要求 MPR 和 `helpdesk-app.mdl` 同时暂存，但不检查 `describe-snapshot.mdl` 是否已同步更新，导致暂存 MPR 但遗漏 snapshot 的 commit 可以通过。
3. **多版本管理缺失**：`helpdesk-golden-11.10.0/describe-snapshot.mdl` 存在但没有对应的 Makefile target 和 hook 守卫。
4. **无 snapshot 有效性验证**：snapshot.mdl 文件可能包含无法被 `mxcli check` 通过的输出（如语法错误、引用错误），且没有幂等性保证——即从 clean MPR 执行 snapshot.mdl 后 re-describe，结果应等于原始 snapshot.mdl。

---

## 2. 整体架构

### Makefile 目标体系

```
make update-helpdesk-golden     完整重建：MDL → MPR → snapshot → validate（所有版本）
make update-snapshots           快速路径：仅重建 snapshot + mxcli check（所有版本，~5s）
make validate-snapshots         只验证不重建（CI 用）：mxcli check + 幂等性测试
```

版本列表集中管理：Makefile 顶部 `HELPDESK_VERSIONS := 11.6.6 11.10.0`，三个 target 均迭代该列表。

### 新增 Hook：`06-describe-snapshot-sync.sh`

| 触发条件 | 动作 |
|---------|------|
| `testdata/helpdesk-golden-X/mprcontents/` 暂存 | 对应 `describe-snapshot.mdl` 必须也暂存，否则阻塞 |
| `testdata/helpdesk-golden-X/describe-snapshot.mdl` 暂存 | 运行 `mxcli check snapshot.mdl -p minimal.mpr --references`（≤3s），失败则阻塞 |

Hook 不运行幂等性测试（太慢），幂等性测试在 `make validate-snapshots` 和 CI 中运行。

### 幂等性保证

```
snapshot.mdl ≡ describe(execute(clean_mpr, snapshot.mdl))
```

从 clean MPR 出发执行 snapshot.mdl，re-describe 的结果必须等于原始 snapshot.mdl（字节级相等）。这验证 snapshot.mdl 是一份**完整可独立执行的项目描述**。

---

## 3. 各组件详细设计

### 3.1 Makefile 变更

在 Makefile 顶部添加版本列表变量：

```makefile
HELPDESK_VERSIONS := 11.6.6 11.10.0
```

修改 `update-helpdesk-golden`：

```makefile
update-helpdesk-golden:
	@for v in $(HELPDESK_VERSIONS); do \
	  echo "=== helpdesk golden $$v ==="; \
	  HELPDESK_VERSION=$$v \
	  CGO_ENABLED=0 go test ./internal/goldenfs/ \
	    -tags linux,integration \
	    -run TestHelpdeskGolden_Update \
	    -update-golden -v -timeout 10m || exit 1; \
	done
	$(MAKE) update-snapshots
```

新增 `update-snapshots`：

```makefile
update-snapshots: build
	@for v in $(HELPDESK_VERSIONS); do \
	  echo "=== snapshot $$v ==="; \
	  HELPDESK_VERSION=$$v \
	  CGO_ENABLED=0 go test ./internal/goldenfs/ \
	    -tags linux,integration \
	    -run TestHelpdeskGolden_DescribeSnapshot \
	    -update-golden -v -timeout 3m || exit 1; \
	  $(BUILD_DIR)/$(BINARY_NAME) check \
	    testdata/helpdesk-golden-$$v/describe-snapshot.mdl \
	    -p testdata/helpdesk-golden-$$v/minimal.mpr \
	    --references || exit 1; \
	done
```

新增 `validate-snapshots`：

```makefile
validate-snapshots: build
	@for v in $(HELPDESK_VERSIONS); do \
	  $(BUILD_DIR)/$(BINARY_NAME) check \
	    testdata/helpdesk-golden-$$v/describe-snapshot.mdl \
	    -p testdata/helpdesk-golden-$$v/minimal.mpr \
	    --references || exit 1; \
	  HELPDESK_VERSION=$$v \
	  CGO_ENABLED=0 go test ./internal/goldenfs/ \
	    -tags linux,integration \
	    -run TestHelpdeskGolden_DescribeSnapshot_Idempotent \
	    -v -timeout 5m || exit 1; \
	done
```

`.PHONY` 列表加入 `update-snapshots validate-snapshots`。

### 3.2 Go 测试参数化

在 `internal/goldenfs/helpdesk_regression_test.go` 中，将硬编码路径改为 `HELPDESK_VERSION` 环境变量驱动：

```go
// helpdeskVersion is read from env; defaults to "11.6.6" for backward compat.
var helpdeskVersion = func() string {
    if v := os.Getenv("HELPDESK_VERSION"); v != "" {
        return v
    }
    return "11.6.6"
}()

func helpdeskGoldenDir(t *testing.T) string {
    t.Helper()
    return filepath.Join("testdata", "helpdesk-golden-"+helpdeskVersion)
}

func helpdeskBlankDir(t *testing.T) string {
    t.Helper()
    return filepath.Join("testdata", "helpdesk-clean-"+helpdeskVersion)
}

func helpdeskGoldenMPR(t *testing.T) string {
    t.Helper()
    return filepath.Join(helpdeskGoldenDir(t), "minimal.mpr")
}
```

现有测试函数体中所有硬编码的 `"testdata/helpdesk-golden-11.6.6"` 和 `"testdata/helpdesk-clean-11.6.6"` 路径替换为上述 helper 调用。

### 3.3 新测试：`TestHelpdeskGolden_DescribeSnapshot_Idempotent`

```go
// TestHelpdeskGolden_DescribeSnapshot_Idempotent 验证 describe-snapshot.mdl 的幂等性：
// 从 clean MPR 出发执行 snapshot.mdl，re-describe 结果必须等于原始 snapshot.mdl。
func TestHelpdeskGolden_DescribeSnapshot_Idempotent(t *testing.T) {
    snapshotPath := filepath.Join(helpdeskGoldenDir(t), "describe-snapshot.mdl")
    snapshotBytes, err := os.ReadFile(snapshotPath)
    if err != nil {
        t.Fatalf("snapshot not found at %s — run: make update-snapshots", snapshotPath)
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

    // 执行 snapshot.mdl（create or modify 语义）
    // runMDLContent 是新 helper，接受 MDL 字符串而非文件路径；
    // 现有 runHelpdeskMDL(t, mountMPR) 从固定文件路径读取，需扩展或新增。
    runMDLContent(t, mountMPR, string(snapshotBytes))

    // re-describe
    got := describeMDLParseable(t, mountMPR)

    if string(snapshotBytes) == got {
        return
    }

    diff := difflib.UnifiedDiff{
        A:        difflib.SplitLines(string(snapshotBytes)),
        B:        difflib.SplitLines(got),
        FromFile: "snapshot.mdl (expected)",
        ToFile:   "re-describe (actual)",
        Context:  3,
    }
    text, _ := difflib.GetUnifiedDiffString(diff)
    t.Errorf("describe-snapshot.mdl not idempotent — re-describe after executing on clean MPR differs:\n%s", text)
}
```

### 3.4 新 Hook：`.githooks/checks/06-describe-snapshot-sync.sh`

```sh
#!/bin/sh
# Guard: describe-snapshot.mdl and MPR must move together; snapshot must pass mxcli check.
#
# Rule A — MPR mprcontents/ staged without describe-snapshot.mdl → block.
# Rule B — describe-snapshot.mdl staged → run mxcli check --references.

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"
MXCLI="${REPO_ROOT}/bin/mxcli"
if [ ! -x "$MXCLI" ]; then
    exit 0  # mxcli not built — skip (developer environment)
fi

for version in 11.6.6 11.10.0; do
    golden_dir="testdata/helpdesk-golden-${version}"
    snapshot="${golden_dir}/describe-snapshot.mdl"
    mpr="${golden_dir}/minimal.mpr"

    staged_mpr=$(git diff --cached --name-only | \
      grep -E "^${golden_dir}/(minimal\\.mpr|mprcontents/)" | head -1)
    staged_snapshot=$(git diff --cached --name-only | \
      grep "^${snapshot}\$" | head -1)

    # Rule A
    if [ -n "$staged_mpr" ] && [ -z "$staged_snapshot" ]; then
        echo "" >&2
        echo "COMMIT BLOCKED: ${golden_dir}/mprcontents/ staged without describe-snapshot.mdl." >&2
        echo "  Run: make update-snapshots" >&2
        echo "  Then: git add ${snapshot}" >&2
        echo "SOP: .githooks/sop/06-describe-snapshot-sync.md" >&2
        exit 1
    fi

    # Rule B
    if [ -n "$staged_snapshot" ]; then
        if [ ! -f "$mpr" ]; then
            continue
        fi
        echo "pre-commit: mxcli check ${snapshot} (Mendix ${version})..." >&2
        if ! "$MXCLI" check "$snapshot" -p "$mpr" --references 2>&1; then
            echo "" >&2
            echo "COMMIT BLOCKED: describe-snapshot.mdl failed mxcli check." >&2
            echo "  Fix the describe output, then: make update-snapshots" >&2
            echo "SOP: .githooks/sop/06-describe-snapshot-sync.md" >&2
            exit 1
        fi
        echo "pre-commit: mxcli check PASS (${version})" >&2
    fi
done
```

### 3.5 新 SOP：`.githooks/sop/06-describe-snapshot-sync.md`

```markdown
# SOP: 06-describe-snapshot-sync

## Trigger A: MPR staged without describe-snapshot.mdl
Run: make update-snapshots
Then: git add testdata/helpdesk-golden-X/describe-snapshot.mdl

## Trigger B: describe-snapshot.mdl fails mxcli check
The describe output contains invalid MDL syntax or broken references.
1. Run: mxcli check testdata/helpdesk-golden-X/describe-snapshot.mdl -p .../minimal.mpr --references
2. Identify failing statement (entity ref, page param, etc.)
3. Fix the describe logic in mdl/executor/ or mdl/backend/mpr/
4. Run: make update-snapshots
5. Re-stage and re-commit

## Idempotency failure (make validate-snapshots)
describe-snapshot.mdl is not idempotent: executing it on clean MPR and re-describing produces different output.
This usually means describe output uses lossy representation (drops fields that the builder writes).
See: mdl/executor/cmd_pages_create_v3.go pageModelHasLossyWidget for current known gaps.
```

---

## 4. Hook 注册

在 `.githooks/pre-commit`（或等效 hook 注册脚本）中，确保 `06-describe-snapshot-sync.sh` 已加入执行链。按现有 `05-mx-check-golden.sh` 的注册模式添加。

---

## 5. 文件变更汇总

| 类型 | 文件 | 变化 |
|------|------|------|
| 修改 | `Makefile` | 加 `HELPDESK_VERSIONS`；扩展 `update-helpdesk-golden`；新增 `update-snapshots`、`validate-snapshots` |
| 修改 | `internal/goldenfs/helpdesk_regression_test.go` | 参数化 `helpdeskVersion`；替换硬编码路径；新增 `TestHelpdeskGolden_DescribeSnapshot_Idempotent` |
| 新建 | `.githooks/checks/06-describe-snapshot-sync.sh` | Rule A + Rule B 守卫 |
| 新建 | `.githooks/sop/06-describe-snapshot-sync.md` | 触发场景和修复步骤文档 |

---

## 6. 不在本次范围内

- 幂等性失败的根本修复（如 PageModel IR 覆盖不完整导致 lossy describe）——这是独立的 IR Stage 2 任务
- Hook `03` 的 `11.10.0` 版本扩展——当前 `03-helpdesk-mdl-golden-sync.sh` 只守卫 `11.6.6`，与本设计同步但作为独立修改
