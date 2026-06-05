# MDL 语法设计优化 + Syntax Showcase 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 MDL grammar 中为所有控制流语句添加 `{ }` 规范形式（向后兼容 `BEGIN…END`），并建立覆盖全语法的 syntax-showcase 回归测试套件（Layer 1：parse-only）。

**Architecture:** Grammar 修改采用 ANTLR4 `|` 分支，两种形式并存；showcase 文件放在 `mdl-examples/syntax-showcase/`，通过 `make test-showcase` 在 CI 中验证语法层（无需 MPR）。

**Tech Stack:** ANTLR4 (Go target)、MDL、Makefile。

---

## 变更说明（相较原始计划）

以下变更发生在计划写定后，均已反映在对应 Task 中：

| 变更 | commit | 影响 |
|------|--------|------|
| Stage 3.2.2 describe 修复 | 7c0baba9f | `split type`/`raise error`/`java action` 的 DESCRIBE 已正确输出；微流 body 的 `begin/end` 输出不变 |
| helpdesk 账号管理 | e6ef4b5c0 | 确认了 `split type + cast`、`raise error;`、`show home page;` 的真实用法 |
| notifyWorkflow ACTIVITY 子句 | 25f4b8d7d | `notify workflow $Wf activity M.WF_Name;` 新增 ACTIVITY 可选子句，showcase 需覆盖 |
| **语法错误修正** | — | `raise error` 语法是无参数 `RAISE ERROR`；带消息用 `throw expression`（throwStatement），两者独立，需各有示例 |
| Java action 命名源块 | b1b47dc50 | 命名块语法为 `imports $$ ... $$ code $$ ... $$ extra $$ ... $$` |

---

## 文件结构

| 文件 | 操作 | 职责 |
|------|------|------|
| `mdl/grammar/domains/MDLMicroflow.g4` | 修改 | 添加 `{}` 分支到 7 个规则 |
| `mdl/grammar/parser/` | 重新生成 | `make grammar` 自动更新 |
| `mdl-examples/syntax-showcase/00-setup.mdl` | 新建 | 所有 showcase 文件共用的前提 module/entity/enum |
| `mdl-examples/syntax-showcase/MANIFEST.md` | 新建 | 目录索引 |
| `mdl-examples/syntax-showcase/expr-01..19.mdl` | 新建 | Mendix 官方表达式层（19 个文件） |
| `mdl-examples/syntax-showcase/xpath-01..06.mdl` | 新建 | XPath 约束（6 个文件） |
| `mdl-examples/syntax-showcase/act-01..30.mdl` | 新建 | 活动层（30 个文件） |
| `mdl-examples/syntax-showcase/ctrl-01..13.mdl` | 新建 | 控制流（新旧两种形式，13 个文件） |
| `mdl-examples/syntax-showcase/ddl-01..10.mdl` | 新建 | DDL 层（10 个文件） |
| `Makefile` | 修改 | 添加 `test-showcase` target |
| `.claude/skills/design-mdl-syntax.md` | 修改 | 记录 `{}` 规范形式 |
| `docs/01-project/MDL_QUICK_REFERENCE.md` | 修改 | 控制流表格添加两种形式 |

---

## Task 1: Grammar — 添加 `{}` 到 microflow/nanoflow body

**Files:**
- Modify: `mdl/grammar/domains/MDLMicroflow.g4:16-33` (createMicroflowStatement, createNanoflowStatement)

- [ ] **Step 1: 替换 createMicroflowStatement**

在 `MDLMicroflow.g4` 第 16-22 行，把：
```antlr
createMicroflowStatement
    : MICROFLOW qualifiedName
      LPAREN microflowParameterList? RPAREN
      microflowReturnType?
      microflowOptions?
      BEGIN microflowBody END SEMICOLON? SLASH?
    ;
```
替换为：
```antlr
createMicroflowStatement
    : MICROFLOW qualifiedName
      LPAREN microflowParameterList? RPAREN
      microflowReturnType?
      microflowOptions?
      ( BEGIN microflowBody END
      | LBRACE microflowBody RBRACE
      ) SEMICOLON? SLASH?
    ;
```

- [ ] **Step 2: 替换 createNanoflowStatement**

第 27-33 行，把：
```antlr
createNanoflowStatement
    : NANOFLOW qualifiedName
      LPAREN microflowParameterList? RPAREN
      microflowReturnType?
      microflowOptions?
      BEGIN microflowBody END SEMICOLON? SLASH?
    ;
```
替换为：
```antlr
createNanoflowStatement
    : NANOFLOW qualifiedName
      LPAREN microflowParameterList? RPAREN
      microflowReturnType?
      microflowOptions?
      ( BEGIN microflowBody END
      | LBRACE microflowBody RBRACE
      ) SEMICOLON? SLASH?
    ;
```

- [ ] **Step 3: commit**

```bash
git add mdl/grammar/domains/MDLMicroflow.g4
git commit -m "feat(grammar): add {} body form to createMicroflowStatement and createNanoflowStatement"
```

---

## Task 2: Grammar — 添加 `{}` 到 if/loop/while/case/splitType

**Files:**
- Modify: `mdl/grammar/domains/MDLMicroflow.g4:174-198, 264-280`

- [ ] **Step 1: 替换 ifStatement（第 264-269 行附近）**

当前：
```antlr
ifStatement
    : IF expression THEN microflowBody
      (ELSIF expression THEN microflowBody)*
      (ELSE microflowBody)?
      END IF
    ;
```
替换为：
```antlr
ifStatement
    : IF expression
      ( THEN microflowBody
        (ELSIF expression THEN microflowBody)*
        (ELSE microflowBody)?
        END IF
      | LBRACE microflowBody RBRACE
        (ELSIF expression LBRACE microflowBody RBRACE)*
        (ELSE LBRACE microflowBody RBRACE)?
      )
    ;
```

- [ ] **Step 2: 替换 loopStatement（第 272-275 行附近）**

当前：
```antlr
loopStatement
    : LOOP VARIABLE IN (VARIABLE | attributePath)
      BEGIN microflowBody END LOOP
    ;
```
替换为：
```antlr
loopStatement
    : LOOP VARIABLE IN (VARIABLE | attributePath)
      ( BEGIN microflowBody END LOOP
      | LBRACE microflowBody RBRACE
      )
    ;
```

- [ ] **Step 3: 替换 whileStatement（第 277-280 行附近）**

当前：
```antlr
whileStatement
    : WHILE expression
      BEGIN? microflowBody END WHILE?
    ;
```
替换为：
```antlr
whileStatement
    : WHILE expression
      ( BEGIN? microflowBody END WHILE?
      | LBRACE microflowBody RBRACE
      )
    ;
```

- [ ] **Step 4: 替换 caseStatement（第 174-179 行附近）**

当前：
```antlr
caseStatement
    : CASE enumSplitSource
      (WHEN enumSplitCaseValue (COMMA enumSplitCaseValue)* THEN microflowBody)+
      (ELSE microflowBody)?
      END CASE
    ;
```
替换为：
```antlr
caseStatement
    : CASE enumSplitSource
      ( (WHEN enumSplitCaseValue (COMMA enumSplitCaseValue)* THEN microflowBody)+
        (ELSE microflowBody)?
        END CASE
      | LBRACE
        (WHEN enumSplitCaseValue (COMMA enumSplitCaseValue)* LBRACE microflowBody RBRACE)+
        (ELSE LBRACE microflowBody RBRACE)?
        RBRACE
      )
    ;
```

