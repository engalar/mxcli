# HelpDesk 实体容器页面模式 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现两个 MDL 语法缺口，然后将 helpdesk-app.mdl 升级为四类实体容器模式（DataGrid2 CRUD Overview、Gallery master-detail、关联路径嵌套 DataGrid、弹窗选择）。

**Architecture:** Phase 1 先修语法缺口（`action: microflow ... close_page` + Gallery filter bar 验证），Phase 2 升级四个页面，Phase 3 验证 mx check 通过并更新 golden。所有页面变更只修改 `helpdesk-app.mdl`。

**Tech Stack:** Go，ANTLR4（`make grammar` 再生成 parser），BSON（`go.mongodb.org/mongo-driver/bson`），`./bin/mxcli check`，`mx check`。

---

## 文件映射

| 文件 | 动作 | 用途 |
|------|------|------|
| `mdl/grammar/domains/MDLPage.g4` | 修改 | 在 MICROFLOW/NANOFLOW action 规则末尾加 `(CLOSE_PAGE)?` |
| `mdl/visitor/visitor_page_v3.go` | 修改 | `buildActionV3()` microflow/nanoflow 分支加 `ClosePage` 解析 |
| `mdl/executor/cmd_pages_builder_v3.go` | 修改 | `buildClientActionBSON` 和 `buildClientActionV3` 的 microflow case 加 ClosePage |
| `mdl/executor/cmd_pages_builder_v3_test.go` | 修改 | 新增 `TestBuildClientActionBSON_MicroflowClosePage` 单元测试 |
| `mdl-examples/doctype-tests/pages/action-microflow-close-page.mdl` | 新建 | L1/L2 语法测试脚本 |
| `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl` | 修改 | 替换/新增四个页面 |
| `testdata/helpdesk-golden/describe-snapshot.mdl` | 修改 | 更新 golden 文件 |

---

## Task 1：写失败测试 — `action: microflow M.F (...) close_page` 单元测试

**Files:**
- Modify: `mdl/executor/cmd_pages_builder_v3_test.go`

- [ ] **Step 1.1：在 `cmd_pages_builder_v3_test.go` 末尾追加以下测试**

```go
// TestBuildClientActionBSON_MicroflowClosePage verifies that an action with
// Type="microflow" and ClosePage=true emits ClosePage:true in BSON.
// Regression guard for the `action: microflow M.F (...) close_page` syntax.
func TestBuildClientActionBSON_MicroflowClosePage(t *testing.T) {
	pb := &pageBuilder{ctx: &ExecContext{}}
	action := &ast.ActionV3{
		Type:      "microflow",
		Target:    "MyMod.SomeMF",
		ClosePage: true,
	}
	got, err := pb.buildClientActionBSON(action)
	if err != nil {
		t.Fatalf("buildClientActionBSON: %v", err)
	}
	for _, kv := range got {
		if kv.Key == "ClosePage" {
			if v, ok := kv.Value.(bool); ok && v {
				return // pass
			}
			t.Errorf("ClosePage = %v, want true", kv.Value)
			return
		}
	}
	t.Error("ClosePage key not found in BSON for microflow action with ClosePage=true")
}

// TestBuildClientActionBSON_MicroflowNoClosePage verifies that a microflow
// action without close_page emits ClosePage:false (not missing).
func TestBuildClientActionBSON_MicroflowNoClosePage(t *testing.T) {
	pb := &pageBuilder{ctx: &ExecContext{}}
	action := &ast.ActionV3{
		Type:      "microflow",
		Target:    "MyMod.SomeMF",
		ClosePage: false,
	}
	got, err := pb.buildClientActionBSON(action)
	if err != nil {
		t.Fatalf("buildClientActionBSON: %v", err)
	}
	for _, kv := range got {
		if kv.Key == "ClosePage" {
			if v, ok := kv.Value.(bool); ok && !v {
				return
			}
			t.Errorf("ClosePage = %v, want false", kv.Value)
			return
		}
	}
	t.Error("ClosePage key not found in BSON for microflow action without close_page")
}
```

- [ ] **Step 1.2：运行测试，确认失败**

