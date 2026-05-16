# Wave 1: sdk/agenteditor 退役实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 删除 `sdk/agenteditor` 包，将 9 个 sdk/mpr 文件及 1 个测试文件中的 import 全部改为直接引用 `mdl/types`，零残留引用。

**Architecture:** `sdk/agenteditor/types.go` 是纯 type alias 文件（38 行），所有类型已指向 `mdl/types/agenteditor.go`。退役路径：将 sdk/mpr 内 8 个文件的 `import sdk/agenteditor` 改为 `mdl/types`，将所有 `agenteditor.X` 引用改为 `types.X`，同步更新 `mdl/types/agenteditor_test.go`，最后 `git rm sdk/agenteditor/`。

**Tech Stack:** Go 1.26，无 CGO，`GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go`

---

## 文件变动清单

| 操作 | 路径 |
|------|------|
| 修改 | `sdk/mpr/reader_agenteditor.go` — import 替换，16 处引用 |
| 修改 | `sdk/mpr/parser_customblob.go` — import 替换，25 处引用 |
| 修改 | `sdk/mpr/writer_agenteditor_agent.go` — import 替换，13 处引用 |
| 修改 | `sdk/mpr/serialize_exports.go` — import 替换，12 处引用 |
| 修改 | `sdk/mpr/writer_agenteditor_kb.go` — import 替换，8 处引用 |
| 修改 | `sdk/mpr/writer_agenteditor_model.go` — import 替换，8 处引用 |
| 修改 | `sdk/mpr/writer_agenteditor_mcpservice.go` — import 替换，7 处引用 |
| 修改 | `sdk/mpr/writer_customblob.go` — import 替换，2 处引用 |
| 修改 | `mdl/types/agenteditor_test.go` — 删除 sdkagenteditor import，更新断言 |
| 删除 | `sdk/agenteditor/types.go`（整包 git rm） |

---

## Task 1: 替换 sdk/mpr/reader_agenteditor.go

**Files:**
- Modify: `sdk/mpr/reader_agenteditor.go`

- [ ] **1.1 确认当前 import 和引用类型**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
grep -n "agenteditor\." sdk/mpr/reader_agenteditor.go
```

期望：看到 `agenteditor.Model`、`agenteditor.KnowledgeBase`、`agenteditor.ConsumedMCPService`、`agenteditor.Agent`、`agenteditor.CustomType*` 等引用。

- [ ] **1.2 替换 import**

找到文件中：
```go
"github.com/mendixlabs/mxcli/sdk/agenteditor"
```
替换为：
```go
"github.com/mendixlabs/mxcli/mdl/types"
```

注意：如果文件已有 `mdl/types` import，则只删除 agenteditor 行，不重复添加。

- [ ] **1.3 全文替换 agenteditor. 前缀**

```bash
sed -i 's/agenteditor\./types./g' sdk/mpr/reader_agenteditor.go
```

- [ ] **1.4 验证编译**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./sdk/mpr/...
```

期望：无错误（如有 "imported and not used" 错误则检查 import 块）。

---

## Task 2: 替换 sdk/mpr/parser_customblob.go

**Files:**
- Modify: `sdk/mpr/parser_customblob.go`

- [ ] **2.1 确认当前引用（25 处）**

```bash
grep -n "agenteditor\." sdk/mpr/parser_customblob.go
```

- [ ] **2.2 替换 import**

找到：
```go
"github.com/mendixlabs/mxcli/sdk/agenteditor"
```
替换为：
```go
"github.com/mendixlabs/mxcli/mdl/types"
```

- [ ] **2.3 全文替换 agenteditor. 前缀**

```bash
sed -i 's/agenteditor\./types./g' sdk/mpr/parser_customblob.go
```

- [ ] **2.4 验证编译**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./sdk/mpr/...
```

期望：无错误。

---

## Task 3: 替换 sdk/mpr/writer_agenteditor_agent.go

**Files:**
- Modify: `sdk/mpr/writer_agenteditor_agent.go`

- [ ] **3.1 确认当前引用（13 处）**

```bash
grep -n "agenteditor\." sdk/mpr/writer_agenteditor_agent.go
```

- [ ] **3.2 替换 import 和前缀**

```bash
# 替换 import 路径（用 sed 或手动编辑）
sed -i 's|"github.com/mendixlabs/mxcli/sdk/agenteditor"|"github.com/mendixlabs/mxcli/mdl/types"|g' sdk/mpr/writer_agenteditor_agent.go
sed -i 's/agenteditor\./types./g' sdk/mpr/writer_agenteditor_agent.go
```

- [ ] **3.3 验证编译**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./sdk/mpr/...
```

期望：无错误。

---

## Task 4: 替换 sdk/mpr/serialize_exports.go

**Files:**
- Modify: `sdk/mpr/serialize_exports.go`

- [ ] **4.1 确认当前引用（12 处）**

```bash
grep -n "agenteditor\." sdk/mpr/serialize_exports.go
```

期望：看到函数签名如 `SerializeAgentEditorModel(m *agenteditor.Model)` 等。

- [ ] **4.2 替换 import 和前缀**

```bash
sed -i 's|"github.com/mendixlabs/mxcli/sdk/agenteditor"|"github.com/mendixlabs/mxcli/mdl/types"|g' sdk/mpr/serialize_exports.go
sed -i 's/agenteditor\./types./g' sdk/mpr/serialize_exports.go
```

