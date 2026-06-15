# Academy Phase 3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 Academy 扩展阶段（模块 08–11）、AI 协作总结（模块 12），以及英文版内容填充。

**Architecture:**
- 模块 08–11 各自独立，可在 Phase 1–2 完成后任意顺序执行
- 模块 08（Java Action）：提供 Java 源码 + 调用 MDL；不要求学员有 Studio Pro
- 模块 09（JS Action）：内联 JavaScript 写入 MDL，只需 mxcli
- 模块 10（Widget 开发）：提供预编译 `.mpk`；可选学完整 React 开发路径
- 模块 11（主题定制）：CSS 变量方案（无需 Studio Pro）+ SCSS 高级路径说明
- 模块 12：纯文档（AI 协作模式总结，无 MDL）
- 英文版：翻译所有 Markdown 文档，MDL 文件共享（语言无关）

**Tech Stack:** MDL、Java（JDK 11+）、TypeScript/React（Widget）、SCSS/CSS（主题）、Markdown

**前提：** Phase 1 + Phase 2 计划已完整执行。

---

## 文件清单（Phase 3 新增）

```
academy/zh/
├── 08-扩展-Java-Action/
│   ├── 业务需求.md
│   ├── AI协作指南.md
│   ├── java-source/
│   │   └── JA_HashPassword.java
│   └── 参考实现/
│       └── call-java-action.mdl
├── 09-扩展-JS-Action/
│   ├── 业务需求.md
│   ├── AI协作指南.md
│   └── 参考实现/
│       └── js-actions.mdl
├── 10-扩展-Widget开发/
│   ├── 业务需求.md
│   ├── AI协作指南.md
│   ├── widget-source/
│   │   ├── package.json
│   │   └── src/
│   │       ├── TicketStatusBadge.tsx
│   │       └── TicketStatusBadge.editorPreview.tsx
│   └── 参考实现/
│       └── use-widget.mdl
├── 11-扩展-主题定制/
│   ├── 业务需求.md
│   ├── AI协作指南.md
│   └── theme/
│       └── helpdesk-theme.css
└── 12-AI协作模式/
    ├── README.md
    ├── prompt-templates/
    │   ├── 创建实体.md
    │   ├── 调试错误.md
    │   └── 审查代码.md
    └── 案例分析/
        └── 从需求到实现-完整案例.md

academy/en/
├── 00-getting-started/
│   ├── application-vision.md
│   ├── ai-collaboration-guide.md
│   └── 参考实现/ → 软链接或说明：MDL files shared with zh/
├── 01-domain-modeling/
├── 02-microflows/
├── 03-nanoflows/
├── 04-pages-ui/
├── 05-security/
├── 06-knowledge-base/
├── 07-escalation-workflow/
├── 08-extension-java-action/
├── 09-extension-js-action/
├── 10-extension-widget-dev/
├── 11-extension-theming/
├── 12-ai-collaboration/
└── capstone-helpdesk/
    ├── business-requirements.md
    └── execution-guide.md
```

---

## Task 1：创建 Phase 3 目录

- [ ] **Step 1：创建中文扩展模块目录**

```bash
mkdir -p academy/zh/08-扩展-Java-Action/java-source
mkdir -p academy/zh/08-扩展-Java-Action/参考实现
mkdir -p academy/zh/09-扩展-JS-Action/参考实现
mkdir -p academy/zh/10-扩展-Widget开发/widget-source/src
mkdir -p academy/zh/10-扩展-Widget开发/参考实现
mkdir -p academy/zh/11-扩展-主题定制/theme
mkdir -p academy/zh/12-AI协作模式/prompt-templates
mkdir -p academy/zh/12-AI协作模式/案例分析
```

- [ ] **Step 2：创建英文版目录（含所有模块占位）**

```bash
for dir in \
  "00-getting-started" \
  "01-domain-modeling" \
  "02-microflows" \
  "03-nanoflows" \
  "04-pages-ui" \
  "05-security" \
  "06-knowledge-base" \
  "07-escalation-workflow" \
  "08-extension-java-action" \
  "09-extension-js-action" \
  "10-extension-widget-dev" \
  "11-extension-theming" \
  "12-ai-collaboration" \
  "capstone-helpdesk"; do
  mkdir -p "academy/en/$dir"
done
```

- [ ] **Step 3：提交**

```bash
git add academy/zh/08-扩展-Java-Action academy/zh/09-扩展-JS-Action \
        academy/zh/10-扩展-Widget开发 academy/zh/11-扩展-主题定制 \
        academy/zh/12-AI协作模式 academy/en/
git commit -m "chore(academy): create Phase 3 directory structure"
```

---

## Task 2：Module 08 — Java Action 扩展

**Files:**
- Create: `academy/zh/08-扩展-Java-Action/业务需求.md`
- Create: `academy/zh/08-扩展-Java-Action/AI协作指南.md`
- Create: `academy/zh/08-扩展-Java-Action/java-source/JA_HashPassword.java`
- Create: `academy/zh/08-扩展-Java-Action/参考实现/call-java-action.mdl`

- [ ] **Step 1：写 业务需求.md**

```markdown
# 模块 08：Java Action 扩展 — 业务需求

## 业务背景

系统管理员反映，Helpdesk 的密码修改功能目前是明文比较——这在任何生产环境都是不可接受的安全风险。
密码必须经过加密（哈希）后才能存储和比对。

Mendix 内置的业务逻辑（微流/纳流）不能直接调用加密算法库，这需要通过 **Java Action** 来扩展。

---

## 用户故事

- 作为系统，我想对用户提交的密码进行 BCrypt 哈希，然后再存储，这样即使数据库泄露，密码也是安全的
- 作为系统，我想比对用户输入的密码与存储的哈希值，以便安全地验证登录
- 作为开发者，我想通过在微流中调用 Java Action 来复用加密逻辑，不需要在多处重写

---

## 业务规则

- 密码最短 8 位，必须包含数字和字母
- 哈希算法：BCrypt（cost factor: 12）
- 同一密码每次哈希结果不同（BCrypt 包含随机 salt）
- 验证：`BCrypt.checkpw(rawPassword, storedHash)` 返回 true/false

---

## 验收标准

- [ ] 微流可以调用 JA_HashPassword Java Action，传入明文密码，获得哈希字符串
- [ ] 微流可以调用 JA_VerifyPassword Java Action，传入明文密码和哈希，获得 boolean 结果
- [ ] 所有密码相关操作经过 Java Action，不出现明文密码比较
```

