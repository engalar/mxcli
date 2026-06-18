# Theme Index Acceptance Verification Plan

> 所有验收步骤必须输出 PASS / FAIL。每步参考规范文件中的对应 Task。

---

## Phase 1 — 单元测试验收（CI 基线）

### 1.1 SCSS 解析器

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./internal/mxgraph/scss/... -v -count=1
```

**预期：11 tests PASS**

| 测试 | 验证点 |
|------|--------|
| `TestParse_BasicSassVar` | `$brand-primary: #264ae5` → Name=`$brand-primary`, Value=`#264ae5`, IsCSSVar=false |
| `TestParse_CssCustomProperty` | `:root { --brand-primary: #1565C0; }` → Name=`--brand-primary`, IsCSSVar=true, IsInRoot=true |
| `TestParse_DefaultFlag` | `$x: 8px !default;` → IsDefault=true |
| `TestParse_CommentedVar` | `// $x: #000;` → IsActive=false |
| `TestParse_MixedContent` | 混合文件 → 3 vars found, 9 lines preserved |
| `TestSetVar_NewVar` | 在 @import 前插入新变量 |
| `TestSetVar_UpdateExisting` | 修改已有变量值 |
| `TestWrite_RoundTrip` | `Parse` → `Write` 与原始内容一致 |
| `TestWrite_AfterSetVar` | 插入后序列化包含新行 |
| `TestWrite_AfterUpdate` | 更新后序列化值已改变 |

### 1.2 ThemeScssAdapter

```bash
go test ./internal/mxgraph/adapter/themescss/... -v -count=1
```

**预期：4 tests PASS**

| 测试 | 验证点 |
|------|--------|
| `TestThemeScssAdapter_Name` | Name() == "themescss" |
| `TestThemeScssAdapter_Schema` | Schema 包含 "ThemeVariable" label |
| `TestThemeScssAdapter_Build` | 临时项目结构 → 9 个 NodeCreated 事件，含 $brand-primary=#1565C0 (project-main)、--brand-warning=commented (IsActive=false) |
| `TestThemeScssAdapter_Build_NoFiles` | 空目录 → 0 事件 |

### 1.3 DesignPropertyAdapter

```bash
go test ./internal/mxgraph/adapter/designdprops/... -v -count=1
```

**预期：3 tests PASS**

| 测试 | 验证点 |
|------|--------|
| `TestDesignPropertyAdapter_Name` | Name() == "designdprops" |
| `TestDesignPropertyAdapter_Schema` | Schema 包含 "DesignProperty" label |
| `TestDesignPropertyAdapter_Build` | 2 个 JSON 文件 → 4 个 DP 节点，含 "Background color" 的 ReferencedVars |

### 1.4 WidgetInstanceAdapter

```bash
go test ./internal/mxgraph/adapter/mpr/... -run "TestWidgetInstance" -v -count=1
```

**预期：4 tests PASS**

| 测试 | 验证点 |
|------|--------|
| `TestWidgetInstanceAdapter_Name` | Name() == "widgetinstance" |
| `TestWidgetInstanceAdapter_Schema` | Schema 包含 "WidgetInstance" + "HAS_WIDGET_INSTANCE" |
| `TestIsWidgetType` | DivContainer/Button=true, LayoutCall/FormCall=false |
| `TestShortTypeName` | "Forms$DivContainer" → "DivContainer" |

### 1.5 mxgraph 内核

```bash
go test ./internal/mxgraph/... -v -count=1 -run "TestDeltaLog|TestRestoreFromSnapshot"
```

**预期：3 tests PASS**

### 1.6 graphcatalog 接口

```bash
go test ./mdl/graphcatalog/... -v -count=1
```

**预期：11+ tests PASS，含 TestInterfaceCompliance（编译期检查 ThemeReader + StylingReader）**

### 1.7 Executor 注册完整性

```bash
go test ./mdl/executor/... -run "TestNewRegistry" -v -count=1
```

**预期：3 tests PASS（NoPanic, Completeness, HandlerCountSnapshot）**

---

## Phase 2 — 集成验收（用 HelpDeskE2E 项目验证）

### 2.1 构建图并验证主题变量

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go run cmd/mxcli/main.go -p HelpDeskE2E/HelpDeskE2E.mpr -c "REFRESH CATALOG"
go run cmd/mxcli/main.go -p HelpDeskE2E/HelpDeskE2E.mpr -c "SHOW THEME VARIABLES"
```

**预期输出应包含：**
```
brand (X):
  $brand-primary                = #1565C0                [project-main]
  --brand-primary               = #264ae5                [project-custom-variables]
  --brand-success               = #16aa16                [project-custom-variables]
```

**验证点：**
- `$brand-primary` 值 = `#1565C0`（来自 main.scss 覆盖）
- `--brand-primary` 值 = `#264ae5`（来自 custom-variables.scss）
- `--brand-warning` 标记为 `(commented)`
- 至少包含 `project-custom-variables`、`project-main`、`atlas-core-default` 三种来源

### 2.2 验证 SHOW THEME VARIABLES DEFAULT

```bash
go run cmd/mxcli/main.go -p HelpDeskE2E/HelpDeskE2E.mpr -c "SHOW THEME VARIABLES DEFAULT"
```

