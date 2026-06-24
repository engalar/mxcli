# mxcli 测试开发指南

本指南规范所有 mxcli 测试的编写方式，确保测试按流水线阶段分层，报告可读，质量一致。

## 测试流水线层次

mxcli 的核心业务流水线：

```
MDL → syntax check → AST parse → visitor → semantic check
   → modelsdk → encode → BSON → decode → modelsdk → MDL
```

每个阶段对应一个测试层：

| 层 | 名称 | 测试文件后缀 | 包路径 | 不允许 |
|----|------|-------------|--------|--------|
| L1/L2 | Parser + Visitor | `*_test.go`（在 visitor/ 包） | `mdl/visitor/` | 打开 MPR 文件 |
| L3 | Executor Mock | `*_mock_test.go` | `mdl/executor/` | 打开真实 MPR；直接用 gen 类型 |
| L4 | Executor Gen | `*_gen_test.go` | `mdl/executor/` + `mdl/backend/mpr/` | 字节级 BSON 对比 |
| L5 | Decode | `*_roundtrip_test.go`（在 backend/mpr/ 包） | `mdl/backend/mpr/` | — |
| L6a | Roundtrip | `roundtrip_*.go`（在 executor/ 包） | `mdl/executor/` | — |
| L6b | Describe Sanity | `describe_sanity_test.go` | `mdl/executor/` | 需要 Docker |
| L7 | Resource Profile | `coverage/test-profiles/*.json` | 任意包 | 需要 `-resource-record` flag |
| Bench | Benchmark | `*_bench_test.go` | 任意包 | `time.Sleep` |

**文件后缀决定层**——`cmd/testreport/` 的分类器读取后缀，无需 build tag（L6a/roundtrip 集成测试除外）。

## 核心原则

1. **测试先行（TDD）** — 先写失败测试，再写实现。所有 PR 必须包含在实现之前提交的测试。
2. **最低有效层** — 在能验证逻辑的最低层写测试。改 visitor 逻辑写 L1/L2，改 executor 写 L3，改 BSON 写 L4。不要用重型 gen 测试去测纯逻辑。
3. **Mock 隔离** — L3 必须用 `mock.MockBackend`。禁止在 mock 测试里打开真实 MPR 文件（这会使测试变慢且不可靠）。
4. **Gen 测试用 shape 断言** — L4 不做字节级 BSON 对比。断言 `TypeName()`、具体字段值、集合长度。字节对比在实现细节变更后必然失败。
5. **Benchmark 隔离 setup** — 必须调用 `b.ResetTimer()`，确保 setup 代码不计入耗时。

## 各层写法模板

### L1/L2 — Parser + Visitor

验证 MDL 语法合法性和 AST 节点结构。

```go
func TestBuild_CreateMicroflow_ValidSyntax(t *testing.T) {
    mdl := `create microflow MyModule.DoWork () returns Nothing begin return; end;`
    prog, errs := visitor.Build(mdl)
    if len(errs) > 0 {
        t.Fatalf("unexpected parse errors: %v", errs)
    }
    if len(prog.Statements) != 1 {
        t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
    }
    stmt, ok := prog.Statements[0].(*ast.CreateMicroflowStmt)
    if !ok {
        t.Fatalf("expected CreateMicroflowStmt, got %T", prog.Statements[0])
    }
    if stmt.Name.Module != "MyModule" || stmt.Name.Name != "DoWork" {
        t.Errorf("unexpected name: %v", stmt.Name)
    }
}

func TestBuild_CreateMicroflow_MissingBegin_ReturnsError(t *testing.T) {
    mdl := `create microflow MyModule.DoWork () returns Nothing return;`
    _, errs := visitor.Build(mdl)
    if len(errs) == 0 {
        t.Fatal("expected parse error, got none")
    }
}
```

### L3 — Executor Mock

验证 executor 逻辑正确调用 backend 并产生正确输出。

```go
func TestCreateEntity_Mock_CallsBackend(t *testing.T) {
    var capturedEntity *genDm.Entity
    mb := &mock.MockBackend{
        IsConnectedFunc: func() bool { return true },
        ListModulesFunc: func() ([]*model.Module, error) {
            return []*model.Module{mkModule("MyModule")}, nil
        },
        SaveEntityFunc: func(e *genDm.Entity) error {
            capturedEntity = e
            return nil
        },
    }
    ctx, _ := newMockCtx(t, withBackend(mb))
    stmt := &ast.CreateEntityStmt{Name: ast.QualifiedName{Module: "MyModule", Name: "Order"}}

    if err := execCreateEntityGen(ctx, stmt); err != nil {
        t.Fatalf("execCreateEntityGen: %v", err)
    }
    if capturedEntity == nil {
        t.Fatal("SaveEntity was not called")
    }
    if capturedEntity.Name() != "Order" {
        t.Errorf("expected entity name Order, got %q", capturedEntity.Name())
    }
}
```

### L4 — Executor Gen

验证通过真实 gen 类型构造的对象字段正确。

```go
func TestGenMicroflowParameters_EntityType(t *testing.T) {
    w := openMprWriterForTest(t)
    // construct gen type and verify shape
    mf := &genMf.Microflow{}
    mf.SetName("TestFlow")
    param := genMf.NewMicroflowParameter()
    param.SetName("Input")
    // ... set type
    if param.Name() != "Input" {
        t.Errorf("expected Input, got %q", param.Name())
    }
    if param.TypeName() != "Microflows$MicroflowParameter" {
        t.Errorf("unexpected TypeName: %q", param.TypeName())
    }
}
```

