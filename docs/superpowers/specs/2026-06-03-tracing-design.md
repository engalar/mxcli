# 链路追踪设计

**日期**：2026-06-03  
**状态**：已批准，待实现  
**范围**：mxcli 工具本身 + Mendix 应用运行时，统一接入 Grafana Tempo

---

## 目标

- mxcli 每条命令（`exec`、`lint`、`export` 等）产生 OpenTelemetry span
- mxcli daemon 作为子进程，通过 Unix socket 协议继承 CLI 的 trace context，产生 child span
- Mendix 应用（Docker 模式 / 本地模式）通过内置 OTel Java Agent 产生微流调用链 span
- 所有 span 统一发送至 Grafana Tempo（OTLP HTTP）
- Tempo 生命周期不归 mxcli 管理：检测到 `OTEL_EXPORTER_OTLP_ENDPOINT` 已设则启用，否则静默 no-op

---

## 架构

```
┌─────────────────────────────────────────────────────┐
│  开发机                                              │
│                                                     │
│  mxcli CLI ──span──► OTLP exporter ─────────────┐  │
│       │                                          │  │
│       │ traceparent (Unix socket JSON req.Env)   │  │
│       ▼                                          │  │
│  mxcli daemon ──child span──► OTLP exporter ────┤  │
│                                                  │  │
│  ┌──── Docker 模式 ────────────────────────┐     │  │
│  │  Mendix container                      │     │  │
│  │  JAVA_TOOL_OPTIONS=..javaagent..       │     │  │
│  │  └► OTel agent ──span──────────────────┼─────┤  │
│  └────────────────────────────────────────┘     │  │
│                                                  │  │
│  ┌──── 本地模式 ───────────────────────────┐     │  │
│  │  bin/start (subprocess)                │     │  │
│  │  env JAVA_TOOL_OPTIONS=..javaagent..   │     │  │
│  │  └► OTel agent ──span──────────────────┼─────┤  │
│  └────────────────────────────────────────┘     │  │
│                                                  ▼  │
│                                        Grafana Tempo │
│                                        :4318         │
│                                        (OTLP HTTP)   │
└─────────────────────────────────────────────────────┘
```

---

## 配置模型

环境变量驱动，与 OTel 生态惯例一致，无需新配置文件。

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | （未设）| 设置后启用追踪，如 `http://localhost:4318` |
| `OTEL_SERVICE_NAME` | `mxcli` / `mxcli-daemon` | 服务名，daemon 自动使用 `mxcli-daemon` |
| `OTEL_TRACES_EXPORTER` | `otlp` | 可设 `none` 强制关闭 |

**启用逻辑**（CLI 和 daemon 相同）：

```
OTEL_EXPORTER_OTLP_ENDPOINT 已设？
  ├─ 是 → 启动 OTLP exporter，发 span
  └─ 否 → no-op tracer，零开销，代码路径完全相同
```

---

## 进程间追踪传播

### 现有协议

`internal/launcherproto.Request` 已有 `Env map[string]string` 字段，天然适合携带 W3C trace context。

### 传播流程

```
CLI 进程                              daemon 进程
────────────────────────────────────────────────────
1. PersistentPreRunE 创建 root span
   span = tracer.Start(ctx, "mxcli <command>")

2. 转发给 daemon 前：
   telemetry.InjectEnv(ctx, req.Env)
   → req.Env["traceparent"] = "00-<traceId>-<spanId>-01"
   → req.Env["tracestate"]  = ""

3. WriteMsg(conn, req) ──────────────► ReadMsg(conn, &req)

                                     4. parentCtx = telemetry.ExtractEnv(req.Env)
                                        childCtx, span = tracer.Start(
                                            parentCtx, "daemon:<command>")

                                     5. runCommand(childCtx, req.Argv, ...)

                                     6. span.End() → flush to Tempo

7. CLI span.End() → flush to Tempo
```

### 新包：`internal/telemetry`

| 函数 | 说明 |
|------|------|
| `Init(serviceName string) (shutdown func(), err error)` | 初始化 OTLP provider；`OTEL_EXPORTER_OTLP_ENDPOINT` 未设时返回 no-op |
| `InjectEnv(ctx context.Context, env map[string]string)` | 将 W3C `traceparent`/`tracestate` 写入 env map |
| `ExtractEnv(env map[string]string) context.Context` | 从 env map 还原 parent span context |

实现基于 `go.opentelemetry.io/otel/propagation`（W3C TraceContext propagator）。

---

## Mendix 应用侧：两种启动模式

### 共用函数

```go
// cmd/mxcli/docker/tracing.go（新文件）
// BuildOTelJVMArgs 返回启用 OTel agent 所需的 JVM 参数列表。
// 若 padDir 内找不到 agent jar，返回空切片（旧版 Mendix 静默跳过）。
func BuildOTelJVMArgs(padDir, endpoint, serviceName string) []string
```

agent jar 路径从 PAD 目录推导，不硬编码版本：

```
{padDir}/runtime/agents/opentelemetry-javaagent.jar
{padDir}/runtime/agents/mendix-opentelemetry-agent-extension.jar
```

生成的 JVM 参数示例：

```
-javaagent:{padDir}/runtime/agents/opentelemetry-javaagent.jar
-Dotel.javaagent.extensions={padDir}/runtime/agents/mendix-opentelemetry-agent-extension.jar
-Dotel.service.name=MyApp
-Dotel.exporter.otlp.traces.endpoint={endpoint}/v1/traces
-Dotel.exporter.otlp.traces.protocol=http/protobuf
```

