---
title: MDL 语法设计优化 + 语法 Showcase 回归测试
status: proposed
date: 2026-06-05
branch: dev
---

# MDL 语法设计优化 + 语法 Showcase 回归测试

**日期**: 2026-06-05
**状态**: Proposed
**目标分支**: dev（`/mnt/data_sdd/gh/mxcli-wt-02`）
**范围**: MDL 控制流语法规范化 + 全语法覆盖回归测试套件

---

## 背景

MDL 是 Mendix Studio Pro 表达式系统（XPath、表达式语言、活动）的**超集**。MDL 在官方 Mendix 语义之上，增加了以下 DDL 层和语句层：

- **DDL 层**：`CREATE`、`ALTER`、`DROP`、`LIST`、`DESCRIBE` 等结构声明
- **语句层**：微流/纳流体内的控制流（`if`、`loop`、`while`、`case`、`split type`）和变量管理（`declare`、赋值）

当前 MDL 存在五个可测量的不一致点，全部集中在**语句层**：

| 编号 | 问题 | 影响 |
|------|------|------|
| I1 | 变量声明无冒号：`declare $x boolean` 而 page 参数有冒号 `$x: Boolean` | 每个 microflow 必经 |
| I2 | 两种赋值关键字：`set $x = val` 与结果赋值 `$x = call ...` 不一致 | 变量赋值逻辑 |
| I3 | Block 分隔符不统一：microflow 用 `BEGIN…END`/`THEN…END IF`，page 用 `{ }` | LLM 最常犯的生成错误 |
| I4 | `on error { }` 已用 `{ }`（先例存在），但 `if`/`loop`/`while` 未跟进 | Grammar 内部不一致 |
| I5 | `show`（遗留）与 `list`（新规范）并存，两个动词做同一件事 | DDL 动词歧义 |

同时，项目**缺少一个以语法覆盖为目标的回归测试**：现有的 `mdl-examples/doctype-tests/` 是业务场景示例，不保证每个 grammar production rule 至少被执行一次。

---

## 核心原则

### P0：Mendix 平台语义不可侵犯（最高优先级）

MDL 分两层：

```
Expression Layer（表达式层）— Mendix 官方定义，MDL 只使用，不修改
  ├── XPath 约束：//Module.Entity[constraint], ., /, [], ()
  ├── 条件表达式：if cond then val else other_val（返回值，不是语句）
  ├── 运算符：and, or, div, mod, not(), empty 等
  ├── 函数：contains(), beginOfDay(), formatDecimal() 等 80+ 个
  └── 路径语法：$var/Assoc/Attr, [%SystemVar%]

Statement Layer（语句层）— MDL 自己发明，可以设计
  ├── 控制流：if/loop/while/case/split type
  ├── 活动语句：create, change, delete, retrieve, call microflow 等
  └── 变量管理：declare, 赋值
```

**官方 Mendix `if` 表达式**（P0 保护，永不改变）：
```
if $package/weight < 1.00 then 0.00 else 5.00
```
这是内联的条件值表达式，返回一个值，不是控制流语句。

### P1：两层使用不同的块边界

| 分隔符 | 层 | 用途 |
|--------|---|------|
| `{ }` | Statement 层 | 语句块边界（控制流） |
| `( )` | DDL / Expression 层 | 属性列表、函数调用 |
| `:` | DDL / Declaration | 类型声明、属性 key:value |
| `=` | Statement / Expression | 运行时赋值、比较、参数绑定 |
| `then … else` | Expression 层专属 | Mendix 官方条件值，不用于语句 if |

`then` 关键字从此只属于 Expression Layer（官方规定）。Statement Layer 的 `if` 语句改用 `{ }`，与表达式 `if` 形成视觉上零歧义的区分。

### P2：声明与赋值的分隔符语义

- `:` = "声明为/类型为"（结构性，静态）→ 类型注解、DDL 属性
- `=` = "运行时赋值为"（命令性，动态）→ 变量赋值、活动参数、比较运算

### P3：变量生命周期（P2 推论）

- `declare $x: Type [= init]` — 引入新变量，`declare` 关键字必须保留（视觉标记），`:` 标注类型
- `$x = expr` — 重新赋值，无需 `set` 关键字
- `set $x = expr` — 向后兼容保留，DESCRIBE 不输出

### P4：英语可读性仍是首位（继承 ADR-0003）