```bash
cd /mnt/data_sdd/gh/mxcli-wt-02
go test ./mdl/executor/ -run TestBuildClientActionBSON_Microflow -v 2>&1 | tail -10
```

期望：`FAIL` — `ClosePage key not found in BSON`

---

## Task 2：写失败测试 — L1 语法测试脚本（`close_page` after microflow）

**Files:**
- Create: `mdl-examples/doctype-tests/pages/action-microflow-close-page.mdl`

- [ ] **Step 2.1：新建测试脚本**

```sql
-- @title: action: microflow ... close_page syntax test
-- @expect: no error

-- Minimal page to verify microflow action with close_page parses correctly.
-- This page demonstrates Pattern D (popup selection):
-- a controlbar button calls a microflow AND closes the popup.

create page MyMod.TestPopup_Select (
  title: 'Select Item',
  layout: Atlas_Core.PopupLayout,
  params: { $Parent: MyFirstModule.User }
) {
  datagrid dgItems (
    datasource: database from MyFirstModule.User sort by Name asc,
    selection: single
  ) {
    controlbar cb1 {
      actionbutton btnSelect (
        caption: 'Select',
        action: microflow MyFirstModule.ACT_Select (Parent: $Parent, Item: $currentObject) close_page,
        buttonstyle: primary
      )
    }
    column colName (attribute: Name, caption: 'Name')
  }
};
```

- [ ] **Step 2.2：运行 mxcli check，确认当前失败**

```bash
./bin/mxcli check mdl-examples/doctype-tests/pages/action-microflow-close-page.mdl 2>&1 | head -20
```

期望：parse error（`close_page` after microflow args 不被识别）。

---

## Task 3：Grammar 修改 — 为 MICROFLOW/NANOFLOW action 加 `(CLOSE_PAGE)?`

**Files:**
- Modify: `mdl/grammar/domains/MDLPage.g4` (lines 339-340)

- [ ] **Step 3.1：修改 grammar 规则**

在 `mdl/grammar/domains/MDLPage.g4` 中，将以下两行：

```
    | MICROFLOW qualifiedName microflowArgsV3?        // MICROFLOW Module.Flow
    | NANOFLOW qualifiedName microflowArgsV3?         // NANOFLOW Module.Flow
```

改为：

```
    | MICROFLOW qualifiedName microflowArgsV3? (CLOSE_PAGE)?   // MICROFLOW Module.Flow [close_page]
    | NANOFLOW qualifiedName microflowArgsV3? (CLOSE_PAGE)?    // NANOFLOW Module.Flow [close_page]
```

- [ ] **Step 3.2：再生成 parser**

```bash
make grammar 2>&1 | tail -5
```

期望：无错误输出（ANTLR4 生成成功）。

- [ ] **Step 3.3：确认 build 通过**

```bash
go build ./... 2>&1 | head -20
```

期望：无编译错误。

- [ ] **Step 3.4：Commit grammar 修改**

```bash
git add mdl/grammar/domains/MDLPage.g4
git commit -m "feat(grammar): add optional CLOSE_PAGE after microflow/nanoflow action"
```

---

## Task 4：Visitor 修改 — 解析 CLOSE_PAGE token

**Files:**
- Modify: `mdl/visitor/visitor_page_v3.go` (around lines 741-755)

- [ ] **Step 4.1：在 `buildActionV3()` 的 microflow 和 nanoflow 分支中添加 ClosePage 解析**

当前 microflow 分支（约 740-747 行）：

```go
	} else if actCtx.MICROFLOW() != nil {
		action.Type = "microflow"
		if qn := actCtx.QualifiedName(); qn != nil {
			action.Target = getQualifiedNameText(qn)
		}
		if argsCtx := actCtx.MicroflowArgsV3(); argsCtx != nil {
			action.Args = buildMicroflowArgsV3(argsCtx)
		}
	} else if actCtx.NANOFLOW() != nil {
		action.Type = "nanoflow"
		if qn := actCtx.QualifiedName(); qn != nil {
			action.Target = getQualifiedNameText(qn)
		}
		if argsCtx := actCtx.MicroflowArgsV3(); argsCtx != nil {
			action.Args = buildMicroflowArgsV3(argsCtx)
		}
```

改为（在每个分支末尾加 CLOSE_PAGE 解析）：

