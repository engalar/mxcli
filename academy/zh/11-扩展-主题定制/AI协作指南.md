# 模块 11：AI 协作指南 — 主题定制

## 方法：CSS 变量覆盖（无需 Studio Pro）

Atlas UI 使用 CSS 自定义属性（CSS Variables）控制颜色、圆角等视觉参数。
通过在自定义 CSS 文件中覆盖这些变量，可以不触碰 Atlas 源码实现主题定制。

## 三种实现方式

### 方式 A：CSS 变量覆盖（推荐，最简单）

1. 在 Studio Pro 中：App → Styling → 找到自定义 CSS 位置（通常是 `theme/web/custom-variables.css`）
2. 把 `theme/helpdesk-theme.css` 的内容粘贴进去
3. 重新运行项目

### 方式 B：SCSS 变量（需要 Atlas SCSS 工具链）

```bash
# 在 Studio Pro 中 App → Styling → 导出 Atlas SCSS 源码
# 修改 _variables.scss 中的颜色变量
# 重新编译：npm run build（Studio Pro 会自动触发）
```

### 方式 C：与 Claude 协作生成 CSS

```
帮我为 Mendix Atlas UI 生成主题覆盖 CSS：
- 主色：#1565C0（品牌蓝）
- 主色悬停：#0D47A1
- 按钮圆角：4px
- 基于 Atlas UI 的 CSS 变量命名规范
```

## 放置位置

| Studio Pro 版本 | 文件位置 |
|----------------|---------|
| Mendix 9.x     | `theme/web/custom-variables.css` |
| Mendix 10/11.x | `theme/web/main.css`（在文件末尾追加） |

## 验证

无需 mx check——CSS 修改不影响 MPR 结构。
在浏览器中运行应用，检查按钮颜色和圆角是否正确。
