# OCP 修复：pages_builder_v3 三大 switch 改为注册表

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 `pages_builder_v3.go` 中三个大型 switch 语句（widget 类型 ~40 case、datasource 类型 6 case、action 类型 11 case）替换为包级注册表 map，使新增 widget/datasource/action 类型只需在对应注册文件追加一行，不再修改 switch 函数本身。

**Architecture:** 新建三个注册文件：`page_widget_registry.go`、`page_datasource_registry.go`、`page_action_registry.go`，各持有一个 `map[string]builderFn` var。`buildWidgetV3`/`buildDataSourceV3`/`buildClientActionV3` 改为查表 + 错误回退，原有的 pluggable widget fallback 逻辑保留在 `buildWidgetV3` 的 default 路径中。现有 `pages_builder_v3_test.go` 中的直接调用无需修改（函数签名不变）。

**Tech Stack:** Go 1.24，`mdl/executor` 包，`mdl/ast` 包。与 `show_registry.go`/`describe_registry.go` 同模式。

---

## 影响文件概览

| 文件 | 操作 |
|------|------|
| `mdl/executor/page_widget_registry.go` | 新建：`widgetBuilderFn` 类型 + `widgetBuilders` map（~40 entry） |
| `mdl/executor/page_datasource_registry.go` | 新建：`dataSourceBuilderFn` 类型 + `dataSourceBuilders` map（6 entry） |
| `mdl/executor/page_action_registry.go` | 新建：`actionBuilderFn` 类型 + `actionBuilders` map（11 entry） |
| `mdl/executor/page_widget_registry_test.go` | 新建：覆盖率测试，防止新 widget 类型没注册 |
| `mdl/executor/page_datasource_registry_test.go` | 新建：覆盖率测试 |
| `mdl/executor/page_action_registry_test.go` | 新建：覆盖率测试 |
| `mdl/executor/pages_builder_v3.go` | 修改：三个 switch 函数体压缩为查表 + fallback |

---

## Task 1：Action 注册表（最简单，11 case，无复杂 fallback）

### 步骤

- [ ] **Step 1.1：写覆盖率测试（先让它编译失败）**

新建 `mdl/executor/page_action_registry_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0
package executor_test

import (
	"testing"
)

// TestActionBuilderRegistryCoverage 确认所有已知 action type 都有注册 handler。
// 当 ast.ActionV3.Type 增加新值时，此测试提醒同步注册。
func TestActionBuilderRegistryCoverage(t *testing.T) {
	// All Type values from ast.ActionV3 doc comment in ast_page_v3.go:86
	knownTypes := []string{
		"save", "cancel", "close", "delete", "create",
		"showPage", "microflow", "nanoflow",
		"openLink", "signOut", "completeTask",
	}
	handlers := ActionBuilders()
	for _, typ := range knownTypes {
		if _, ok := handlers[typ]; !ok {
			t.Errorf("action type %q has no handler in actionBuilders — add it to page_action_registry.go", typ)
		}
	}
}
```

- [ ] **Step 1.2：运行确认测试因 `ActionBuilders` 不存在而失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/... -run TestActionBuilderRegistryCoverage 2>&1 | head -5
```

预期：`undefined: ActionBuilders`。

- [ ] **Step 1.3：新建 `mdl/executor/page_action_registry.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// actionBuilderFn builds a client action gen element from an ActionV3 AST node.
// pb gives access to backend, caches, and resolvers.
type actionBuilderFn func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error)