活动调用前缀（`call microflow`、`call java action`）保持不变。去掉 `call` 节省的 token 不值得损失明确性。

### P5：每种操作一种规范形式

- DESCRIBE 输出规范形式
- 旧形式向后兼容接受
- lint 规则 `MDL-DEPR001` 警告旧形式

### P6：LLM 一致性（P1 的推论）

有了 P1 的分层，LLM 的区分规则变为：
- 看到 `then`（不跟 `{`）= 进入 Mendix Expression Layer（值）
- 看到 `{` = 进入 MDL Statement Layer（语句块）

---

## 语法设计：各区域

### 区域 A：顶层 microflow/nanoflow body

```mdl
-- 规范形式（P1: Statement Layer 用 {}）
create microflow Shop.ACT_Process ($order: Shop.Order) returns Nothing {
  ...
}

-- 向后兼容（dev 分支上仍可解析）
create microflow Shop.ACT_Process ($order: Shop.Order) returns Nothing
begin
  ...
end;
```

**Grammar 修改**（`MDLMicroflow.g4`）：
```antlr
createMicroflowStatement
    : MICROFLOW qualifiedName LPAREN microflowParameterList? RPAREN
      microflowReturnType? microflowOptions?
      ( BEGIN microflowBody END          -- 兼容旧形式
      | LBRACE microflowBody RBRACE      -- 新规范形式
      ) SEMICOLON? SLASH?
    ;
```
nanoflow 同理。

### 区域 B：if/elsif/else 语句

```mdl
-- 规范形式
if $order/Status = 'pending' {
  change $order (Status = 'active');
} elsif $order/Status = 'cancelled' {
  raise error;
} else {
  log warning node 'Shop' 'Unexpected status';
}

-- 官方表达式 if（永不改变）
declare $fee: Decimal = if $order/Weight < 1.0 then 0.0 else 5.0;

-- 向后兼容
if $order/Status = 'pending' then
  change $order (Status = 'active');
end if;
```

**Grammar 修改**：
```antlr
ifStatement
    : IF expression
      ( THEN microflowBody (ELSIF expression THEN microflowBody)* (ELSE microflowBody)? END IF
      | LBRACE microflowBody RBRACE
        (ELSIF expression LBRACE microflowBody RBRACE)*
        (ELSE LBRACE microflowBody RBRACE)?
      )
    ;
```

两种形式**不可混用**于同一个 if 语句（`if expr { body } end if` 非法）。

### 区域 C：loop（for-each）

```mdl
-- 规范形式
loop $product in $products {
  change $product (IsActive = true);
}

-- 向后兼容
loop $product in $products begin
  change $product (IsActive = true);
end loop;
```

**Grammar 修改**：
```antlr
loopStatement
    : LOOP VARIABLE IN (VARIABLE | attributePath)
      ( BEGIN microflowBody END LOOP
      | LBRACE microflowBody RBRACE
      )
    ;
```

注：`$currentIndex` 是 Mendix 官方在 loop 内提供的变量（官方文档），MDL 不重命名。

### 区域 D：while

```mdl
-- 规范形式
while $count < 10 {
  $count = $count + 1;
}

-- 向后兼容
while $count < 10 begin
  set $count = $count + 1;
end while;
```

**Grammar 修改**：
```antlr
whileStatement
    : WHILE expression
      ( BEGIN? microflowBody END WHILE?
      | LBRACE microflowBody RBRACE
      )
    ;
```

### 区域 E：case（枚举分支）

```mdl
-- 规范形式（when 后紧跟 {}，无 then）
case $order/Priority {
  when High {
    log warning node 'Shop' 'High priority';
  }
  when Low, Medium {
    log info node 'Shop' 'Standard';
  }
  else {
    log debug node 'Shop' 'Unknown';
  }
}

-- 向后兼容
case $order/Priority
  when High then
    log warning node 'Shop' 'High priority';
end case;
```

`then` 关键字在新形式中被 `{` 替代，与 P1 一致。

### 区域 F：split type（继承分支）

```mdl
-- 规范形式
split type $animal {
  case Shop.Dog {
    change $animal (Breed = 'Labrador');
  }
  case Shop.Cat {
    change $animal (Indoor = true);
  }
  else {
    log info node 'Shop' 'Unknown';
  }
}

-- 向后兼容
split type $animal
  case Shop.Dog
    change $animal (Breed = 'Labrador');
end split;
```