- [ ] **Step 2：写 AI协作指南.md**

```markdown
# 模块 08：AI 协作指南 — Java Action 扩展

## 工具要求

| 工具 | 用途 | 最低版本 |
|------|------|---------|
| Mendix Studio Pro | 创建 Java Action 框架（`.java` 文件生成）| 11.x |
| JDK | 编译 Java 代码 | 11+ |
| mxcli | 调用 Java Action 的微流（MDL） | 最新版 |

> **快速路径（无 Studio Pro）：**
> 使用 `java-source/JA_HashPassword.java` 中提供的源码，在 Studio Pro 中创建同名 Java Action 后，
> 将源码粘贴进去。或直接使用 Marketplace 中的 `CommunityCommons` 模块，它已内置 `BCrypt` 工具类。

## 两种路径

### 路径 A：使用 CommunityCommons（最快，推荐演示用）

1. 在 Studio Pro 中安装 CommunityCommons Marketplace 模块
2. 在微流中调用 `CommunityCommons.BCryptHash` 和 `CommunityCommons.BCryptCheck`
3. 对应 MDL（调用方式）见 `参考实现/call-java-action.mdl`

### 路径 B：自己实现 Java Action（完整学习路径）

1. Studio Pro → App Explorer → 右键模块 HD → Add Java Action
2. 命名：`JA_HashPassword`，添加参数 `Password`（String），返回类型 `String`
3. 打开 `javasource/helpdesk/actions/JA_HashPassword.java`
4. 把 `java-source/JA_HashPassword.java` 的内容粘贴进去
5. 在 Studio Pro 中添加 `bcrypt.jar` 依赖（或用 Maven）
6. 用 mxcli exec 运行 `参考实现/call-java-action.mdl` 创建调用微流

## 与 Claude 协作

```
帮我用 MDL 实现一个微流 HD.ACT_ChangePassword，
它调用 Java Action JA_HashPassword（模块内，参数：Password: string，返回 string），
然后把哈希结果存储到当前用户的 PasswordHash 属性。

另外实现 HD.ACT_VerifyPassword，
调用 JA_VerifyPassword（参数：Password: string，HashedPassword: string，返回 boolean）
```

## 关键 MDL 语法

```mdl
-- 调用 Java Action 的语法
declare $Hash: string = call java action HD.JA_HashPassword (
  Password = $Password
);
```

## 常见坑

| 坑 | 解决 |
|----|------|
| `call java action` 找不到 Action | Java Action 必须在 Studio Pro 中创建后，mxcli 才能引用 |
| 返回类型不匹配 | 确认 Java Action 在 Studio Pro 中的返回类型与 MDL 声明一致 |
| BCrypt jar 缺失 | 在 Studio Pro 的 `userlib/` 目录放入 bcrypt jar，或用 Maven |
```

- [ ] **Step 3：写 java-source/JA_HashPassword.java**

```java
// ============================================================
// JA_HashPassword — BCrypt 密码哈希 Java Action
// 在 Mendix Studio Pro 中创建同名 Java Action 后，粘贴此内容
// 依赖：bcrypt-0.10.2.jar（放入项目 userlib/ 目录）
// ============================================================
package helpdesk.actions;

import com.mendix.systemwideinterfaces.core.IContext;
import com.mendix.webui.CustomJavaAction;
import at.favre.lib.crypto.bcrypt.BCrypt;

public class JA_HashPassword extends CustomJavaAction<String> {

    private final String password;

    public JA_HashPassword(IContext context, String password) {
        super(context);
        this.password = password;
    }

    @Override
    public String executeAction() throws Exception {
        if (password == null || password.isEmpty()) {
            throw new RuntimeException("Password cannot be empty");
        }
        // cost factor 12 — good balance between security and performance
        return BCrypt.withDefaults().hashToString(12, password.toCharArray());
    }
}
```

- [ ] **Step 4：写 参考实现/call-java-action.mdl**

```mdl
-- ============================================================
-- 模块 08：调用 Java Action — 参考实现
-- 前提：
--   1. 在 Studio Pro 中已创建 HD.JA_HashPassword（返回 String）
--   2. 在 Studio Pro 中已创建 HD.JA_VerifyPassword（返回 Boolean）
--   3. 先运行模块 01–05 的参考实现
-- 运行：mxcli exec call-java-action.mdl -p MyProject.mpr
-- ============================================================

-- ============================================================
-- 密码哈希：调用 Java Action 返回哈希字符串
-- 实际应用：在修改密码前调用此微流
-- ============================================================

create or modify microflow HD.ACT_HashPassword
  ($Password: string)
  returns string as $Hash
  folder 'Security'
{
  if $Password = '' or length($Password) < 8 {
    show message 'Password must be at least 8 characters.' type warning;
    return '';
  }
  declare $Hash: string = call java action HD.JA_HashPassword (
    Password = $Password
  );
  return $Hash;
}
/

-- ============================================================
-- 密码验证：输入明文 + 哈希，返回 true/false
-- ============================================================

create or modify microflow HD.ACT_VerifyPassword
  ($Password: string, $HashedPassword: string)
  returns boolean as $IsValid
  folder 'Security'
{
  declare $IsValid: boolean = call java action HD.JA_VerifyPassword (
    Password       = $Password,
    HashedPassword = $HashedPassword
  );
  return $IsValid;
}
/
```

- [ ] **Step 5：提交**

