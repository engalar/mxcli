# AI 辅助开发学院 / AI-Assisted Development Academy

通过构建一个完整的 IT 支持 Helpdesk 应用，学习如何使用 Claude Code + mxcli 实现 AI 辅助 Mendix 开发。

Learn AI-assisted Mendix development by building a complete IT helpdesk application using Claude Code + mxcli.

---

## 核心学习循环 / Core Learning Loop

```
业务需求文档            AI 协作                  验证
(业务语言)    →    Claude Code + mxcli    →    mx check
"需要工单系统"     /mendix:create-entity       0 错误 = 完成
                       ↓
               参考实现（标准答案）
               参考实现/*.mdl
```

## 课程语言 / Languages

| 语言 | 状态 |
|------|------|
| 🇨🇳 中文（zh/） | ✅ 完整实现 |
| 🇬🇧 English（en/） | 🚧 Coming soon |

## 中文课程目录 / Chinese Curriculum

| 模块 | 主题 | 先决条件 |
|------|------|---------|
| [00 入门准备](zh/00-入门准备/) | 环境搭建 & 第一次 exec | 无 |
| [01 领域建模](zh/01-领域建模/) | 实体、枚举、关联 | 00 |
| [02 微流业务逻辑](zh/02-微流业务逻辑/) | 工单状态机 & 业务规则 | 01 |
| [03 纳流与客户端](zh/03-纳流与客户端/) | 客户端计算 & 快速操作 | 01 |
| [04 页面与UI](zh/04-页面与UI/) | Atlas 布局 & 美观界面 | 01–03 |
| [05 安全与权限](zh/05-安全与权限/) | 角色 & 行级过滤 | 01–04 |
| [Capstone](zh/capstone-helpdesk/) | 完整应用交付 | 所有模块 |

## 技术要求

- mxcli（最新版）：`mxcli --version`
- Claude Code：`claude --version`
- Mendix 项目文件（.mpr）：用于 exec 和 mx check 验证
- mx check 工具：`mxcli setup mxbuild -p app.mpr`（自动下载）
