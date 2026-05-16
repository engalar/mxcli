# Wave 4: sdk/pages 退役实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 退役 `sdk/pages` 包（1470行），使 `sdk/mpr` 内部改用 `modelsdk/gen/pages` 类型，最终 git rm。

**Architecture:** `sdk/pages` 是 sdk/mpr 内部用于 widget/page BSON 解析的类型包。`modelsdk/gen/pages/` 有完整实现（36,327行，codec.Element 风格）。这是最复杂的一波，涉及约 16 个 sdk/mpr 文件。已有 `mdl-examples/widget-roundtrip/` 作为端到端安全网。

**Tech Stack:** Go 1.26，`GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go`

**前提条件：** Waves 1-3 全部完成。

---

## 文件变动清单（~16 个，待调查后确认）

| 操作 | 路径 |
|------|------|
| 修改 | `sdk/mpr/reader_widgets.go` |
| 修改 | `sdk/mpr/writer_pages.go` |
| 修改 | `sdk/mpr/writer_widgets.go` |
| 修改 | `sdk/mpr/writer_widgets_custom.go` |
| 修改 | `sdk/mpr/writer_widgets_input.go` |
| 修改 | `sdk/mpr/writer_widgets_layout.go` |
| 修改 | `sdk/mpr/writer_widgets_display.go` |
| 修改 | `sdk/mpr/writer_widgets_action.go` |
| 修改 | `sdk/mpr/parser_page.go` |
| 修改 | `sdk/mpr/parser_misc.go`（pages 相关部分） |
| 修改 | `sdk/mpr/reader_types.go` |
| 修改 | `sdk/mpr/reader_documents.go`（pages 相关函数） |
| 修改 | 相关测试文件 |
| 删除 | `sdk/pages/`（整包） |

---

## Task 1: 类型映射调查（前置，必须完成才能继续）

