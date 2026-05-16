# PR5 Phase 2: 切换 sdkReader + 删除 Bridge + 删除 sdk/mpr

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 切换 sdkReader 类型别名到 modelsdk/mpr，删除所有 bridge 文件，最终删除 sdk/mpr 目录。

**Architecture:** 前提条件：Phase 1 已完成（modelsdk/mpr.Reader 有全部 48 个方法）。本 Phase 纯粹是切换和清理，无新业务逻辑。

**Tech Stack:** Go 1.26，`GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go`

**前提条件验证：**
```bash
# Phase 1 完成标志
grep -c "^func (r \*Reader)" modelsdk/mpr/reader_documents.go modelsdk/mpr/reader_raw.go
# 期望：≥ 30 个方法
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./...
# 期望：全 ok
```

---

## Task 1: 切换 sdkReader 类型别名

**Files:**
- Modify: `mdl/backend/mpr/sdkmpr_bridge.go`

- [ ] **1.1 读取当前 sdkmpr_bridge.go 的 sdkReader 定义**

```bash
grep -n "sdkReader\|sdkOpenReader\|sdkmpr\." mdl/backend/mpr/sdkmpr_bridge.go | head -15
```

期望看到：
```
type sdkReader = sdkmpr.Reader
func sdkOpenReader(path string) (*sdkReader, error) {
    return sdkmpr.OpenWithOptions(path, sdkmpr.OpenOptions{ReadOnly: false})
}
```

- [ ] **1.2 替换为 modelsdk/mpr 版本**

找到（约第 75-85 行）：
```go
// sdkReader is a type alias for sdk/mpr.Reader
type sdkReader = sdkmpr.Reader

// sdkOpenReader opens a project MPR file for read-write access
func sdkOpenReader(path string) (*sdkReader, error) {
    return sdkmpr.OpenWithOptions(path, sdkmpr.OpenOptions{ReadOnly: false})
}
```

替换为：
```go
// sdkReader is now a type alias for modelsdk/mpr.Reader.
// sdk/mpr.Reader is no longer used as the backend reader.
type sdkReader = modelsdkmpr.Reader

// sdkOpenReader opens a project MPR file for read-write access using modelsdk/mpr.
func sdkOpenReader(path string) (*sdkReader, error) {
    return modelsdkmpr.Open(path)
}
```

其中 `modelsdkmpr` 已在文件顶部 import（验证：`grep modelsdkmpr mdl/backend/mpr/sdkmpr_bridge.go`）。

- [ ] **1.3 从 sdkmpr_bridge.go 删除 sdkmpr import（如果不再需要）**

检查是否还有其他 `sdkmpr.` 调用：
```bash
grep "sdkmpr\." mdl/backend/mpr/sdkmpr_bridge.go
```

期望：空（所有 Serialize/Patch 都已改为 modelsdkmpr）。若为空，删除 import 行：
```go
// 删除这行：
sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
```

- [ ] **1.4 编译验证**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./mdl/backend/mpr/...
```

若有编译错误（某个 `b.reader.*` 调用返回类型变化），在 `convert_reader.go` 补充对应的转换函数，在 `backend.go` 加包装。

- [ ] **1.5 全量编译**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./...
```