// actionBuilders maps each action type string to its builder.
// New action types only need an entry added here.
var actionBuilders = map[string]actionBuilderFn{
	"save": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		act := genPg.NewSaveChangesClientAction()
		assignFreshID(act)
		act.SetClosePage(action.ClosePage)
		return act, nil
	},
	"cancel": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		act := genPg.NewCancelChangesClientAction()
		assignFreshID(act)
		act.SetClosePage(action.ClosePage)
		return act, nil
	},
	"close": func(pb *pageBuilder, _ *ast.ActionV3) (element.Element, error) {
		act := genPg.NewClosePageClientAction()
		assignFreshID(act)
		return act, nil
	},
	"delete": func(pb *pageBuilder, _ *ast.ActionV3) (element.Element, error) {
		act := genPg.NewDeleteClientAction()
		assignFreshID(act)
		return act, nil
	},
	"create": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		entityID, err := pb.resolveEntity(ast.QualifiedName{
			Module: pb.extractModule(action.Target),
			Name:   pb.extractName(action.Target),
		})
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve entity for create", err)
		}
		_ = entityID

		act := genPg.NewCreateObjectClientAction()
		assignFreshID(act)
		ref := genDm.NewDirectEntityRef()
		assignFreshID(ref)
		ref.SetEntityQualifiedName(action.Target)
		act.SetEntityRef(ref)

		if action.ThenAction != nil && action.ThenAction.Type == "showPage" {
			if _, err := pb.resolvePageRef(action.ThenAction.Target); err != nil {
				log.Printf("warning: then show_page %s not found (will still create action by name)", action.ThenAction.Target)
			}
			ps := genPg.NewPageSettings()
			assignFreshID(ps)
			ps.SetPageQualifiedName(action.ThenAction.Target)
			act.SetPageSettings(ps)
		}
		return act, nil
	},
	"showPage": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		if _, err := pb.resolvePageRef(action.Target); err != nil {
			log.Printf("warning: action show_page %s not found (will still create action by name)", action.Target)
		}
		act := genPg.NewPageClientAction()
		assignFreshID(act)
		ps := genPg.NewPageSettings()
		assignFreshID(ps)
		ps.SetPageQualifiedName(action.Target)
		act.SetPageSettings(ps)
		return act, nil
	},
	"microflow": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		if _, err := pb.resolveMicroflow(action.Target); err != nil {
			log.Printf("warning: action microflow %s not found (will still create action by name)", action.Target)
		}
		act := genPg.NewMicroflowClientAction()
		assignFreshID(act)
		settings := genPg.NewMicroflowSettings()
		assignFreshID(settings)
		settings.SetMicroflowQualifiedName(action.Target)
		for _, arg := range action.Args {
			mm := genPg.NewMicroflowParameterMapping()
			assignFreshID(mm)
			mm.SetParameterQualifiedName(action.Target + "." + arg.Name)
			if strVal, ok := arg.Value.(string); ok {
				mm.SetExpression(strVal)
			}
			settings.AddParameterMappings(mm)
		}
		act.SetMicroflowSettings(settings)
		if action.ClosePage {
			setRawBSONField(act, "ClosePage", true)
		}
		return act, nil
	},
	"nanoflow": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		nfID, err := pb.resolveNanoflowByName(action.Target)
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve nanoflow", err)
		}
		_ = nfID
		act := genPg.NewCallNanoflowClientAction()
		assignFreshID(act)
		act.SetNanoflowQualifiedName(action.Target)
		for _, arg := range action.Args {
			nm := genPg.NewNanoflowParameterMapping()
			assignFreshID(nm)
			// Use fully-qualified "Module.NanoflowName.ParamName" form —
			// bare param name causes mx check crash with null Parameter reference.
			nm.SetParameterQualifiedName(action.Target + "." + arg.Name)
			if strVal, ok := arg.Value.(string); ok {
				if strings.HasPrefix(strVal, "$") {
					pv := genPg.NewPageVariable()
					assignFreshID(pv)
					pv.SetPageParameterQualifiedName(strVal)
					nm.SetVariable(pv)
				} else {
					nm.SetExpression(strVal)
				}
			}
			act.AddParameterMappings(nm)
		}
		return act, nil
	},
	"openLink": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		act := genPg.NewOpenLinkClientAction()
		assignFreshID(act)
		act.SetLinkType("Web")
		addr := genPg.NewStaticOrDynamicString()
		assignFreshID(addr)
		addr.SetValue(action.LinkURL)
		act.SetAddress(addr)
		return act, nil
	},
	"signOut": func(pb *pageBuilder, _ *ast.ActionV3) (element.Element, error) {
		act := genPg.NewSignOutClientAction()
		assignFreshID(act)
		return act, nil
	},
	"completeTask": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		act := genPg.NewSetTaskOutcomeClientAction()
		assignFreshID(act)
		act.SetClosePage(true)
		act.SetCommit(true)
		act.SetOutcomeValue(action.OutcomeValue)
		return act, nil
	},
}

