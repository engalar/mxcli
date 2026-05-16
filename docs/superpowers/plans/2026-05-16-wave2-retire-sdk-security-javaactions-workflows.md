# Wave 2: sdk/security + sdk/javaactions + sdk/workflows 退役实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 退役 `sdk/security`、`sdk/javaactions`、`sdk/workflows` 三个包，使 `sdk/mpr` 内部不再 import 它们，最终 git rm。

**Architecture:** 三个包均为 sdk/mpr 内部使用的类型定义包。退役路径：先做类型映射调查（Task 1），确认 modelsdk/gen 等价类型后，在 sdk/mpr 内部做类型切换，最后删包。每个包独立迁移，互不干扰。

**Tech Stack:** Go 1.26，`GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go`

**前提条件：** Wave 1（sdk/agenteditor）已完成。

---

## 文件变动清单（待调查后确认）

| 操作 | 路径 |
|------|------|
| 修改 | `sdk/mpr/parser_security.go` |
| 修改 | `sdk/mpr/reader_documents.go`（security 相关函数） |
| 修改 | `sdk/mpr/writer_javaactions.go` |
| 修改 | `sdk/mpr/parser_javaactions.go` |
| 修改 | `sdk/mpr/serialize_exports.go`（javaactions/workflows 部分） |
| 修改 | `sdk/mpr/parser_misc.go`（javaactions 部分） |
| 修改 | `sdk/mpr/parser_workflow.go` |
| 修改 | `sdk/mpr/writer_workflow.go` |
| 修改 | `sdk/mpr/reader_documents.go`（workflows 相关函数） |
| 删除 | `sdk/security/`（整包） |
| 删除 | `sdk/javaactions/`（整包） |
| 删除 | `sdk/workflows/`（整包） |

---

## Task 1: 类型映射调查（前置，必须完成才能继续）

**目标：** 确认每个旧类型在新位置的等价映射，避免在迁移过程中发现类型缺口。

- [ ] **1.1 调查 sdk/security 与 modelsdk/gen/security 的类型对应关系**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02

# 查看 sdk/security 的所有类型和字段
cat sdk/security/security.go

# 查看 modelsdk/gen/security/types.go 的类型（codec.Element 风格）
grep -n "^type\|func.*Get\|func.*Set" modelsdk/gen/security/types.go | head -50

# 检查 mdl/types/ 中是否有 security 相关结构体
grep -rn "ProjectSecurity\|ModuleSecurity\|UserRole\|DemoUser\|PasswordPolicy" mdl/types/ --include="*.go" | head -20
```

记录结果：`sdk/security.ProjectSecurity` 对应哪个新类型？字段名是否匹配？

- [ ] **1.2 调查 sdk/javaactions 与 modelsdk/gen/javaactions 的类型对应关系**

```bash
# 查看 sdk/javaactions 的类型
cat sdk/javaactions/javaactions.go

# 查看 modelsdk/gen/javaactions/ 的内容
ls modelsdk/gen/javaactions/
cat modelsdk/gen/javaactions/refs.go

# 检查 mdl/types/ 中是否有 javaactions 相关类型
grep -rn "JavaAction\|CodeActionParameter" mdl/types/ --include="*.go" | head -20

# 检查 sdk/mpr/parser_javaactions.go 的完整内容（了解 BSON 解析逻辑）
cat sdk/mpr/parser_javaactions.go
```

记录结果：javaactions 的类型是否在 modelsdk/gen 或 mdl/types 中有等价实现？

- [ ] **1.3 调查 sdk/workflows 与 modelsdk/gen/workflows 的类型对应关系**

```bash
# 查看 sdk/workflows 的类型（全文）
cat sdk/workflows/workflow.go

# 查看 modelsdk/gen/workflows/ 的内容
ls modelsdk/gen/workflows/
cat modelsdk/gen/workflows/enums.go | head -50
cat modelsdk/gen/workflows/ext.go

# 检查 sdk/mpr/parser_workflow.go（了解哪些类型被用于 BSON 解析）
head -80 sdk/mpr/parser_workflow.go
```

记录结果：workflow 类型是否在 modelsdk/gen/workflows 中有等价实现？

- [ ] **1.4 根据调查结果，确认迁移路径**

对每个包，选择以下路径之一：

**路径 A**：直接类型替换（如果 modelsdk/gen 或 mdl/types 有字段完全对等的类型）
  - 操作：在 sdk/mpr 文件中替换 import 和类型名

**路径 B**：需要补齐 modelsdk/gen 类型（如果 gen 类型缺少字段）
  - 操作：先在 modelsdk/gen/supplements.json 或扩展文件中补字段，再做路径 A

**路径 C**：类型不兼容，需要重写解析逻辑
  - 操作：重写 parser_X.go，产出新的 gen 类型而非旧 struct

如果调查发现某个包需要路径 C（大幅重写），请拆分为单独的子任务，每次只改一个文件。

---

## Task 2: 退役 sdk/security

> 在完成 Task 1.1 且确认迁移路径后执行。

- [ ] **2.1 写 security roundtrip 失败测试**

在 `sdk/mpr/parser_security_test.go`（或现有测试文件）中添加：

```go
// 此测试在迁移前必须通过，迁移后必须仍然通过
func TestSecurityRoundtrip(t *testing.T) {
    // 使用现有 MPR 测试数据（如果有）读取 ProjectSecurity 并验证字段不变
    // 如果没有测试数据，至少验证函数签名存在
    // （根据 Task 1.1 调查结果填写具体断言）
}
```

- [ ] **2.2 运行确认测试当前通过**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./sdk/mpr/... -run TestSecurity -v
```

