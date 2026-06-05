# Widget 生命周期改进 — 设计文档

**日期：** 2026-06-05  
**来源：** study 项目端到端验证（mxcli new → widget 开发 → 应用构建运行）  
**参考仓库：** https://github.com/engalar/mxcli-taskdemo — 包含完整 widget 生命周期演示

---

## 背景

在 `mxcli-taskdemo` 端到端验证中（新建项目 → 开发自定义 widget → MDL 集成 → 本地运行），发现 9 个问题。本文档覆盖其中可通过代码或文档修复的 8 个（Issue 9 首次启动 NPE 需独立排查）。

按实现方式分三批：

| 批次 | Issues | 类型 |
|------|--------|------|
| Batch 1 | 2, 6 | 代码 Bug |
| Batch 2 | 1, 4, 8 + mxcli-taskdemo 引用 | Skill/文档 |
| Batch 3 | 3, 5/7 | 行为变更 |

---

## Batch 1：代码 Bug 修复

### Issue 6 — package.xml xmlns 大小写错误（1 行修复）

**文件：** `cmd/mxcli/widget_scaffold.go:299`

**根因：** `generatePackageXML` 模板把 `clientModule` 的 xmlns 写成小写 `clientmodule/1.0/`，XML namespace 大小写敏感，导致 Mendix CE0462（widget 找不到）。

**mxcli-taskdemo 证据：** `StudyWidgets/package.xml` 使用正确大小写：
```xml
<clientModule name="StudyWidgets" version="1.0.0"
              xmlns="http://www.mendix.com/clientModule/1.0/">
```

**修复：**
```diff
- xmlns="http://www.mendix.com/clientmodule/1.0/"
+ xmlns="http://www.mendix.com/clientModule/1.0/"
```

**测试：** `widget_scaffold_test.go` 断言生成的 package.xml 包含正确大小写 namespace。

---

### Issue 2 — `--dir` 路径重复（esbuild 入口路径被拼接两次）

**文件：** `cmd/mxcli/widget_build.go`，`runWidgetBuild` 函数

**根因：** `compileWidget` 里 `cmd.Dir = projectDir`（设工作目录为相对路径如 `./StudyWidgets`），同时 `src = filepath.Join(projectDir, "src", ...)` 生成相对于 CWD 的路径 `./StudyWidgets/src/Widget.jsx`。esbuild 在 `cmd.Dir=./StudyWidgets` 下解析 `./StudyWidgets/src/...`，路径被拼接两次。

**修复：** 在 `runWidgetBuild` 入口将 `dir` 转为绝对路径：
```go
dir, _ := cmd.Flags().GetString("dir")
if dir == "" {
    dir = "."
}
dir, _ = filepath.Abs(dir)   // 新增：确保绝对路径，消除 cmd.Dir + src 的叠加
```

**临时绕过（已知）：** `cd StudyWidgets && mxcli widget build`（不传 `--dir`）。

**测试：** 新增 `--dir` 相对路径场景的单元测试，验证传入 `compileWidget` 的 `src` 是绝对路径。

---

## Batch 2：Skill / 文档更新

### Issue 1 — mxcli binary 检测逻辑

**背景：** CLAUDE.md 多处提到 `./mxcli -p project.mpr`，但 `mxcli new` / `mxcli init` 不自动生成该 binary。`./mxcli` 仅在用户手动运行 `mxcli setup mxcli --os linux` 后才存在（devcontainer 场景）。

**设计：** 不改代码行为，改文档逻辑。在 `check-syntax.md`（最高频 skill）顶部加入 binary 探测规则：

```bash
# 自动探测可用的 mxcli binary（不感知 devcontainer，感知 binary 是否存在）
if   [ -f ./mxcli ];               then MXCLI=./mxcli   # devcontainer: setup mxcli 下载的 Linux binary
elif command -v mxcli &>/dev/null; then MXCLI=mxcli     # Windows/Mac 全局安装
else echo "请先运行: mxcli setup mxcli --os linux" && exit 1; fi
```

CLAUDE.md `mxcli setup mxcli` 条目补注：
> Windows/Mac：mxcli 全局安装，直接用 `mxcli`。  
> Devcontainer：下载 Linux binary 到 `./mxcli`，使用 `./mxcli`。  
> Skill 会自动探测哪个可用，无需手动区分。

---

### Issue 4 — PLUGGABLEWIDGET attribute 绑定语法（`create-page.md`）

**验证结论：** 经代码分析（`buildPropertyValueV3`，`visitor_page_v3.go:1028`）和 mxcli-taskdemo 真实用例确认，attribute 绑定使用**裸属性名**，不加 `@` 或实体前缀。