// ActionBuilders returns the action builder map (exported for tests).
func ActionBuilders() map[string]actionBuilderFn {
	return actionBuilders
}
```

> **注意**：`page_action_registry.go` 需要 import `"log"` 和 `"strings"` — 检查并在 import 块补充。这两个包已在 `pages_builder_v3.go` 中使用，移过来后需要确保新文件的 import 完整。

- [ ] **Step 1.4：运行覆盖率测试确认通过**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/... -run TestActionBuilderRegistryCoverage -v
```

预期：`PASS`。

- [ ] **Step 1.5：在 `pages_builder_v3.go` 替换 `buildClientActionV3` 的 switch 为查表**

找到 `buildClientActionV3`（行 971-1139），将其函数体替换为：

```go
func (pb *pageBuilder) buildClientActionV3(action *ast.ActionV3) (element.Element, error) {
	if fn, ok := actionBuilders[action.Type]; ok {
		return fn(pb, action)
	}
	return nil, mdlerrors.NewUnsupported("unsupported action type: " + action.Type)
}
```

原 switch 的 168 行压缩为 5 行。

- [ ] **Step 1.6：编译并运行全量测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/executor/...
go test ./mdl/executor/... -count=1 2>&1 | tail -10
```

预期：无错误，全部 `ok`。

- [ ] **Step 1.7：确认 `pages_builder_v3.go` 的 import 无孤立引用**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go vet ./mdl/executor/... 2>&1 | grep "imported and not used\|unused import"
```

若 `pages_builder_v3.go` 中某些 import 只被删除的 switch 用，去掉它们（`go build` 会报告）。

- [ ] **Step 1.8：commit**

```bash
git add mdl/executor/page_action_registry.go mdl/executor/page_action_registry_test.go mdl/executor/pages_builder_v3.go
git commit -m "$(cat <<'EOF'
refactor(executor): replace 11-case action switch with handler map (OCP)

buildClientActionV3 in pages_builder_v3.go had a 168-line switch over
action.Type. Every new action type required modifying the switch (OCP
violation). Extract actionBuilders map to page_action_registry.go;
buildClientActionV3 becomes a 5-line table lookup. Coverage test
TestActionBuilderRegistryCoverage prevents gaps at test-time.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2：DataSource 注册表（6 case，中等复杂度）

### 步骤

- [ ] **Step 2.1：写覆盖率测试**

新建 `mdl/executor/page_datasource_registry_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0
package executor_test

import (
	"testing"
)

// TestDataSourceBuilderRegistryCoverage 确认所有 DataSourceV3.Type 值都有注册 handler。
func TestDataSourceBuilderRegistryCoverage(t *testing.T) {
	// All Type values from ast.DataSourceV3 doc comment in ast_page_v3.go:64
	knownTypes := []string{
		"parameter", "database", "microflow", "nanoflow", "association", "selection",
	}
	handlers := DataSourceBuilders()
	for _, typ := range knownTypes {
		if _, ok := handlers[typ]; !ok {
			t.Errorf("datasource type %q has no handler in dataSourceBuilders — add it to page_datasource_registry.go", typ)
		}
	}
}
```

- [ ] **Step 2.2：运行确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/... -run TestDataSourceBuilderRegistryCoverage 2>&1 | head -5
```

预期：`undefined: DataSourceBuilders`。

