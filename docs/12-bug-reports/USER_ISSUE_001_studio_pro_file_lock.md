# Issue 001: Studio Pro 必须关闭才能使用 mxcli

## 元数据

| 字段 | 值 |
|------|-----|
| Reporter | Miwa |
| 分类 | MxCli |
| 状态 | Open |
| 优先级 | 需架构决策 |
| 发现日期 | 2026-06-02 |

## 问题描述

使用 mxcli 时必须关闭 Studio Pro，导致开发-验证循环非常繁琐。用户希望能在 Studio Pro 保持打开的同时使用 mxcli，或从 Studio Pro 的 chat 界面调用 mxcli。

## 根因分析

这是操作系统级别的文件锁竞争，无法通过 mxcli 单方面绕过：

- **MPR v1**：`.mpr` 是 SQLite 文件，Studio Pro 持有排他写锁
- **MPR v2**：`mprcontents/` 目录中的各文档文件被 Studio Pro 独占打开

当 mxcli 尝试写入时，SQLite/OS 会返回 SQLITE_BUSY 或文件锁错误。

## 用户影响

每次 mxcli exec 后需要：关闭 Studio Pro → 执行 → 重新打开 Studio Pro → 验证。循环成本高。

## 可能的解决方向

1. **检测锁冲突并给出明确提示**（短期）：当检测到文件被锁定时，输出 "Studio Pro 正在使用此项目，请先关闭 Studio Pro" 而非底层 SQLite 错误
2. **只读模式下共存**（中期）：mxcli 读操作（DESCRIBE/SHOW）在 Studio Pro 打开时仍可工作
3. **Studio Pro 插件集成**（长期）：需要 Mendix 官方配合，超出 mxcli 单方面能力范围

## 关联文件

- `modelsdk/mpr/` — MPR 读写入口
- `modelsdk/mpr/writer.go` — 写事务实现
