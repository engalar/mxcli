# bson diff — 设计文档

**日期：** 2026-06-07  
**状态：** 已批准

---

## 背景与动机

调试 BSON 序列化问题时，常见工作流是：在 Studio Pro 里手工创建一个"参考"元素，提取其 `.mxunit` 文件作为基准，然后与 mxcli 生成的 `.mxunit` 对比，找出字段差异。

现有 `mxcli bson compare` 需要指定对象类型和名称，要求项目上下文（SQLite 查询），无法直接对比两个裸文件。用户需要一个能直接接收文件路径、输出 git-diff 风格差异的命令，结果可直接粘贴给 AI 阅读和理解。

---

## 命令设计

```bash
# 基本用法
mxcli bson diff <file1.mxunit> <file2.mxunit>

# 强制彩色（默认 auto-detect）
mxcli bson diff a.mxunit b.mxunit --color

# 纯文本（用于管道 / 粘贴给 AI）
mxcli bson diff a.mxunit b.mxunit --no-color
```

**退出码：**
- `0` — 文件相同
- `1` — 存在差异
- `2` — 错误（文件不存在、BSON 解析失败）

---

## 输出格式

NDSL unified diff，与 `git diff` 视觉一致：

```diff
--- a.mxunit
+++ b.mxunit
@@ -12,7 +12,7 @@
   Forms$Page
     Name = "P_Overview"
-    Title = "Customer Overview"
+    Title = "Customer List"
     CanClose = false
```

- `---`/`+++`/`@@` 行：青色（`colorCyan`）
- `-` 行：红色（`colorRed`）
- `+` 行：绿色（`colorGreen`）
- 上下文行：无色

---

## 架构

无项目依赖，纯文件操作：

```
os.ReadFile(file1)  os.ReadFile(file2)
       ↓                    ↓
bson.Unmarshal → bson.D   bson.Unmarshal → bson.D
       ↓                    ↓
bsondebug.Render(doc, 0)  bsondebug.Render(doc, 0)
       ↓                    ↓
           difflib.UnifiedDiff
                   ↓
         彩色 unified diff 输出
```

---

## 实现文件

| 文件 | 变更 |
|------|------|
| `cmd/mxcli/cmd_bson_diff.go` | 新建，实现 `bsonDiffCmd` |
| `cmd/mxcli/cmd_bson.go` | 注册 `bsonDiffCmd` |

**零新依赖**，复用：
- `cmd/mxcli/bson/render.go` — `bsondebug.Render()`
- `github.com/pmezard/go-difflib/difflib` — `UnifiedDiff` / `GetUnifiedDiffString`
- `go.mongodb.org/mongo-driver/v2/bson` — `Unmarshal`
- executor 的 `colorGreen/colorRed/colorCyan/colorReset` 常量（或直接内联 ANSI 码）

---

## 颜色自动检测

优先级：
1. `--color` flag → 强制彩色
2. `--no-color` flag → 强制无色
3. `NO_COLOR` 环境变量（标准，见 no-color.org）→ 无色
4. `os.Stdout` 是否为终端（`term.IsTerminal`）→ 自动判断

---

## 范围界定

**包含：**
- 两个 `.mxunit` 文件的 NDSL unified diff
- 彩色/无色输出控制
- 文件不存在/BSON 解析失败的清晰错误信息

**不包含：**
- 目录递归 diff（超出本次范围）
- TUI 集成（可后续添加）
- JSON 格式输出（NDSL 对 AI 更友好，JSON 可用现有 `bson dump` 获取）