- [ ] **Step 2.3：新建 `mdl/executor/page_datasource_registry.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"log"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// dataSourceBuilderFn builds a datasource gen element and returns it alongside
// the resolved entity qualified name (may be empty for some sources).
type dataSourceBuilderFn func(pb *pageBuilder, ds *ast.DataSourceV3) (element.Element, string, error)

// dataSourceBuilders maps each datasource type string to its builder.
// New datasource types only need an entry added here.
var dataSourceBuilders = map[string]dataSourceBuilderFn{
	"parameter": func(pb *pageBuilder, ds *ast.DataSourceV3) (element.Element, string, error) {
		paramName := strings.TrimPrefix(ds.Reference, "$")
		entityID, ok := pb.paramScope[paramName]
		entityName := pb.paramEntityNames[paramName]
		if !ok {
			entityID, ok = pb.paramScope["$"+paramName]
			entityName = pb.paramEntityNames["$"+paramName]
		}
		if !ok {
			return nil, "", mdlerrors.NewNotFound("parameter", ds.Reference)
		}

		if entityName == "" {
			var err error
			entityName, err = pb.getEntityNameByID(entityID)
			if err != nil {
				log.Printf("warning: could not resolve entity name for ID %s: %v", entityID, err)
			}
		}

		dvs := genPg.NewDataViewSource()
		assignFreshID(dvs)
		dvs.SetForceFullObjects(false)
		if pb.isSnippet {
			dvs.SetSnippetParameterQualifiedName(paramName)
		} else {
			// SP11.6.6: use SourceVariable (nested PageVariable) instead of flat PageParameter
			sv := genPg.NewPageVariable()
			assignFreshID(sv)
			sv.SetPageParameterQualifiedName(paramName)
			dvs.SetSourceVariable(sv)
		}
		if entityName != "" {
			ref := genDm.NewDirectEntityRef()
			assignFreshID(ref)
			ref.SetEntityQualifiedName(entityName)
			dvs.SetEntityRef(ref)
		}
		return dvs, entityName, nil
	},
	"database": func(pb *pageBuilder, ds *ast.DataSourceV3) (element.Element, string, error) {
		entityID, err := pb.resolveEntity(ast.QualifiedName{
			Module: pb.extractModule(ds.Reference),
			Name:   pb.extractName(ds.Reference),
		})
		if err != nil {
			return nil, "", mdlerrors.NewBackend("resolve entity", err)
		}
		_ = entityID

		dvs := genPg.NewDataViewSource()
		assignFreshID(dvs)
		dvs.SetEntityPath(ds.Reference)
		ref := genDm.NewDirectEntityRef()
		assignFreshID(ref)
		ref.SetEntityQualifiedName(ds.Reference)
		dvs.SetEntityRef(ref)
		return dvs, ds.Reference, nil
	},
	"microflow": func(pb *pageBuilder, ds *ast.DataSourceV3) (element.Element, string, error) {
		mfID, err := pb.resolveMicroflow(ds.Reference)
		if err != nil {
			return nil, "", mdlerrors.NewBackend("resolve microflow", err)
		}
		_ = mfID

		entityName := pb.getMicroflowReturnEntityName(ds.Reference)
		ms := genPg.NewMicroflowSource()
		assignFreshID(ms)
		settings := genPg.NewMicroflowSettings()
		assignFreshID(settings)
		settings.SetMicroflowQualifiedName(ds.Reference)
		ms.SetMicroflowSettings(settings)
		return ms, entityName, nil
	},
	"nanoflow": func(pb *pageBuilder, ds *ast.DataSourceV3) (element.Element, string, error) {
		// Delegates to the shared helper used by DataGrid2 as well.
		return pb.buildNanoflowSourceGen(ds)
	},
	"association": func(pb *pageBuilder, ds *ast.DataSourceV3) (element.Element, string, error) {
		ctxVar := ds.ContextVariable
		if ctxVar == "currentObject" {
			ctxVar = ""
		}

		path := ds.Reference
		destEntity := ""
		if idx := strings.Index(path, "/"); idx >= 0 {
			destEntity = path[idx+1:]
			path = path[:idx]
		} else {
			destEntity = pb.resolveAssociationDestination(path, pb.entityContext)
		}

		as := genPg.NewAssociationSource()
		assignFreshID(as)
		as.SetEntityPath(path + "/" + destEntity)
		if ctxVar != "" {
			pv := genPg.NewPageVariable()
			assignFreshID(pv)
			pv.SetPageParameterQualifiedName(ctxVar)
			as.SetSourceVariable(pv)
		}
		return as, destEntity, nil
	},
	"selection": func(pb *pageBuilder, ds *ast.DataSourceV3) (element.Element, string, error) {
		widgetName := ds.Reference
		widgetID, ok := pb.widgetScope[widgetName]
		if !ok {
			return nil, "", mdlerrors.NewNotFound("widget", widgetName)
		}
		_ = widgetID

		entityName := pb.paramEntityNames[widgetName]
		lts := genPg.NewListenTargetSource()
		assignFreshID(lts)
		lts.SetListenTarget(widgetName)
		return lts, entityName, nil
	},
}

// DataSourceBuilders returns the datasource builder map (exported for tests).
func DataSourceBuilders() map[string]dataSourceBuilderFn {
	return dataSourceBuilders
}
```

