# mxcli local run — Deploy Target Optimization

## Background

`mxcli local build` 原来使用 `--target=portable-app-package`，MxBuild 总是产生一个 ZIP 文件，
然后 mxcli 再解压。这是一个不必要的往返开销。

## 优化方案

当 Studio Pro **同版本**已安装时，使用 `--target=deploy` 代替：

- `--target=portable-app-package` → MxBuild 创建 ZIP → mxcli 解压 → 合计 2 次 IO
- `--target=deploy` → MxBuild 直接写入目录 → 0 ZIP → 无解压开销

### 版本匹配要求（关键）

必须使用与项目 Mendix 版本**完全一致**的 Studio Pro + runtime。
`mxcli local build` 通过 `ResolveStudioProDir(pv.ProductVersion)` 确保版本匹配。
`mxcli local run` 通过读取 `deployment/model/metadata.json` 的 `RuntimeVersion` 字段
来查找正确版本的 runtime，避免版本错配导致的运行时错误。

## Deploy 目标的输出结构

`mxbuild --target=deploy` 输出到 `{project_dir}/deployment/`（固定路径，无法通过 `-o` 更改）：

```
deployment/
  model/
    config.json    # DB 配置 + 常量默认值（mxcli 用于生成 HOCON 配置）
    metadata.json  # 包含 RuntimeVersion、ModelVersion 等（mxcli 用于匹配 runtime 版本）
    model.mdp      # 编译后的模型
    ...
  web/           # 前端资源
  run/bin/       # 模块 JAR（用户 Java 代码）
  native/        # native 文件
  sass/          # SCSS 源文件
  data/          # 运行时数据（HSQLDB、上传文件等）
  tmp/           # 临时目录
```

**不包含**：`lib/runtime/`（runtime JARs）、`etc/`（配置模板）、`bin/start.bat`

## 启动方式

Runtime 从 Studio Pro 安装目录读取，版本由 `metadata.json` 决定：

```
MX_INSTALL_PATH = C:\Program Files\Mendix\{RuntimeVersion}
runtime launcher = {MX_INSTALL_PATH}\runtime\launcher\runtimelauncher.jar
```

启动命令：
```
java -DMX_LOG_LEVEL=INFO -Dfile.encoding=UTF-8 \
  -jar "{MX_INSTALL_PATH}\runtime\launcher\runtimelauncher.jar" \
  "{project_dir}/deployment" \
  {config_file}
```

## 环境变量

| 变量 | 说明 |
|------|------|
| `MX_INSTALL_PATH` | 覆盖自动检测，手动指定 Studio Pro 路径 |
| `M2EE_ADMIN_PASS` | M2EE 管理 API 密码 |
| `ADMIN_ADMINPASSWORD` | 同上（另一个读取路径） |
| `RUNTIME_ADMINUSER_PASSWORD` | MxAdmin 登录密码 |

## config.json 的作用

`deployment/model/config.json` 包含所有需要的配置：

```json
{
  "Configuration": {
    "DatabaseName": "default",
    "DatabaseType": "HSQLDB",
    "ApplicationRootUrl": "http://localhost:8080/",
    "ScheduledEventExecution": "None"
  },
  "Constants": {
    "HD.SLA_HIGH_HOURS": "8",
    "HD.SLA_CRITICAL_HOURS": "2"
  },
  "AdminPassword": "1"
}
```

mxcli 读取此文件生成临时 HOCON 配置，**无需读取 MPR**。

## metadata.json 的作用

`deployment/model/metadata.json` 提供运行时版本信息：

```json
{
  "RuntimeVersion": "11.6.6",
  "ModelVersion": "unversioned",
  "JavaVersion": 21
}
```

`mxcli local run` 读取 `RuntimeVersion`，用于：
1. 精确查找对应版本的 Studio Pro runtime（`resolveMxInstallPathForVersion`）
2. 版本不匹配时报清晰错误，避免用错误版本 runtime 加载模型

## 生成的 HOCON 配置关键点

**必须包含 `logging` 部分**（有至少一个订阅者），否则 launcher 仅启动 M2EE admin server
而不自动启动 runtime：

```hocon
logging = [
  {
    name = console
    type = console
    autoSubscribe = INFO
    levels {}
  }
]
```

## HSQLDB 预检

函数 `preflightLocal` 在启动前：
1. 检查端口 8090 是否被占用（= 旧实例还在运行，返回清晰错误）
2. 自动清理 `data/database/hsqldb/**/*.lck` 残留锁文件（避免进程崩溃后重启失败）

## 已知限制

### 首次启动 demo 用户警告

首次启动（空数据库）时出现：
```
WARNING - ModelStore: Failed to synchronize demo users
NullPointerException: ... because "language" is null
```

原因：Mendix 在语言实体初始化前尝试同步 demo 用户密码。第二次启动（语言数据已在 DB）后消失。
这是 Mendix runtime 的已知限制，不影响启动。

## 实现文件

| 文件 | 变更 |
|------|------|
| `cmd/mxcli/docker/build.go` | `ResolveStudioProDir(version) != ""` 时自动用 `--target=deploy` |
| `cmd/mxcli/docker/local.go` | `isDeployLayout`、`startFromDeployLayout`、`resolveMxInstallPathForVersion`、`preflightLocal` |
| `cmd/mxcli-local/cmd_run.go` | 自动检测 deploy layout 优先于 PAD layout |