- [ ] **Step 5: 替换 inheritanceSplitStatement（第 191-198 行附近）**

当前：
```antlr
inheritanceSplitStatement
    : SPLIT TYPE VARIABLE
      (inheritanceSplitCase+ (ELSE microflowBody)? END SPLIT)?
    ;

inheritanceSplitCase
    : CASE qualifiedName microflowBody
    ;
```
替换为：
```antlr
inheritanceSplitStatement
    : SPLIT TYPE VARIABLE
      ( inheritanceSplitCase+ (ELSE microflowBody)? END SPLIT
      | LBRACE inheritanceSplitCaseModern* (ELSE LBRACE microflowBody RBRACE)? RBRACE
      )?
    ;

inheritanceSplitCase
    : CASE qualifiedName microflowBody
    ;

inheritanceSplitCaseModern
    : CASE qualifiedName LBRACE microflowBody RBRACE
    ;
```

- [ ] **Step 6: commit**

```bash
git add mdl/grammar/domains/MDLMicroflow.g4
git commit -m "feat(grammar): add {} form to if/loop/while/case/split-type statements"
```

---

## Task 3: Grammar — declare 加冒号 + simpleAssignStatement

**Files:**
- Modify: `mdl/grammar/domains/MDLMicroflow.g4:170-172, 111-168`

- [ ] **Step 1: 修改 declareStatement（第 170-172 行附近）**

当前：
```antlr
declareStatement
    : DECLARE VARIABLE dataType (EQUALS expression)?
    ;
```
替换为：
```antlr
declareStatement
    : DECLARE VARIABLE COLON? dataType (EQUALS expression)?
    ;
```

- [ ] **Step 2: 在 `MDLMicroflow.g4` 中添加 simpleAssignStatement 规则**

在 `setStatement`（第 205-207 行附近）之后添加新规则：
```antlr
simpleAssignStatement
    : (VARIABLE | attributePath) EQUALS expression
    ;
```

- [ ] **Step 3: 在 microflowStatement 末尾添加 simpleAssignStatement**

在第 167 行（`annotation* applyJumpToStatement SEMICOLON?`）之后、`;` 之前，添加：
```antlr
    | annotation* simpleAssignStatement SEMICOLON?
```

注意：必须放在所有其他 alternative 之后，避免与 `createObjectStatement`（`(VARIABLE EQUALS)? CREATE ...`）等已有规则产生歧义。

- [ ] **Step 4: commit**

```bash
git add mdl/grammar/domains/MDLMicroflow.g4
git commit -m "feat(grammar): add COLON? to declareStatement, add simpleAssignStatement"
```

---

## Task 4: 重新生成 Parser 并验证

**Files:**
- Auto-generated: `mdl/grammar/parser/` (由 make grammar 更新)

- [ ] **Step 1: 重新生成 parser**

```bash
make grammar
```

期望输出末尾：`Generated Go parser in mdl/grammar/parser/`（无报错，无 ANTLR warning）

- [ ] **Step 2: 验证构建成功**

```bash
make build
```

期望：`go build` 无错误，`bin/mxcli` 可执行。

- [ ] **Step 3: 快速冒烟测试——旧形式仍工作**

```bash
echo "create microflow Test.M () returns Nothing begin return; end;" | ./bin/mxcli check /dev/stdin
```

期望：输出无错误（exit code 0）。

- [ ] **Step 4: 快速冒烟测试——新形式可解析**

```bash
echo "create microflow Test.M () returns Nothing { return; }" | ./bin/mxcli check /dev/stdin
```

期望：输出无错误（exit code 0）。

如果有错误，检查 grammar 文件中的括号是否匹配，并重新运行 `make grammar`。

- [ ] **Step 5: commit parser 重新生成**

```bash
git add mdl/grammar/parser/
git commit -m "build(grammar): regenerate ANTLR parser after {} form additions"
```

---

## Task 5: 创建 00-setup.mdl 和 MANIFEST.md

**Files:**
- Create: `mdl-examples/syntax-showcase/00-setup.mdl`
- Create: `mdl-examples/syntax-showcase/MANIFEST.md`

- [ ] **Step 1: 创建目录**

```bash
mkdir -p mdl-examples/syntax-showcase
```

- [ ] **Step 2: 创建 00-setup.mdl**

此文件被所有其他 showcase 文件依赖（parse-only 模式下不需要真正 execute，但作为文档约定 module/entity/enum 名称）。

创建 `mdl-examples/syntax-showcase/00-setup.mdl`：
```mdl
-- ============================================================
-- SHOWCASE SETUP: Module, Entities, Enumerations
-- MDL grammar rule: createModule, createEnumeration, createEntity
-- Purpose: Defines Showcase.* types referenced by all other showcase files.
--          Run this first if executing against a real MPR.
-- ============================================================

create module Showcase;

-- [SETUP-ENUM-01] Order status
create enumeration Showcase.OrderStatus (
  Pending 'Pending',
  Active  'Active',
  Closed  'Closed'
);
/

-- [SETUP-ENUM-02] Priority
create enumeration Showcase.Priority (
  High   'High',
  Medium 'Medium',
  Low    'Low'
);
/

-- [SETUP-ENT-01] Order (persistent)
@position(100,100)
create persistent entity Showcase.Order (
  OrderNumber: string(50) not null,
  TotalAmount: decimal,
  Weight:      decimal,
  Status:      enumeration(Showcase.OrderStatus),
  Priority:    enumeration(Showcase.Priority),
  OrderDate:   datetime,
  IsActive:    boolean default true,
  Quantity:    integer
);
/

-- [SETUP-ENT-02] Product (persistent)
@position(300,100)
create persistent entity Showcase.Product (
  Name:     string(200) not null,
  Code:     string(50),
  IsActive: boolean default true,
  Price:    decimal
);
/

-- [SETUP-ENT-03] Customer (persistent)
@position(500,100)
create persistent entity Showcase.Customer (
  Name:  string(200) not null,
  Email: string(200)
);
/

-- [SETUP-ENT-04] Animal base (for split type showcase)
@position(100,300)
create persistent entity Showcase.Animal (
  Name: string(200)
);
/

-- [SETUP-ENT-05] Dog (extends Animal)
@position(300,300)
create persistent entity Showcase.Dog extends Showcase.Animal (
  Breed: string(200)
);
/

-- [SETUP-ENT-06] Cat (extends Animal)
@position(500,300)
create persistent entity Showcase.Cat extends Showcase.Animal (
  Indoor: boolean default false
);
/

-- [SETUP-ASSOC-01] Order → Product
create association Showcase.Order_Product
  from Showcase.Order to Showcase.Product
  type reference;
/
```

- [ ] **Step 3: 創建 MANIFEST.md**

创建 `mdl-examples/syntax-showcase/MANIFEST.md`：
```markdown
# MDL Syntax Showcase

Grammar coverage test suite. Each file covers one or more MDL grammar production rules.

## Running

```bash
# Layer 1: syntax check (no MPR needed)
make test-showcase

# Or manually:
find mdl-examples/syntax-showcase -name "*.mdl" | sort | \
    xargs -I{} sh -c './bin/mxcli check {} && echo "OK: {}" || exit 1'