- [ ] **2.3 替换 sdk/mpr/parser_security.go 中的 sdk/security 引用**

（根据 Task 1 确定的迁移路径执行，此处为模板）

```bash
# 路径 A 示例：
sed -i 's|"github.com/mendixlabs/mxcli/sdk/security"|"<新包路径>"|g' sdk/mpr/parser_security.go
# 然后手动替换类型名（字段名可能不同，不能简单 sed）
```

- [ ] **2.4 替换 sdk/mpr/reader_documents.go 中的 sdk/security 引用**

```bash
grep -n "security\." sdk/mpr/reader_documents.go
# 手动编辑，替换函数返回类型
```

- [ ] **2.5 全量编译验证**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./sdk/mpr/...
```

- [ ] **2.6 运行测试确认通过**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./sdk/mpr/... -run TestSecurity -v
```

- [ ] **2.7 验证零残留并删包**

```bash
grep -r '"github.com/mendixlabs/mxcli/sdk/security"' . --include="*.go"
# 期望：空
git rm -r sdk/security/
```

- [ ] **2.8 提交**

```bash
git add sdk/mpr/parser_security.go sdk/mpr/reader_documents.go
git commit -m "refactor(sdk): retire sdk/security

sdk/mpr now uses <新类型路径> for security types.
Delete sdk/security/ (<N> lines)."
```

---

## Task 3: 退役 sdk/javaactions

> 在完成 Task 1.2 且确认迁移路径后执行。

受影响文件（~6 个）：`writer_javaactions.go`、`parser_javaactions.go`、`serialize_exports.go`、`parser_misc.go` 及相关测试。

- [ ] **3.1 写 javaactions roundtrip 失败测试**

在现有测试文件或新建 `sdk/mpr/parser_javaactions_test.go` 中添加验证 JavaAction 字段的测试。

- [ ] **3.2 运行确认测试通过（基线）**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./sdk/mpr/... -run TestJava -v
```

- [ ] **3.3 逐文件替换 sdk/javaactions 引用**

按此顺序（依赖少的先做）：
1. `sdk/mpr/parser_javaactions.go`
2. `sdk/mpr/writer_javaactions.go`
3. `sdk/mpr/parser_misc.go`（只替换 javaactions 相关部分）
4. `sdk/mpr/serialize_exports.go`（只替换 javaactions 相关部分）

每改一个文件后运行 `go build ./sdk/mpr/...` 验证。

- [ ] **3.4 验证零残留并删包**

```bash
grep -r '"github.com/mendixlabs/mxcli/sdk/javaactions"' . --include="*.go"
# 期望：空
git rm -r sdk/javaactions/
```

- [ ] **3.5 全量测试**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./... 2>&1 | grep -E "FAIL|ok"
```

- [ ] **3.6 提交**

```bash
git commit -m "refactor(sdk): retire sdk/javaactions

sdk/mpr now uses <新类型路径> for JavaAction types.
Delete sdk/javaactions/ (<N> lines)."
```

---

## Task 4: 退役 sdk/workflows

> 在完成 Task 1.3 且确认迁移路径后执行。

受影响文件（~4 个）：`parser_workflow.go`、`writer_workflow.go`、`serialize_exports.go`、`reader_documents.go`。

- [ ] **4.1 写 workflow roundtrip 失败测试**

在现有测试文件中添加验证 Workflow 字段的测试。

- [ ] **4.2 运行确认测试通过（基线）**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./sdk/mpr/... -run TestWorkflow -v
```

- [ ] **4.3 逐文件替换 sdk/workflows 引用**

按此顺序：
1. `sdk/mpr/parser_workflow.go`
2. `sdk/mpr/writer_workflow.go`
3. `sdk/mpr/reader_documents.go`（只替换 workflow 相关部分）
4. `sdk/mpr/serialize_exports.go`（只替换 workflow 相关部分）

每改一个文件后运行 `go build ./sdk/mpr/...` 验证。

- [ ] **4.4 验证零残留并删包**

```bash
grep -r '"github.com/mendixlabs/mxcli/sdk/workflows"' . --include="*.go"
# 期望：空
git rm -r sdk/workflows/
```

- [ ] **4.5 全量测试**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./... 2>&1 | grep -E "FAIL|ok"
```

- [ ] **4.6 提交**

```bash
git commit -m "refactor(sdk): retire sdk/workflows

sdk/mpr now uses <新类型路径> for Workflow types.
Delete sdk/workflows/ (<N> lines). Package was marked DEPRECATED
since Stage 3.3.3."
```

---

## 验收清单

```bash
# 1. 零残留
grep -r '"github.com/mendixlabs/mxcli/sdk/security"\|"sdk/javaactions"\|"sdk/workflows"' . --include="*.go"
# 期望：空

# 2. 全量编译
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./...
# 期望：无错误

# 3. 全量测试
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./...
# 期望：全 ok
```

---

## ⚠️ 重要提示：Task 1 是强制前置步骤

Task 1（类型映射调查）**必须在任何代码修改之前完成**。如果发现 modelsdk/gen 类型与 sdk 类型字段不兼容，需要先做补齐工作，否则迁移会在中途失败。

如果调查结果表明某个包的迁移路径是路径 C（完全重写解析逻辑），请将其拆分为独立的规划会话，不要在本计划内强行完成。