```bash
git add academy/zh/08-扩展-Java-Action/
git commit -m "docs(academy): add module 08 — Java Action extension (zh)"
```

---

## Task 3：Module 09 — JS Action 扩展

**Files:**
- Create: `academy/zh/09-扩展-JS-Action/业务需求.md`
- Create: `academy/zh/09-扩展-JS-Action/AI协作指南.md`
- Create: `academy/zh/09-扩展-JS-Action/参考实现/js-actions.mdl`

- [ ] **Step 1：写 业务需求.md**

```markdown
# 模块 09：JS Action 扩展 — 业务需求

## 业务背景

客服要把工单编号复制给用户时，总是要手动框选、Ctrl+C，很不顺手。
另外，他们希望浏览器在收到新高优先级工单时能弹出系统通知，而不是每隔几分钟刷新一次页面。

这些操作涉及**浏览器 API**（剪贴板、通知），Mendix 微流运行在服务器端无法直接调用，
需要通过 **JavaScript Action** 在客户端完成。

---

## 用户故事

- 作为客服，我想点一个"复制"按钮就能把工单标题复制到剪贴板，省去手动复制
- 作为客服，我想在浏览器后台收到新工单通知，这样我不需要一直盯着屏幕刷新
- 作为系统，我想把时间戳格式化为友好显示（"3 分钟前"），而不是原始 ISO 格式

---

## 验收标准

- [ ] "复制工单号"按钮：点击后工单标题复制到剪贴板，显示"已复制"提示
- [ ] 浏览器通知：工单优先级为 Critical 时弹出桌面通知（需用户授权）
- [ ] 相对时间格式化：`formatRelativeTime(datetime)` 返回"刚刚"/"3 分钟前"/"2 小时前"等
```

- [ ] **Step 2：写 AI协作指南.md**

```markdown
# 模块 09：AI 协作指南 — JS Action 扩展

## JS Action vs Java Action

| 特性 | JS Action | Java Action |
|------|-----------|-------------|
| 运行环境 | 客户端（浏览器） | 服务器 |
| MDL 语法 | `call javascript action` | `call java action` |
| 适合场景 | 浏览器 API、UI 交互 | 服务器计算、数据库操作 |
| 创建方式 | Studio Pro 或内联 MDL | 只能在 Studio Pro 中创建 |

## 与 Claude 协作

```
帮我用 MDL 实现三个 JavaScript Action 纳流：

1. HD.NF_CopyToClipboard($Text: string)：
   用 navigator.clipboard.writeText() 把 $Text 写入剪贴板
   成功后调用 mx.ui.info('已复制') 显示提示

2. HD.NF_NotifyHighPriority($Subject: string)：
   检查 Notification.permission，如果是 granted 则创建通知
   标题：'高优先级工单'，正文：$Subject

3. HD.NF_FormatRelativeTime($DateTime: DateTime) returns string：
   计算 now - datetime 的差值（毫秒），转换为"刚刚/N分钟前/N小时前/N天前"
```

## JS Action 的 MDL 写法

```mdl
-- 内联 JavaScript Action（Mendix 10.6+ 支持在纳流中内联 JS）
create or modify nanoflow HD.NF_CopyToClipboard
  ($Text: string)
  returns void
  folder 'UI'
{
  call javascript action (
    code = 'navigator.clipboard.writeText($Text).then(() => mx.ui.info("已复制"));',
    parameters = { $Text: string }
  );
}
/
```

## 常见坑

| 坑 | 解决 |
|----|------|
| 剪贴板 API 需要 HTTPS | 本地开发用 localhost 会自动授权；生产环境必须 HTTPS |
| 通知权限未授权 | 先调用 `Notification.requestPermission()`，再检查 permission |
| DateTime 传入 JS | Mendix 的 DateTime 是 JS Date 对象，直接 `.getTime()` 获取毫秒 |
```

- [ ] **Step 3：写 参考实现/js-actions.mdl**

```mdl
-- ============================================================
-- 模块 09：JS Action 扩展 — 参考实现
-- 前提：先运行模块 01（HD 模块必须存在）
-- 运行：mxcli exec js-actions.mdl -p MyProject.mpr
-- 注：call javascript action 语法要求 Mendix 10.6+
-- ============================================================

-- ============================================================
-- 复制到剪贴板（浏览器 Clipboard API）
-- ============================================================

create or modify nanoflow HD.NF_CopyToClipboard
  ($Text: string)
  returns void
  folder 'UI'
{
  call javascript action (
    code = 'if (navigator.clipboard) { navigator.clipboard.writeText($Text).then(() => { mx.ui.info("已复制到剪贴板"); }).catch(() => { mx.ui.error("复制失败，请手动复制"); }); } else { mx.ui.error("浏览器不支持剪贴板 API"); }',
    parameters = { $Text: string }
  );
}
/

-- ============================================================
-- 浏览器桌面通知（Web Notifications API）
-- ============================================================

create or modify nanoflow HD.NF_NotifyHighPriority
  ($Subject: string)
  returns void
  folder 'UI'
{
  call javascript action (
    code = 'if (typeof Notification !== "undefined") { if (Notification.permission === "granted") { new Notification("高优先级工单", { body: $Subject, icon: "/favicon.ico" }); } else if (Notification.permission !== "denied") { Notification.requestPermission().then(function(p) { if (p === "granted") { new Notification("高优先级工单", { body: $Subject }); } }); } }',
    parameters = { $Subject: string }
  );
}
/

-- ============================================================
-- 相对时间格式化："N分钟前" / "N小时前" / "N天前"
-- ============================================================

create or modify nanoflow HD.NF_FormatRelativeTime
  ($DateTime: DateTime)
  returns string as $Result
  folder 'UI'
{
  declare $Result: string = call javascript action (
    code = 'var diff = Date.now() - $DateTime.getTime(); var seconds = Math.floor(diff / 1000); if (seconds < 60) { return "刚刚"; } var minutes = Math.floor(seconds / 60); if (minutes < 60) { return minutes + "分钟前"; } var hours = Math.floor(minutes / 60); if (hours < 24) { return hours + "小时前"; } return Math.floor(hours / 24) + "天前"; ',
    parameters = { $DateTime: DateTime },
    return_type = string
  );
  return $Result;
}
/
```