```

## File Groups

| Prefix | Count | Content |
|--------|-------|---------|
| `00-setup` | 1 | Shared module/entity/enum definitions |
| `expr-01..19` | 19 | Mendix expression layer (arithmetic, functions, date, etc.) |
| `xpath-01..06` | 6 | XPath constraint syntax |
| `act-01..30` | 30 | Microflow activities |
| `ctrl-01..13` | 13 | Control flow (legacy + modern {} forms) |
| `ddl-01..10` | 10 | DDL layer (entity, page, security, etc.) |
```

- [ ] **Step 4: commit**

```bash
git add mdl-examples/syntax-showcase/
git commit -m "feat(showcase): add 00-setup.mdl and MANIFEST.md"
```

---

## Task 6: 创建 expr-01..10 (表达式层 Part 1)

**Files:**
- Create: `mdl-examples/syntax-showcase/expr-01-arithmetic.mdl` 到 `expr-10-date-subtract.mdl`

每个文件的写法范式（以 expr-01 为例）：

```mdl
-- ============================================================
-- SHOWCASE: Arithmetic Operators
-- Mendix doc: /refguide/arithmetic-expressions/
-- MDL grammar rule: expression (addExpression, mulExpression)
-- ============================================================

create module Showcase;

-- [EXPR-ARITH-01] 整数算术
create microflow Showcase.Expr_Arithmetic_01 ($a: integer, $b: integer)
returns integer as $r
begin
  declare $r: integer = $a + $b * 2 - 1;
  return $r;
end;
/

-- [EXPR-ARITH-02] Decimal div/mod
create microflow Showcase.Expr_Arithmetic_02 ($x: decimal, $y: decimal)
returns decimal as $r
begin
  declare $r: decimal = $x div $y + $x mod $y;
  return $r;
end;
/
```

- [ ] **Step 1: 创建 expr-01-arithmetic.mdl**

参考范式，覆盖 `+`, `-`, `*`, `div`, `mod`（整数和 decimal 各一例）。

- [ ] **Step 2: 创建 expr-02-relational.mdl**

覆盖 `=`, `!=`, `<`, `<=`, `>`, `>=`，在 microflow return 表达式中使用。

- [ ] **Step 3: 创建 expr-03-boolean.mdl**

覆盖 `and`, `or`, `not()`，组合表达式。

- [ ] **Step 4: 创建 expr-04-if-expression.mdl**

**关键**：这是 Mendix 官方 `if cond then val else other` 表达式（不是控制流）。示例：
```mdl
create microflow Showcase.Expr_IfExpr_01 ($weight: decimal)
returns decimal as $fee
begin
  declare $fee: decimal = if $weight < 1.00 then 0.00 else 5.00;
  return $fee;
end;
/
```

- [ ] **Step 5: 创建 expr-05-special-checks.mdl**

覆盖 `empty`, `isNew()`, `isSynced()`。

- [ ] **Step 6: 创建 expr-06-string-functions.mdl**

覆盖 `contains()`, `startsWith()`, `endsWith()`, `toLowerCase()`, `toUpperCase()`, `trim()`, `length()`, `substring()`, `find()`, `replaceAll()`, `urlEncode()`, `urlDecode()`, `htmlEncode()`, `formatString()`（各一例）。

- [ ] **Step 7: 创建 expr-07-math-functions.mdl**

覆盖 `abs()`, `round()`, `roundDown()`, `roundUp()`, `floor()`, `ceiling()`, `sqrt()`, `pow()`, `max(a,b)`, `min(a,b)`（各一例）。

- [ ] **Step 8: 创建 expr-08-date-create.mdl**

覆盖 `dateTime(year,month,day,hour,min,sec)`, `dateTimeUTC(...)`。

- [ ] **Step 9: 创建 expr-09-date-add.mdl**

覆盖 `addDays()`, `addWeeks()`, `addMonths()`, `addYears()`, `addHours()`, `addMinutes()`, `addSeconds()`, `addMilliseconds()`, `addQuarters()`, `addMilliseconds()`（各一例）。

- [ ] **Step 10: 创建 expr-10-date-subtract.mdl**

覆盖 `subtractDays()`, `subtractWeeks()`, `subtractMonths()`, `subtractYears()`, `subtractHours()`, `subtractMinutes()`, `subtractSeconds()`（各一例）。

- [ ] **Step 11: 验证 10 个文件都能 parse**

```bash
find mdl-examples/syntax-showcase -name "expr-0[1-9]-*.mdl" -o -name "expr-10-*.mdl" | sort | \
    xargs -I{} sh -c './bin/mxcli check {} && echo "OK: {}" || { echo "FAIL: {}"; exit 1; }'
```

期望：全部 `OK`。

- [ ] **Step 12: commit**

```bash
git add mdl-examples/syntax-showcase/expr-0{1..9}-*.mdl mdl-examples/syntax-showcase/expr-10-*.mdl
git commit -m "feat(showcase): add expr-01..10 expression layer files"
```

---

## Task 7: 创建 expr-11..19 (表达式层 Part 2)

**Files:**
- Create: `mdl-examples/syntax-showcase/expr-11-date-between.mdl` 到 `expr-19-enumerations.mdl`

- [ ] **Step 1: 创建 expr-11-date-between.mdl**

覆盖 `daysBetween()`, `weeksBetween()`, `monthsBetween()`, `yearsBetween()`, `hoursBetween()`, `minutesBetween()`, `secondsBetween()`, `millisecondsBetween()`。

- [ ] **Step 2: 创建 expr-12-date-trim.mdl**

覆盖 `trimToSeconds()`, `trimToMinutes()`, `trimToHours()`, `trimToDays()`, `trimToWeeks()`, `trimToMonths()`, `trimToYears()`, `trimToQuarters()`。

- [ ] **Step 3: 创建 expr-13-date-begin-end.mdl**

覆盖 `beginOfDay()`, `beginOfWeek()`, `beginOfMonth()`, `beginOfYear()`, `endOfDay()`, `endOfWeek()`, `endOfMonth()`, `endOfYear()`。

- [ ] **Step 4: 创建 expr-14-date-format.mdl**

覆盖 `formatDateTime()`, `parseDateTime()`, `formatDateTimeUTC()`, `parseDateTimeUTC()`，含格式字符串参数。

- [ ] **Step 5: 创建 expr-15-decimal-format.mdl**

覆盖 `formatDecimal()`, `parseDecimal()`, `formatDecimalWithCurrency()`。

- [ ] **Step 6: 创建 expr-16-parse-integer.mdl**

覆盖 `parseInt()`, `parseBoolean()`, `toString()`（各种类型转字符串）。

- [ ] **Step 7: 创建 expr-17-length.mdl**

覆盖 `length(list)`, `length(string)`（MDL 支持重载），及 `$list = empty` 判空。

- [ ] **Step 8: 创建 expr-18-system-variables.mdl**

覆盖 `[%CurrentDateTime%]`, `[%CurrentUser%]`, `[%CurrentObject%]`, `[%CurrentSession%]`, `[%ProjectName%]`（各一例）。

- [ ] **Step 9: 创建 expr-19-enumerations.mdl**

覆盖枚举值比较（`$order/Status = Showcase.OrderStatus.Active`）和 `getCaption($order/Status)`。

- [ ] **Step 10: 验证并 commit**