**mxcli-taskdemo 证据（`TaskDemo/mdlsource/02-pages.mdl`）：**

```sql
-- 场景 1：DataView 内（可编辑）
dataview dvTask (datasource: $Task) {
  PLUGGABLEWIDGET 'com.mendix.widget.custom.PrioritySelector.PrioritySelector' wPriority (
    priority: Priority,   -- 裸属性名，与 widget XML key="priority" 一致（大小写敏感）
    editable: true        -- boolean 直接写 true/false
  )
  PLUGGABLEWIDGET 'com.mendix.widget.custom.ProgressRing.ProgressRing' wProgressRing (
    progress: Progress, size: 'large', showLabel: true
  )
}

-- 场景 2：DataGrid 列内（只读）
column colPriority (caption: 'Priority', ShowContentAs: customContent) {
  PLUGGABLEWIDGET 'com.mendix.widget.custom.PrioritySelector.PrioritySelector' wColPriority (
    priority: Priority, editable: false
  )
}
```

**`create-page.md` 补充内容（加入 PLUGGABLEWIDGET 章节）：**

1. attribute 类型属性用裸属性名，与 dataview 内的 `attribute: Priority` 写法一致
2. property key 名称来自 widget XML 的 `key=` 属性（大小写敏感）
3. `mxcli widget list -p app.mpr`（CLI）列出所有已安装 widget（含未实例化）；`SHOW WIDGETS`（MDL）只列出页面中已实例化的 widget——两者用途不同
4. 参考：`github.com/engalar/mxcli-taskdemo` — `TaskDemo/mdlsource/02-pages.mdl`

---

### Issue 8 — `mxcli local run -p` 必填（文档 + 程序输出）

**现状：** 错误输出 `"either -p or --pad-dir is required"`，无示例，无下一步引导。

**运行逻辑（经代码验证）：**
- `mxcli local build`：有 Studio Pro → 输出到 `deployment/`；无 → 输出到 `.docker/build/`
- `mxcli local run -p app.mpr`：自动探测两种产物目录，用户无需感知具体路径

**程序输出改进（`cmd/mxcli-local/cmd_run.go:47`）：**
```
Error: -p is required

Specify the .mpr file path so mxcli can find the build output:

  mxcli local run -p /path/to/app.mpr [--admin-password pw]

If you haven't built yet, run first:

  mxcli local build -p /path/to/app.mpr
```

**文档改进（`run-app.md`）：** 在"Local Path"开头加一行：
> `-p <path/to/app.mpr>` 对所有 `mxcli local` 命令均为**必填**，不支持自动检测项目路径。始终通过 `mxcli local`（launcher）调用，不要直接调用 `~/.mxcli/local/mxcli-local`。

---

### mxcli-taskdemo 引用强化

**在 `create-custom-widget.md`（创建 widget 阶段）加入完整参考章节：**

> **完整参考：** github.com/engalar/mxcli-taskdemo
>
> 该项目演示了从 scaffold 到运行的完整 widget 生命周期：
>
> | 文件 | 内容 |
> |------|------|
> | `StudyWidgets/src/PrioritySelector.xml` | attribute + boolean 属性声明 |
> | `StudyWidgets/src/ProgressRing.xml` | attribute + string + boolean 混用 |
> | `StudyWidgets/src/PrioritySelector.jsx` | attribute 双向绑定（`priority.value` / `priority.setValue`） |
> | `StudyWidgets/src/ProgressRing.jsx` | 只读 attribute（`progress.value`）+ SVG 渲染 |
> | `StudyWidgets/package.xml` | 多 widget 单包结构（xmlns 大小写：`clientModule/1.0/`，M 大写） |
> | `TaskDemo/mdlsource/02-pages.mdl` | DataView + DataGrid 列两种使用场景 |
>
> 构建命令（必须在包目录内运行）：
> ```bash
> cd StudyWidgets
> mxcli widget build        # 不能用 --dir（已知 bug，修复中）
> cp StudyWidgets.mpk ../TaskDemo/widgets/
> ```

---

## Batch 3：行为变更

### Issue 3 — `SHOW INSTALLED WIDGETS` 新命令

**背景：** `SHOW WIDGETS`（MDL）查 catalog 实例化记录；`mxcli widget list -p`（CLI）扫描 `widgets/*.mpk`。AI agent 在纯 MDL 上下文中无法感知已安装但未实例化的 widget。

**设计：**

新增 MDL 命令：
```sql
SHOW INSTALLED WIDGETS;
```

- 需要连接项目（`-p`）以定位 `widgets/` 目录
- 不需要 `refresh catalog`，直接扫描 `.mpk` 文件
- 复用已有 `WidgetRegistry.MPKDiscovered()` 逻辑
- 输出格式：