- [ ] **Step 2.4：运行覆盖率测试确认通过**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/... -run TestDataSourceBuilderRegistryCoverage -v
```

预期：`PASS`。

- [ ] **Step 2.5：在 `pages_builder_v3.go` 替换 `buildDataSourceV3` 的 switch 为查表**

找到 `buildDataSourceV3`（行 602-730），将其函数体替换为：

```go
func (pb *pageBuilder) buildDataSourceV3(ds *ast.DataSourceV3) (element.Element, string, error) {
	if fn, ok := dataSourceBuilders[ds.Type]; ok {
		return fn(pb, ds)
	}
	return nil, "", mdlerrors.NewUnsupported("unsupported datasource type: " + ds.Type)
}
```

原 128 行压缩为 5 行。`buildListViewDataSourceV3` 不受影响（它对 `database` 类型有特殊处理，保持不变）。

- [ ] **Step 2.6：编译并运行全量测试（含现有的 pages_builder_v3_test.go）**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/executor/...
go test ./mdl/executor/... -count=1 2>&1 | tail -10
```

预期：无错误，全部 `ok`。现有的 `TestNanoflowDatasource*` 和 selection 测试继续通过。

- [ ] **Step 2.7：commit**

```bash
git add mdl/executor/page_datasource_registry.go mdl/executor/page_datasource_registry_test.go mdl/executor/pages_builder_v3.go
git commit -m "$(cat <<'EOF'
refactor(executor): replace 6-case datasource switch with handler map (OCP)

buildDataSourceV3 in pages_builder_v3.go had a 128-line switch over
ds.Type (parameter/database/microflow/nanoflow/association/selection).
Extract dataSourceBuilders map to page_datasource_registry.go;
buildDataSourceV3 becomes a 5-line table lookup. Coverage test
TestDataSourceBuilderRegistryCoverage guards against missing registrations.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3：Widget 注册表（~40 case，default 保留 pluggable fallback）

Widget switch 比前两者复杂，因为：
1. 案例 `"image"` 有特殊逻辑：先尝试 pluggable，失败则 fallback 到 static image
2. `default` 路径有 3 层 pluggable widget 查找逻辑，需要保留在函数本身

**设计决策**：map 只存放**确定性**的内置 widget 类型（包含 `image` 的 fallback 逻辑）；`buildWidgetV3` 在 map miss 后执行 pluggable 查找，行为与原 `default` case 完全一致。

### 步骤

- [ ] **Step 3.1：写覆盖率测试**

新建 `mdl/executor/page_widget_registry_test.go`：

```go
// SPDX-License-Identifier: Apache-2.0
package executor_test

import (
	"testing"
)

// TestWidgetBuilderRegistryCoverage 确认所有内置 widget 类型都有注册 handler。
// pluggable widget（default case）由 widgetRegistry 处理，不在此列表。
func TestWidgetBuilderRegistryCoverage(t *testing.T) {
	// All explicitly handled cases from the original buildWidgetV3 switch.
	// "legacydatagrid" is intentionally absent (returns error, not a builder).
	// "tabpage" and "item" are intentionally absent (return validation errors).
	knownTypes := []string{
		"dataview", "datagrid", "listview", "layoutgrid",
		"row", "column", "container", "customcontainer",
		"textbox", "textarea", "datepicker", "dropdown",
		"checkbox", "fileinput", "text", "statictext",
		"dynamictext", "title", "button", "actionbutton",
		"tabcontainer", "groupbox", "radiobuttons",
		"navigationlist", "snippetcall",
		"footer", "header", "controlbar",
		"template", "filter", "staticimage", "dynamicimage", "image",
	}
	builders := WidgetBuilders()
	for _, typ := range knownTypes {
		if _, ok := builders[typ]; !ok {
			t.Errorf("widget type %q has no handler in widgetBuilders — add it to page_widget_registry.go", typ)
		}
	}
}
```

- [ ] **Step 3.2：运行确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/... -run TestWidgetBuilderRegistryCoverage 2>&1 | head -5
```

预期：`undefined: WidgetBuilders`。

- [ ] **Step 3.3：新建 `mdl/executor/page_widget_registry.go`**

