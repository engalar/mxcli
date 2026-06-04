# mxcli local run — Deploy Target Optimization

## Background

`mxcli local build` 原来使用 `--target=portable-app-package`，MxBuild 总是产生一个 ZIP 文件，
然后 mxcli 再解压。这是一个不必要的往返开销。

## 验证过程（2026-06-04）

### 发现

当 Studio Pro 已安装时，可以使用 `--target=deploy` 代替：

- `--target=portable-app-package` → MxBuild 创建 ZIP → mxcli 解压 → 合计 2 次 IO
- `--target=deploy` → MxBuild 直接写入目录 → 0 ZIP → 无解压开销

### Deploy 目标的输出结构

`mxbuild --target=deploy` 输出到 `{project_dir}/deployment/`（固定路径，无法通过 `-o` 更改）：

```
deployment/
  model/         # 编译后的模型文件（model.mdp 等）
  web/           # 前端资源
  run/bin/       # 模块 JAR（用户 Java 代码）
  native/        # native 文件
  sass/          # SCSS 源文件
  data/          # 运行时数据（HSQLDB、上传文件等）
  tmp/           # 临时目录
```

**不包含**：`lib/runtime/`（runtime JARs）、`etc/`（配置模板）、`bin/start.bat`

### 启动方式

Runtime 直接从 Studio Pro 安装目录读取：

```
MX_INSTALL_PATH = C:\Program Files\Mendix\{version}
runtime launcher = {MX_INSTALL_PATH}\runtime\launcher\runtimelauncher.jar
```

启动命令：
```
java -DMX_LOG_LEVEL=INFO -Dfile.encoding=UTF-8 \
  -jar "{MX_INSTALL_PATH}\runtime\launcher\runtimelauncher.jar" \
  "{project_dir}/deployment" \
  {config_file}
```

### 必需的环境变量

| 变量 | 说明 |
|------|------|
| `MX_INSTALL_PATH` | Studio Pro 安装目录，如 `C:\Program Files\Mendix\11.6.6` |
| `M2EE_ADMIN_PASS` | M2EE 管理 API 密码（= admin-password） |
| `ADMIN_ADMINPASSWORD` | 同上（另一个读取路径） |
| `RUNTIME_ADMINUSER_PASSWORD` | MxAdmin 登录密码 |

### 必需的配置文件内容

必须包含 `logging` 部分（有至少一个订阅者），否则 launcher 不会自动启动 runtime：

```hocon
runtime.params {
  DatabaseType = HSQLDB
  DatabaseName = default
  ApplicationRootUrl = "http://localhost:8080/"
  HashAlgorithm = "BCRYPT:12"
  DTAPMode = D
  ScheduledEventExecution = NONE
  MyScheduledEvents = ""
  CACertificates = ""
  ClientCertificates = ""
  ClientCertificatePasswords = ""
}

runtime.params.MicroflowConstants {
  # 从项目常量读取默认值（必须提供，否则包含 @Constant 表达式的 microflow 会崩溃）
  "Module.ConstantName" = "default_value"
}

admin {
  port = 8090
  addresses = [ localhost ]
  adminPassword = ${?ADMIN_ADMINPASSWORD}
}

runtime {
  http {
    port = 8080
    addresses = [ "*" ]
  }
  adminUser.password = ${?RUNTIME_ADMINUSER_PASSWORD}
}

# 关键：没有 logging 订阅者时 launcher 仅启动 M2EE admin server 而不继续启动 runtime
logging = [
  {
    name = MySubscriber
    type = console
    autoSubscribe = INFO
    levels {}
  }
]
```

### 常量缺失导致崩溃

如果 microflow 中使用了 `@Module.ConstantName` 表达式，但配置文件中没有提供对应常量值，
runtime 会在模块初始化时崩溃：

```
Could not find value for constant 'HD.SLA_CRITICAL_HOURS'.
Input 'addHours([%CurrentDateTime%], @HD.SLA_CRITICAL_HOURS)' could not be parsed
```

mxcli 需要从 MPR 读取所有常量的默认值并注入配置文件。

### 验证结果

```
INFO - Workflow Engine: Initializing Workflow Engine...
INFO - Workflow Engine: Workflow Engine is initialized.
INFO - Core: Mendix Runtime successfully started, the application is now available.
```

`curl http://localhost:8080/` → HTTP 200 ✓

## 实现计划

### `mxcli local build` 变更

文件：`cmd/mxcli/docker/build.go`

- 新增 `LocalBuildOptions.UseDeployTarget bool`
- 当 `ResolveStudioProDir(version) != ""` 时自动切换为 deploy 模式
- deploy 模式：调用 `mxbuild --target=deploy`，输出到 `{project_dir}/deployment/`
- 跳过 `extractPADZip`、`flattenPADDir`、`generateDockerfile`、`injectRuntime` 步骤

### `mxcli local run` 变更

文件：`cmd/mxcli/docker/local.go`

- 新增 `isDeployLayout(dir string) bool` — 检测目录是否为 deploy 格式（有 `model/` 但没有 `bin/start.bat`）
- 新增 `StartLocalFromDeploy(opts LocalRunOptions)` — 从 deploy 目录启动
  1. 读取项目常量默认值（从 MPR 或从 `constants/defaults.conf` 等）
  2. 在临时目录生成配置文件
  3. 找到 Studio Pro runtime launcher
  4. 启动 java 进程

### 常量读取

`--target=deploy` 在 `deployment/model/config.json` 中已包含所有需要的信息：

```json
{
  "Configuration": {
    "DatabaseName": "default",
    "DatabaseType": "HSQLDB",
    "ApplicationRootUrl": "http://localhost:8080/",
    ...
  },
  "Constants": {
    "HD.SLA_HIGH_HOURS": "8",
    "HD.SLA_CRITICAL_HOURS": "2",
    "FeedbackModule.LocalStorageKey": "mxfeedback-form-data",
    ...
  },
  "AdminPassword": "1"
}
```

直接读取此文件生成 HOCON 配置，**无需读取 MPR**。

## HSQLDB 锁文件预检（同期实现）

文件：`cmd/mxcli/docker/local.go`

函数 `preflightLocal(padDir, stderr)` 已实现：
1. 检查端口 8090 是否被占用（= 旧实例还在运行）
2. 清理 `app/data/database/hsqldb/**/*.lck` 中的残留锁文件

对 deploy 目录，lck 路径为 `deployment/data/database/hsqldb/**/*.lck`。
