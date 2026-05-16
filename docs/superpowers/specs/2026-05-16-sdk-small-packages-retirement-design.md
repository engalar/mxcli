# sdk/* 小包退役设计文档

**日期：** 2026-05-16
**状态：** 已批准，待实施
**范围：** sdk/agenteditor, sdk/security, sdk/versions, sdk/javaactions, sdk/workflows, sdk/domainmodel, sdk/pages

---

## 背景

`sdk/widgets` 已于 2026-05-16 完成退役（8 commits，删除 ~60,000 行）。本文档设计其余 sdk/* 小包的退役路线图，采用方案 A（叶子优先的自底向上波浪式）。

---

## 核心退役模型

每个小包的退役分三步：

1. **类型替换**：在 `sdk/mpr/` 内部，将对 `sdk/TYPE` 的引用改为 `modelsdk/gen/TYPE`
2. **接口清理**：更新导出签名，使调用方（`mdl/backend/mpr/`）不感知变化
3. **删包**：`git rm sdk/TYPE/`，确认零残留引用

---

## 各包现状

| 包 | 代码量 | modelsdk 对应 | sdk/mpr 内需改文件数 | 难度 |
|---|---|---|---|---|
| `sdk/agenteditor` | 38行（纯代理） | ❌（已在 mdl/types） | ~8 个小改 | 极简 |
| `sdk/security` | 209行 | ✅ gen/security/ | ~2 个 | 简 |
| `sdk/versions` | 数据/注册 | ✅ modelsdk/version/ | — | 简 |
| `sdk/javaactions` | 269行 | ✅ gen/javaactions/ | ~6 个 | 中 |
| `sdk/workflows` | 395行（已 DEPRECATED） | ✅ gen/workflows/ | ~6 个 | 中 |
| `sdk/domainmodel` | 607行 | ✅ gen/domainmodels/ | ~8 个 | 复杂 |
| `sdk/pages` | 1470行 | ✅ gen/pages/ | ~16 个 | 复杂 |

---

## 四波 PR 结构

```
Wave 1 (PR1)  agenteditor + security + versions    ← 无内部 sdk/mpr 改写，纯类型切换
Wave 2 (PR2)  javaactions + workflows              ← sdk/mpr 内 ~12 文件改写
Wave 3 (PR3)  domainmodel                          ← sdk/mpr 内 ~8 文件，独立 PR
Wave 4 (PR4)  pages                                ← sdk/mpr 内 ~16 文件，最复杂
PR5 (可选)    sdk/mpr 本体退役                     ← 前四波完成后才解锁，单独设计
```

每波结束时 `go build ./...` 和 `go test ./...` 必须全绿。Wave 2~4 不修改 `mdl/backend/mpr/` 的公开接口。

---

## 逐波详细设计

### Wave 1 — agenteditor + security + versions（PR1）

**sdk/agenteditor**
- 已是 `mdl/types` 的纯代理（type alias），无需 modelsdk/gen 对应
- 将 `sdk/mpr/` 中 ~8 个文件的 import 改为 `mdl/types`
- 无需 TDD，编译即验证
- `git rm sdk/agenteditor/`

**sdk/security**
- `modelsdk/gen/security/` 已有完整实现
- `sdk/mpr/reader_documents.go` + `parser_security.go` 改用 gen 类型
- 写 security document roundtrip 测试确认读写不变
- `git rm sdk/security/`

**sdk/versions**
- 改写前先 diff `sdk/versions/` 与 `modelsdk/version/` 的版本注册数据
- 若完全同步：`sdk/mpr/version/` 改引用路径，`git rm sdk/versions/`
- 若不同步：补齐后再删

---

### Wave 2 — javaactions + workflows（PR2）

**sdk/javaactions**（~6 个文件）
- 目标文件：`parser_javaactions.go`, `writer_javaactions.go`, `system_java_actions.go`, `serialize_exports.go` 及相关测试
- 先写 javaaction roundtrip 失败测试，改写后验证通过
- `git rm sdk/javaactions/`

**sdk/workflows**（~6 个文件）
- 目标文件：`parser_workflow.go`, `writer_workflow.go`, `reader_documents.go`, `serialize_exports.go` 及相关测试
- 与 javaactions 合入同一 PR（两者互不依赖）
- `git rm sdk/workflows/`

---

### Wave 3 — domainmodel（PR3）

**sdk/domainmodel**（~8 个文件）
- 目标文件：`parser_domainmodel.go`, `writer_domainmodel.go`, `system_module.go`, `writer_modules.go`, `writer_units.go` 及相关测试
- 改写前对比 `sdk/domainmodel` 与 `modelsdk/gen/domainmodels/` struct 字段，缺口先补 gen 侧
- 需要专门的 entity/association/attribute roundtrip 测试套件
- `git rm sdk/domainmodel/`

---

### Wave 4 — pages（PR4）

**sdk/pages**（~16 个文件）
- 目标文件：`reader_widgets.go`, `writer_pages.go`, `writer_widgets_*.go`, `parser_page.go`, `parser_misc.go`, `reader_types.go`, `reader_documents.go` 等
- `modelsdk/gen/pages/` 已有完整实现（36,327 行）
- 以 `mdl-examples/widget-roundtrip/` 为安全网
- 每文件改完立即 `go build`，不批量改
- `git rm sdk/pages/`

---

## 测试策略

每波统一流程：

| 阶段 | 动作 |
|---|---|
| 改写前 | 写失败测试（roundtrip：读旧 MPR → 写 → 读回，字段一致） |
| 改写中 | `go build ./sdk/mpr/...` 逐文件验证 |
| 改写后 | `go build ./...` + `go test ./...` 全绿 |
| 删包后 | `grep -r "sdk/TYPE" . --include="*.go"` 空输出 |

Wave 3/4 额外：在真实 `.mpr` 文件上跑 `mxcli exec mdl-examples/` 验证端到端不退化。

---

## 风险与缓解

| 风险 | 缓解 |
|---|---|
| modelsdk/gen 类型字段与 sdk/TYPE 不完全对等 | 改写前对比两侧 struct，缺字段先补 gen 侧 |
| sdk/mpr 内部使用了 sdk/TYPE 的私有辅助函数 | 将辅助函数内联或迁移到 gen 扩展文件（ext.go） |
| Wave 2~4 改写量大，中间态编译失败 | 每文件改完立即验证，不批量改 |
| sdk/versions 数据与 modelsdk/version 不同步 | Wave 1 开始前先 diff，发现缺口立即补 |

---

## 验收标准（每波）

```bash
# 1. 零残留
grep -r '"github.com/mendixlabs/mxcli/sdk/TYPE"' . --include="*.go"  # 空

# 2. 全量编译
go build ./...  # 无错误

# 3. 全量测试
go test ./...   # 全 ok
```

Wave 4 额外：
```bash
mxcli exec mdl-examples/widget-roundtrip/*.mdl -p /path/to/app.mpr  # 成功
```

---

## 不在范围内

- `sdk/mpr` 本体退役（PR5）：依赖前四波全完成，需单独设计文档
- 任何对 `mdl/backend/mpr/` 公开接口的修改
- `modelsdk/gen` 类型的功能扩展（仅在发现缺口时作为 Wave 前置条件补齐）