```bash
find mdl-examples/syntax-showcase -name "expr-1[1-9]-*.mdl" | sort | \
    xargs -I{} sh -c './bin/mxcli check {} && echo "OK: {}" || { echo "FAIL: {}"; exit 1; }'
git add mdl-examples/syntax-showcase/expr-1[1-9]-*.mdl
git commit -m "feat(showcase): add expr-11..19 expression layer files"
```

---

## Task 8: 创建 xpath-01..06 (XPath 约束层)

**Files:**
- Create: `mdl-examples/syntax-showcase/xpath-01-tokens.mdl` 到 `xpath-06-system-variables.mdl`

每个文件的 xpath 示例嵌入 `retrieve from database` 活动的 `where` 子句中：

```mdl
-- ============================================================
-- SHOWCASE: XPath Tokens
-- Mendix doc: /refguide/xpath/
-- MDL grammar rule: xpathConstraint
-- ============================================================
create module Showcase;

-- [XPATH-01] 基本路径 // . / [] ()
create microflow Showcase.XPath_Tokens_01 ()
returns list of Showcase.Order as $result
begin
  declare $result: list of Showcase.Order = empty;
  $result = retrieve Showcase.Order from database
    where $currentObject/Status = Showcase.OrderStatus.Active;
  return $result;
end;
/
```

- [ ] **Step 1: 创建 xpath-01-tokens.mdl** — `//`, `.`, `/`, `[]`, `()`
- [ ] **Step 2: 创建 xpath-02-operators.mdl** — `=`, `!=`, `<`, `>`, `and`, `or` 在 XPath 中
- [ ] **Step 3: 创建 xpath-03-functions.mdl** — `contains()`, `starts-with()`, `not()`, `string-length()`
- [ ] **Step 4: 创建 xpath-04-associations.mdl** — 跨关联路径约束（`$currentObject/Order_Customer/Email = 'x'`）
- [ ] **Step 5: 创建 xpath-05-nested.mdl** — 嵌套子约束
- [ ] **Step 6: 创建 xpath-06-system-variables.mdl** — `[%CurrentUser%]`, `[%CurrentDateTime%]` 在 XPath 中

- [ ] **Step 7: 验证并 commit**

```bash
find mdl-examples/syntax-showcase -name "xpath-*.mdl" | sort | \
    xargs -I{} sh -c './bin/mxcli check {} && echo "OK: {}" || { echo "FAIL: {}"; exit 1; }'
git add mdl-examples/syntax-showcase/xpath-*.mdl
git commit -m "feat(showcase): add xpath-01..06 XPath constraint files"
```

---

## Task 9: 创建 act-01..15 (活动层 Part 1)

**Files:**
- Create: `mdl-examples/syntax-showcase/act-01-object-create.mdl` 到 `act-15-call-microflow.mdl`

范式（以 act-01 为例）：
```mdl
-- ============================================================
-- SHOWCASE: Create Object Activity
-- Mendix doc: /refguide/create-object/
-- MDL grammar rule: createObjectStatement
-- ============================================================

create module Showcase;

-- [ACT-CREATE-01] 最小创建
create microflow Showcase.Act_Create_01 ()
returns Showcase.Order as $r
begin
  declare $r: Showcase.Order = empty;
  $r = create Showcase.Order (
    OrderNumber = '001',
    Status = Showcase.OrderStatus.Pending
  );
  return $r;
end;
/

-- [ACT-CREATE-02] 带 on error continue
create microflow Showcase.Act_Create_02 ()
returns Nothing
begin
  create Showcase.Order (OrderNumber = '002') on error continue;
end;
/
```

- [ ] **Step 1: 创建 act-01-object-create.mdl** — `create object`，至少包含：最小创建、全属性类型、on error 变体
- [ ] **Step 2: 创建 act-02-object-change.mdl** — `change object`，包含 REFRESH 变体
- [ ] **Step 3: 创建 act-03-object-delete.mdl** — `delete object`，带 on error continue/rollback
- [ ] **Step 4: 创建 act-04-object-commit.mdl** — `commit`，WITH EVENTS、REFRESH 变体
- [ ] **Step 5: 创建 act-05-object-rollback.mdl** — `rollback`
- [ ] **Step 6: 创建 act-06-object-retrieve-db.mdl** — `retrieve from database`（where/sort by/limit/offset 组合）
- [ ] **Step 7: 创建 act-07-object-retrieve-assoc.mdl** — `retrieve from association`（`$order/Order_Customer`）
- [ ] **Step 8: 创建 act-08-object-cast.mdl** — `cast $var` 和 `$x = cast $var`
- [ ] **Step 9: 创建 act-09-list-create.mdl** — `declare $list: list of Type = empty`
- [ ] **Step 10: 创建 act-10-list-change.mdl** — `add to list`, `remove from list`
- [ ] **Step 11: 创建 act-11-list-aggregate.mdl** — `count()`, `sum()`, `avg()`, `min()`, `max()`（在 `aggregate list` 活动中）
- [ ] **Step 12: 创建 act-12-list-operation.mdl** — `filter list`, `sort list`, `find in list`, `union`, `intersect`, `subtract`
- [ ] **Step 13: 创建 act-13-variable-declare.mdl** — 全数据类型的 declare（integer/decimal/string/boolean/datetime/list/object），新旧两种形式

  新形式（规范）：
  ```mdl
  declare $count: integer = 0;
  declare $name: string;
  declare $products: list of Showcase.Product;
  declare $result: Showcase.Order = empty;
  ```
  旧形式（向后兼容，需在同一文件中验证解析）：
  ```mdl
  declare $count integer = 0;
  ```

- [ ] **Step 14: 创建 act-14-variable-assign.mdl** — 新赋值形式 `$x = expr` 和旧形式 `set $x = expr`

  ```mdl
  create microflow Showcase.Act_Assign_01 ()
  returns integer as $r
  begin
    declare $r: integer = 0;
    $r = $r + 1;          -- 新规范形式
    set $r = $r + 1;      -- 向后兼容
    return $r;
  end;
  /
  ```

- [ ] **Step 15: 创建 act-15-call-microflow.mdl** — 参见 spec 中的 [MF-CALL-01..04] 范例（已在 spec 文档中写好）

- [ ] **Step 16: 验证并 commit**

```bash
find mdl-examples/syntax-showcase -name "act-0[1-9]-*.mdl" -o -name "act-1[0-5]-*.mdl" | sort | \
    xargs -I{} sh -c './bin/mxcli check {} && echo "OK: {}" || { echo "FAIL: {}"; exit 1; }'
git add mdl-examples/syntax-showcase/act-0{1..9}-*.mdl mdl-examples/syntax-showcase/act-1[0-5]-*.mdl
git commit -m "feat(showcase): add act-01..15 activity showcase files"
```

---

## Task 10: 创建 act-16..30 (活动层 Part 2)

**Files:**
- Create: `mdl-examples/syntax-showcase/act-16-call-nanoflow.mdl` 到 `act-30-import-export-mapping.mdl`

