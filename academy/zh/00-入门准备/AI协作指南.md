# 模块 00：AI 协作指南 — 入门准备

## 你需要准备什么

### 1. 安装 mxcli

**Windows（PowerShell，推荐）：**

```powershell
irm https://github.com/engalar/mxcli/raw/refs/heads/dev/install.ps1 | iex
```

**Windows（Git Bash）/ Linux / Mac：**

```bash
curl -fsSL https://github.com/engalar/mxcli/raw/refs/heads/dev/install.sh | bash
```

安装完成后验证：

```bash
mxcli --version
```

### 2. 安装 Claude Code

前往 [claude.ai/code](https://claude.ai/code) 下载桌面版，或通过 npm 安装 CLI：

```bash
npm install -g @anthropic-ai/claude-code
claude --version
```

### 3. 准备一个 Mendix 项目

**方式 A：新建项目（推荐）**

```bash
# 自动下载 Mendix 11.6.6 并创建空项目
mxcli new MyHelpdesk --version 11.6.6
cd MyHelpdesk
```

**方式 B：使用已有项目**

```bash
# 确认项目可以打开
mxcli -p MyProject.mpr -c "show structure"
```

### 4. 准备 mx check 工具

mxcli 会自动按以下顺序查找 `mx`，**无需手动配置**：

| 平台 | 自动查找路径（按优先级）|
|------|----------------------|
| Windows | `C:\Program Files\Mendix\{version}\modeler\mx.exe`（Studio Pro 安装目录） |
| Windows | `~/.mxcli/mxbuild/{version}/modeler/mx.exe`（mxbuild 缓存） |
| Linux/Mac | `~/.mxcli/mxbuild/{version}/modeler/mx` |

**如果你已安装 Mendix Studio Pro，mx 已经可用，跳过此步。**

如果没有安装 Studio Pro，用 mxcli 自动下载 mxbuild：

```bash
mxcli setup mxbuild -p MyProject.mpr
```

---

## 你的第一次 AI 协作

打开 Claude Code（在项目目录内）：

```bash
claude
```

试着输入：

```
用 MDL 为我创建一个简单的模块 HD，包含一个 Customer 实体，有 Name 和 Email 两个属性
```

Claude 会生成 MDL 代码。把生成的代码保存为 `test.mdl`，然后：

```bash
# 语法检查
mxcli check test.mdl

# 执行到项目
mxcli exec test.mdl -p MyProject.mpr

# Mendix 平台验证（mxcli 自动查找 mx，无需手动指定路径）
mxcli docker check -p MyProject.mpr
```

> **Windows 用户**：如果已安装 Studio Pro，也可以直接调用：
> ```
> "C:\Program Files\Mendix\11.6.6\modeler\mx.exe" check MyProject.mpr
> ```

**0 错误 = 成功！**

---

## 理解三步验证循环

```
1. mxcli check file.mdl              → 语法是否正确？
2. mxcli exec file.mdl -p app.mpr    → 能否写入项目？
3. mxcli docker check -p app.mpr     → Mendix 平台是否接受？
   （Windows Studio Pro 用户也可直接：mx.exe check app.mpr）
```

每写完一段 MDL，都要跑这三步。这是本课程贯穿始终的验证节奏。

---

## 参考实现

如果遇到困难，查看 `参考实现/hello-world.mdl`。

**规则：先自己尝试，实在卡住再看参考实现。**
