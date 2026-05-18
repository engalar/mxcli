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

# 管理后台 daemon
mxcli expr daemon start   -p app.mpr
mxcli expr daemon status
mxcli expr daemon stop    -p app.mpr
```

## 重要选项

| 选项 | 适用命令 | 说明 |
|------|----------|------|
| `--no-daemon` | `validate` | 跳过 daemon，仅做语法校验（适合 CI） |
| `--socket PATH` | `daemon start` | 自定义 Unix socket 路径 |
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

### SEM — 语义规则（需要 -p 和 daemon）

| 码 | 含义 | 严重程度 |
|----|------|----------|
| `SEM-04` | 枚举值引用不存在（如 `Status.Active` 但枚举无此值） | ERROR |
| `SEM-05` | 常量引用不存在（如 `MyModule.CONST_X`） | ERROR |
| `SEM-07` | 实体属性或关联路径不存在（如 `$Var/Module.Entity/UnknownAttr`） | ERROR |

## 典型工作流

### 快速语法扫描（无需打开项目）

```bash
mxcli expr validate -p app.mpr --no-daemon --format text
```

### 完整语义检查（需要 MPR）

```bash
# daemon 会自动启动并缓存 index
mxcli expr validate -p app.mpr --format json | jq '.[] | select(.Severity=="ERROR")'
```

### CI 集成

```bash
# 仅语法检查，不启动 daemon，非零退出码表示有 ERROR
mxcli expr validate -p app.mpr --no-daemon --severity ERROR --format json
echo "Exit: $?"
```

### 生成 HTML 报告

```bash
mxcli expr report -p app.mpr --format html -o expr-report.html
open expr-report.html
```

## Daemon 工作原理

`mxcli expr validate`（不带 `--no-daemon`）会自动启动一个后台 daemon，daemon：
- 为 MPR 建立 JIT 语义索引（实体属性、枚举值、常量）
- 通过 Unix socket 提供校验服务（socket 路径从 MPR 路径派生，默认 `/tmp/mxexpr-*.sock`）
- 空闲超时后自动退出

手动管理：`mxcli expr daemon start|status|stop -p app.mpr`

## 与 LSP / VS Code 的关系

`mxcli expr` 是独立的批量检查工具，与 LSP 的实时表达式诊断是不同路径。LSP 诊断在编辑器里逐表达式触发；`mxcli expr` 适合全项目扫描和 CI 场景。