- [ ] **Step 1: 创建 act-16-call-nanoflow.mdl** — `call nanoflow M.NF(Param = $val)`
- [ ] **Step 2: 创建 act-17-call-java-action.mdl** — java action 创建（含命名源块）+ call java action

  Java action source block 语法（已验证，`b1b47dc50`）：
  ```mdl
  -- ============================================================
  -- SHOWCASE: Java Action — Create + Call
  -- Mendix doc: /refguide/java-actions/
  -- MDL grammar rule: createJavaActionStatement, callJavaActionStatement
  -- ============================================================

  create module Showcase;

  -- [ACT-JAVA-01] 纯接口声明（无源块，运行时从 .java 文件加载）
  create or modify java action Showcase.JA_Hash (
    PlainText: string
  ) returns string;

  -- [ACT-JAVA-02] 含命名源块（imports/code/extra，各用 $$ ... $$ 包裹）
  create or modify java action Showcase.JA_Trim (
    Input: string
  ) returns string
  imports $$
  import java.util.Objects;
  $$
  code $$
  return Objects.toString(Input, "").trim();
  $$
  extra $$
  private static boolean isEmpty(String s) { return s == null || s.isEmpty(); }
  $$
  ;

  -- [ACT-JAVA-03] call java action（有返回值）
  create microflow Showcase.Act_JavaCall_01 ($text: string)
  returns string as $result
  begin
    declare $result: string = empty;
    $result = call java action Showcase.JA_Trim (Input = $text);
    return $result;
  end;
  /

  -- [ACT-JAVA-04] call java action（无返回值 + on error continue）
  create microflow Showcase.Act_JavaCall_02 ($text: string)
  returns Nothing
  begin
    call java action Showcase.JA_Trim (Input = $text) on error continue;
  end;
  /
  ```
- [ ] **Step 3: 创建 act-18-call-javascript-action.mdl** — `call javascript action`
- [ ] **Step 4: 创建 act-19-call-workflow.mdl** — `call workflow`, `notify workflow`（含 ACTIVITY 子句）

  `notify workflow` 语法（commit 25f4b8d7d 扩展，ACTIVITY 子句可选）：
  ```mdl
  -- [ACT-WF-01] call workflow
  create microflow Showcase.Act_Wf_Call_01 ($ctx: Showcase.Order)
  returns Nothing
  begin
    call workflow Showcase.MyWorkflow ($ctx = $ctx);
  end;
  /

  -- [ACT-WF-02] notify workflow（基本形式）
  create microflow Showcase.Act_Wf_Notify_01 ($wf: System.Workflow)
  returns Nothing
  begin
    notify workflow $wf;
  end;
  /

  -- [ACT-WF-03] notify workflow + ACTIVITY 子句（指定 workflow 活动类型）
  create microflow Showcase.Act_Wf_Notify_02 ($wf: System.Workflow)
  returns Nothing
  begin
    notify workflow $wf activity Showcase.MyWorkflow;
  end;
  /
  ```
- [ ] **Step 5: 创建 act-20-workflow-jump-to.mdl** — `generate jump to options`, `apply jump to option`
- [ ] **Step 6: 创建 act-21-client-show-page.mdl** — `show page`（带参数/不带参数）
- [ ] **Step 7: 创建 act-22-client-close-page.mdl** — `close page`
- [ ] **Step 8: 创建 act-23-client-show-message.mdl** — `show message`（info/warning/error/confirm）
- [ ] **Step 9: 创建 act-24-client-validation.mdl** — `validation feedback`
- [ ] **Step 10: 创建 act-25-client-download-file.mdl** — `download file`
- [ ] **Step 11: 创建 act-26-log-message.mdl** — `log` 六个级别 + template 参数

  ```mdl
  create microflow Showcase.Act_Log_01 () returns Nothing
  begin
    log trace node 'Showcase' 'trace message';
    log debug node 'Showcase' 'debug message';
    log info  node 'Showcase' 'info message';
    log warning node 'Showcase' 'warning message';
    log error node 'Showcase' 'error message';
    log critical node 'Showcase' 'critical message';
  end;
  /
  ```

- [ ] **Step 12: 创建 act-27-raise-error.mdl** — `raise error` 和 `throw` 语句

  **重要**：grammar 中有两个独立语句：
  - `raiseErrorStatement: RAISE ERROR` — 无参数，终止到 Mendix ErrorEvent（对应 Studio Pro "Raise Error" 活动）
  - `throwStatement: THROW expression` — 带表达式（字符串消息），不同于 raise error

  ```mdl
  -- ============================================================
  -- SHOWCASE: raise error & throw statements
  -- MDL grammar rule: raiseErrorStatement, throwStatement
  -- ============================================================

  create module Showcase;

  -- [ACT-ERR-01] raise error（终止到 ErrorEvent，无消息参数）
  create microflow Showcase.Act_RaiseError_01 ($ok: boolean)
  returns Nothing
  begin
    if not($ok) then
      raise error;
    end if;
  end;
  /

  -- [ACT-ERR-02] throw（带字符串表达式，不同于 raise error）
  create microflow Showcase.Act_Throw_01 ($msg: string)
  returns Nothing
  begin
    throw $msg;
  end;
  /
  ```
- [ ] **Step 13: 创建 act-28-rest-call.mdl** — `call rest service`（基本 GET 变体）
- [ ] **Step 14: 创建 act-29-web-service.mdl** — `call web service`
- [ ] **Step 15: 创建 act-30-import-export-mapping.mdl** — `import with mapping`, `export with mapping`

- [ ] **Step 16: 验证并 commit**

```bash
find mdl-examples/syntax-showcase -name "act-1[6-9]-*.mdl" -o -name "act-2[0-9]-*.mdl" -o -name "act-30-*.mdl" | sort | \
    xargs -I{} sh -c './bin/mxcli check {} && echo "OK: {}" || { echo "FAIL: {}"; exit 1; }'
git add mdl-examples/syntax-showcase/act-1[6-9]-*.mdl mdl-examples/syntax-showcase/act-2[0-9]-*.mdl mdl-examples/syntax-showcase/act-30-*.mdl
git commit -m "feat(showcase): add act-16..30 activity showcase files"
```

---

## Task 11: 创建控制流 showcase (ctrl-01..13)

**这是本次实施的核心**——验证新旧两种形式均可解析。

**Files:**
- Create: `mdl-examples/syntax-showcase/ctrl-01-if-legacy.mdl` 到 `ctrl-13-on-error.mdl`

- [ ] **Step 1: 创建 ctrl-01-if-legacy.mdl** （向后兼容形式）

```mdl
-- ============================================================
-- SHOWCASE: if Statement — Legacy Form (BEGIN/THEN/END IF)
-- MDL grammar rule: ifStatement (THEN form)
-- ============================================================

create module Showcase;

-- [CTRL-IF-LEGACY-01] 基本 if/end if
create microflow Showcase.Ctrl_If_Legacy_01 ($status: Showcase.OrderStatus)
returns Nothing
begin
  if $status = Showcase.OrderStatus.Pending then
    log info node 'Showcase' 'Pending';
  end if;
end;
/

-- [CTRL-IF-LEGACY-02] if/else/end if
create microflow Showcase.Ctrl_If_Legacy_02 ($status: Showcase.OrderStatus)
returns Nothing
begin
  if $status = Showcase.OrderStatus.Active then
    log info node 'Showcase' 'Active';
  else
    log info node 'Showcase' 'Not active';
  end if;
end;
/

-- [CTRL-IF-LEGACY-03] if/elsif/else/end if
create microflow Showcase.Ctrl_If_Legacy_03 ($status: Showcase.OrderStatus)
returns Nothing
begin
  if $status = Showcase.OrderStatus.Pending then
    log info node 'Showcase' 'Pending';
  elsif $status = Showcase.OrderStatus.Active then
    log info node 'Showcase' 'Active';
  else
    log info node 'Showcase' 'Closed';
  end if;
end;
/
```