### 区域 G：declare 变量声明（P2 + P3）

```mdl
-- 规范形式（: 标注类型，与 page 参数 $x: Type 对齐）
declare $count: integer = 0;
declare $name: string;
declare $products: list of Shop.Product;
declare $result: Shop.Order = empty;

-- 向后兼容
declare $count integer = 0;
```

**Grammar 修改**：
```antlr
declareStatement
    : DECLARE VARIABLE COLON? dataType (EQUALS expression)?
    ;
```

DESCRIBE 输出带冒号的规范形式。

### 区域 H：变量赋值（P3）

```mdl
-- 规范形式（无 set 关键字）
$count = $count + 1;
$order/Status = 'active';

-- 向后兼容
set $count = $count + 1;
```

**Grammar 修改**：在 `microflowStatement` 末尾新增（放在最后避免歧义）：
```antlr
simpleAssignStatement
    : (VARIABLE | attributePath) EQUALS expression
    ;
```

### 不改动的区域

| 区域 | 保持不变 | 原因 |
|------|---------|------|
| `call microflow M.MF(Param = $val)` | ✓ | P4：前缀明确意图；`=` 符合 P2 |
| `change $x (Attr = val)` | ✓ | P2：`=` 是运行时赋值 |
| `create Entity (Attr = val)` | ✓ | 同上 |
| DDL 属性 `(key: value)` | ✓ | 已是规范形式 |
| 表达式 `if cond then val else other` | ✓ | P0：Mendix 官方，永不改 |
| XPath、表达式函数、系统变量 | ✓ | P0：Mendix 官方，永不改 |
| `on error { }` | ✓ | 已是规范形式，本次工作的先例 |

---

## 语法 Showcase 回归测试

### 定位与目的

**Showcase ≠ doctype-tests（业务场景）≠ golden MPR（Studio Pro 验证）**

| | 目的 | 内容 | 验证方式 |
|--|------|------|---------|
| **syntax-showcase** | 语法目录 + 回归测试基线 | 最小化，一构造一示例 | `mxcli check` + CI |
| **doctype-tests** | 功能集成测试 | 按文档类型综合示例 | `mxcli exec` |
| **golden MPR** | 业务场景 + Studio Pro 验证 | 完整业务流程 | `mx check` + Studio Pro |

Showcase 是覆盖 MDL 作为 Mendix 超集的**语法目录**：每个 grammar production rule、每个官方 Mendix 表达式函数、每个活动类型都至少有一个最小化示例。

### 三层验证架构

```
层 1：语法层（parse-only，无需 MPR）
  位置：mdl-examples/syntax-showcase/*.mdl
  CI:   find mdl-examples/syntax-showcase -name "*.mdl" | xargs -n1 mxcli check
  验证：所有语法构造被 ANTLR grammar 接受

层 2：表达式层（exprcheck type-check，无需 MPR）
  位置：mdl/exprcheck/showcase_test.go
  工具：TestExec（mock backend）+ exprcheck 适配器
  验证：每个 SlotExpectations slot 的示例表达式通过类型检查
        对照 generated/exprgrammar/mined.go 中的 slot 定义

层 3：写路径层（write + BSON roundtrip，需要 MPR）
  位置：mdl/executor/showcase_test.go（package executor_test）
  工具：TestExec（testutil 包）+ FUSE 内存 MPR + bsoncompare
  验证：每个 write 活动产生正确 BSON 结构
        describe roundtrip 产生等价 MDL（旧形式或新形式均通过）
```

### 文件目录结构

