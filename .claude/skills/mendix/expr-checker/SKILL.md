# mxcli expr — Mendix 表达式检查器

`mxcli expr` 是一条完整的表达式检查流水线，可扫描 MPR 中所有表达式字符串，发现语法和语义错误，并提供修复建议。

## 核心子命令

```bash
# 扫描 mprcontents/ 中所有表达式，输出 JSONL
mxcli expr scan <mprcontents>...

# 解析收集到的表达式（检测 token 级错误）
mxcli expr parse <mprcontents>...

# 应用 SYN + SEM 验证规则（推荐：与 -p 一起用）
mxcli expr validate -p app.mpr

# 为可修复问题生成修复建议
mxcli expr repair <mprcontents>...

# 完整流水线：scan → parse → validate → 生成报告
mxcli expr report -p app.mpr --format html -o report.html
```

## 重要选项

| 选项 | 适用命令 | 说明 |
|------|----------|------|
| `--format json\|html\|text` | `report`, `scan`, `validate` | 输出格式（默认 json） |
| `--filter <substring>` | `validate`, `report` | 按 unit_type 过滤（如 `Microflow`） |
| `--severity ERROR\|WARNING\|INFO` | `validate`, `report` | 按严重程度过滤 |
| `--summary` | `scan` | 输出人类可读统计而非 JSONL |

## 错误码体系

### SYN — 语法规则

| 码 | 含义 | 严重程度 |
|----|------|----------|
| `SYN-01` | 表达式解析失败（token 级错误） | ERROR |
| `SYN-02` | 字段存储了 URL 而非表达式 | INFO |
| `SYN-03` | if-then 缺少 else 分支（启发式） | WARNING |

### SEM — 语义规则（需要 --project / -p）

| 码 | 含义 | 严重程度 |
|----|------|----------|
| `SEM-04` | 枚举值引用不存在（如 `Status.Active` 但枚举无此值） | ERROR |
| `SEM-05` | 常量引用不存在（如 `MyModule.CONST_X`） | ERROR |
| `SEM-07` | 实体属性或关联路径不存在（如 `$Var/Module.Entity/UnknownAttr`） | ERROR |

## 典型工作流

### 快速语法扫描（无需打开项目）

```bash
mxcli expr validate -p app.mpr --format text
```

### 完整语义检查（需要 MPR）

```bash
mxcli expr validate -p app.mpr --format json | jq '.[] | select(.Severity=="ERROR")'
```

### CI 集成

```bash
mxcli expr validate -p app.mpr --severity ERROR --format json
echo "Exit: $?"
```

### 生成 HTML 报告

```bash
mxcli expr report -p app.mpr --format html -o expr-report.html
open expr-report.html
```

## Index 工作原理

`mxcli expr validate` 自动构建语义 index（实体属性、枚举值、常量）：

1. **建立 index** — 打开 MPR 并构建实体/枚举/常量/关联/微流变量的内存索引（无持久化 daemon 进程）
2. **扫描表达式** — 从 mprcontents/ 中提取所有表达式字符串
3. **解析 + 校验** — 对每个表达式进行语法解析和语义校验
4. **输出结果** — 带错误码的格式化报告

语义校验（SEM-04/05/07）需要 `--project` / `-p` 参数；没有时只做语法校验（SYN 规则）。

## 与 LSP / VS Code 的关系

`mxcli expr` 是独立的批量检查工具，与 LSP 的实时表达式诊断是不同路径。LSP 诊断在编辑器里逐表达式触发；`mxcli expr` 适合全项目扫描和 CI 场景。
