# MEMV Phase 2.5 — JIT 语义验证设计规格
日期：2026-05-18
仓库：/mnt/data_sdd/gh/mxcli-wt-02（feature/expression-checker 分支）

## 目标

为 `mxcli expr` 子命令组增加语义验证层（SEM 规则），在现有纯语法验证（Phase 2）之上，利用 MPR 文件中的完整项目元数据，检测枚举值引用错误、常量引用错误、实体属性路径错误、XPath 实体路径错误。

## 两种模式

### Daemon 模式（默认）
```
mxcli expr validate -p project.mpr
```
- 自动启动后台 daemon 进程（首次调用，冷启动 ~3s）
- Daemon 从 MPR 构建 JIT 元数据索引（meta.Index）并保持在内存中
- 后续调用通过 Unix socket JSON-RPC 连接 daemon（热路径 ~10ms）
- 5 分钟空闲后自动退出，MPR 文件变化时自动重建索引
- 输出：语法规则（SYN）+ 语义规则（SEM）全量结果

### No-Daemon 模式（显式禁用）
```
mxcli expr validate -p project.mpr --no-daemon
MXCLI_NO_DAEMON=1 mxcli expr validate -p project.mpr
```
- **完全禁用语义层**，不加载 MPR，不启动后台进程
- 仅执行语法验证（SYN-01~03、E006 等已有规则）
- 恒定执行时间 ~100ms，适合 CI / 容器 / 快速扫描场景
- No-Daemon 模式下 mprcontents/ 从 dirname(mpr)/mprcontents/ 推导，用户只需传 -p

## V1 / V2 MPR 格式支持

| 格式 | 识别方式 | 表达式来源 | 元数据来源 |
|------|----------|-----------|-----------|
| V2（默认） | `dirname(mpr)/mprcontents/` 目录存在 | mprcontents/*.mxunit BSON | mxcli engine（含继承链展开） |
| V1（经典） | mprcontents/ 不存在，MPR 为 SQLite | mxcli engine（SQLite 查询） | mxcli engine |

`mpr.Open(path)` 自动检测格式，上层代码无需感知差异。

## 元数据索引（meta.Index）

从 MPR 通过 mxcli engine（modelsdk）构建，一次性加载，常驻 daemon 内存。

| 数据 | 来源 | 说明 |
|------|------|------|
| EntityAttrs：`map[entityQN]map[attrName]TypeKind` | engine | 含继承链展开、System 模块实体 |
| EnumValues：`map[enumQN][]string` | engine | 完整枚举值名称列表（纯 BSON 扫描只有 ValueCount） |
| Constants：`map[@Module.Name]TypeKind` | engine | 项目所有常量及类型 |
| Associations：`map[fromQN][]AssocInfo` | engine | 跨模块关联，含正确 UUID→QN 解析 |

meta.Index 实现 `exprcheck.CatalogReader` 接口（已定义于 `mdl/exprcheck/interfaces.go`），传入 `exprcheck.Context.Catalog`，自动激活 exprcheck 的 E001/E002 等语义检查。

## 语义规则覆盖（Phase 2.5）

| 规则 | 描述 | 实现方式 |
|------|------|---------|
| SEM-02 | 属性路径 `$Var/Attr` 中属性不存在 | CatalogReader.AttributeKind() 返回 (_, false) |
| SEM-04 | 枚举值 `Module.Enum.Value` 不存在 | CatalogReader.EnumCases() 返回值列表，检查 Value 在其中 |
| SEM-05 | 常量引用 `@Module.Name` 不存在 | meta.Index.Constants 查找 |
| SEM-07 | XPath 实体路径无效 | meta.Index.EntityAttrs 查找实体 QN |

Phase 3 规则（暂不实现）：SEM-01（变量未声明，需 MDL AST）、SEM-03（类型不符，需作用域推导）。

## Daemon 生命周期

```
未启动 → [首次调用触发 fork] → 加载中（Open+BuildMetaIndex）
       → 就绪（监听 socket）
       → [收到请求] → 处理 → 重置空闲计时器 → 就绪
       → [MPR 文件变化（fsnotify）] → 重新加载 → 就绪
       → [5min 空闲] → 自动退出（删除 socket 文件）
```

### Socket 文件命名
```
~/.mxcli/expr-daemon/<sha256(abspath(mpr))[:8]>.sock
```
每个 MPR 文件对应独立 daemon，支持多项目并行。

### IPC 协议：Unix Socket + JSON-RPC
```json
// 请求
{"method": "expr.Validate", "params": {"mprPath": "...", "filter": "...", "severity": "..."}}

// 响应
{"indexAge": "2m34s", "mprMtime": "...", "results": [...]}
```

## 配置项

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `--no-daemon` | false | 禁用 daemon，仅语法验证 |
| `MXCLI_NO_DAEMON=1` | unset | 同上（环境变量方式） |
| `MXCLI_DAEMON_TIMEOUT` | `5m` | 空闲超时时长 |
| `MXCLI_DAEMON_DIR` | `~/.mxcli/expr-daemon/` | socket 文件目录 |

新增子命令：
- `mxcli expr daemon status` — 列出运行中的 daemon 及状态
- `mxcli expr daemon stop -p project.mpr` — 终止指定 daemon

## 新增文件结构

```
internal/expr/
├── mpr/
│   ├── reader.go          MPRReader interface + Open() 自动检测
│   ├── v1_reader.go       V1 SQLite 实现
│   └── v2_reader.go       V2 mprcontents/ BSON 实现（复用 scan 逻辑）
├── meta/
│   ├── index.go           Index struct + BuildFromReader()
│   ├── entity.go          EntityAttrs 构建（含继承展开）
│   ├── enum.go            EnumValues 构建
│   ├── const.go           Constants 构建
│   └── catalog_reader.go  实现 exprcheck.CatalogReader
└── daemon/
    ├── daemon.go           Daemon struct，Serve()/Stop()/Status()
    ├── client.go           DaemonClient，Connect()/Call()/StartIfNeeded()
    ├── proto.go            JSON-RPC 类型定义
    └── socket.go           SocketPath() / IsAlive() 工具函数

cmd/mxcli/
├── cmd_expr.go            修改：--no-daemon flag，-p 路径推导，daemon client 接入
└── cmd_expr_daemon.go     新增：expr daemon start/stop/status 子命令

internal/expr/
├── parse/parse.go         修改：daemon 模式下 ctx.Catalog = idx
└── validate/validate.go   修改：SEM 规则分派（daemon 有 / no-daemon 无）
```

## 性能目标

| 场景 | 目标延迟 | 条件 |
|------|---------|------|
| No-Daemon 模式 | < 200ms | 任何时候 |
| Daemon 热路径 | < 30ms | daemon 已运行 |
| Daemon 冷启动 | < 10s | 首次或 MPR 变更后 |
| Daemon 空闲退出 | 5min（可配置） | 无请求时 |

## 规格自检

- **无占位符**：所有接口、文件路径、数据结构均已定义 ✓
- **内部一致性**：V1/V2 读取方式、CatalogReader 绑定、daemon 模式分发路径一致 ✓
- **范围合理**：Phase 2.5 限定 SEM-02/04/05/07；SEM-01/03 明确为 Phase 3 ✓
- **歧义消除**：No-Daemon = 禁用语义层，不做进程内 MPR 加载 ✓