- [ ] **Step 4：提交**

```bash
git add academy/zh/09-扩展-JS-Action/
git commit -m "docs(academy): add module 09 — JS Action extension (zh)"
```

---

## Task 4：Module 10 — Widget 开发

**Files:**
- Create: `academy/zh/10-扩展-Widget开发/业务需求.md`
- Create: `academy/zh/10-扩展-Widget开发/AI协作指南.md`
- Create: `academy/zh/10-扩展-Widget开发/widget-source/package.json`
- Create: `academy/zh/10-扩展-Widget开发/widget-source/src/TicketStatusBadge.tsx`
- Create: `academy/zh/10-扩展-Widget开发/参考实现/use-widget.mdl`

- [ ] **Step 1：写 业务需求.md**

```markdown
# 模块 10：Widget 开发 — 业务需求

## 业务背景

工单列表里，状态（待处理/处理中/已解决）目前只是普通文字。
客服反映很难快速区分不同状态——他们需要**彩色徽章**来一眼看清工单优先级。

Mendix Atlas UI 的内置组件不提供条件颜色徽章，这需要开发一个**自定义 Widget**。

---

## 用户故事

- 作为客服，我想在工单列表中看到彩色状态徽章（绿=已解决，蓝=处理中，黄=待处理，灰=草稿/关闭），这样我一眼就能知道工单状态
- 作为开发者，我想用一个简单的属性绑定来使用这个 Widget，不需要写 CSS 类名

---

## Widget 规格

**Widget 名称：** TicketStatusBadge

**属性：**
- `statusValue`（Enumeration: HD.TicketStatus）：工单状态

**颜色映射：**
- Draft → 灰色 `#9E9E9E`
- Open → 橙色 `#FF9800`（待处理）
- InProgress → 蓝色 `#2196F3`
- Resolved → 绿色 `#4CAF50`
- Closed → 灰绿色 `#607D8B`

---

## 验收标准

- [ ] Widget 打包为 `.mpk` 文件，可在 Studio Pro 中导入
- [ ] 在页面中绑定 Status 属性，显示对应颜色的圆形徽章 + 文字
- [ ] Widget 在 Mendix 11.x 中正常显示，无 CE0463 错误
```

- [ ] **Step 2：写 AI协作指南.md**

```markdown
# 模块 10：AI 协作指南 — Widget 开发

## 工具要求

| 工具 | 版本 | 用途 |
|------|------|------|
| Node.js | 18+ | 运行 Widget 工具链 |
| pluggable-widgets-tools | 最新 | 编译和打包 Widget |
| Mendix Studio Pro | 11.x | 导入 .mpk，测试 Widget |

## 两种学习路径

### 路径 A：只使用 Widget（5 分钟）

1. 把预编译的 `TicketStatusBadge.mpk`（从此模块提供的构建产物）拖入 Studio Pro
2. 运行 `参考实现/use-widget.mdl` 将 Widget 添加到页面
3. 启动应用，查看效果

### 路径 B：从零开发 Widget（完整路径）

```bash
# 1. 克隆 Widget 模板
npx @mendix/pluggable-widgets-tools@latest create-widget TicketStatusBadge

# 2. 替换源码（从 widget-source/src/ 复制）
cp widget-source/src/TicketStatusBadge.tsx src/TicketStatusBadge.tsx

# 3. 编译 + 打包
npm run build
# 产出：dist/TicketStatusBadge.mpk

# 4. 在 Studio Pro 中导入 .mpk
# App → Import module package → 选择 .mpk

# 5. 用 MDL 在页面中使用
mxcli exec 参考实现/use-widget.mdl -p MyProject.mpr
```

## 与 Claude 协作

```
我有一个 Mendix Pluggable Widget 叫 TicketStatusBadge，
def.json 已定义属性 statusValue（HD.TicketStatus 枚举）。
帮我用 MDL 的 PLUGGABLEWIDGET 语法把它添加到 HD.Ticket_Overview 工单列表的 Status 列中，
替换原来的文字显示。
```

## Widget 使用的 MDL 语法

```mdl
-- 在 DataGrid column 中使用自定义 Widget
column colStatus (caption: 'Status', ColumnWidth: manual, Size: 120, ShowContentAs: customContent) {
  PLUGGABLEWIDGET 'helpdesk.TicketStatusBadge' wdgStatus (
    statusValue: attribute Status
  )
}
```
```

- [ ] **Step 3：写 widget-source/package.json**

```json
{
  "name": "ticket-status-badge",
  "widgetName": "TicketStatusBadge",
  "version": "1.0.0",
  "description": "Colored status badge widget for Helpdesk tickets",
  "packagePath": "helpdesk",
  "dependencies": {},
  "devDependencies": {
    "@mendix/pluggable-widgets-tools": "^10.0.0"
  },
  "scripts": {
    "build": "pluggable-widgets-tools build:prod",
    "dev": "pluggable-widgets-tools start:web"
  }
}
```

- [ ] **Step 4：写 widget-source/src/TicketStatusBadge.tsx**

```tsx
import { Component, ReactNode, createElement } from "react";
import { ValueStatus } from "mendix";
import { TicketStatusBadgeContainerProps } from "../typings/TicketStatusBadgeProps";

const STATUS_COLORS: Record<string, { bg: string; label: string }> = {
    Draft:      { bg: "#9E9E9E", label: "Draft" },
    Open:       { bg: "#FF9800", label: "Open" },
    InProgress: { bg: "#2196F3", label: "In Progress" },
    Resolved:   { bg: "#4CAF50", label: "Resolved" },
    Closed:     { bg: "#607D8B", label: "Closed" }
};