```go
	} else if actCtx.MICROFLOW() != nil {
		action.Type = "microflow"
		if qn := actCtx.QualifiedName(); qn != nil {
			action.Target = getQualifiedNameText(qn)
		}
		if argsCtx := actCtx.MicroflowArgsV3(); argsCtx != nil {
			action.Args = buildMicroflowArgsV3(argsCtx)
		}
		action.ClosePage = actCtx.CLOSE_PAGE() != nil
	} else if actCtx.NANOFLOW() != nil {
		action.Type = "nanoflow"
		if qn := actCtx.QualifiedName(); qn != nil {
			action.Target = getQualifiedNameText(qn)
		}
		if argsCtx := actCtx.MicroflowArgsV3(); argsCtx != nil {
			action.Args = buildMicroflowArgsV3(argsCtx)
		}
		action.ClosePage = actCtx.CLOSE_PAGE() != nil
```

- [ ] **Step 4.2：确认 build 通过**

```bash
go build ./... 2>&1 | head -10
```

- [ ] **Step 4.3：运行 L1 语法测试，确认 parse 通过（BSON 还未对）**

```bash
./bin/mxcli check mdl-examples/doctype-tests/pages/action-microflow-close-page.mdl 2>&1 | head -10
```

期望：parse 通过（不再报 grammar 错误）；可能还有其他错误（entity/microflow 未找到），但不是 parse error。

---

## Task 5：Executor BSON 修改 — `buildClientActionBSON` + `buildClientActionV3`

**Files:**
- Modify: `mdl/executor/cmd_pages_builder_v3.go`

- [ ] **Step 5.1：修改 `buildClientActionBSON` 的 microflow case（约 2225-2246 行）**

将当前 microflow case 的 `return bson.D{...}` 改为（加 `ClosePage` key）：

```go
	case "microflow":
		// Resolution is for validation only — BSON stores the qualified name string.
		// Log a warning for missing microflows but continue building the action.
		if _, err := pb.resolveMicroflow(action.Target); err != nil {
			log.Printf("warning: action microflow %s not found (will still create action by name)", action.Target)
		}
		return bson.D{
			{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
			{Key: "$Type", Value: "Forms$MicroflowAction"},
			{Key: "ClosePage", Value: action.ClosePage},
			{Key: "DisabledDuringExecution", Value: true},
			{Key: "MicroflowSettings", Value: bson.D{
				{Key: "$ID", Value: bsonutil.NewIDBsonBinary()},
				{Key: "$Type", Value: "Forms$MicroflowSettings"},
				{Key: "Asynchronous", Value: false},
				{Key: "ConfirmationInfo", Value: nil},
				{Key: "FormValidations", Value: "All"},
				{Key: "Microflow", Value: action.Target},
				{Key: "ParameterMappings", Value: bson.A{int32(3)}},
				{Key: "ProgressBar", Value: "None"},
				{Key: "ProgressMessage", Value: nil},
			}},
		}, nil
```

- [ ] **Step 5.2：修改 `buildClientActionV3` 的 microflow case（约 1038 行，`act.SetMicroflowSettings(settings)` 之后）**

在 `act.SetMicroflowSettings(settings)` 之后、`return act, nil` 之前，加：

```go
		if action.ClosePage {
			setRawBSONField(act, "ClosePage", true)
		}
		return act, nil
```

- [ ] **Step 5.3：运行单元测试，确认通过**

```bash
go test ./mdl/executor/ -run TestBuildClientActionBSON_Microflow -v 2>&1
```

期望：两个测试 `PASS`。

- [ ] **Step 5.4：运行全量 executor 测试，确认无回归**

```bash
go test ./mdl/executor/ -timeout 120s 2>&1 | tail -5
```

期望：`ok github.com/mendixlabs/mxcli/mdl/executor`。

- [ ] **Step 5.5：运行 L1 syntax 测试确认全通过**

```bash
./bin/mxcli check mdl-examples/doctype-tests/pages/action-microflow-close-page.mdl 2>&1 | head -10
```

期望：无 parse 错误（entity/microflow 引用错误可接受，因为测试没有关联的 MPR）。

- [ ] **Step 5.6：Commit**

