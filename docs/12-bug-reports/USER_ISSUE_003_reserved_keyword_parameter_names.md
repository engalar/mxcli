# Issue 003: 保留关键词与参数名冲突，DESCRIBE 输出不自动加引号

## 元数据

| 字段 | 值 |
|------|-----|
| Reporter | Miwa |
| 分类 | MxCli |
| 状态 | Open |
| 优先级 | 高 |
| 发现日期 | 2026-06-02 |

## 问题描述

MDL 将常见英文单词作为保留关键词（Template、Attribute、Column、List 等），与 Marketplace 模块参数名（如 ExcelImporter）产生命名冲突。`DESCRIBE MICROFLOW` 输出的标识符未自动加引号，直接使用会导致 `mxcli check` 报错。

## 已确认为保留字的标识符

根据 `mdl/grammar/MDLLexer.g4`：

| 标识符 | 行号 |
|--------|------|
| `TEMPLATE` | 第 380 行 |
| `ATTRIBUTE` | 第 75 行 |
| `COLUMN` / `COLUMNS` | 第 76–77 行 |
| `LIST` | 第 217 行 |

## Bug 代码位置

**`mdl/executor/cmd_microflows_show_gen.go`**（约第 136–144 行）：

```go
lines = append(lines, fmt.Sprintf("  $%s: %s%s", p.name, p.declType, comma))
```

`p.name` 直接输出，未检查是否为保留字，未自动添加反引号或双引号。

## 复现步骤

1. 在 Mendix 项目中存在参数名为 `Template` 的微流（来自 ExcelImporter 等 Marketplace 模块）
2. 执行 `DESCRIBE MICROFLOW Module.SomeMicroflow`
3. 将输出复制后执行，`mxcli check` 报语法错误

## 期望行为

DESCRIBE 输出应自动检测保留字并加引号：
```sql
create or modify microflow ExcelImporter.ImportExcel (
  $`Template`: ExcelImporter.Template,
  $`Column`: ExcelImporter.Column
)
```

或在 `mxcli check` 时给出带建议的错误：
```
error: 'Template' is a reserved keyword. Use $`Template` instead.
```

## 实际行为

输出未加引号的标识符，使用时报语法错误，错误信息未提示解决方法。

## 修复建议

在 `genMicroflowParameters()` 输出参数名时，调用保留字检测函数，若匹配则加反引号。保留字列表可从 lexer token 常量表生成。

## 关联文件

- `mdl/executor/cmd_microflows_show_gen.go` — DESCRIBE MICROFLOW 输出逻辑（第 136–144 行）
- `mdl/grammar/MDLLexer.g4` — 保留字定义（第 75、76–77、217、380 行）
- `mdl/grammar/domains/MDLDomainModel.g4` — `attributeName` 规则（支持 keyword 作为属性名）