export class TicketStatusBadge extends Component<TicketStatusBadgeContainerProps> {
    render(): ReactNode {
        const { statusValue } = this.props;

        if (!statusValue || statusValue.status !== ValueStatus.Available) {
            return <span style={{ color: "#ccc" }}>—</span>;
        }

        const key = statusValue.value as string;
        const config = STATUS_COLORS[key] ?? { bg: "#ccc", label: key };

        return (
            <span
                style={{
                    display:       "inline-flex",
                    alignItems:    "center",
                    gap:           "6px",
                    padding:       "2px 10px",
                    borderRadius:  "12px",
                    background:    config.bg,
                    color:         "#fff",
                    fontSize:      "12px",
                    fontWeight:    600,
                    whiteSpace:    "nowrap"
                }}
            >
                <span
                    style={{
                        width:        "8px",
                        height:       "8px",
                        borderRadius: "50%",
                        background:   "rgba(255,255,255,0.7)"
                    }}
                />
                {config.label}
            </span>
        );
    }
}
```

- [ ] **Step 5：写 参考实现/use-widget.mdl**

```mdl
-- ============================================================
-- 模块 10：Widget 使用示例
-- 前提：
--   1. TicketStatusBadge.mpk 已在 Studio Pro 中导入
--   2. 模块 01–05 参考实现已执行（HD 模块存在）
-- 运行：mxcli exec use-widget.mdl -p MyProject.mpr
-- ============================================================

-- 重建工单概览页，将 Status 列替换为自定义 Widget
create or replace page HD.Ticket_Overview
(
  title:  'All Tickets',
  layout: Atlas_Core.Atlas_Default,
  folder: 'Ticket'
)
{
  layoutgrid lgMain {
    row rHeader {
      column cTitle (desktopwidth: 12) {
        dynamictext txtTitle (content: 'All Tickets', rendermode: H2)
      }
    }
    row rMain {
      column cMain (desktopwidth: 12) {
        datagrid dgTickets (
          datasource:     database from HD.Ticket sort by SLADueAt asc,
          PageSize:       20,
          PagingPosition: both
        ) {
          controlbar cb {
            actionbutton btnNew (
              caption:     'New Ticket',
              action:      create_object HD.Ticket then show_page HD.Ticket_NewEdit,
              buttonstyle: primary
            )
          }
          column colSubject  (attribute: Subject,  caption: 'Subject') {
            textfilter fSubject
          }
          column colStatus (caption: 'Status', ShowContentAs: customContent, ColumnWidth: manual, Size: 130) {
            PLUGGABLEWIDGET 'helpdesk.TicketStatusBadge' wdgStatus (
              statusValue: attribute Status
            )
          }
          column colPriority (attribute: Priority, caption: 'Priority', ColumnWidth: manual, Size: 100) {
            dropdownfilter fPriority
          }
          column colSLADue   (attribute: SLADueAt, caption: 'SLA Due',  ColumnWidth: manual, Size: 140)
          column colOverdue  (attribute: IsOverSLA, caption: 'Overdue',  ColumnWidth: manual, Size: 80)
          column colActions (caption: 'Actions', ShowContentAs: customContent, ColumnWidth: manual, Size: 80) {
            actionbutton btnOpen (
              caption:     'Open',
              action:      show_page HD.Ticket_Detail (Ticket: $currentObject),
              buttonstyle: link
            )
          }
        }
      }
    }
  }
};
```

- [ ] **Step 6：提交**

```bash
git add academy/zh/10-扩展-Widget开发/
git commit -m "docs(academy): add module 10 — widget development (zh)"
```

---

## Task 5：Module 11 — 主题定制

**Files:**
- Create: `academy/zh/11-扩展-主题定制/业务需求.md`
- Create: `academy/zh/11-扩展-主题定制/AI协作指南.md`
- Create: `academy/zh/11-扩展-主题定制/theme/helpdesk-theme.css`

- [ ] **Step 1：写 业务需求.md**

```markdown
# 模块 11：主题定制 — 业务需求

## 业务背景

TechCorp 的品牌色是深蓝色（`#1565C0`），但 Mendix Atlas UI 默认是紫色调。
IT 部门希望 Helpdesk 应用和公司其他系统保持视觉一致性——至少主色调要对上。

另外，按钮的圆角太大（Atlas 默认 8px），TechCorp 的 UI 规范要求更小的圆角（4px）。

---

## 用户故事

- 作为产品负责人，我想让 Helpdesk 的主色调变为公司品牌蓝色，这样和其他系统风格统一
- 作为 UI 设计师，我想把按钮圆角调小一点，让界面看起来更专业
- 作为开发者，我不想改 Atlas 源码，而是通过 CSS 变量覆盖，这样升级 Atlas 时不会冲突

---

## 品牌要求

| 元素 | 默认 Atlas | TechCorp 要求 |
|------|-----------|--------------|
| 主色（按钮/链接/高亮） | 紫色 `#264AE5` | 品牌蓝 `#1565C0` |
| 主色悬停 | `#1E3AB8` | `#0D47A1` |
| 按钮圆角 | `8px` | `4px` |
| 字体 | Atlas 默认 | 保持不变 |

---

## 验收标准

- [ ] 主要操作按钮显示为品牌蓝色
- [ ] 导航栏主色为品牌蓝色
- [ ] 按钮圆角为 4px
- [ ] 主题变更不影响 mx check（纯 CSS，不动 MDL）
```

- [ ] **Step 2：写 AI协作指南.md**

```markdown
# 模块 11：AI 协作指南 — 主题定制

## 方法：CSS 变量覆盖（无需 Studio Pro）

Atlas UI 使用 CSS 自定义属性（CSS Variables）控制颜色、圆角等视觉参数。
通过在自定义 CSS 文件中覆盖这些变量，可以不触碰 Atlas 源码实现主题定制。

## 三种实现方式

### 方式 A：CSS 变量覆盖（推荐，最简单）