```
MPK File              Widget ID                                                    Display Name
StudyWidgets.mpk      com.mendix.widget.custom.PrioritySelector.PrioritySelector  Priority Selector
StudyWidgets.mpk      com.mendix.widget.custom.ProgressRing.ProgressRing          Progress Ring

MDL 用法: PLUGGABLEWIDGET '<Widget ID>' name (prop: val, ...)
参考: github.com/engalar/mxcli-taskdemo — TaskDemo/mdlsource/02-pages.mdl
```

**实现要点：**
- Grammar：新增 `SHOW INSTALLED WIDGETS` 规则（`MDLParser.g4`）
- AST：`ShowInstalledWidgetsStmt`
- Executor：`execShowInstalledWidgets`，从 `ctx.Backend.GetProjectDir()` 获取路径，调用 `WidgetRegistry.SetProjectDir()` + `MPKDiscovered()`
- Backend 接口：若 `GetProjectDir()` 不存在则新增

---

### Issue 5/7 — 重复 module role 导致 CE1613（根因修复）

**根因（代码证据）：**

`addModuleRoleViaModelsdk`（`mdl/backend/mpr/security_module_modelsdk.go:24`）是纯追加，无去重：
```go
ms.AddModuleRoles(mr)  // 始终 append，不检查同名
```

`execCreateModuleRoleGen`（`mdl/executor/cmd_security_write_modulerole_gen.go:56`）在检测到 auto-provisioned 角色时，直接调用 `AddModuleRole` 而**没有先 `RemoveModuleRole`**：
```go
if typed.Description() == autoDocumentRoleDescription {
    // ← 缺少 RemoveModuleRole 调用
    ctx.Backend.AddModuleRole(...)  // 追加第二个同名角色 → CE1613
}
```

**修复（`cmd_security_write_modulerole_gen.go` overwrite 分支）：**

```go
if typed.Description() == autoDocumentRoleDescription {
    oldQualified := s.Name.Module + "." + typed.Name()
    newQualified := s.Name.Module + "." + s.Name.Name

    // 先删旧角色，因为 AddModuleRole 是纯追加不去重
    if err := ctx.Backend.RemoveModuleRole(model.ID(ms.ID()), typed.Name()); err != nil {
        return mdlerrors.NewBackend("remove auto-provisioned role", err)
    }
    if err := ctx.Backend.AddModuleRole(model.ID(ms.ID()), s.Name.Name, s.Description); err != nil {
        return mdlerrors.NewBackend("create module role", err)
    }
    // ... rename references（逻辑不变）
}
```

`RemoveModuleRole` 已存在于：
- 接口：`mdl/backend/security.go:42`
- 实现：`mdl/backend/mpr/backend.go:655`
- Mock：`mdl/backend/mock/backend.go:160`（`RemoveModuleRoleFunc` 字段）

**错误消息改进：** 当遇到用户自定义的同名角色（非 auto-provisioned）时，当前只打印 `"already exists"` 然后返回，改为：
```
Module role TaskMgr.User already exists.
To link it to a user role: ALTER USER ROLE "User" ADD MODULE ROLES (TaskMgr."User");
```

**测试：** 新增 executor 测试，验证：auto-provisioned → `CREATE MODULE ROLE` → 只有一个同名角色（无 CE1613）。

---

## 不在本次范围

- **Issue 9**（首次启动 NPE）：需运行时日志诊断，与本批次代码修改无关，单独排查。
- **Issue 1 代码修复路径**（让 `mxcli new` 自动下载 binary）：成本高、有网络依赖，本次选择文档路径（B 方案）。

---

## 参考文件索引

| 文件 | 关联 Issues |
|------|-------------|
| `cmd/mxcli/widget_scaffold.go:299` | Issue 6 |
| `cmd/mxcli/widget_build.go:264` | Issue 2 |
| `cmd/mxcli-local/cmd_run.go:47` | Issue 8 |
| `mdl/executor/cmd_security_write_modulerole_gen.go:53` | Issue 5/7 |
| `mdl/backend/mpr/security_module_modelsdk.go:24` | Issue 5/7（根因） |
| `mdl/backend/security.go:42` | Issue 5/7（RemoveModuleRole 接口） |
| `.claude/skills/mendix/check-syntax.md` | Issue 1 |
| `.claude/skills/mendix/create-page.md` | Issue 4 |
| `.claude/skills/mendix/create-custom-widget.md` | mxcli-taskdemo 引用 |
| `.claude/skills/mendix/run-app.md` | Issue 8 |
| `github.com/engalar/mxcli-taskdemo` | Issues 4, 8, 3（参考） |