**预期：只显示 atlas-core-default 来源的变量，值带 !default 标记**

### 2.3 验证 SHOW THEME VARIABLES LIKE

```bash
go run cmd/mxcli/main.go -p HelpDeskE2E/HelpDeskE2E.mpr -c "SHOW THEME VARIABLES LIKE '%brand%'"
```

**预期：只显示 Name 包含 "brand" 的变量**

### 2.4 验证图持久化

```bash
# 确认 .mxcli/graph.gob 文件存在
ls -la HelpDeskE2E/.mxcli/graph.gob
```

```bash
# 确认 delta 日志存在
ls -la HelpDeskE2E/.mxcli/delta.log
```

```bash
# 二次加载（走缓存路径）
go run cmd/mxcli/main.go -p HelpDeskE2E/HelpDeskE2E.mpr -c "SHOW THEME VARIABLES"
```

**预期：二次加载输出 "Graph restored from cache" + 正确的变量列表**

### 2.5 验证设计属性查询

由于 SHOW DESIGN PROPERTIES 已有完整实现，确认 mxgraph 中的 DesignProperty 节点与 JSON 定义一致：

```bash
# 通过 SHOW DESIGN PROPERTIES 验证（走文件路径）
go run cmd/mxcli/main.go -p HelpDeskE2E/HelpDeskE2E.mpr -c "SHOW DESIGN PROPERTIES FOR CONTAINER" | head -30
```

**预期：输出 DivContainer 的设计属性列表（Spacing, Card style, Background color 等）**

---

## Phase 3 — 增量 Watch 验收

### 3.1 SCSS 文件变更监测试验

```bash
# 步骤 1: 构建图
go run cmd/mxcli/main.go -p HelpDeskE2E/HelpDeskE2E.mpr -c "REFRESH GRAPH"

# 步骤 2: 在另一个终端启动测试程序（验证 watch 事件）
# 说明：watch 会在后台 goroutine 运行，以下用 touch 模拟变更
touch HelpDeskE2E/theme/web/custom-variables.scss

# 步骤 3: 等待 1 秒后重新查询
sleep 1
go run cmd/mxcli/main.go -p HelpDeskE2E/HelpDeskE2E.mpr -c "SHOW THEME VARIABLES LIKE '%brand-primary%'"
```

**验证点：** 如果 --watch 模式开启，touched 文件应重新解析并更新图（图查询返回最新值）

### 3.2 设计属性文件变更监测试验

```bash
touch HelpDeskE2E/themesource/atlas_core/web/design-properties.json
sleep 1
go run cmd/mxcli/main.go -p HelpDeskE2E/HelpDeskE2E.mpr -c "SHOW DESIGN PROPERTIES FOR CONTAINER" | head -10
```

---

## Phase 4 — 边界条件验收

### 4.1 空项目

```bash
# 创建临时空项目目录
mkdir -p /tmp/test-empty-project/theme/web
go run cmd/mxcli/main.go -p /tmp/test-empty-project -c "SHOW THEME VARIABLES" 2>&1
```

**预期：** `graph not built — run REFRESH CATALOG first`（或 `No theme variables found`）

### 4.2 项目无主题目录

```bash
mkdir -p /tmp/test-no-theme
go run cmd/mxcli/main.go -p /tmp/test-no-theme -c "REFRESH GRAPH" 2>&1
```

**预期：** 不崩溃，图构建成功但无主题节点

### 4.3 损坏的 SCSS 文件

```bash
mkdir -p /tmp/test-broken-scss/theme/web
echo "this is not valid scss ::::" > /tmp/test-broken-scss/theme/web/custom-variables.scss
go run cmd/mxcli/main.go -p /tmp/test-broken-scss -c "REFRESH GRAPH" 2>&1
```

**预期：** 不崩溃，跳过错行，解析正常行

---

## Phase 5 — 非回归验收

### 5.1 现有测试全部 PASS

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
# 运行新代码影响的全部包
go test ./internal/mxgraph/... ./mdl/graphcatalog/... -count=1 2>&1 | grep -E "(ok|FAIL)"

# 运行 executor 中的非预存失败测试（排除已知失败的 JavaScript action 测试）
go test ./mdl/executor/... -run "TestAddCallJavaScript" -count=1 2>&1 | grep -E "FAIL"
# 预期：只有 TestAddCallJavaScriptActionActionGenSetsCallAndMappings 失败（预存问题）
```

### 5.2 编译无错误

```bash
go build ./internal/... ./mdl/... 2>&1
# 预期：无输出（编译成功）
```

---

## 验收结论标准

| 级别 | 通过条件 |
|------|----------|
| **P0** | Phase 1 全部 25 个单元测试 PASS |
| **P1** | Phase 2 集成测试中 3 个命令执行成功 |
| **P2** | Phase 3 增量 watch 文件变更可触发图更新 |
| **P3** | Phase 4 边界条件不崩溃 |
| **P4** | Phase 5 非回归确认无新测试失败 |

**P0 必须在提交前通过。P1-P2 通过后标记为可发布。P3-P4 确保长期稳定性。**