```
mdl-examples/syntax-showcase/
├── MANIFEST.md                        -- 目录索引，说明如何运行
├── 00-setup.mdl                       -- 被其他文件依赖的 module/entity/enum 前提
│
├── -- ① Mendix 官方：表达式系统（P0 约束，照单全收）
├── expr-01-arithmetic.mdl             -- +, -, *, div, mod
├── expr-02-relational.mdl             -- =, !=, <, <=, >, >=
├── expr-03-boolean.mdl                -- and, or, not()
├── expr-04-if-expression.mdl          -- if cond then val else other（Mendix 官方内联 if）
├── expr-05-special-checks.mdl         -- empty, isNew(), isSynced()
├── expr-06-string-functions.mdl       -- contains(), startsWith(), toLowerCase() 等 14 个
├── expr-07-math-functions.mdl         -- abs(), round(), floor(), pow() 等 9 个
├── expr-08-date-create.mdl            -- dateTime(), dateTimeUTC()
├── expr-09-date-add.mdl               -- addDays(), addMonths() 等 14 个
├── expr-10-date-subtract.mdl          -- subtractDays() 等 14 个
├── expr-11-date-between.mdl           -- daysBetween(), calendarMonthsBetween() 等 8 个
├── expr-12-date-trim.mdl              -- trimToDays(), trimToMonths() 等 10 个
├── expr-13-date-begin-end.mdl         -- beginOfDay/Week/Month/Year, endOf* 等 8 个
├── expr-14-date-format.mdl            -- formatDateTime(), parseDateTime() 等 10 个
├── expr-15-decimal-format.mdl         -- formatDecimal(), parseDecimal()
├── expr-16-parse-integer.mdl          -- parseInt(), toString()
├── expr-17-length.mdl                 -- length()
├── expr-18-system-variables.mdl       -- [%CurrentDateTime%], [%CurrentUser%] 等 26 个
├── expr-19-enumerations.mdl           -- 枚举值比较，getCaption()
│
├── -- ② Mendix 官方：XPath 约束
├── xpath-01-tokens.mdl                -- //, ., /, [], ()
├── xpath-02-operators.mdl             -- =, !=, <, >, and, or 在 XPath 中
├── xpath-03-functions.mdl             -- contains(), starts-with(), not() 等
├── xpath-04-associations.mdl          -- 跨关联路径约束
├── xpath-05-nested.mdl                -- 嵌套子约束
├── xpath-06-system-variables.mdl      -- [%CurrentUser%] 等系统变量在 XPath 中
│
├── -- ③ Mendix 官方：活动（按官方 Activities 分类对应）
├── act-01-object-create.mdl           -- create object（all 属性类型）
├── act-02-object-change.mdl           -- change object（all 变体 + refresh）
├── act-03-object-delete.mdl           -- delete object + on error 变体
├── act-04-object-commit.mdl           -- commit（with events, without events, refresh）
├── act-05-object-rollback.mdl         -- rollback
├── act-06-object-retrieve-db.mdl      -- retrieve from database（where/sort/limit/offset）
├── act-07-object-retrieve-assoc.mdl   -- retrieve from association
├── act-08-object-cast.mdl             -- cast + split type
├── act-09-list-create.mdl             -- declare list = empty
├── act-10-list-change.mdl             -- add to list, remove from list
├── act-11-list-aggregate.mdl          -- count(), sum(), avg(), min(), max()
├── act-12-list-operation.mdl          -- filter(), sort(), find(), union(), intersect(), subtract()
├── act-13-variable-declare.mdl        -- declare $x: type（all 数据类型，旧形式 + 新形式）
├── act-14-variable-assign.mdl         -- $x = expr（新形式）, set $x = expr（兼容形式）
├── act-15-call-microflow.mdl          -- call microflow（all 参数 + on error 变体）
├── act-16-call-nanoflow.mdl           -- call nanoflow
├── act-17-call-java-action.mdl        -- call java action（含 dev 新增多块 source）
├── act-18-call-javascript-action.mdl  -- call javascript action
├── act-19-call-workflow.mdl           -- call workflow + notify activity（dev 新增）
├── act-20-workflow-jump-to.mdl        -- generate jump to options + apply jump to option（dev 新增）
├── act-21-client-show-page.mdl        -- show page（带参数/不带参数）
├── act-22-client-close-page.mdl       -- close page
├── act-23-client-show-message.mdl     -- show message（info/warning/error/confirm）
├── act-24-client-validation.mdl       -- validation feedback
├── act-25-client-download-file.mdl    -- download file
├── act-26-log-message.mdl             -- log（trace/debug/info/warning/error/critical + template）
├── act-27-raise-error.mdl             -- raise error
├── act-28-rest-call.mdl               -- call rest service（all 变体）
├── act-29-web-service.mdl             -- call web service
├── act-30-import-export-mapping.mdl   -- import/export with mapping
│
├── -- ④ MDL 超集：控制流语句层（新旧两种形式各一个文件）
├── ctrl-01-if-legacy.mdl              -- if … then … end if（向后兼容形式）
├── ctrl-02-if-modern.mdl              -- if … { } elsif … else（新规范形式）
├── ctrl-03-if-nested.mdl              -- 嵌套 if，多个 elsif（新形式）
├── ctrl-04-loop-legacy.mdl            -- loop $x in $list begin … end loop
├── ctrl-05-loop-modern.mdl            -- loop $x in $list { }（新规范形式）
├── ctrl-06-while-legacy.mdl           -- while expr begin … end while
├── ctrl-07-while-modern.mdl           -- while expr { }（新规范形式）
├── ctrl-08-break-continue.mdl         -- break, continue, $currentIndex
├── ctrl-09-case-legacy.mdl            -- case … when … then … end case
├── ctrl-10-case-modern.mdl            -- case … { when A { } else { } }（新形式）
├── ctrl-11-split-type-legacy.mdl      -- split type … case … end split
├── ctrl-12-split-type-modern.mdl      -- split type … { case M.T { } }（新形式）
├── ctrl-13-on-error.mdl               -- on error continue / rollback / { body }（已存在 {} 先例）
│
├── -- ⑤ MDL 超集：DDL 层
├── ddl-01-entity.mdl                  -- create entity（all 属性类型、约束、索引、event handlers）
├── ddl-02-enumeration.mdl             -- create enumeration（含 doc comment）
├── ddl-03-association.mdl             -- create association（all 类型、owner、delete_behavior）
├── ddl-04-microflow-signature.mdl     -- 微流签名（all 参数类型、返回类型、reset layout，dev 新增）
├── ddl-05-page.mdl                    -- create page（代表性 widget 集）
├── ddl-06-security.mdl                -- grant/revoke（all 形式）
├── ddl-07-constants.mdl               -- create constant（all 数据类型）
├── ddl-08-annotations.mdl             -- @caption, @position, @excluded, @comment
├── ddl-09-alter.mdl                   -- alter entity/page（all 操作）
└── ddl-10-translate.mdl               -- 翻译命令（dev 新增：translate/translations/supported）
```

