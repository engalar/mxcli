# 设计：mxcli --verbose 追踪日志 + diag bundle 增强

**日期：** 2026-05-21  
**状态：** 已批准，待实现

---

## 背景与目标

用户遇到 mxcli 运行时问题时，当前缺乏两个能力：

1. **实时诊断输出**：`mxcli` 日志全部写入文件，终端无任何追踪信息，难以判断执行卡在哪个阶段。
2. **充分的 bug 报告包**：`mxcli diag --bundle` 已生成 tar.gz，但缺少运行环境、项目元信息和错误追栈。

本设计增加全局 `-v`/`-vv` verbose flag 和 bundle 内容增强，帮助用户自助诊断并为维护者提供充分的故障数据。

---

## 范围

**包含：**
- 全局 `-v`（trace）/ `-vv`（debug）flag
- diaglog 原生支持 stderr 输出（方案 B）
- `mxcli diag --bundle` 新增 env-dump、project-meta、error-stacks 三类收集项

**不包含：**
- REPL 模式（正在单独规划删除）
- 格式变更（保留 tar.gz）
- 新顶层子命令（入口不变）

---

## 方案选择

选择**方案 B — diaglog 原生感知 verbose**，理由：

- 日志统一由 `diaglog.Logger` 管控，格式一致
- TextHandler（-v）和 JSONHandler（-vv）可精确控制哪些记录走 verbose
- 不引入额外的 slog 分支或临时文件

---

## 设计详情

### 1. CLI 接口

#### 全局 verbose flag

在 `rootCmd.PersistentFlags()` 中增加 count flag：

```go
rootCmd.PersistentFlags().CountP("verbose", "v",
    "Enable verbose output (-v trace, -vv debug)")
```

**使用方式：**

```bash
mxcli -v   -p app.mpr -c "show entities"   # 执行追踪：语句、耗时
mxcli -vv  -p app.mpr -c "show entities"   # 调试：所有内部状态
```

`CountP` flag 语义：`-v` → verboseLevel=1，`-vv` 或 `-v -v` → verboseLevel=2。

verbose 不适用于 REPL 模式（REPL 功能独立规划删除）。

#### bundle 命令（入口不变，内容增强）

```bash
mxcli diag --bundle              # 现有用法，新增 env-dump、error-stacks
mxcli diag --bundle -p app.mpr  # 额外包含 project-meta.txt
```

---

### 2. diaglog 内部架构

#### 数据结构变化

```go
// mdl/diaglog/diaglog.go
type Logger struct {
    slog      *slog.Logger    // 现有：JSON handler → 文件
    stderrLog *slog.Logger    // 新增：Text/JSON handler → stderr
    verbose   int             // 0=关，1=trace，2=debug
    file      *os.File
    cmdCount  int
    errCount  int
    startTime time.Time
}
```

#### Init 函数签名

```go
// 新增 verboseLevel 参数（0/1/2），main.go 读取 PersistentFlags 后传入
func Init(version, mode string, verboseLevel int) *Logger
```

内部逻辑：

```go
func Init(version, mode string, verboseLevel int) *Logger {
    // ... 现有文件 handler 初始化不变 ...

    l.verbose = verboseLevel
    if verboseLevel == 1 {
        // trace: 人类可读 TextHandler
        l.stderrLog = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
            Level: slog.LevelInfo,
        }))
    } else if verboseLevel >= 2 {
        // debug: JSON handler，与文件格式一致，便于对比
        l.stderrLog = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
            Level: slog.LevelDebug,
        }))
    }
    return l
}
```

#### 输出内容对照

| 操作 | `-v` trace 输出（人类可读） | `-vv` debug 输出（JSON） |
|------|----------------------------|-----------------------------|
| 执行 MDL 语句 | `level=INFO msg="execute" stmt=SHOW\ ENTITIES duration_ms=12` | 完整 JSON 含 backend 方法列表 |
| 连接项目 | `level=INFO msg="connect" project=app.mpr version=11.6.6` | + module 数量、文档计数 |
| 错误 | `level=ERROR msg="parse_error" preview=...` | + 完整 error 字段和上下文 |

verbose 输出写入 stderr（保持 stdout 用于正常命令输出，不混淆管道）。

#### 现有方法适配

`diaglog.Logger` 的 `Command()`、`Connect()`、`ParseError()`、`Info()`、`Warn()`、`Error()` 方法各自增加对 `stderrLog` 的镜像调用（仅当 `verbose > 0`）。