```bash
git add mdl/visitor/visitor_page_v3.go \
        mdl/executor/cmd_pages_builder_v3.go \
        mdl/executor/cmd_pages_builder_v3_test.go \
        mdl-examples/doctype-tests/pages/action-microflow-close-page.mdl
git commit -m "feat(pages): action: microflow/nanoflow M.F (...) close_page support"
```

---

## Task 6：Gallery filter bar 验证

**Goal:** 确认 Gallery `filter {}` 内的 `dropdownfilter` 能通过 `mxcli check` 并被 `mx check` 接受。

- [ ] **Step 6.1：新建验证脚本**

新建 `mdl-examples/doctype-tests/pages/gallery-filterbar.mdl`：

```sql
-- @title: Gallery filter bar with dropdownfilter
-- @expect: no error

create module GalTest;
create enumeration GalTest.ArticleStatus (Published 'Published', Draft 'Draft');
create persistent entity GalTest.Article (
  Title: string(500) not null,
  Status: GalTest.ArticleStatus
);

create page GalTest.Article_Gallery (
  title: 'Articles',
  layout: Atlas_Core.Atlas_Default
) {
  gallery artGallery (
    datasource: database from GalTest.Article sort by Title asc,
    selection: single
  ) {
    filter filterBar {
      dropdownfilter fStatus (attributes: [GalTest.Article.Status])
    }
    template template1 {
      dynamictext txtTitle (content: '{1}', contentparams: [{1} = Title], rendermode: H4)
    }
  }
};
```

- [ ] **Step 6.2：针对 corpus-b MPR 运行 check（确认引用可用）**

```bash
./bin/mxcli check mdl-examples/doctype-tests/pages/gallery-filterbar.mdl 2>&1 | head -20
```

**若通过（无错误或仅有 Atlas_Core 引用警告）：** 跳到 Step 6.3。

**若报 BSON 构造错误（关于 filter bar 或 dropdownfilter）：** 按以下步骤修复：

在 `mdl/executor/cmd_pages_builder_v3.go` 中，找到 `buildFilterV3`（约 2838 行）。当前它创建了一个 `DivContainer`，但 Gallery 的 filter bar 不是 `DivContainer`——Gallery 作为 pluggable widget，其 filter bar 需要通过 pluggable widget engine 处理。

检查 Gallery 的 widget definition（`sdk/widgets/templates/` 或 pluggable registry）确认 filter bar 属性名。如果 Gallery 通过 pluggable engine 处理，filter bar children 需要走 pluggable path，不能走 gen path。

具体修复：在 `buildGalleryV3` 或 Gallery pluggable widget builder 中，识别 `filter` 类型的子 widget 并作为 Gallery 的 filterList/filterProps 处理（路径取决于 Gallery widget definition 的结构）。

- [ ] **Step 6.3：确认 `mx check` 通过（在 corpus-b MPR 上执行）**

```bash
./bin/mxcli -p testdata/corpus-b/app.mpr -c "$(cat mdl-examples/doctype-tests/pages/gallery-filterbar.mdl)" 2>&1 | head -10
~/.mxcli/mxbuild/11.6.4/modeler/mx check testdata/corpus-b/app.mpr 2>&1 | grep -i "StorageLoadException\|CE0" | head -10
git restore testdata/corpus-b/
```

- [ ] **Step 6.4：Commit**

```bash
git add mdl-examples/doctype-tests/pages/gallery-filterbar.mdl
# 若有 executor 修改也加进来
git commit -m "test(pages): add Gallery filter bar doctype test + fix if needed"
```

---

## Task 7：Pattern A — 替换 `HD.Ticket_Overview`

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 7.1：在 helpdesk-app.mdl 中，找到现有 `create page HD.Ticket_Overview` 块，替换为以下内容**

（当前位置：约第 957-979 行，`-- Ticket overview with DataGrid2 column filters` 注释到下一个 `create page` 前）

