# HelpDesk 实体容器驱动页面设计

**日期：** 2026-05-23  
**目标文件：** `mdl-examples/use-cases/helpdesk/helpdesk-app.mdl`  
**约束：** 禁止降级实现——遇到未支持的 MDL 语法必须先实现，再应用

---

## 1. 目标

将 `helpdesk-app.mdl` 的骨架页面升级为真实的实体容器驱动设计，覆盖四类核心模式：

| 模式 | 页面 | 演示重点 |
|------|------|----------|
| A — DataGrid2 CRUD Overview | `HD.Ticket_Overview` | controlbar、列过滤、操作列、分页 |
| B — Gallery + DataView master-detail | `KB.Article_Overview` | selection 绑定、Gallery filter bar |
| C — DataView + 关联路径嵌套 DataGrid | `HD.Ticket_Detail` | association datasource、ActionButton 微流调用 |
| D — 弹窗选择（Popup Selection） | `HD.Agent_Select`（新增） | PopupLayout DataGrid selection + microflow close_page |

---

## 2. MDL 语法缺口（必须先实现）

### 缺口 1：`action: microflow M.F (...) close_page`

**现状：** `ActionV3` 的 microflow/nanoflow 动作不支持 `close_page` 修饰符。`close_page` 仅对 `save_changes` 和 `cancel_changes` 有效。

**需求：** 弹窗选择按钮调用指派微流后必须关闭弹窗：
```sql
actionbutton btnAssign (
  caption: 'Assign',
  action: microflow HD.ACT_Ticket_Assign (Ticket: $Ticket, Agent: $currentObject) close_page,
  buttonstyle: primary
)
```

**实现范围：**
- `mdl/grammar/domains/MDLPage.g4`：在 microflow/nanoflow action 规则末尾添加可选 `CLOSE_PAGE?`
- `mdl/ast/ast_page_v3.go`：`ActionV3` 添加 `CloseAfterCall bool` 字段
- `mdl/visitor/visitor_page_v3.go`：`buildActionV3()` 解析 CLOSE_PAGE token
- `mdl/executor/cmd_pages_builder_v3.go` / `mdl/backend/mpr/`：BSON 写入时设置 `ClosePageAction` 组合动作或 `MicroflowClientAction.closePageAfterMicroflow = true`

**验证方法：** `mx check` 通过，弹窗按钮调用微流后页面自动关闭。

---

### 缺口 2：Gallery filter bar 验证

**现状：** `master-detail-pages.md` 技能文件标记为"已实现"，但 explore 搜索未找到 executor 中 Gallery filter bar 的完整 BSON 写入路径。

**动作：** 在实现阶段先运行 `./bin/mxcli check` 验证 Gallery filter bar 语法是否通过解析。若 `mx check` 报 BSON 错误，则补全 executor 中的 filter bar BSON 构造。

**语法目标：**
```sql
gallery artGallery (datasource: database from KB.Article sort by PublishedAt desc, selection: single) {
  filter filterBar {
    dropdownfilter fStatus (attributes: [KB.Article.Status])
  }
  template template1 {
    dynamictext txtTitle  (content: '{1}', contentparams: [{1} = Title],  rendermode: H4)
    dynamictext txtStatus (content: '{1}', contentparams: [{1} = Status])
  }
}
```

---

## 3. 页面设计详情

### 模式 A — `HD.Ticket_Overview`（替换现有页面）

**现状问题：** `datagrid dgTickets` 无 controlbar、无列过滤、无分页、无操作列。