### 文件写法范式

```mdl
-- ============================================================
-- SHOWCASE: Call Microflow Activity
-- Mendix doc: /refguide/microflow-call/
-- MDL grammar rule: callMicroflowStatement
-- SlotExpectations: CallArgument.Value
-- ============================================================

create module Showcase;

-- 前提
create microflow Showcase.Target ($name: string) returns string as $out
begin
  declare $out: string = $name;
  return $out;
end;
/

-- [MF-CALL-01] 无返回值
create microflow Showcase.MF_Call_NoReturn () returns Nothing
begin
  call microflow Showcase.Target(name = 'test');
end;
/

-- [MF-CALL-02] 有返回值
create microflow Showcase.MF_Call_WithReturn () returns string as $r
begin
  declare $r: string = empty;
  $r = call microflow Showcase.Target(name = 'hello');
  return $r;
end;
/

-- [MF-CALL-03] 带 on error continue
create microflow Showcase.MF_Call_OnError () returns Nothing
begin
  call microflow Showcase.Target(name = 'x') on error continue;
end;
/

-- [MF-CALL-04] 带 on error 自定义处理（已有 {} 先例）
create microflow Showcase.MF_Call_OnErrorBody () returns Nothing
begin
  call microflow Showcase.Target(name = 'x') on error {
    log error node 'Showcase' 'call failed';
  };
end;
/
```

每个构造的注释须包含：
- Mendix 官方文档链接（表达式/活动类）或 MDL ADR 引用（MDL 自创类）
- 对应的 grammar rule 名称
- 相关的 `SlotExpectations` slot 路径（如有）
- 构造编号（如 `[MF-CALL-01]`），供 CI 日志追踪

### CI 集成

```makefile
# Makefile 新增 target
test-showcase:
	@echo "=== Syntax showcase: grammar check ==="
	find mdl-examples/syntax-showcase -name "*.mdl" | sort | \
	    xargs -I{} sh -c './bin/mxcli check {} && echo "OK: {}" || exit 1'
	@echo "Passed: $$(find mdl-examples/syntax-showcase -name '*.mdl' | wc -l) files"

# 并入 make test
test: build test-unit test-integration test-showcase
```

Go 层验证（层 2、层 3）并入现有 `go test ./...`，因为 showcase 测试文件在标准包路径下。

### exprgrammar-mine 与 showcase 的关系

showcase 的 `expr-*` 文件提供了真实可执行的表达式示例。应在 showcase MPR 建立后，用 `exprgrammar-mine` 重新扫描，更新 `generated/exprgrammar/mined.go`：