- [ ] **Step 2: 创建 ctrl-02-if-modern.mdl** （新规范形式）

```mdl
-- ============================================================
-- SHOWCASE: if Statement — Modern Form ({})
-- MDL grammar rule: ifStatement (LBRACE form)
-- ============================================================

create module Showcase;

-- [CTRL-IF-MODERN-01] 基本 if {}
create microflow Showcase.Ctrl_If_Modern_01 ($status: Showcase.OrderStatus)
returns Nothing
begin
  if $status = Showcase.OrderStatus.Pending {
    log info node 'Showcase' 'Pending';
  }
end;
/

-- [CTRL-IF-MODERN-02] if {} else {}
create microflow Showcase.Ctrl_If_Modern_02 ($status: Showcase.OrderStatus)
returns Nothing
begin
  if $status = Showcase.OrderStatus.Active {
    log info node 'Showcase' 'Active';
  } else {
    log info node 'Showcase' 'Not active';
  }
end;
/

-- [CTRL-IF-MODERN-03] if {} elsif {} else {}
create microflow Showcase.Ctrl_If_Modern_03 ($status: Showcase.OrderStatus)
returns Nothing
begin
  if $status = Showcase.OrderStatus.Pending {
    log info node 'Showcase' 'Pending';
  } elsif $status = Showcase.OrderStatus.Active {
    log info node 'Showcase' 'Active';
  } else {
    log info node 'Showcase' 'Closed';
  }
end;
/
```

- [ ] **Step 3: 创建 ctrl-03-if-nested.mdl** — 嵌套 if（新形式），多个 elsif

```mdl
-- ============================================================
-- SHOWCASE: Nested if Statements — Modern Form
-- MDL grammar rule: ifStatement nested
-- ============================================================

create module Showcase;

-- [CTRL-IF-NESTED-01] 嵌套 if（外层新形式，内层新形式）
create microflow Showcase.Ctrl_If_Nested_01 ($amount: decimal, $priority: Showcase.Priority)
returns string as $r
begin
  declare $r: string = 'none';
  if $amount > 100.00 {
    if $priority = Showcase.Priority.High {
      $r = 'high-large';
    } elsif $priority = Showcase.Priority.Medium {
      $r = 'medium-large';
    } else {
      $r = 'low-large';
    }
  } else {
    $r = 'small';
  }
  return $r;
end;
/
```

- [ ] **Step 4: 创建 ctrl-04-loop-legacy.mdl** — `loop $x in $list begin … end loop`

```mdl
-- ============================================================
-- SHOWCASE: loop Statement — Legacy Form
-- MDL grammar rule: loopStatement (BEGIN form)
-- ============================================================

create module Showcase;

-- [CTRL-LOOP-LEGACY-01]
create microflow Showcase.Ctrl_Loop_Legacy_01 ($products: list of Showcase.Product)
returns Nothing
begin
  loop $product in $products
  begin
    change $product (IsActive = true);
  end loop;
end;
/
```

- [ ] **Step 5: 创建 ctrl-05-loop-modern.mdl** — `loop $x in $list { }`

```mdl
-- ============================================================
-- SHOWCASE: loop Statement — Modern Form ({})
-- MDL grammar rule: loopStatement (LBRACE form)
-- ============================================================

create module Showcase;

-- [CTRL-LOOP-MODERN-01]
create microflow Showcase.Ctrl_Loop_Modern_01 ($products: list of Showcase.Product)
returns Nothing
begin
  loop $product in $products {
    change $product (IsActive = true);
  }
end;
/

-- [CTRL-LOOP-MODERN-02] 带 $currentIndex
create microflow Showcase.Ctrl_Loop_Modern_02 ($products: list of Showcase.Product)
returns Nothing
begin
  loop $product in $products {
    if $currentIndex = 0 {
      log info node 'Showcase' 'First item';
    }
  }
end;
/
```

- [ ] **Step 6: 创建 ctrl-06-while-legacy.mdl** — `while expr begin … end while`

```mdl
-- ============================================================
-- SHOWCASE: while Statement — Legacy Form
-- MDL grammar rule: whileStatement (BEGIN form)
-- ============================================================

create module Showcase;

-- [CTRL-WHILE-LEGACY-01]
create microflow Showcase.Ctrl_While_Legacy_01 ()
returns integer as $r
begin
  declare $r: integer = 0;
  while $r < 10
  begin
    $r = $r + 1;
  end while;
  return $r;
end;
/
```

- [ ] **Step 7: 创建 ctrl-07-while-modern.mdl** — `while expr { }`

```mdl
-- ============================================================
-- SHOWCASE: while Statement — Modern Form ({})
-- MDL grammar rule: whileStatement (LBRACE form)
-- ============================================================

create module Showcase;

-- [CTRL-WHILE-MODERN-01]
create microflow Showcase.Ctrl_While_Modern_01 ()
returns integer as $r
begin
  declare $r: integer = 0;
  while $r < 10 {
    $r = $r + 1;
  }
  return $r;
end;
/
```

- [ ] **Step 8: 创建 ctrl-08-break-continue.mdl** — `break`, `continue`, `$currentIndex` 使用

```mdl
-- ============================================================
-- SHOWCASE: break, continue, $currentIndex
-- MDL grammar rule: breakStatement, continueStatement
-- ============================================================

create module Showcase;

-- [CTRL-BREAK-01] break in loop
create microflow Showcase.Ctrl_Break_01 ($products: list of Showcase.Product)
returns Nothing
begin
  loop $product in $products {
    if $currentIndex = 5 {
      break;
    }
    change $product (IsActive = true);
  }
end;
/

-- [CTRL-CONTINUE-01] continue in loop
create microflow Showcase.Ctrl_Continue_01 ($products: list of Showcase.Product)
returns Nothing
begin
  loop $product in $products {
    if not($product/IsActive) {
      continue;
    }
    change $product (Name = $product/Name + '_ok');
  }
end;
/
```

- [ ] **Step 9: 创建 ctrl-09-case-legacy.mdl** — `case … when … then … end case`

```mdl
-- ============================================================
-- SHOWCASE: case Statement — Legacy Form
-- MDL grammar rule: caseStatement (THEN/END CASE form)
-- ============================================================

create module Showcase;

-- [CTRL-CASE-LEGACY-01]
create microflow Showcase.Ctrl_Case_Legacy_01 ($priority: Showcase.Priority)
returns Nothing
begin
  case $priority
  when High then
    log warning node 'Showcase' 'High priority';
  when Low, Medium then
    log info node 'Showcase' 'Standard';
  else
    log debug node 'Showcase' 'Unknown';
  end case;
end;
/
```

- [ ] **Step 10: 创建 ctrl-10-case-modern.mdl** — `case … { when A { } else { } }`

```mdl
-- ============================================================
-- SHOWCASE: case Statement — Modern Form ({})
-- MDL grammar rule: caseStatement (LBRACE form)
-- ============================================================

create module Showcase;

-- [CTRL-CASE-MODERN-01]
create microflow Showcase.Ctrl_Case_Modern_01 ($priority: Showcase.Priority)
returns Nothing
begin
  case $priority {
    when High {
      log warning node 'Showcase' 'High priority';
    }
    when Low, Medium {
      log info node 'Showcase' 'Standard';
    }
    else {
      log debug node 'Showcase' 'Unknown';
    }
  }
end;
/
```