1. 在 Studio Pro 中：App → Styling → 找到自定义 CSS 位置（通常是 `theme/web/custom-variables.css`）
2. 把 `theme/helpdesk-theme.css` 的内容粘贴进去
3. 重新运行项目

### 方式 B：SCSS 变量（需要 Atlas SCSS 工具链）

```bash
# 在 Studio Pro 中 App → Styling → 导出 Atlas SCSS 源码
# 修改 _variables.scss 中的颜色变量
# 重新编译：npm run build（Studio Pro 会自动触发）
```

### 方式 C：与 Claude 协作生成 CSS

```
帮我为 Mendix Atlas UI 生成主题覆盖 CSS：
- 主色：#1565C0（品牌蓝）
- 主色悬停：#0D47A1
- 按钮圆角：4px
- 基于 Atlas UI 的 CSS 变量命名规范
```

## 放置位置

| Studio Pro 版本 | 文件位置 |
|----------------|---------|
| Mendix 9.x     | `theme/web/custom-variables.css` |
| Mendix 10/11.x | `theme/web/main.css`（在文件末尾追加） |

## 验证

无需 mx check——CSS 修改不影响 MPR 结构。
在浏览器中运行应用，检查按钮颜色和圆角是否正确。
```

- [ ] **Step 3：写 theme/helpdesk-theme.css**

```css
/*
 * TechCorp Helpdesk — 品牌主题
 * 覆盖 Atlas UI CSS 变量，无需修改 Atlas 源码。
 * 放置位置：theme/web/main.css（末尾）或 custom-variables.css
 */

:root {
    /* ====================================================
     * 主色（品牌蓝 #1565C0，替换 Atlas 默认紫色）
     * ==================================================== */
    --color-brand-primary:           #1565C0;
    --color-brand-primary-dark:      #0D47A1;
    --color-brand-primary-light:     #1976D2;
    --color-brand-primary-contrast:  #FFFFFF;

    /* ====================================================
     * 按钮样式
     * ==================================================== */
    --border-radius-button:          4px;     /* Atlas 默认 8px */
    --border-radius-default:         4px;

    /* ====================================================
     * 链接颜色
     * ==================================================== */
    --color-link:                    #1565C0;
    --color-link-hover:              #0D47A1;

    /* ====================================================
     * 导航栏颜色（Mendix Atlas Sidebar）
     * ==================================================== */
    --color-nav-background:          #1565C0;
    --color-nav-item-text:           #FFFFFF;
    --color-nav-item-text-hover:     #E3F2FD;
    --color-nav-item-bg-hover:       rgba(255, 255, 255, 0.12);
    --color-nav-item-bg-active:      rgba(255, 255, 255, 0.20);
}

/*
 * 主要操作按钮（Primary）
 */
.btn-primary,
.mx-button.btn-primary {
    background-color: #1565C0;
    border-color:     #1565C0;
}

.btn-primary:hover,
.mx-button.btn-primary:hover {
    background-color: #0D47A1;
    border-color:     #0D47A1;
}

/*
 * 输入框焦点边框颜色
 */
.form-control:focus,
.mx-input:focus {
    border-color: #1565C0;
    box-shadow:   0 0 0 3px rgba(21, 101, 192, 0.15);
}
```

- [ ] **Step 4：提交**

```bash
git add academy/zh/11-扩展-主题定制/
git commit -m "docs(academy): add module 11 — theme customization (zh)"
```

---

## Task 6：Module 12 — AI 协作模式

**Files:**
- Create: `academy/zh/12-AI协作模式/README.md`
- Create: `academy/zh/12-AI协作模式/prompt-templates/创建实体.md`
- Create: `academy/zh/12-AI协作模式/prompt-templates/调试错误.md`
- Create: `academy/zh/12-AI协作模式/prompt-templates/审查代码.md`
- Create: `academy/zh/12-AI协作模式/案例分析/从需求到实现-完整案例.md`

- [ ] **Step 1：写 README.md**

```markdown
# 模块 12：AI 协作模式

## 本模块目的

不是教你用哪个命令，而是教你**怎么想**：当拿到一个需求，如何和 AI 高效协作，
从设计到实现到验证，走完整个循环而不浪费时间。

---

## 核心原则

### 1. 需求优先，代码其次

先让 AI 帮你澄清需求，再让 AI 写代码。
跳过设计直接要代码，往往要改很多次。

**好的工作流：**
```
你：读取需求.md，你理解这个功能的核心业务规则是什么？
AI：[解释业务逻辑]
你：对。现在用 MDL 实现这个微流。
AI：[生成 MDL]
你：执行，检查错误，回来修。
```

**差的工作流：**
```
你：给我写一个工单微流
AI：[猜测生成，通常缺少状态检查]
你：不对，要加状态校验
AI：[修改，又漏了其他东西]
...（循环往复）
```

### 2. 验证驱动：每步都跑 check

不要等写完所有 MDL 再验证——每写完一个微流或页面，就跑一次：

```bash
mxcli check my-file.mdl
mxcli exec  my-file.mdl -p app.mpr
```

越早发现错误，修复成本越低。

### 3. 错误信息是对话的一部分

mxcli 或 mx check 报错时，直接把错误信息粘贴给 Claude：

```
执行后报了这个错误：
[粘贴错误信息]
帮我分析原因并修复
```

不要自己猜，AI 分析错误信息比大多数人快得多。

### 4. 分而治之：一次只做一件事

不要让 AI 一次生成 500 行代码——先让它做领域模型，验证后再做微流，再做页面。

每个阶段完成后都要 commit，这样出问题可以回滚。

---

## 五种典型场景

| 场景 | 推荐提示词模板 |
|------|--------------|
| 创建实体 | `prompt-templates/创建实体.md` |
| 调试 mxcli 错误 | `prompt-templates/调试错误.md` |
| 审查生成的 MDL | `prompt-templates/审查代码.md` |
| 从头设计新功能 | `案例分析/从需求到实现-完整案例.md` |

---

## 常见陷阱与规避