```bash
# 更新 slot expectations（在 showcase MPR 建立后运行）
go run ./cmd/exprgrammar-mine -mpr testdata/showcase.mpr \
    -out generated/exprgrammar/mined.go

# CI 验证 mined.go 没有意外变化
git diff --exit-code generated/exprgrammar/mined.go
```

这确保 showcase 的表达式示例和 slot expectations 始终同步。

---

## 迁移策略

```
Phase 1（本 spec 实现范围）：
  - Grammar 同时接受 {} 和 BEGIN/THEN/END 两种形式（ANTLR | 分支）
  - DESCRIBE 继续输出旧形式（零 golden test 破坏）
  - 建立 syntax-showcase/ 完整文件集（层 1 + 层 2 + 层 3）
  - CI 集成 test-showcase target
  - 更新 design-mdl-syntax.md skill 和 MDL_QUICK_REFERENCE.md

Phase 2（独立 ADR，下一季度）：
  - DESCRIBE 切换输出 {} 规范形式
  - 更新所有 mdl-examples/ golden test 和 doctype-tests
  - lint 规则 MDL-DEPR001 警告旧 BEGIN/END/THEN/END IF 形式

Phase 3（v1.0+，独立 ADR）：
  - 旧形式停止支持（parser 移除 BEGIN/END/THEN 分支）
  - ctrl-XX-*-legacy.mdl 文件归档删除
```

---

## 实现路径（dev 分支上）

```
Step 1: 从 dev 分支开 feature/mdl-syntax-showcase 分支
Step 2: 建立 mdl-examples/syntax-showcase/00-setup.mdl（基础实体/枚举）
Step 3: 写 expr-* 文件（表达式层，parse-only）— 基线，current grammar 全通过
Step 4: 写 xpath-* 文件
Step 5: 写 act-* 文件（活动层，含 dev 新增的 jump-to、java source blocks、notify activity）
Step 6: 运行 CI 确认层 1 基线全通过
Step 7: Grammar 修改（MDLMicroflow.g4：6 处添加 LBRACE/RBRACE 分支）
Step 8: 运行 make grammar（重新生成 ANTLR parser）
Step 9: 写 ctrl-*-modern.mdl（新 {} 形式），ctrl-*-legacy.mdl（旧形式验证向后兼容）
Step 10: 写 ddl-* 文件，含 dev 新增的 translate 和 reset layout
Step 11: 写层 2（exprcheck）Go 测试文件
Step 12: 写层 3（write-path）Go 测试文件（需 showcase MPR）
Step 13: 更新 design-mdl-syntax.md skill（记录 {} 规范形式）
Step 14: 更新 MDL_QUICK_REFERENCE.md（每个控制流构造记录两种形式）
Step 15: PR → dev
```

---

## 成功标准

- [ ] `mxcli check mdl-examples/syntax-showcase/*.mdl` 全部通过（层 1）
- [ ] `go test ./mdl/exprcheck/...` 覆盖所有 `SlotExpectations` slot（层 2）
- [ ] `go test ./mdl/executor/...` write-path showcase 测试通过（层 3）
- [ ] `make grammar` 无报错，parser 重新生成成功
- [ ] 旧形式（`begin…end`、`then…end if`）仍被 parser 接受（向后兼容）
- [ ] 新形式（`{ }`）被 parser 接受（grammar 正确扩展）
- [ ] DESCRIBE 输出不变（继续输出旧形式，Phase 1）
- [ ] `git diff --exit-code generated/exprgrammar/mined.go` 通过（slot expectations 同步）
- [ ] design-mdl-syntax.md skill 和 MDL_QUICK_REFERENCE.md 已更新

---

## 关联文件

- Grammar: `mdl/grammar/domains/MDLMicroflow.g4`
- 语法设计决策: `docs/13-decisions/0003-mdl-is-sql-shaped.md`
- 语法设计 skill: `.claude/skills/design-mdl-syntax.md`
- 语法快速参考: `docs/01-project/MDL_QUICK_REFERENCE.md`
- Expression checker 设计: `docs/superpowers/specs/2026-05-12-exprcheck-inferkind-systematic-design.md`
- Slot expectations: `generated/exprgrammar/mined.go`
- TestExec helper: `mdl/executor/testutil/testutil.go`
- FUSE mount: `mdl/executor/testutil/fuse.go`
- bsoncompare: `internal/bsoncompare/`