```go
// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"log"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// widgetBuilderFn builds a gen widget element from a WidgetV3 AST node.
// All type strings are lowercase (matched via strings.ToLower in buildWidgetV3).
type widgetBuilderFn func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error)

// widgetBuilders maps each built-in widget type (lowercase) to its builder.
// Pluggable/unknown types fall through to the pluggable widget engine in
// buildWidgetV3's fallback path — they are NOT listed here.
//
// "legacydatagrid", "tabpage", and "item" are intentionally absent: they
// return validation errors rather than elements.
var widgetBuilders = map[string]widgetBuilderFn{
	"dataview":        func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildDataViewV3(w) },
	"datagrid":        func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildDataGridV3(w) },
	"listview":        func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildListViewV3(w) },
	"layoutgrid":      func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildLayoutGridV3(w) },
	"row":             func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildContainerWithRowV3(w) },
	"column":          func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildContainerWithColumnV3(w) },
	"container":       func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildContainerV3(w) },
	"customcontainer": func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildContainerV3(w) },
	"textbox":         func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildTextBoxV3(w) },
	"textarea":        func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildTextAreaV3(w) },
	"datepicker":      func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildDatePickerV3(w) },
	"dropdown":        func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildDropdownV3(w) },
	"checkbox":        func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildCheckBoxV3(w) },
	"fileinput":       func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildFileManagerV3(w) },
	"text":            func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildTextWidgetV3(w) },
	"statictext":      func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildTextWidgetV3(w) },
	"dynamictext":     func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildDynamicTextV3(w) },
	"title":           func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildTitleV3(w) },
	"button":          func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildButtonV3(w) },
	"actionbutton":    func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildButtonV3(w) },
	"tabcontainer":    func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildTabContainerV3(w) },
	"groupbox":        func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildGroupBoxV3(w) },
	"radiobuttons":    func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildRadioButtonsV3(w) },
	"navigationlist":  func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildNavigationListV3(w) },
	"snippetcall":     func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildSnippetCallV3(w) },
	"footer":          func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildFooterV3(w) },
	"header":          func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildHeaderV3(w) },
	"controlbar":      func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildControlBarV3(w) },
	"template":        func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildTemplateV3(w) },
	"filter":          func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildFilterV3(w) },
	"staticimage":     func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildStaticImageV3(w) },
	"dynamicimage":    func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) { return pb.buildDynamicImageV3(w) },
	// image: tries pluggable widget first, falls back to static image
	"image": func(pb *pageBuilder, w *ast.WidgetV3) (element.Element, error) {
		pb.initPluggableEngine()
		if pb.widgetRegistry != nil {
			if def, ok := pb.widgetRegistry.Get("image"); ok {
				cw, err := pb.pluggableEngine.Build(def, w)
				if err != nil {
					return nil, err
				}
				return pb.customWidgetToElement(cw)
			}
		}
		log.Printf("warning: pluggable image widget not found, using static image fallback")
		return pb.buildStaticImageV3(w)
	},
}

// WidgetBuilders returns the widget builder map (exported for tests).
func WidgetBuilders() map[string]widgetBuilderFn {
	return widgetBuilders
}
```