### Docker 模式（`mxcli docker up`）

1. `docker compose up` 前检查 `OTEL_EXPORTER_OTLP_ENDPOINT`
2. 已设 → 将 `JAVA_TOOL_OPTIONS=<otel args>` 写入 `.docker/.env`（追加或更新）
3. 未设 → 不改 `.docker/.env`，compose 照常启动

`.docker/docker-compose.yml` 模板中 `command` 已有 `-J` 参数，JVM 自动读取 `JAVA_TOOL_OPTIONS`，**无需修改 compose 模板**。

### 本地模式（`mxcli local run`）

`StartLocal()` 构造 `exec.Cmd` 时条件注入：

```go
if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
    jvmArgs := docker.BuildOTelJVMArgs(opts.PadDir, endpoint, serviceName)
    if len(jvmArgs) > 0 {
        cmd.Env = append(os.Environ(),
            "JAVA_TOOL_OPTIONS="+strings.Join(jvmArgs, " "))
    }
}
```

---

## 代码变更清单

| 文件 | 类型 | 改动说明 |
|------|------|---------|
| `internal/telemetry/telemetry.go` | 新建 | `Init`、`InjectEnv`、`ExtractEnv` |
| `internal/telemetry/telemetry_test.go` | 新建 | 单元测试 |
| `cmd/mxcli/main.go` | 修改 | `PersistentPreRunE` 初始化 OTel，创建 root span |
| `cmd/mxcli/daemon_backend.go` | 修改 | 发送 request 前调用 `telemetry.InjectEnv` |
| `cmd/mxcli/daemon_server.go` | 修改 | `handleConn` 调用 `telemetry.ExtractEnv`，传 ctx 给 `runCommand` |
| `cmd/mxcli/docker/tracing.go` | 新建 | `BuildOTelJVMArgs()` |
| `cmd/mxcli/docker/tracing_test.go` | 新建 | 单元测试 |
| `cmd/mxcli/docker/local.go` | 修改 | `StartLocal()` 注入 `JAVA_TOOL_OPTIONS` |
| `cmd/mxcli/docker/runtime.go` | 修改 | `docker up` 前更新 `.docker/.env` |
| `go.mod` | 修改 | 添加 `go.opentelemetry.io/otel` 及 OTLP exporter 依赖 |

---

## 版本兼容性

| Mendix 版本 | 支持情况 |
|-------------|---------|
| < 10.17（估算）| agent jar 不存在，`BuildOTelJVMArgs` 返回空，静默跳过 |
| 10.17+ / 11.0+ | 微流 span 自动生成 |
| 11.5.0+ | 支持 `mendix.tracing.filter` 过滤规则 |
| 11.10.0+ | 支持 Java Action 自定义 span（`Core.tracing()` API） |

---

## Java Action 自定义 Span

Mendix 11.10.0 引入了 `Core.tracing()` API，允许在 Java Action 中创建自定义 span，将业务逻辑的关键步骤纳入链路追踪。

### 自动生命周期（`.run()`）

`run` 方法自动管理 span 的启动和关闭，设置状态并处理异常：

```java
Core.tracing()
  .createSpan("my span name")
  .withAttribute("attribute_key", "attribute value")
  .run(span -> {
    // the code here will be wrapped by the span
  });
```

### 手动生命周期（`.start()` / `.close()`）

当控制流程较复杂时（如分支、循环、异步回调），可以手动控制 span 生命周期：

```java
var span = Core.tracing()
  .createSpan("my span name")
  .withAttribute("attribute_key", "attribute value")
  .start();
try {
  // your code
  span.setStatus(Span.Status.OK);
} catch (Throwable exc) {
  span.setError(exc);
} finally {
  span.close();
}
```

### 最佳实践

- span name 使用**有意义的操作名称**，格式建议 `Module.Action_Description`，便于在 Tempo 中快速定位
- `withAttribute` 添加**结构化上下文**（如实体 ID、请求 URL、关键参数值），避免将大量数据塞入 span name
- 优先使用 `.run()` 避免忘记 `close()` 导致内存泄漏
- 仅在确实需要精细控制（如跨越多个 try-catch 块）时使用手动模式，且必须保证 `close()` 在 `finally` 中执行

### 与 MDL TracingConfiguration 的关系

| 配置 | 作用 |
|------|------|
| `TracingEnabled = true` | 开启 Mendix 运行时的 OTel agent |
| `TracingEndpoint = "http://..."` | 指定 OTLP HTTP 接收端 |
| `TracingServiceName = "MyApp"` | 指定服务名 |
| Java Action 中 `Core.tracing()` | 在上述基础上创建**自定义业务 span** |

三者是**叠加**关系：启用 TracingConfiguration 后，微流调用链 span 自动生成；Java Action 中的 `Core.tracing()` 在此基础上添加额外业务 span。若无 `TracingConfiguration`，`Core.tracing()` 仍可工作（使用进程级 OTel agent 配置）。

---

## 不在此次范围内

- mxcli 命令与 Mendix 运行时 span 的 trace context 跨进程关联（HTTP `traceparent` header 注入 M2EE/OQL 请求）
- Tempo 容器生命周期管理
- `mendix.tracing.filter` 配置的 MDL 语法封装
- metrics / logs 的 OTel 接入（仅做 traces）