### L5 — Decode (Roundtrip)

验证写入 MPR 后能正确读回。

```go
func TestEntityRoundtrip(t *testing.T) {
    dst := copyMPRFixture(t, fixtureMprPath, t.TempDir())
    w, _ := mmpr.NewWriter(dst)
    t.Cleanup(func() { _ = w.Close() })

    // write
    // ... create entity

    // read back
    r, _ := mmpr.NewReader(dst)
    t.Cleanup(func() { _ = r.Close() })
    // ... find entity and compare
}
```

### L6a — Roundtrip（集成，需要 `//go:build integration`）

```go
//go:build integration

func TestRoundtrip_Microflow_Basic(t *testing.T) {
    env := setupTestEnv(t)
    defer env.teardown()

    env.executeMDL(`create microflow RoundtripTest.MyFlow () returns Nothing begin return; end;`)
    out, _ := env.describeMDL(`describe microflow RoundtripTest.MyFlow;`)
    _, errs := visitor.Build(extractMDLContent(out))
    if len(errs) > 0 {
        t.Errorf("roundtrip MDL invalid: %v", errs)
    }
}
```

### Benchmark

```go
func BenchmarkCreateMicroflow_10Activities(b *testing.B) {
    w := openMprWriterForTest(b)
    ctx := newGenDescribeContext(b, w)
    // setup: prepare 10-activity stmt
    stmt := buildLargeMicroflowStmt(10)

    b.ResetTimer() // <- 必须
    for i := 0; i < b.N; i++ {
        _ = execCreateMicroflowGen(ctx, stmt)
    }
}
```

### L7 — Resource Profile（集成测试资源分析）

记录每个集成测试的资源使用量，用于调度和回归检测。

```go
func TestRoundtrip_Microflow_Basic(t *testing.T) {
    monitor := testresource.NewMonitor(t)
    defer monitor.Done() // profile captured at test end

    env := setupTestEnv(t)
    defer env.teardown()
    // ... test logic ...
}
```

**资源分类阈值：**

| 指标 | IO Heavy | CPU Heavy |
|------|----------|-----------|
| ReadBytes | > 10MB | — |
| WriteBytes | > 1MB | — |
| CPUTime/Duration | — | > 50% |

**调度规则：** IO Heavy 测试跑在 IO lane（默认 2 并发），CPU Heavy 跑在 CPU lane（默认 nproc 并发）。Mixed 测试跑在单独 lane。每个 lane 按 Duration 降序排列（长测试先启动）。

## 命名规范

| 类型 | 格式 | 示例 |
|------|------|------|
| 正常测试 | `Test<Subject>_<Scenario>` | `TestCreateMicroflow_MissingModule` |
| 期望错误 | `Test<Subject>_<Scenario>_ReturnsError` | `TestBuild_MissingBegin_ReturnsError` |
| Benchmark | `Benchmark<Operation><Scale>` | `BenchmarkCreateMicroflow_10Activities` |
| 辅助函数 | `mk<Type>`, `make<Type>`, `open<Type>ForTest` | `mkModule`, `openMprWriterForTest` |

## PR 测试要求清单

每个 PR 合并前必须满足：

```
[ ] L1/L2: 至少1个合法语法测试 + 至少1个非法语法测试（如改动了 visitor/ 或 grammar/）
[ ] L3: happy path + 至少1个错误路径（如改动了 executor/）
[ ] L4: shape 断言新字段的 TypeName() 和值（如涉及 BSON 写入）
[ ] L6b: 新文档类型已加入 describe_sanity_test.go 的 testdataMPRs 批量列表
[ ] _bench_test.go: 如改动涉及热路径（import、create、encode 操作）
[ ] L7: 如新增集成测试（roundtrip_*），须包含 `testresource.NewMonitor(t)`（或用 `make test-profile-record` 重新记录 profile）
[ ] make report 通过，无新增 FAIL
```

## 本地验证流程

```bash
# 全量测试 + 生成报告（终端摘要 + HTML）
make report

# 查看 HTML 报告（浏览器打开）
open coverage/report.html         # macOS
xdg-open coverage/report.html     # Linux

# 仅跑 benchmark 并更新基线
make report-bench

# 重置 benchmark 基线（大重构后使用）
make report-reset-baseline

# 仅跑特定层（开发时快速验证）
go test ./mdl/visitor/... -v                    # L1/L2
go test ./mdl/executor/... -run '.*Mock.*' -v   # L3
go test ./mdl/executor/... -run 'DescribeSanity' -v  # L6b
```

## 常见错误

| 错误 | 原因 | 解决 |
|------|------|------|
| mock 测试打开了 MPR 文件 | 违反 L3 隔离原则 | 改用 `mock.MockBackend` |
| gen 测试做字节对比 | 实现细节变化就会失败 | 改为断言 TypeName() 和字段值 |
| benchmark 没有 ResetTimer | setup 时间计入 b.N | 在 loop 前加 `b.ResetTimer()` |
| L6b 里的新文档类型报错 | 对应 DESCRIBE 函数有 bug | 修复 DESCRIBE，不要 skip |