- [ ] **Step 3.4：运行覆盖率测试确认通过**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/... -run TestWidgetBuilderRegistryCoverage -v
```

预期：`PASS`。

- [ ] **Step 3.5：在 `pages_builder_v3.go` 替换 `buildWidgetV3` 的 switch 为查表 + fallback**

找到 `buildWidgetV3`（行 342-468），将其函数体替换为：

```go
func (pb *pageBuilder) buildWidgetV3(w *ast.WidgetV3) (element.Element, error) {
	typeLower := strings.ToLower(w.Type)

	// Explicit error cases: these types are not legal standalone widgets.
	switch typeLower {
	case "legacydatagrid":
		return nil, mdlerrors.NewUnsupported(
			"LEGACYDATAGRID (native Forms$DataGrid) is not yet implemented. " +
				"Use DATAGRID for the pluggable equivalent on Mendix 11+, " +
				"or open the project in Studio Pro to add native datagrids manually.")
	case "tabpage":
		return nil, mdlerrors.NewValidation("tabpage must be a direct child of tabcontainer")
	case "item":
		return nil, mdlerrors.NewValidation("item must be a direct child of navigationlist")
	}

	// Registered built-in widget types.
	if fn, ok := widgetBuilders[typeLower]; ok {
		widget, err := fn(pb, w)
		if err != nil {
			return nil, err
		}
		applyWidgetAppearanceGen(widget, w, pb.themeRegistry)
		applyConditionalSettingsGen(widget, w)
		return widget, nil
	}

	// Pluggable widget fallback: look up by type ID in the registry.
	pb.initPluggableEngine()
	if pb.widgetRegistry != nil {
		if def, ok := pb.widgetRegistry.Get(strings.ToUpper(w.Type)); ok {
			cw, err := pb.pluggableEngine.Build(def, w)
			if err != nil {
				return nil, err
			}
			return pb.customWidgetToElement(cw)
		}
		if w.Type == "pluggablewidget" || w.Type == "customwidget" {
			if widgetType, ok := w.Properties["WidgetType"].(string); ok {
				if def, ok := pb.widgetRegistry.GetByWidgetID(widgetType); ok {
					cw, err := pb.pluggableEngine.Build(def, w)
					if err != nil {
						return nil, err
					}
					return pb.customWidgetToElement(cw)
				}
				return nil, mdlerrors.NewNotFoundMsg("widget", widgetType,
					"no definition for widget "+widgetType+" (run 'mxcli widget init -p app.mpr')")
			}
		}
	}
	if pb.pluggableEngineErr != nil {
		return nil, mdlerrors.NewUnsupported(fmt.Sprintf("unsupported widget type: %s (%v)", w.Type, pb.pluggableEngineErr))
	}
	return nil, mdlerrors.NewUnsupported("unsupported widget type: " + w.Type)
}
```

原 126 行压缩为 ~50 行（含 pluggable fallback 逻辑保留）。

- [ ] **Step 3.6：编译并运行全量测试**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go build ./mdl/executor/...
go test ./mdl/executor/... -count=1 2>&1 | tail -10
```

预期：无错误，全部 `ok`。

- [ ] **Step 3.7：确认 `pages_builder_v3.go` 统计行数下降**

```bash
wc -l /mnt/data_sdd/gh/mxcli-wt-02/mdl/executor/pages_builder_v3.go
```

预期：比 3088 行少约 250 行（三个 switch 删减约 168+128+126=422 行，三个注册文件承接这些代码）。

- [ ] **Step 3.8：commit**

```bash
git add mdl/executor/page_widget_registry.go mdl/executor/page_widget_registry_test.go mdl/executor/pages_builder_v3.go
git commit -m "$(cat <<'EOF'
refactor(executor): replace ~40-case widget switch with handler map (OCP)

buildWidgetV3 in pages_builder_v3.go had a 126-line switch over
strings.ToLower(w.Type). Every new built-in widget type required
modifying the switch (OCP violation). Extract widgetBuilders map to
page_widget_registry.go; buildWidgetV3 becomes a table lookup + pluggable
widget fallback (unchanged behavior). Explicit error cases (legacydatagrid,
tabpage, item) kept as a small switch before the table lookup for clarity.
Coverage test TestWidgetBuilderRegistryCoverage prevents gaps.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## 自检 Checklist

- [ ] `go build ./mdl/executor/...` 无错误
- [ ] `go test ./mdl/executor/... -run TestActionBuilderRegistryCoverage` PASS
- [ ] `go test ./mdl/executor/... -run TestDataSourceBuilderRegistryCoverage` PASS
- [ ] `go test ./mdl/executor/... -run TestWidgetBuilderRegistryCoverage` PASS
- [ ] `go test ./mdl/executor/... -count=1` 全部 `ok`，无 FAIL
- [ ] `grep -c "case " mdl/executor/pages_builder_v3.go` 输出 < 10（只剩三个显式错误 case + 少量小 switch）
- [ ] `wc -l mdl/executor/pages_builder_v3.go` 比原来少 ~250 行

## 范围外（后续 PR）

以下改善方向超出本次重构范围：

1. **`buildListViewDataSourceV3` 的 `database` 特殊分支** — 当前使用 `if ds.Type != "database"` fallback，考虑注册独立的 `listViewDataSourceBuilders` map
2. **`pageBuilder` 结构体 SRP** — pageBuilder 承担 builder + resolver + cache 多职责；可进一步提取 `pageResolver`（只含 resolveEntity/resolveMicroflow/resolvePageRef 等）
3. **`pages_builder_v3.go` 中的其他小型 switch** — `genPageParamType`、`pageParamBSONType`、`bsonTypeToMDLType` 等类型映射可改为 map，但行数少，优先级低