| 陷阱 | 症状 | 规避方法 |
|------|------|---------|
| 让 AI 一次做太多 | 生成的代码有很多小错误 | 分步：先实体，再微流，再页面 |
| 不提供上下文 | AI 生成的代码与现有模型不匹配 | 先 `show structure` 或 `describe entity` |
| 跳过验证 | 发现错误时已经写了很多行 | 每步都跑 mxcli check |
| 接受 AI 第一稿 | 缺少边界条件处理 | 询问"这个实现有哪些边界情况没有处理？" |
```

- [ ] **Step 2：写 prompt-templates/创建实体.md**

```markdown
# 提示词模板：创建实体

## 使用场景

当你要在 Mendix 项目中创建新的数据实体时。

## 模板

```
我有一个 Mendix 项目，使用 MDL（Mendix Definition Language）。

当前上下文：
[粘贴 `mxcli -p app.mpr -c "show structure"` 的输出，或描述现有模块]

我需要创建一个新实体，业务需求如下：
[粘贴业务需求描述]

请：
1. 先列出你理解的实体属性（类型 + 约束 + 默认值）
2. 列出需要的关联（方向 + 多对一/多对多 + 是否可空）
3. 再生成完整的 MDL

要求：
- 使用 `create or modify persistent entity` 语法
- string 类型必须指定长度（如 string(200)）
- 布尔属性必须有 default 值
- 枚举属性使用 `ModuleName.EnumName default Value` 格式
- 关联格式：`create or modify association X from A to B type reference owner default;`
```

## 使用技巧

- 在"当前上下文"中提供足够信息，避免 AI 生成与现有模块冲突的代码
- 步骤 1（列属性）和步骤 2（生成代码）分开，方便你确认设计再执行
```

- [ ] **Step 3：写 prompt-templates/调试错误.md**

```markdown
# 提示词模板：调试 mxcli 错误

## 使用场景

运行 `mxcli exec` 或 `mxcli check` 时报错。

## 模板

```
我在运行 mxcli 时遇到了错误，请帮我分析原因和修复方法。

**执行的命令：**
mxcli exec my-file.mdl -p app.mpr

**错误信息：**
[完整粘贴错误输出，包括行号]

**相关 MDL 代码（出错的部分）：**
[粘贴报错行附近的代码，上下各 5 行]

**我的预期：**
[描述这段代码应该做什么]

请：
1. 解释错误原因
2. 给出修复后的代码
3. 如果有类似的坑，告诉我下次怎么避免
```

## 常见错误类型速查

| 错误关键词 | 通常原因 |
|-----------|---------|
| `unknown type` | 枚举或实体名称拼写错误，或还未创建 |
| `StorageLoadException` | BSON 结构错误，通常是属性类型不匹配 |
| `association not found` | 关联名称错误，或关联方向相反 |
| `parse error` | MDL 语法错误（括号/分号/引号不匹配） |
| `CE0463` | Widget 属性与 def.json 定义不匹配 |
```

- [ ] **Step 4：写 prompt-templates/审查代码.md**

```markdown
# 提示词模板：审查生成的 MDL

## 使用场景

AI 生成了一段 MDL，你想在执行前确认质量。

## 模板

```
请审查以下 MDL 代码，检查是否有问题：

**代码：**
[粘贴要审查的 MDL]

**业务需求：**
[粘贴对应的业务需求描述]

**请检查：**
1. 业务规则是否完整（边界条件、状态校验）
2. 属性类型是否正确（string 有没有长度？boolean 有没有 default？）
3. 微流有没有少 commit？有没有少 return？
4. 关联方向是否正确？
5. 是否存在可能的空指针（如访问了可能为 empty 的对象属性）？

对发现的每个问题，请给出修复后的代码。
```
```

- [ ] **Step 5：写 案例分析/从需求到实现-完整案例.md**

```markdown
# 案例分析：从需求到实现 — 工单提交功能

## 背景

本案例完整展示如何从一段业务需求出发，和 Claude 协作，一步步实现"提交工单"功能。

## Step 1：读取并理解需求

**你：**
```
读取这段业务需求，告诉我这个功能的关键业务规则：

- 作为客户，提交工单时标题不能为空
- 不同优先级的 SLA 截止时间不同（紧急=2h，高=8h，其他=24h）
- 提交成功后工单状态从草稿变为待处理
```

**Claude：**
> 关键规则：(1) 标题非空校验，(2) 按优先级分支计算截止时间，(3) 状态从 Draft → Open，(4) 需要 commit。

## Step 2：生成 MDL

**你：**
```
用 MDL 实现微流 HD.ACT_Ticket_Submit($Ticket: HD.Ticket) returns boolean
```

## Step 3：验证

```bash
mxcli check my-submit.mdl
mxcli exec  my-submit.mdl -p app.mpr
~/.mxcli/mxbuild/*/modeler/mx check app.mpr 2>&1 | grep -c "StorageLoadException"
# 期望：0
```

## Step 4：发现问题，迭代

如果报错：
```
错误：Unknown constant HD.SLA_CRITICAL_HOURS
```

**你告诉 Claude：**
```
报了这个错：Unknown constant HD.SLA_CRITICAL_HOURS
常量还没创建，帮我加上 create or modify constant 语句
```

**Claude 给出修复代码，再跑一次验证。**

## 关键洞察

