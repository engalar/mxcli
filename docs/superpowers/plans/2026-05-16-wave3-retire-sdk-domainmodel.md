# Wave 3: sdk/domainmodel 退役实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 退役 `sdk/domainmodel` 包，使 `sdk/mpr` 内部改用 `modelsdk/gen/domainmodels` 类型，最终 git rm。

**Architecture:** `sdk/domainmodel` 是 sdk/mpr 内部用于 BSON 解析的类型包（607行）。`modelsdk/gen/domainmodels/` 有完整实现（5061行，codec.Element 风格）。迁移需要先调查两侧类型字段对应关系，再逐文件改写解析逻辑。

**Tech Stack:** Go 1.26，`GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go`

**前提条件：** Wave 1（sdk/agenteditor）和 Wave 2（security/javaactions/workflows）已完成。

---

## 文件变动清单（待调查后确认）

| 操作 | 路径 |
|------|------|
| 修改 | `sdk/mpr/parser_domainmodel.go` |
| 修改 | `sdk/mpr/writer_domainmodel.go` |
| 修改 | `sdk/mpr/system_module.go` |
| 修改 | `sdk/mpr/writer_modules.go` |
| 修改 | `sdk/mpr/writer_units.go` |
| 修改 | `sdk/mpr/reader_documents.go`（domainmodel 相关函数） |
| 修改 | 相关测试文件（_test.go） |
| 删除 | `sdk/domainmodel/`（整包） |

---

## Task 1: 类型映射调查（前置，必须完成才能继续）

- [ ] **1.1 对比 sdk/domainmodel 与 modelsdk/gen/domainmodels 的类型**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02

# 查看 sdk/domainmodel 的所有类型和字段
cat sdk/domainmodel/domainmodel.go

# 查看 modelsdk/gen/domainmodels/types.go 的类型列表
grep -n "^type " modelsdk/gen/domainmodels/types.go | head -40

# 对比关键类型字段：Entity
grep -A 20 "type Entity struct" sdk/domainmodel/domainmodel.go
grep -n "entityParts\|Attributes\|Associations\|Documentation" modelsdk/gen/domainmodels/types.go | head -20

# 对比关键类型字段：Association
grep -A 20 "type Association struct" sdk/domainmodel/domainmodel.go
```

- [ ] **1.2 调查 parser_domainmodel.go 的解析逻辑**

```bash
cat sdk/mpr/parser_domainmodel.go
```

记录：parser 产出的是 `sdk/domainmodel.DomainModel`（平结构体）还是已经是 gen 类型？

- [ ] **1.3 调查 mdl/backend/mpr 中的 domainmodel 消费方**

```bash
# mdl/backend/mpr 中谁使用 sdk/mpr 读取的 domainmodel 类型？
grep -rn "GetDomainModel\|ListEntities\|sdk/domainmodel" mdl/backend/mpr/ --include="*.go" | head -20
```

记录：mdl/backend/mpr 是否直接使用 sdk/domainmodel 类型，还是通过转换函数？

- [ ] **1.4 决定迁移路径**

基于以上调查，选择：
- **路径 A**：gen 类型与 sdk 类型字段对等 → 直接替换 import 和类型名
- **路径 B**：gen 类型缺少字段 → 先补 gen 类型，再做路径 A
- **路径 C**：解析逻辑需要完全重写（sdk 解析到平结构体，gen 使用 codec.Element）

---

## Task 2: 建立 DomainModel roundtrip 测试基线

- [ ] **2.1 写 domainmodel roundtrip 失败测试**

在 `sdk/mpr/parser_domainmodel_test.go`（新建或追加）中：

```go
func TestDomainModelRoundtrip(t *testing.T) {
    // 验证读取并重新序列化后字段不变
    // 根据 Task 1 调查结果填写具体字段断言
    // 至少验证：Entity.Name, Entity.Attributes 数量, Association.Name
}
```

- [ ] **2.2 运行确认测试当前通过（基线）**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./sdk/mpr/... -run TestDomainModel -v
```

---

## Task 3: 迁移 sdk/mpr/parser_domainmodel.go

- [ ] **3.1 替换 import 和类型**

根据 Task 1 选定的路径执行。每次只改一个函数，改完立即编译：

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./sdk/mpr/...
```

- [ ] **3.2 运行 roundtrip 测试确认通过**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./sdk/mpr/... -run TestDomainModel -v
```

---

## Task 4: 迁移 sdk/mpr/writer_domainmodel.go

- [ ] **4.1 替换 import 和类型**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./sdk/mpr/...
```

---

## Task 5: 迁移剩余文件

按此顺序，每改完一个文件立即编译验证：

- [ ] **5.1** `sdk/mpr/system_module.go` — 替换 domainmodel 引用
- [ ] **5.2** `sdk/mpr/writer_modules.go` — 替换 domainmodel 引用
- [ ] **5.3** `sdk/mpr/writer_units.go` — 替换 domainmodel 引用
- [ ] **5.4** `sdk/mpr/reader_documents.go` — 只替换 domainmodel 相关函数

每步验证：
```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./sdk/mpr/...
```

---

## Task 6: 全量验证、删包、提交

- [ ] **6.1 验证零残留**

```bash
grep -r '"github.com/mendixlabs/mxcli/sdk/domainmodel"' . --include="*.go"
# 期望：空
```

- [ ] **6.2 全量编译**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./...
```

- [ ] **6.3 全量测试**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./... 2>&1 | grep -E "FAIL|ok"
```

- [ ] **6.4 删包**

```bash
git rm -r sdk/domainmodel/
```

- [ ] **6.5 提交**

```bash
git commit -m "refactor(sdk): retire sdk/domainmodel

sdk/mpr now uses modelsdk/gen/domainmodels types for entity/association
parsing. Delete sdk/domainmodel/ (<N> lines)."
```

---

## ⚠️ 注意

Task 1 的调查结果决定了 Tasks 3-5 的具体代码。如果发现解析逻辑需要路径 C（完全重写），`parser_domainmodel.go` 的改写可能需要单独拆分为多个子任务。不要跳过 Task 1 直接开始写代码。