---

### 3. bundle 内容增强

`mxcli diag --bundle` 生成的 tar.gz 新增三个条目：

```
mxcli-diag-20260521-143022.tar.gz
├── system-info.txt          # 现有：版本、Go、OS/Arch
├── env-dump.txt             # 新增：运行环境
├── project-meta.txt         # 新增（仅 -p 指定时）：MPR 元信息
├── error-stacks.txt         # 新增：最近 20 条 ERROR 记录
└── logs/
    ├── mxcli-2026-05-20.log # 现有日志文件
    └── mxcli-2026-05-21.log
```

#### env-dump.txt

```
=== Go Runtime ===
MemSys: 45 MB
HeapAlloc: 12 MB
NumGC: 3
NumCPU: 8
NumGoroutine: 4

=== Environment Variables ===
PATH=/usr/local/go/bin:/usr/bin:...
HOME=/home/user
MXCLI_LOG=1
# 敏感变量（*_TOKEN, *_KEY, *_SECRET, *_PASSWORD, *_PASS）自动过滤，显示 [REDACTED]
```

实现：遍历 `os.Environ()`，对 key 进行大写匹配，命中敏感模式则值替换为 `[REDACTED]`。

#### project-meta.txt（仅当 -p 有效时）

```
MendixVersion: 11.6.6
MPRFormat: v2
ModuleCount: 5
DocumentCount: 142
MPRFileHash: sha256:abc123...   # 文件哈希，不含任何模型内容
```

数据来源：已有 `backend.Reader`，通过 `--project` flag 打开只读连接提取元信息。  
若未指定 `-p` 或打开失败，则跳过该条目（打印提示 `project-meta: skipped (no -p)`）。

#### error-stacks.txt

从 `~/.mxcli/logs/` 中最近 7 天的日志文件里提取 `"level":"ERROR"` 行，最多 20 条，  
格式为：

```
=== 2026-05-21T14:30:22Z ===
{"level":"ERROR","msg":"execute","error":"...","stmt_type":"..."}

=== 2026-05-21T09:12:01Z ===
...
```

按时间倒序排列。实现在现有 `collectRecentErrors()` 函数基础上扩展。

---

### 4. 数据流

```
用户执行命令
    │
    ├── main.go: 读取 --verbose flag → verboseLevel (0/1/2)
    │
    ├── diaglog.Init(version, mode, verboseLevel)
    │       ├── JSON handler → ~/.mxcli/logs/mxcli-YYYY-MM-DD.log  (始终)
    │       └── stderr handler → os.Stderr                          (verbose > 0)
    │
    ├── executor 执行 MDL 语句
    │       └── diaglog.Command() / Info() / Error()
    │               ├── 写入日志文件
    │               └── 若 verbose > 0: 写入 stderr
    │
    └── 用户执行 mxcli diag --bundle
            ├── 现有：打包 logs/ 目录
            ├── 新增：生成 env-dump.txt
            ├── 新增：生成 error-stacks.txt（扫描日志文件）
            └── 新增（有 -p 时）：生成 project-meta.txt
```

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `cmd/mxcli/main.go` | 修改 | 增加全局 `-v` PersistentFlag；传 verboseLevel 给 diaglog.Init |
| `mdl/diaglog/diaglog.go` | 修改 | Logger 增加 stderrLog、verbose 字段；Init 增加 verboseLevel 参数；各方法镜像输出 |
| `cmd/mxcli/diag.go` | 修改 | runDiagBundle() 增加 env-dump、error-stacks、project-meta 收集逻辑 |

---

## 测试策略

- `diaglog` 单元测试：验证 verboseLevel=0/1/2 时 stderr 的输出内容
- `diag --bundle` 集成测试：创建临时日志目录，调用 bundle，解压验证新条目存在
- 敏感变量过滤测试：注入含 `_TOKEN` 的 env var，验证 env-dump 中显示 `[REDACTED]`

---

## 不设计的内容（YAGNI）

- 不支持 `--log-level` 完整枚举（fatal/error/warn/info/debug/trace）—— 两级足够
- 不引入第三方日志库（保持 slog）
- bundle 不改为 zip 格式（tar.gz 足够，跨平台一致性）
- verbose 日志不写临时文件（方案 C 被明确拒绝）