```sql
-- Ticket overview: DataGrid2 CRUD overview
-- Pattern A: controlbar New button, column filters, sort, Edit action column
create page HD.Ticket_Overview
(
  title: 'Tickets',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Ticket'
)
{
  layoutgrid lgMain {
    row rMain {
      column cMain (desktopwidth: 12) {
        datagrid dgTickets (
          datasource: database from HD.Ticket sort by SLADueAt asc,
          PageSize: 20,
          PagingPosition: both
        ) {
          controlbar cb1 {
            actionbutton btnNew (
              caption: 'New Ticket',
              action: create_object HD.Ticket then show_page HD.Ticket_NewEdit,
              buttonstyle: primary
            )
          }
          column colSubject  (attribute: Subject,   caption: 'Subject') {
            textfilter fSubject
          }
          column colStatus   (attribute: Status,    caption: 'Status',   ColumnWidth: manual, Size: 120) {
            dropdownfilter fStatus
          }
          column colPriority (attribute: Priority,  caption: 'Priority', ColumnWidth: manual, Size: 100) {
            dropdownfilter fPriority
          }
          column colSLADue   (attribute: SLADueAt,  caption: 'SLA Due',  ColumnWidth: manual, Size: 140) {
            datefilter fSLADue
          }
          column colIsOver   (attribute: IsOverSLA, caption: 'Overdue',  ColumnWidth: manual, Size: 90)
          column colActions  (caption: 'Actions', ShowContentAs: customContent, ColumnWidth: manual, Size: 80) {
            actionbutton btnEdit (
              caption: 'Edit',
              action: show_page HD.Ticket_Detail (Ticket: $currentObject),
              buttonstyle: default
            )
          }
        }
      }
    }
  }
};
```

- [ ] **Step 7.2：运行 mxcli check 确认语法通过**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | grep -i "error\|fatal" | head -10
```

期望：无 parse 或 fatal 错误。

- [ ] **Step 7.3：Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): Pattern A — HD.Ticket_Overview DataGrid2 CRUD + column filters"
```

---

## Task 8：Pattern B — 替换 `KB.Article_Overview`

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 8.1：找到现有 `create page KB.Article_Overview` 块，替换为**

（当前位置：约第 903-923 行，`-- MARK: Pages — KnowledgeBase` 下的第一个页面）

```sql
-- KB Article overview: Gallery (left) + DataView master-detail (right)
-- Pattern B: Gallery selection:single + filter bar, DataView datasource:selection
create page KB.Article_Overview
(
  title: 'Knowledge Base',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Article'
)
{
  layoutgrid lgMain {
    row rMain {
      column cList (desktopwidth: 5) {
        gallery artGallery (
          datasource: database from KB.Article sort by PublishedAt desc,
          selection: single
        ) {
          filter filterBar {
            dropdownfilter fStatus (attributes: [KB.Article.Status])
          }
          template template1 {
            dynamictext txtTitle  (content: '{1}', contentparams: [{1} = Title],  rendermode: H4)
            dynamictext txtStatus (content: '{1}', contentparams: [{1} = Status])
          }
        }
      }
      column cDetail (desktopwidth: 7) {
        dataview dvArticleDetail (datasource: selection artGallery) {
          dynamictext txtTitle     (content: '{1}', contentparams: [{1} = Title],       rendermode: H2)
          dynamictext txtStatus    (content: '{1}', contentparams: [{1} = Status])
          dynamictext txtPublished (content: '{1}', contentparams: [{1} = PublishedAt])
          dynamictext txtContent   (content: '{1}', contentparams: [{1} = Content])
          footer ftrActions {
            actionbutton btnPublish (
              caption: 'Publish',
              action: microflow KB.ACT_Article_Publish (Article: $currentObject),
              buttonstyle: primary
            )
            actionbutton btnArchive (
              caption: 'Archive',
              action: microflow KB.ACT_Article_Archive (Article: $currentObject),
              buttonstyle: default
            )
          }
        }
      }
    }
  }
};
```

- [ ] **Step 8.2：运行 mxcli check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | grep -i "error\|fatal" | head -10
```

- [ ] **Step 8.3：Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): Pattern B — KB.Article_Overview Gallery master-detail"
```

---

## Task 9：Pattern C — 替换 `HD.Ticket_Detail`

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 9.1：找到现有 `create page HD.Ticket_Detail` 块（约 983-1015 行），替换为**