- [ ] **1.1 列出 sdk/pages 的所有导出类型**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
grep -n "^type " sdk/pages/*.go | head -60
```

- [ ] **1.2 列出 modelsdk/gen/pages 的对应类型**

```bash
grep -n "^type " modelsdk/gen/pages/types.go | head -60
ls modelsdk/gen/pages/
```

- [ ] **1.3 调查最复杂的 parser_page.go 的解析逻辑**

```bash
head -100 sdk/mpr/parser_page.go
# 重点：它产出 sdk/pages.Page 还是其他结构？
# 关键问题：parser 是否使用 sdk/pages 类型作为中间层？
```

- [ ] **1.4 检查 writer_widgets 系列文件的 sdk/pages 使用方式**

```bash
for f in sdk/mpr/writer_widgets*.go; do
  echo "=== $f ==="
  grep -n "pages\." "$f" | head -10
done
```

- [ ] **1.5 确认 mdl-examples/widget-roundtrip/ 测试可运行（端到端安全网）**

```bash
ls mdl-examples/widget-roundtrip/
# 确认这些 MDL 脚本存在，将作为迁移后的验收测试
```

- [ ] **1.6 决定分批迁移策略**

由于涉及 ~16 个文件，按以下分批原则：
1. **批次 1**：只读文件（reader_*.go, parser_*.go）——先迁移读取端
2. **批次 2**：写入文件（writer_*.go）——再迁移写入端
3. **批次 3**：全量端到端测试

---

## Task 2: 建立 widget roundtrip 测试基线

- [ ] **2.1 确认现有 widget roundtrip 测试通过**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./sdk/mpr/... -v 2>&1 | grep -E "PASS|FAIL|ok" | tail -20
```

- [ ] **2.2 添加页面解析 roundtrip 测试（如果不存在）**

在 `sdk/mpr/parser_page_test.go` 中添加：

```go
func TestPageRoundtrip(t *testing.T) {
    // 验证读取并重新序列化后 widget 树结构不变
    // 至少验证：Page.Widgets 数量, Layout.Name, DataGrid2 widget ID
    // 根据 Task 1 调查结果填写具体断言
}
```

- [ ] **2.3 运行确认基线通过**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./sdk/mpr/... -run TestPage -v
```

---

## Task 3: 迁移读取端（reader/parser 文件）

批次 1：每改一个文件立即编译验证。

- [ ] **3.1** `sdk/mpr/parser_page.go` — 替换 sdk/pages 引用，改为 gen 类型
- [ ] **3.2** `sdk/mpr/reader_types.go` — 替换 sdk/pages 引用
- [ ] **3.3** `sdk/mpr/reader_widgets.go` — 替换 sdk/pages 引用
- [ ] **3.4** `sdk/mpr/parser_misc.go`（pages 相关部分）
- [ ] **3.5** `sdk/mpr/reader_documents.go`（pages 相关函数）

每步验证：
```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./sdk/mpr/...
```

---

## Task 4: 迁移写入端（writer 文件）

批次 2：每改一个文件立即编译验证。

- [ ] **4.1** `sdk/mpr/writer_pages.go`
- [ ] **4.2** `sdk/mpr/writer_widgets.go`
- [ ] **4.3** `sdk/mpr/writer_widgets_custom.go`
- [ ] **4.4** `sdk/mpr/writer_widgets_input.go`
- [ ] **4.5** `sdk/mpr/writer_widgets_layout.go`
- [ ] **4.6** `sdk/mpr/writer_widgets_display.go`
- [ ] **4.7** `sdk/mpr/writer_widgets_action.go`

每步验证：
```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./sdk/mpr/...
```

---

## Task 5: 全量验证、删包、提交

- [ ] **5.1 验证零残留**

```bash
grep -r '"github.com/mendixlabs/mxcli/sdk/pages"' . --include="*.go"
# 期望：空
```

- [ ] **5.2 全量编译**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./...
```

- [ ] **5.3 全量测试**

```bash
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./... 2>&1 | grep -E "FAIL|ok"
```

- [ ] **5.4 端到端 widget roundtrip 验证（如有真实 MPR 项目）**

```bash
# 在有 .mpr 项目的环境中运行：
mxcli exec mdl-examples/widget-roundtrip/*.mdl -p /path/to/app.mpr
# 期望：DataGrid2 等 pluggable widget 正常写入，无 CE0463 错误
```

- [ ] **5.5 删包**

```bash
git rm -r sdk/pages/
```

- [ ] **5.6 提交**

```bash
git commit -m "refactor(sdk): retire sdk/pages — switch to modelsdk/gen/pages

sdk/mpr now uses modelsdk/gen/pages types for page/widget parsing.
Delete sdk/pages/ (1470 lines).
sdk/mpr/ is now free of sdk/pages dependency."
```

---

## 验收清单

```bash
# 1. 零残留
grep -r '"github.com/mendixlabs/mxcli/sdk/pages"' . --include="*.go"
# 期望：空

# 2. 全量编译
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go build ./...
# 期望：无错误

# 3. 全量测试
GOPATH=~/go1.26 GOMODCACHE=~/go1.26/pkg/mod GOPROXY=https://goproxy.cn,direct ~/go1.26/bin/go test ./...
# 期望：全 ok

# 4. widget roundtrip（如有真实项目）
mxcli exec mdl-examples/widget-roundtrip/*.mdl -p /path/to/app.mpr
# 期望：成功
```

---

## ⚠️ 注意

- Wave 4 是 4 个 Wave 中最复杂的，务必在 Wave 3 全部通过后才开始
- Task 1 的类型映射调查是关键——sdk/pages 的类型系统（interface-heavy，50+ widget 类型）与 modelsdk/gen/pages（codec.Element 风格）差异较大，可能需要路径 C（重写解析逻辑）
- 如果某个 writer 文件超过 300 行，考虑拆分为独立 PR

完成 Wave 4 后，`sdk/mpr` 将不再依赖任何 sdk/* 小包，为 PR5（sdk/mpr 本体退役）解锁。