- [ ] **Step 11: 创建 ctrl-11-split-type-legacy.mdl**

  `split type` 的典型用法是与 `cast` 配合，将基类变量绑定到子类变量（如 helpdesk `DS_GetMyProfile`）。
  showcase 文件需同时覆盖：简单 change 形式 + 真实 cast 形式。

```mdl
-- ============================================================
-- SHOWCASE: split type Statement — Legacy Form
-- MDL grammar rule: inheritanceSplitStatement (END SPLIT form),
--                   castObjectStatement
-- Note: cast $Var is the typical use inside split type branches
--       (binds base-type variable to a sub-type variable).
-- ============================================================

create module Showcase;

-- [CTRL-SPLIT-LEGACY-01] 简单分支（change，不含 cast）
create microflow Showcase.Ctrl_Split_Legacy_01 ($animal: Showcase.Animal)
returns Nothing
begin
  split type $animal
  case Showcase.Dog
    change $animal (Name = 'dog');
  case Showcase.Cat
    change $animal (Name = 'cat');
  else
    log info node 'Showcase' 'Unknown animal';
  end split;
end;
/

-- [CTRL-SPLIT-LEGACY-02] 含 cast（典型模式：子类绑定后返回）
-- Grammar rule: castObjectStatement — CAST VARIABLE | VARIABLE EQUALS CAST VARIABLE
create microflow Showcase.Ctrl_Split_Legacy_02 ($animal: Showcase.Animal)
returns Showcase.Dog as $dog
begin
  declare $dog: Showcase.Dog = empty;
  split type $animal
  case Showcase.Dog
    cast $dog;
    return $dog;
  else
    return empty;
  end split;
end;
/
```

- [ ] **Step 12: 创建 ctrl-12-split-type-modern.mdl**

```mdl
-- ============================================================
-- SHOWCASE: split type Statement — Modern Form ({})
-- MDL grammar rule: inheritanceSplitStatement (LBRACE form),
--                   castObjectStatement
-- ============================================================

create module Showcase;

-- [CTRL-SPLIT-MODERN-01] 简单分支（{}，不含 cast）
create microflow Showcase.Ctrl_Split_Modern_01 ($animal: Showcase.Animal)
returns Nothing
begin
  split type $animal {
    case Showcase.Dog {
      change $animal (Name = 'dog');
    }
    case Showcase.Cat {
      change $animal (Name = 'cat');
    }
    else {
      log info node 'Showcase' 'Unknown animal';
    }
  }
end;
/

-- [CTRL-SPLIT-MODERN-02] 含 cast（典型模式：子类绑定后返回）
create microflow Showcase.Ctrl_Split_Modern_02 ($animal: Showcase.Animal)
returns Showcase.Dog as $dog
begin
  declare $dog: Showcase.Dog = empty;
  split type $animal {
    case Showcase.Dog {
      cast $dog;
      return $dog;
    }
    else {
      return empty;
    }
  }
end;
/
```

- [ ] **Step 13: 创建 ctrl-13-on-error.mdl** — `on error continue / rollback / { body }` （已存在 `{}` 先例）

```mdl
-- ============================================================
-- SHOWCASE: on error Clause
-- MDL grammar rule: onErrorClause
-- Note: {} form already existed as a precedent before this spec.
-- ============================================================

create module Showcase;

-- [CTRL-ONERROR-01] on error continue
create microflow Showcase.Ctrl_OnError_01 () returns Nothing
begin
  create Showcase.Order (OrderNumber = '001') on error continue;
end;
/

-- [CTRL-ONERROR-02] on error rollback
create microflow Showcase.Ctrl_OnError_02 () returns Nothing
begin
  create Showcase.Order (OrderNumber = '001') on error rollback;
end;
/

-- [CTRL-ONERROR-03] on error { body }
create microflow Showcase.Ctrl_OnError_03 () returns Nothing
begin
  create Showcase.Order (OrderNumber = '001') on error {
    log error node 'Showcase' 'create failed';
  };
end;
/
```

- [ ] **Step 14: 验证所有控制流文件**

```bash
find mdl-examples/syntax-showcase -name "ctrl-*.mdl" | sort | \
    xargs -I{} sh -c './bin/mxcli check {} && echo "OK: {}" || { echo "FAIL: {}"; exit 1; }'
```

期望：全部 `OK`。如有 FAIL，检查对应 grammar rule 是否正确添加了 `{}` 分支。

- [ ] **Step 15: commit**

```bash
git add mdl-examples/syntax-showcase/ctrl-*.mdl
git commit -m "feat(showcase): add ctrl-01..13 control flow showcase files (legacy + modern forms)"
```

---

## Task 12: 创建 DDL showcase (ddl-01..10)

**Files:**
- Create: `mdl-examples/syntax-showcase/ddl-01-entity.mdl` 到 `ddl-10-translate.mdl`

- [ ] **Step 1: 创建 ddl-01-entity.mdl** — 覆盖所有属性类型、constraints、索引、event handlers

```mdl
-- ============================================================
-- SHOWCASE: Entity DDL
-- MDL grammar rule: createEntityStatement
-- ============================================================

create module Showcase;

-- [DDL-ENT-01] 所有数据类型
create persistent entity Showcase.AllTypes (
  A_String:   string(200),
  A_Integer:  integer,
  A_Decimal:  decimal,
  A_Boolean:  boolean default false,
  A_DateTime: datetime,
  A_Long:     long,
  A_HashStr:  hashed string,
  A_Enum:     enumeration(Showcase.OrderStatus),
  A_AutoNum:  autonumber(1)
);
/
```

- [ ] **Step 2: 创建 ddl-02-enumeration.mdl** — 含 doc comment 的枚举
- [ ] **Step 3: 创建 ddl-03-association.mdl** — 四种 multiplicity、owner、delete_behavior 各一例
- [ ] **Step 4: 创建 ddl-04-microflow-signature.mdl** — 签名（all 参数类型、返回类型、reset layout）

  ```mdl
  create microflow Showcase.Sig_01 (
    $order: Showcase.Order,
    $name: string,
    $count: integer,
    $flag: boolean
  ) returns string as $out
  folder 'Showcase'
  begin
    declare $out: string = 'ok';
    return $out;
  end;
  /
  ```

- [ ] **Step 5: 创建 ddl-05-page.mdl** — 最小 page 语法（不需要真实执行）
- [ ] **Step 6: 创建 ddl-06-security.mdl** — `grant`/`revoke` 各形式
- [ ] **Step 7: 创建 ddl-07-constants.mdl** — `create constant` 所有数据类型
- [ ] **Step 8: 创建 ddl-08-annotations.mdl** — `@caption`, `@position`, `@excluded`, `@comment`
- [ ] **Step 9: 创建 ddl-09-alter.mdl** — `alter entity`（add/drop/rename 属性）
- [ ] **Step 10: 创建 ddl-10-translate.mdl** — translate 命令（dev 分支新增）

- [ ] **Step 11: 验证并 commit**

```bash
find mdl-examples/syntax-showcase -name "ddl-*.mdl" | sort | \
    xargs -I{} sh -c './bin/mxcli check {} && echo "OK: {}" || { echo "FAIL: {}"; exit 1; }'
git add mdl-examples/syntax-showcase/ddl-*.mdl
git commit -m "feat(showcase): add ddl-01..10 DDL layer showcase files"
```