这个循环（需求理解 → 生成 → 验证 → 修复）通常要跑 2–4 次才能零错误。
这是正常的——不是你的问题，也不是 AI 的问题，而是软件开发本身的复杂性。
AI 让你的每一圈跑得更快，而不是直接跳过这个循环。
```

- [ ] **Step 6：提交**

```bash
git add academy/zh/12-AI协作模式/
git commit -m "docs(academy): add module 12 — AI collaboration patterns (zh)"
```

---

## Task 7：英文版内容

**Files:** 各模块的英文 Markdown 文档（MDL 文件共享，不复制）

**策略：** 每个英文模块只创建 `business-requirements.md` 和 `ai-collaboration-guide.md`，MDL 通过相对路径引用中文版的参考实现（不复制，避免维护双份）。

- [ ] **Step 1：为每个模块创建英文 business-requirements.md**

为以下各目录分别创建英文版 `business-requirements.md`，内容为对应中文 `业务需求.md` 的英文翻译：

```
academy/en/00-getting-started/application-vision.md
academy/en/01-domain-modeling/business-requirements.md
academy/en/02-microflows/business-requirements.md
academy/en/03-nanoflows/business-requirements.md
academy/en/04-pages-ui/business-requirements.md
academy/en/05-security/business-requirements.md
academy/en/06-knowledge-base/business-requirements.md
academy/en/07-escalation-workflow/business-requirements.md
academy/en/08-extension-java-action/business-requirements.md
academy/en/09-extension-js-action/business-requirements.md
academy/en/10-extension-widget-dev/business-requirements.md
academy/en/11-extension-theming/business-requirements.md
academy/en/capstone-helpdesk/business-requirements.md
academy/en/capstone-helpdesk/execution-guide.md
```

翻译规则：
- 保持所有标题结构不变
- 用户故事格式：`As a [role], I want to [action], so that [value]`
- 验收标准保持 `- [ ]` checkbox 格式
- 不翻译 MDL 代码块内容
- 技术术语（MDL, mxcli, BSON 等）不翻译

对于 Module 00，内容是操作指南（非业务需求），翻译 `应用愿景.md` → `application-vision.md`。

- [ ] **Step 2：为每个模块创建英文 ai-collaboration-guide.md**

```
academy/en/00-getting-started/ai-collaboration-guide.md
academy/en/01-domain-modeling/ai-collaboration-guide.md
... （以此类推，共 12 个模块 + capstone）
```

翻译规则与 Step 1 相同。表格中的命令、代码块内容不翻译。

- [ ] **Step 3：创建英文版 README**

```bash
# en/README.md 已在 Phase 1 Task 2 创建，更新内容
```

更新内容（完整替换 en/README.md）：

```markdown
# AI-Assisted Development Academy — English Curriculum

Build a complete IT Helpdesk application using Claude Code + mxcli,
mastering AI-assisted Mendix development from domain modeling to production-ready apps.

---

## Course Map

| Module | Topic | Prerequisites |
|--------|-------|---------------|
| [00 Getting Started](00-getting-started/) | Setup & first exec | None |
| [01 Domain Modeling](01-domain-modeling/) | Entities, enums, associations | 00 |
| [02 Microflows](02-microflows/) | Business logic, state machines | 01 |
| [03 Nanoflows](03-nanoflows/) | Client-side logic | 01 |
| [04 Pages & UI](04-pages-ui/) | Atlas layouts, forms, grids | 01–03 |
| [05 Security](05-security/) | Roles, access rules, row-level XPath | 01–04 |
| [06 Knowledge Base](06-knowledge-base/) | Cross-module, many-to-many | 05 |
| [07 Escalation Workflow](07-escalation-workflow/) | Approval state machine | 02 |
| [08 Java Action](08-extension-java-action/) | BCrypt, server extensions | Studio Pro + JDK |
| [09 JS Action](09-extension-js-action/) | Browser API, client extensions | 01 |
| [10 Widget Development](10-extension-widget-dev/) | React pluggable widget | Node.js |
| [11 Theming](11-extension-theming/) | Atlas CSS variables | None |
| [12 AI Collaboration](12-ai-collaboration/) | Prompt design, debugging patterns | All |
| [Capstone](capstone-helpdesk/) | Full app delivery | All modules |

## MDL Reference Files

MDL files are language-neutral and shared with the Chinese curriculum.
See `zh/[module]/参考实现/` for all reference implementations.

## Quick Start

```bash
mxcli new MyHelpdesk --version 11.6.6
claude  # open Claude Code in project directory
# Follow the guide in 00-getting-started/ai-collaboration-guide.md
```
```

- [ ] **Step 4：提交英文内容**

```bash
git add academy/en/
git commit -m "docs(academy): add English curriculum content (business requirements + guides)"
```

---

## Task 8：全量验证与最终提交

- [ ] **Step 1：对所有 Phase 3 参考实现 MDL 运行语法检查**

```bash
for f in \
  academy/zh/08-扩展-Java-Action/参考实现/*.mdl \
  academy/zh/09-扩展-JS-Action/参考实现/*.mdl \
  academy/zh/10-扩展-Widget开发/参考实现/*.mdl; do
  echo "Checking: $f"
  ./bin/mxcli check "$f"
done
```

Expected: 每个文件无语法错误。Module 08 的 `call-java-action.mdl` 可能报 "Java Action not found"（正常，因为 Java Action 需要在 Studio Pro 中创建后才能引用）——此为预期行为，记录在注释中即可。

- [ ] **Step 2：验证英文内容文件存在**

```bash
find academy/en -name "*.md" | sort | head -30
```

Expected: 至少 30 个 Markdown 文件被列出。

- [ ] **Step 3：最终提交**

```bash
git add -A
git commit -m "docs(academy): Phase 3 complete — extensions + AI patterns + English content"
```

---

## 自检：Spec 覆盖确认

| 设计规格要求 | 对应 Task |
|------------|----------|
| 模块 08（Java Action）内容 | Task 2 |
| 模块 09（JS Action）内容 | Task 3 |
| 模块 10（Widget 开发）内容 | Task 4 |
| 模块 11（主题定制）内容 | Task 5 |
| 模块 12（AI 协作模式）内容 | Task 6 |
| 英文版内容填充 | Task 7 |
| 扩展模块标注额外工具要求 | Task 2–5 的 AI协作指南.md |
| Widget 参考 React 源码 | Task 4（widget-source/src/TicketStatusBadge.tsx）|
| 主题 CSS 变量方案（无 Studio Pro 替代）| Task 5（helpdesk-theme.css）|
| 英文版 README 更新 | Task 7 Step 3 |