**升级后结构：**
```sql
create page HD.Ticket_Overview (
  title: 'Tickets',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Ticket'
) {
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
          column colSubject  (attribute: Subject,  caption: 'Subject') {
            textfilter fSubject
          }
          column colStatus   (attribute: Status,   caption: 'Status',   ColumnWidth: manual, Size: 120) {
            dropdownfilter fStatus
          }
          column colPriority (attribute: Priority, caption: 'Priority', ColumnWidth: manual, Size: 100) {
            dropdownfilter fPriority
          }
          column colSLADue   (attribute: SLADueAt, caption: 'SLA Due',  ColumnWidth: manual, Size: 140) {
            datefilter fSLADue
          }
          column colIsOver   (attribute: IsOverSLA, caption: 'Overdue', ColumnWidth: manual, Size: 90)
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

**关键点：**
- `sort by SLADueAt asc`：到期时间升序，紧急工单置顶
- 列过滤：Subject（textfilter）、Status（dropdownfilter）、Priority（dropdownfilter）、SLADueAt（datefilter）
- 操作列：只有 Edit，不加 Delete（工单不应从 overview 删除）
- `$currentObject` 引用当前行的 Ticket

---

### 模式 B — `KB.Article_Overview`（替换现有页面）

**现状问题：** 单列 `datagrid dgArticles`，无 selection，无 DataView 联动。

**升级后结构（二栏 master-detail）：**
```sql
create page KB.Article_Overview (
  title: 'Knowledge Base',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Article'
) {
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
          dynamictext txtTitle     (content: '{1}', contentparams: [{1} = Title],    rendermode: H2)
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

**关键点：**
- Gallery 左列（5列宽）：`selection: single` + filterbar by Status
- DataView 右列（7列宽）：`datasource: selection artGallery` 随选中行变化
- DataView 内 ActionButton 调用微流：使用缺口1实现的 `action: microflow` 语法（此处不需要 close_page）
- `$currentObject` 在 DataView 内指向选中的 Article

---

### 模式 C — `HD.Ticket_Detail`（替换现有页面）

**现状问题：**
1. Comments datagrid 使用 `datasource: database HD.TicketComment`（全表查询，无关联过滤）
2. ActionButton 只有 caption，无 action 绑定
3. 没有 "Assign Agent" 入口

**升级后结构：**
```sql
create page HD.Ticket_Detail (
  title: 'Ticket',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Ticket',
  params: { $Ticket: HD.Ticket }
) {
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
          column colContent    (attribute: Content,    caption: 'Comment')
          column colIsInternal (attribute: IsInternal, caption: 'Internal', ColumnWidth: manual, Size: 90)
        }
      }
    }
  }
};
```

**关键点：**
- `datasource: $Ticket/HD.TicketComment_Ticket/HD.TicketComment`：关联路径 datasource，自动过滤当前工单的评论，**无需 XPath**
- ActionButton 绑定真实微流调用（需要缺口1的 `action: microflow` 支持）
- "Assign Agent" 按钮打开模式 D 的弹窗选择页（传递 $Ticket 参数）
- DataView `datasource: $Ticket` 来自页面参数

---

### 模式 D — `HD.Agent_Select`（新增页面）

**用途：** 弹窗选择 Agent，指派到 Ticket。演示 PopupLayout DataGrid selection + microflow close_page。

```sql
create page HD.Agent_Select (
  title: 'Select Agent',
  layout: Atlas_Core.PopupLayout,
  folder: 'Ticket',
  params: { $Ticket: HD.Ticket }
) {
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

**关键点：**
- `layout: Atlas_Core.PopupLayout`：弹窗布局
- `params: { $Ticket: HD.Ticket }`：从 Ticket_Detail 传入上下文
- `datagrid dgAgents (... selection: single)`：单选模式（DataGrid selection 与 Gallery 不同，需验证 BSON 路径）
- `action: microflow HD.ACT_Ticket_Assign (...) close_page`：**需要缺口1实现**
  - 传参：`Ticket: $Ticket`（页面参数）+ `Agent: $currentObject`（选中行）
  - 调用成功后自动关闭弹窗

---

## 4. 安全授权补充

新增页面 `HD.Agent_Select` 需要页面访问授权：
```sql
grant view on page HD.Agent_Select to HD.AgentRole, HD.ManagerRole;
```

---

## 5. 实现顺序

```
Phase 1 — MDL 语法实现（先实现，再应用）
  1a. 实现 action: microflow ... close_page（grammar + AST + visitor + executor/backend）
  1b. 验证 Gallery filter bar；若有 BSON 缺口则补全

Phase 2 — helpdesk-app.mdl 页面升级
  2a. 替换 HD.Ticket_Overview（模式 A）
  2b. 替换 KB.Article_Overview（模式 B）
  2c. 替换 HD.Ticket_Detail（模式 C）
  2d. 新增 HD.Agent_Select（模式 D）
  2e. 补充 grant view on page HD.Agent_Select

Phase 3 — 验证
  3a. ./bin/mxcli check mdl-examples/use-cases/helpdesk/helpdesk-app.mdl
  3b. mx check testdata/corpus-b/app.mpr（无新增错误）
  3c. 更新 golden 测试文件
```

---

## 6. 不在范围内

- `HD.Ticket_NewEdit`、`KB.Article_NewEdit` 页面不升级（已有 dataview 骨架，不属于本次演示范围）
- `HD.TicketSearch_Form`、`HD.EscalationStart_Form` 不升级
- `combobox`、`datepicker` widget 不引入（已知 CE0463/CE2421 限制，属于独立实现任务）