- [ ] **1.6 全量测试**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./... 2>&1 | grep -E "FAIL|ok" | tail -15
```

期望：全 ok。

- [ ] **1.7 提交**

```bash
git add mdl/backend/mpr/sdkmpr_bridge.go
git commit -m "refactor(sdk/mpr): switch sdkReader from sdk/mpr to modelsdk/mpr — core alias change"
```

---

## Task 2: 删除 sdkmpr_bridge.go

**Files:**
- Modify: `mdl/backend/mpr/backend.go` （将 bridge 内联）
- Delete: `mdl/backend/mpr/sdkmpr_bridge.go`

- [ ] **2.1 确认 sdkmpr_bridge.go 内容**

```bash
cat mdl/backend/mpr/sdkmpr_bridge.go
```

- [ ] **2.2 将 sdkOpenReader 内联到 backend.go 的 Connect 方法**

找到 backend.go 的 `Connect()` 函数：
```go
func (b *MprBackend) Connect(path string) error {
    r, err := sdkOpenReader(path)
    ...
```

替换为：
```go
func (b *MprBackend) Connect(path string) error {
    r, err := modelsdkmpr.Open(path)
    ...
```

- [ ] **2.3 将所有 sdkSerialize* / sdkPatch* 调用改为直接调用 modelsdkmpr**

```bash
# 找到所有使用 bridge 函数的文件
grep -rn "sdkSerialize\|sdkPatch\|sdkOpenReader\|sdkReader\b" mdl/backend/mpr/ --include="*.go" | grep -v "sdkmpr_bridge.go"
```

对每处调用：将 `sdkSerializeProjectSettings(ps)` 改为 `modelsdkmpr.SerializeProjectSettings(ps)`，将 `sdkPatchNavigationProfile(...)` 改为 `modelsdkmpr.PatchNavigationProfile(...)` 等。

- [ ] **2.4 删除 sdkmpr_bridge.go**

```bash
git rm mdl/backend/mpr/sdkmpr_bridge.go
```

- [ ] **2.5 编译验证**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./mdl/backend/mpr/...
```

- [ ] **2.6 提交**

```bash
git add mdl/backend/mpr/backend.go mdl/backend/mpr/*.go
git commit -m "refactor(sdk/mpr): inline sdkmpr_bridge.go — direct modelsdkmpr calls"
```

---

## Task 3: 删除 repos/sdk_bridge.go

**Files:**
- Modify: `mdl/backend/mpr/repos/reference.go`
- Delete: `mdl/backend/mpr/repos/sdk_bridge.go`

- [ ] **3.1 查看 repos/sdk_bridge.go 内容**

```bash
cat mdl/backend/mpr/repos/sdk_bridge.go
```

- [ ] **3.2 将 sdkPatchNavigationProfile 调用内联到 reference.go**

找到 reference.go 中：
```go
patched, err := sdkPatchNavigationProfile(rawBytes, profileName, types.NavigationProfileSpec(spec))
```

替换为：
```go
patched, err := modelsdkmpr.PatchNavigationProfile(rawBytes, profileName, spec)
```

在 reference.go 的 import 块加（如不存在）：
```go
modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
```

- [ ] **3.3 将 sdkOpenBSONScanner 内联**

找到 reference.go 中调用 `sdkOpenBSONScanner` 的地方（若在 reference_test.go）：
```bash
grep -rn "sdkOpenBSONScanner" mdl/backend/mpr/repos/
```

将 `sdkOpenBSONScanner(path)` 替换为：
```go
r, err := modelsdkmpr.Open(path)
if err != nil { return nil, func() {}, err }
scanner := types.BSONScanner(r)
return scanner, func() { _ = r.Close() }, nil
```

- [ ] **3.4 删除 repos/sdk_bridge.go**

```bash
git rm mdl/backend/mpr/repos/sdk_bridge.go
```

- [ ] **3.5 编译验证**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./mdl/backend/mpr/...
```

- [ ] **3.6 提交**

```bash
git add mdl/backend/mpr/repos/reference.go mdl/backend/mpr/repos/reference_test.go
git commit -m "refactor(sdk/mpr): delete repos/sdk_bridge.go — inline into reference.go"
```

---

## Task 4: 删除 bson_reader_bridge.go

**Files:**
- Modify: `cmd/mxcli/cmd_bson_compare.go`, `cmd_bson_discover.go`, `cmd_bson_dump.go`
- Modify: `cmd/mxcli/cmd_extract_templates.go`
- Modify: `cmd/mxcli/project_tree.go`
- Delete: `cmd/mxcli/bson_reader_bridge.go`

- [ ] **4.1 查看 bson_reader_bridge.go 中的接口和工厂函数**

```bash
cat cmd/mxcli/bson_reader_bridge.go
```

期望看到：bsonReader / widgetReader / projectTreeReader 3 个接口 + openBSONReader / openWidgetReader / openProjectTreeReader 3 个工厂函数。

- [ ] **4.2 在各 cmd 文件中替换接口调用为直接的 modelsdk/mpr.Reader**

对 `cmd_bson_compare.go`、`cmd_bson_discover.go`、`cmd_bson_dump.go`：

```bash
# 确认当前调用
grep "openBSONReader\|bsonReader\b" cmd/mxcli/cmd_bson_compare.go cmd/mxcli/cmd_bson_discover.go cmd/mxcli/cmd_bson_dump.go
```

将 `openBSONReader(path)` 替换为 `modelsdkmpr.Open(path)`，将 `bsonReader` 类型替换为 `*modelsdkmpr.Reader`，在 import 块加：
```go
modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
```

对 `cmd_extract_templates.go`：
将 `openWidgetReader(path)` 替换为 `modelsdkmpr.Open(path)`。

对 `project_tree.go`：
将 `openProjectTreeReader(path)` 替换为 `modelsdkmpr.Open(path)`，变量类型改为 `*modelsdkmpr.Reader`（或使用 `:=` 推断）。

- [ ] **4.3 删除 bson_reader_bridge.go**

```bash
git rm cmd/mxcli/bson_reader_bridge.go
```

- [ ] **4.4 全量编译**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./cmd/mxcli/...
```

若有 "method undefined" 错误，说明 modelsdk/mpr.Reader 某个方法还未实现，回到 Phase 1 补充。

- [ ] **4.5 提交**

```bash
git add cmd/mxcli/
git commit -m "refactor(sdk/mpr): delete bson_reader_bridge.go — direct modelsdk/mpr.Reader usage"
```

---

## Task 5: 删除 sdk/mpr 目录

**Files:**
- Delete: `sdk/mpr/`（整目录）

- [ ] **5.1 确认零残留 sdk/mpr import**

```bash
grep -r '"github.com/mendixlabs/mxcli/sdk/mpr"' . --include="*.go"
```

期望：**空输出**。若有残留，回前几步修复。

- [ ] **5.2 删除整目录**

```bash
git rm -r sdk/mpr/
```

- [ ] **5.3 全量编译**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./...
```

期望：无错误。

- [ ] **5.4 全量测试**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./... 2>&1 | grep -E "FAIL|ok" | tail -15
```

期望：全 ok，无 FAIL。

- [ ] **5.5 提交**

```bash
git commit -m "feat(sdk): retire sdk/mpr — delete entire directory

All sdk/mpr functionality has been migrated to modelsdk/mpr:
- Reader methods: all 48 new methods in modelsdk/mpr/reader_documents.go
  and reader_raw.go return gen/* types
- Pure functions: Serialize*, Patch* now in modelsdk/mpr/serialize_*.go,
  nav_patch.go, security_patch.go
- BSONScanner: scanner.go implements ScanRenameReferences and
  ScanQualifiedNameUpdates natively
- All bridge files deleted

sdk/ now contains only: mpr/version (deferred), versions (non-redundant)"
```

---

## Task 6: 善后清理

**Files:**
- Modify: `modelsdk.go` — 删除 sdk/mpr 注释残留
- Modify: `CLAUDE.md` — 更新架构描述
- Modify: `.claude/skills/migrate-sdk-to-modelsdk/SKILL.md` — 标记完成状态

- [ ] **6.1 更新 modelsdk.go 注释**

```bash
grep -n "sdk/mpr" modelsdk.go
```

删除所有引用 `sdk/mpr` 的注释行。

- [ ] **6.2 更新 CLAUDE.md 中的架构说明**

找到 `sdk/mpr/` 相关描述，更新为：
```markdown
sdk/ 目录现在只包含 versions/（版本注册表，供 cmd_features.go 使用）。
sdk/mpr 已于 2026-05-16 完成退役，所有功能迁移到 modelsdk/mpr/。
```

- [ ] **6.3 最终终态验证**

```bash
# 1. sdk/mpr 目录不存在
ls sdk/  # 期望：只有 versions/

# 2. 全局零 sdk/mpr import
grep -r '"github.com/mendixlabs/mxcli/sdk/mpr"' . --include="*.go"
# 期望：空

# 3. 全量编译
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./...
# 期望：无错误

# 4. 全量测试
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./...
# 期望：全 ok
```

- [ ] **6.4 提交**

```bash
git add modelsdk.go CLAUDE.md .claude/skills/migrate-sdk-to-modelsdk/SKILL.md
git commit -m "docs: update architecture docs — sdk/mpr retired, modelsdk/mpr is the sole reader"
```

---

## 验收清单（Phase 2 完成）

```bash
ls sdk/           # 只有 versions/
grep -r '"github.com/mendixlabs/mxcli/sdk/mpr"' . --include="*.go"  # 空
go build ./...    # 无错误
go test ./...     # 全 ok
```