- [ ] **4.3 验证编译**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./sdk/mpr/...
```

期望：无错误。注意 serialize_exports.go 的函数签名变化（`*agenteditor.Model` → `*types.Model`）是公开接口变化，需检查调用方。

- [ ] **4.4 检查 serialize_exports 的外部调用方**

```bash
grep -rn "SerializeAgentEditor\|SerializeWorkflow" . --include="*.go" | grep -v "^sdk/mpr/"
```

期望：如有调用方，确认它们已使用 `mdl/types` 而非 `sdk/agenteditor`（已是 type alias，零影响）。

---

## Task 5: 替换剩余三个 writer 文件

**Files:**
- Modify: `sdk/mpr/writer_agenteditor_kb.go`
- Modify: `sdk/mpr/writer_agenteditor_model.go`
- Modify: `sdk/mpr/writer_agenteditor_mcpservice.go`

- [ ] **5.1 批量替换三个文件**

```bash
for f in sdk/mpr/writer_agenteditor_kb.go sdk/mpr/writer_agenteditor_model.go sdk/mpr/writer_agenteditor_mcpservice.go; do
  sed -i 's|"github.com/mendixlabs/mxcli/sdk/agenteditor"|"github.com/mendixlabs/mxcli/mdl/types"|g' "$f"
  sed -i 's/agenteditor\./types./g' "$f"
done
```

- [ ] **5.2 验证编译**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./sdk/mpr/...
```

期望：无错误。

---

## Task 6: 替换 sdk/mpr/writer_customblob.go

**Files:**
- Modify: `sdk/mpr/writer_customblob.go`

- [ ] **6.1 确认当前引用（2 处）**

```bash
grep -n "agenteditor\." sdk/mpr/writer_customblob.go
```

期望：仅 `agenteditor.CreatedByExtensionID`（一个字符串常量）。

- [ ] **6.2 替换 import 和前缀**

```bash
sed -i 's|"github.com/mendixlabs/mxcli/sdk/agenteditor"|"github.com/mendixlabs/mxcli/mdl/types"|g' sdk/mpr/writer_customblob.go
sed -i 's/agenteditor\./types./g' sdk/mpr/writer_customblob.go
```

- [ ] **6.3 验证编译**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./sdk/mpr/...
```

期望：无错误。

---

## Task 7: 更新 mdl/types/agenteditor_test.go

**Files:**
- Modify: `mdl/types/agenteditor_test.go`

- [ ] **7.1 查看当前测试内容**

```bash
grep -n "sdkagenteditor\|sdk/agenteditor" mdl/types/agenteditor_test.go | head -20
```

期望：看到 `sdkagenteditor "github.com/mendixlabs/mxcli/sdk/agenteditor"` import 和基于它的断言（验证 alias 等价性）。

- [ ] **7.2 删除 sdkagenteditor import 和相关断言**

这个测试文件导入了 `sdk/agenteditor` 来验证 type alias 关系。既然我们要删除该包，这些测试需要：
- 删除 `sdkagenteditor` import 行
- 删除或改写依赖 `sdkagenteditor.X` 的测试（这些是验证 alias 等价性的，删包后不再需要）

手动编辑 `mdl/types/agenteditor_test.go`，移除所有 `sdkagenteditor` 相关代码。

- [ ] **7.3 验证测试仍能编译和运行**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./mdl/types/... -v 2>&1 | tail -20
```

期望：全通，无 agenteditor 相关错误。

---

## Task 8: 全量编译验证

- [ ] **8.1 全量编译**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./...
```

期望：无错误。

- [ ] **8.2 确认零残留引用（sdk/agenteditor 包路径）**

```bash
grep -r '"github.com/mendixlabs/mxcli/sdk/agenteditor"' . --include="*.go"
```

期望：**空输出**。

- [ ] **8.3 全量测试**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./... 2>&1 | grep -E "FAIL|ok" | tail -20
```

期望：全部 `ok`，无 `FAIL`。

---

## Task 9: 删除 sdk/agenteditor 并提交

- [ ] **9.1 删除整包**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
git rm sdk/agenteditor/types.go
```

- [ ] **9.2 最终编译确认**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./...
```

期望：无错误（sdk/agenteditor 包不存在不影响编译）。

- [ ] **9.3 提交**

```bash
git add sdk/mpr/reader_agenteditor.go \
        sdk/mpr/parser_customblob.go \
        sdk/mpr/writer_agenteditor_agent.go \
        sdk/mpr/serialize_exports.go \
        sdk/mpr/writer_agenteditor_kb.go \
        sdk/mpr/writer_agenteditor_model.go \
        sdk/mpr/writer_agenteditor_mcpservice.go \
        sdk/mpr/writer_customblob.go \
        mdl/types/agenteditor_test.go
git commit -m "refactor(sdk): retire sdk/agenteditor — use mdl/types directly

sdk/agenteditor was a pure type-alias bridge to mdl/types (38 lines).
Update 8 files in sdk/mpr and agenteditor_test.go to import mdl/types
directly. Delete sdk/agenteditor/types.go.

Zero API changes for external callers (mdl/types was always the source
of truth; the alias layer is now removed)."
```

---

## 验收清单

```bash
# 1. 零残留
grep -r '"github.com/mendixlabs/mxcli/sdk/agenteditor"' . --include="*.go"
# 期望：空

# 2. 全量编译
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./...
# 期望：无错误

# 3. 全量测试
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./... 2>&1 | grep -E "FAIL|ok"
# 期望：全 ok
```

---

## 注意事项

- **sdk/versions 不在本计划范围内**：`modelsdk/version` 不是 `sdk/versions` 的等价替换（前者无 Registry/YAML 系统），sdk/versions 保留原位。
- **sdk/security 不在本计划范围内**：`modelsdk/gen/security` 使用 codec.Element 类型系统，与 sdk/security 的平结构体架构不同，需单独规划（Wave 2 独立设计）。
- 如果 `sed` 对某文件产生了错误的替换（例如将注释中的 `agenteditor.` 也替换了），用 `git diff` 检查后手动修正。