---

## Task 13: 添加 Makefile `test-showcase` target

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: 在 `check-mdl` target 之后添加 `test-showcase`**

找到 `Makefile` 中 `check-mdl:` target（第 379 行附近），在其后添加：

```makefile
# Syntax showcase: grammar coverage regression test (no MPR needed)
test-showcase: build
	@echo "=== Syntax showcase: grammar check ==="
	@FAILED=0; \
	for f in $$(find mdl-examples/syntax-showcase -name "*.mdl" | sort); do \
		NAME=$$(basename "$$f"); \
		if ./$(BUILD_DIR)/$(BINARY_NAME) check "$$f" > /dev/null 2>&1; then \
			echo "OK: $$NAME"; \
		else \
			echo "FAIL: $$NAME"; \
			./$(BUILD_DIR)/$(BINARY_NAME) check "$$f" 2>&1 | grep -v "^WARNING"; \
			FAILED=1; \
		fi; \
	done; \
	echo "Passed: $$(find mdl-examples/syntax-showcase -name '*.mdl' | wc -l) files"; \
	exit $$FAILED
```

注意：`BUILD_DIR` 和 `BINARY_NAME` 变量已在 Makefile 顶部定义，直接使用。

- [ ] **Step 2: 运行验证**

```bash
make test-showcase
```

期望输出末尾：`Passed: 60 files`（或当前文件数）且 exit code 0。

如有 FAIL，查看具体文件并修复（通常是 grammar 未覆盖的语法或文件本身有 typo）。

- [ ] **Step 3: commit**

```bash
git add Makefile
git commit -m "build: add test-showcase Makefile target for grammar coverage CI"
```

---

## Task 14: 文档更新

**Files:**
- Modify: `.claude/skills/design-mdl-syntax.md`
- Modify: `docs/01-project/MDL_QUICK_REFERENCE.md`

- [ ] **Step 1: 更新 design-mdl-syntax.md**

在该文件的"控制流"或"Block 语法"相关章节（grep `BEGIN\|block\|loop\|while\|if`），添加以下内容：

```markdown
### Statement Block 语法（两种形式）

MDL 语句块支持两种等价形式。`{}` 是**规范形式**，`BEGIN…END` 是**向后兼容形式**。

**DESCRIBE 输出现状（Phase 1）**：
- 微流/纳流 **body**（`begin...end`）：Phase 1 仍输出旧形式；Phase 2 再切换 `{}`
- `split type`、`raise error`、`java action`：Stage 3.2.2（commit 7c0baba9f）已修复 describe 输出，现在与创建语法一致

| 构造 | 规范形式（新） | 兼容形式（旧） |
|------|--------------|--------------|
| microflow/nanoflow body | `create microflow M.F () { ... }` | `create microflow M.F () begin ... end` |
| if 语句 | `if cond { ... } elsif cond { ... } else { ... }` | `if cond then ... elsif cond then ... else ... end if` |
| loop 语句 | `loop $x in $list { ... }` | `loop $x in $list begin ... end loop` |
| while 语句 | `while cond { ... }` | `while cond begin ... end while` |
| case 语句 | `case $x { when A { ... } else { ... } }` | `case $x when A then ... else ... end case` |
| split type | `split type $x { case T { ... } }` | `split type $x case T ... end split` |
| declare | `declare $x: Type = val` | `declare $x Type = val` |
| 赋值 | `$x = expr` | `set $x = expr` |

**关键区分**：`then` 关键字（不跟 `{`）= Mendix 官方条件值表达式（返回值，P0 保护）。
`{` 开头 = MDL Statement Layer 语句块。两者不可混用于同一 if 语句。
```

- [ ] **Step 2: 更新 MDL_QUICK_REFERENCE.md**

找到控制流相关章节，将每个控制流语句的语法表格更新为显示两种形式：

在 `if`、`loop`、`while`、`case`、`split type` 的语法列中，添加 `{ }` 规范形式为主，`BEGIN…END` 为次：

```markdown
| `if` 语句 | `if cond { body } elsif cond { body } else { body }` | `if cond then body elsif cond then body else body end if` |
| `loop` | `loop $x in $list { body }` | `loop $x in $list begin body end loop` |
| `while` | `while cond { body }` | `while cond begin body end while` |
| `case` | `case $x { when V { body } else { body } }` | `case $x when V then body else body end case` |
| `split type` | `split type $x { case T { body } else { body } }` | `split type $x case T body else body end split` |
| `declare` | `declare $x: Type [= expr]` | `declare $x Type [= expr]` |
| 赋值 | `$x = expr` | `set $x = expr` |
```

- [ ] **Step 3: commit**

```bash
git add .claude/skills/design-mdl-syntax.md docs/01-project/MDL_QUICK_REFERENCE.md
git commit -m "docs: update design-mdl-syntax skill and MDL_QUICK_REFERENCE with {} canonical forms"
```

---

## Task 15: 最终验证

- [ ] **Step 1: 完整测试套件**

```bash
make build && make test-showcase
```

期望：`Passed: N files`（N ≥ 60），exit code 0。

- [ ] **Step 2: 验证向后兼容（旧形式仍工作）**

```bash
find mdl-examples/syntax-showcase -name "ctrl-*-legacy.mdl" | sort | \
    xargs -I{} sh -c './bin/mxcli check {} && echo "OK: {}" || { echo "FAIL: {}"; exit 1; }'
```

期望：全部 `OK`。

- [ ] **Step 3: 验证 grammar keyword_coverage_test（如有）**

```bash
CGO_ENABLED=0 go test ./mdl/grammar/ -v -run TestKeyword
```

期望：PASS（新 token ELSIF 等仍在覆盖列表中）。

- [ ] **Step 4: 验证 make test 无回归**

```bash
make test
```

期望：所有单元测试通过。

- [ ] **Step 5: 最终 commit（如有未提交的改动）**

```bash
git status
# 只提交 showcase 新文件，grammar 和 Makefile 应已提交
```

---

## 成功标准

- [ ] `make test-showcase` 通过（Layer 1，≥60 文件全部 `OK`）
- [ ] `make grammar` 无报错，parser 重新生成成功
- [ ] 旧形式（`begin…end`、`then…end if`）仍被 parser 接受（向后兼容）
- [ ] 新形式（`{ }`）被 parser 接受（grammar 正确扩展）
- [ ] `make test` 无回归
- [ ] `design-mdl-syntax.md` 和 `MDL_QUICK_REFERENCE.md` 已更新
- [ ] 微流 body 的 DESCRIBE 仍输出 `begin…end`（Phase 1 不变；Phase 2 才切换 `{}`）
- [ ] `split type`/`raise error`/`java action` describe 输出正确（已由 Stage 3.2.2 fix，commit 7c0baba9f）

---

## 关于 Layer 2 和 Layer 3

Layer 2（exprcheck Go 测试）和 Layer 3（write-path + showcase MPR）属于 Phase 1 范围内的"建立文件"任务，但需要 showcase MPR 构建完成。这超出了本计划的 parse-only 范围。

**Layer 2** 入口：`mdl/exprcheck/showcase_test.go`（新建），测试 `SlotExpectations` slot 覆盖。
**Layer 3** 入口：`mdl/executor/showcase_test.go`（新建），需要 `testutil.TestExec` + FUSE 内存 MPR。

这两层可在 showcase MPR 建立后作为独立 PR 实施。