```sql
-- Ticket detail: DataView (page param) + ActionButtons + nested association DataGrid
-- Pattern C: datasource: $Ticket/HD.TicketComment_Ticket/HD.TicketComment (no XPath needed)
create page HD.Ticket_Detail
(
  title: 'Ticket',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Ticket',
  params: { $Ticket: HD.Ticket }
)
{
  layoutgrid lgMain {
    row rHeader {
      column cHeader (desktopwidth: 12) {
        dataview dvTicket (datasource: $Ticket) {
          dynamictext txtSubject  (content: '{1}', contentparams: [{1} = Subject],  rendermode: H2)
          dynamictext txtStatus   (content: '{1}', contentparams: [{1} = Status])
          dynamictext txtPriority (content: '{1}', contentparams: [{1} = Priority])
          dynamictext txtSLADue   (content: '{1}', contentparams: [{1} = SLADueAt])
          footer ftrActions {
            actionbutton btnSubmit (
              caption: 'Submit',
              action: microflow HD.ACT_Ticket_Submit (Ticket: $Ticket),
              buttonstyle: primary
            )
            actionbutton btnResolve (
              caption: 'Resolve',
              action: microflow HD.ACT_Ticket_Resolve (Ticket: $Ticket),
              buttonstyle: default
            )
            actionbutton btnReopen (
              caption: 'Reopen',
              action: microflow HD.ACT_Ticket_Reopen (Ticket: $Ticket),
              buttonstyle: default
            )
            actionbutton btnAssignAgent (
              caption: 'Assign Agent',
              action: show_page HD.Agent_Select (Ticket: $Ticket),
              buttonstyle: default
            )
          }
        }
      }
    }
    row rComments {
      column cComments (desktopwidth: 12) {
        datagrid dgComments (
          datasource: $Ticket/HD.TicketComment_Ticket/HD.TicketComment
        ) {
          column colContent_    (attribute: Content,    caption: 'Comment')
          column colIsInternal (attribute: IsInternal, caption: 'Internal', ColumnWidth: manual, Size: 90)
        }
      }
    }
  }
};
```

- [ ] **Step 9.2：运行 mxcli check**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | grep -i "error\|fatal" | head -10
```

- [ ] **Step 9.3：Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): Pattern C — HD.Ticket_Detail association datasource + ActionButtons"
```

---

## Task 10：Pattern D — 新增 `HD.Agent_Select` 弹窗选择页 + 安全授权

**Files:**
- Modify: `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`

- [ ] **Step 10.1：在 helpdesk-app.mdl 的 `-- MARK: Pages — HelpDesk` 区域末尾（`HD.TicketSearch_Form` 之后），插入新页面**

在 `HD.TicketSearch_Form` 页面定义之后（约 1017-1033 行之后），插入：

```sql
-- Agent selection popup: DataGrid (selection:single) + microflow close_page
-- Pattern D: popup selection — lists HD.Agent, calls ACT_Ticket_Assign, closes popup
create page HD.Agent_Select
(
  title: 'Select Agent',
  layout: Atlas_Core.PopupLayout,
  folder: 'Ticket',
  params: { $Ticket: HD.Ticket }
)
{
  datagrid dgAgents (
    datasource: database from HD.Agent sort by Name asc,
    PageSize: 15,
    selection: single
  ) {
    controlbar cb1 {
      actionbutton btnAssign (
        caption: 'Assign',
        action: microflow HD.ACT_Ticket_Assign (Ticket: $Ticket, Agent: $currentObject) close_page,
        buttonstyle: primary
      )
    }
    column colName     (attribute: Name,     caption: 'Name')
    column colEmail    (attribute: Email,    caption: 'Email')
    column colIsActive (attribute: IsActive, caption: 'Active', ColumnWidth: manual, Size: 80)
  }
};
```

- [ ] **Step 10.2：在 `-- MARK: Security — User Roles & Grants` 区域的 Page grants 末尾（约 1215-1221 行），加上 `HD.Agent_Select` 的授权**

在最后一个 `grant view on page` 之后，加：

```sql
grant view on page HD.Agent_Select        to HD.AgentRole, HD.ManagerRole;
grant view on page HD.Ticket_NewEdit      to HD.CustomerRole, HD.AgentRole, HD.ManagerRole;
grant view on page KB.Article_NewEdit     to KB.Contributor, KB.Admin;
```

