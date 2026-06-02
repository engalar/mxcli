# Issue 007: CE0066/CE0129/CE0156 一致性错误无法通过 MDL 修复

## 元数据

| 字段 | 值 |
|------|-----|
| Reporter | Miwa |
| 分类 | MxCli |
| 状态 | Open |
| 优先级 | 高 |
| 发现日期 | 2026-06-02 |

## 问题描述

以下 Mendix 一致性错误无法通过 MDL 脚本解决，必须手动在 Studio Pro GUI 中操作：

| 错误码 | 描述 | Studio Pro 操作 |
|--------|------|----------------|
| CE0066 | Entity access is out of date | Domain Model → "Update Security" 按钮 |
| CE0129 | Administrator password has not been set | Security settings 界面 |
| CE0156 | User role should have at least one System module role | System module role 设置 |

## 代码分析

`ALTER PROJECT SECURITY` 命令已实现（`mdl/executor/cmd_security_write_project_gen.go`），但仅能修改：
- 安全级别（Production / Prototype / Off）
- Demo 用户开关

**无法触发的操作：**
- CE0066：`Update Security`（重新计算 entity access rule）— 对应 `modelsdk/mpr/security_patch.go` 中有相关逻辑但无 MDL 命令暴露
- CE0129：设置管理员密码
- CE0156：为 User Role 分配 System 模块角色

## 期望行为（用户请求）

```sql
-- 触发 Update Security（修复 CE0066）
ALTER PROJECT SECURITY UPDATE;

-- 设置管理员密码（修复 CE0129）
ALTER PROJECT SECURITY SET admin password = '...';

-- 为 User Role 绑定 System 模块角色（修复 CE0156）
GRANT SYSTEM ROLE Administrator TO USER ROLE MyModule.Administrator;
```

## 修复难点

- **CE0066**：`security_patch.go` 中已有逻辑，需将其暴露为 MDL 命令（工作量中等）
- **CE0129**：密码存储涉及 Mendix 内部哈希算法，需要逆向确认存储格式
- **CE0156**：System 模块角色绑定的 BSON 结构需验证（参考 reflection data）

## 关联文件

- `mdl/executor/cmd_security_write_project_gen.go` — 现有 ALTER PROJECT SECURITY 实现
- `modelsdk/mpr/security_patch.go` — CE0066 相关补丁逻辑（已有，待暴露）
- `mdl/grammar/domains/MDLSecurity.g4` — 安全相关 grammar
- `mdl/executor/cmd_security_gen.go` — 安全命令执行入口