> 注：`HD.Ticket_NewEdit` 和 `KB.Article_NewEdit` 在现有代码中已存在但没有 grant，顺便补上。

- [ ] **Step 10.3：在 MARK: Folder Organization — move 区域，为新页面加 move 命令**

在 `HD Module — Ticket/Search` 部分之后，加：

```sql
move page      HD.Agent_Select             to folder 'Ticket';
```

- [ ] **Step 10.4：运行 mxcli check 全文件验证**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1 | head -20
```

期望：0 parse/fatal 错误。reference 错误（Atlas_Core/MyFirstModule）可接受。

- [ ] **Step 10.5：Commit**

```bash
git add mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
git commit -m "feat(helpdesk): Pattern D — HD.Agent_Select popup selection + close_page + grants"
```

---

## Task 11：端到端验证 + Golden 更新

**Files:**
- Modify: `testdata/helpdesk-golden/describe-snapshot.mdl`
- Modify: `internal/goldenfs/helpdesk_regression_test.go` (if needed)

- [ ] **Step 11.1：完整 mxcli check 验证**

```bash
./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl 2>&1
```

记录所有错误。合法错误（entity 引用、Atlas_Core 引用）不是问题；parse/BSON 错误必须修复。

- [ ] **Step 11.2：在 corpus-b MPR 上运行完整 MDL 文件，检查 mx check**

```bash
# 建立 baseline
~/.mxcli/mxbuild/11.6.4/modeler/mx check testdata/corpus-b/app.mpr 2>&1 | grep -c "StorageLoadException\|CE0" > /tmp/baseline_errors.txt
cat /tmp/baseline_errors.txt

# 执行 helpdesk MDL
./bin/mxcli -p testdata/corpus-b/app.mpr -c "$(cat mdl-examples/use-cases/helpdesk/helpdesk-app.mdl)" 2>&1 | tail -5

# mx check 再次，确认无新增错误
~/.mxcli/mxbuild/11.6.4/modeler/mx check testdata/corpus-b/app.mpr 2>&1 | grep "StorageLoadException\|CE0" | head -20
```

若出现新的 `StorageLoadException`，需要修复 BSON（回到 Task 5 或 Task 6）。

- [ ] **Step 11.3：恢复 corpus-b**

```bash
git restore testdata/corpus-b/
```

- [ ] **Step 11.4：更新 golden 文件**

```bash
go test ./internal/goldenfs/ -run TestHelpdesk -update 2>&1 | tail -5
```

若 `-update` flag 不存在，手动运行并将输出写入 golden：

```bash
go test ./internal/goldenfs/ -run TestHelpdesk 2>&1 | tail -10
```

若 golden 测试失败，检查 `internal/goldenfs/helpdesk_regression_test.go` 中的更新机制并按其说明更新 `testdata/helpdesk-golden/describe-snapshot.mdl`。

- [ ] **Step 11.5：运行全量测试**

```bash
go test ./... -timeout 180s 2>&1 | grep -E "FAIL|ok" | tail -20
```

期望：所有包 `ok`，无 `FAIL`。

- [ ] **Step 11.6：最终 Commit**

```bash
git add testdata/helpdesk-golden/ internal/goldenfs/
git commit -m "test(golden): regenerate helpdesk golden after entity-container page upgrades"
```

---

## 自检：Spec 覆盖率

| Spec 需求 | 对应 Task |
|-----------|-----------|
| `action: microflow ... close_page` 语法实现 | Task 1-5 |
| Gallery filter bar 验证/实现 | Task 6 |
| Pattern A: HD.Ticket_Overview CRUD DataGrid2 | Task 7 |
| Pattern B: KB.Article_Overview Gallery master-detail | Task 8 |
| Pattern C: HD.Ticket_Detail 关联路径嵌套 DataGrid | Task 9 |
| Pattern D: HD.Agent_Select 弹窗选择 | Task 10 |
| 安全授权 HD.Agent_Select | Task 10 |
| mx check 验证 + golden 更新 | Task 11 |
| 禁止降级实现（先实现缺口再应用） | Task 1-6 必须在 Task 7-10 之前完成 |
