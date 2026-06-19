// internal/fkg/concepts/theme.go
package concepts

import (
	"context"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

func init() { Register(&ThemeAdapter{}) }

type ThemeAdapter struct{}

func (a *ThemeAdapter) Name() string { return "fkg:theme" }
func (a *ThemeAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{LabelConcept, LabelImplDetail, LabelPattern, LabelSkill},
	}
}
func (a *ThemeAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *ThemeAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	return sink.Emit([]mxgraph.Event{
		// ── Concept ─────────────────────────────────────────────────────────
		conceptNode("theme", "Theme (Atlas 3 CSS Custom Properties)",
			"Mendix Atlas 3 使用 CSS Custom Properties 控制全局样式。通过在 :root 中重定义 --brand-primary 等变量覆盖默认主题，无需修改 Atlas 源码。"),

		// ── Implementation details ──────────────────────────────────────────
		implDetailNode("theme.css-custom-properties",
			":root { --brand-primary } 覆盖方式",
			"Atlas 3 所有颜色通过 :root { --brand-primary: #0891B2; } 控制。编译时生成六个色阶 (50/100/200/300/400/500/600/700/800/900) 和 --brand-primary-hover。在文件末尾定义 :root 块，CSS 级联自动取最后声明。"),
		implDetailNode("theme.variable-chain",
			"CSS 变量引用链",
			"Atlas 组件不直接使用 --brand-primary，而是通过链式引用：--btn-primary-bg: var(--brand-primary-600)。按钮实际背景色 = var(--btn-primary-bg)。只需重定义 --brand-primary-600 即可改变按钮色。"),
		implDetailNode("theme.custom-variables-entry",
			"custom-variables.scss 入口",
			"theme/web/custom-variables.scss 被 main.scss 自动 @import。在这里用 SCSS 变量 ($brand-primary) 覆盖 Atlas 默认值，会被 Atlas 的颜色生成系统处理为 CSS 色阶。"),
		implDetailNode("theme.import-partial",
			"SCSS partial @import 方案",
			"创建 _brand-theme.scss 后用 @import 'brand-theme' 追加到 main.scss 末尾。SCSS partial 可独立维护，不影响 Atlas 原始文件。"),
		implDetailNode("theme.font-import",
			"Google Fonts 导入",
			"在 SCSS partial 中 @import url(...) 加载 Google Fonts，然后通过 body { font-family: ... } 设置全局字体。CSS 变量 --font-family-base 需要同时设置。"),
		implDetailNode("theme.not-scss-bottom",
			"坑：SCSS 变量在底部无效",
			"main.scss 顶部的 @import 'theme-dark' 已经编译了 Atlas 样式。底部追加的 $brand-primary: #0891B2 不影响已编译的 Atlas 选择器。必须用 CSS 变量 (:root) 或 SCSS partial @import。"),
		implDetailNode("theme.not-important-conflict",
			"坑：!important 与 CSS 变量链冲突",
			"Atlas 使用 --btn-primary-bg: var(--brand-primary-600) 链式引用。用 !important 直接覆盖 .btn-primary { background: #0891B2 !important; } 虽能生效，但会破坏变量链一致性，且 hover/active 状态仍需单独处理。正确方式：重定义 --brand-primary-600。"),
		implDetailNode("theme.order-module11",
			"坑：模块 11 SCSS 追加顺序",
			"academy-11-theme 的 helpdesk-theme.scss 和品牌 SCSS 都追加到 main.scss 末尾。如果品牌在模块 11 之前，模块 11 的 $brand-primary: #1565C0 会覆盖品牌色。必须保证模块 11 先追加、品牌后追加。"),
		implDetailNode("theme.not-scss-vars-only",
			"坑：SCSS 变量不够 — 还需要 CSS 变量",
			"Atlas 3 在编译时将 SCSS $brand-primary 转为 CSS --brand-primary-600 等。只设 SCSS 变量不够，还需要在 :root 中同时设 CSS 变量以便运行时覆盖。最佳实践：SCSS partial 中同时定义 $var 和 :root { --var }。"),
		implDetailNode("theme.color-variants",
			"色阶生成规则",
			"Atlas 从 $brand-primary 自动生成 10 个色阶 (50/100/…/900)。--brand-primary-500 = 主色，--brand-primary-600 = 按钮色，--brand-primary-700 = hover 色。要自定义按钮色需直接设 --brand-primary-600: #067394。"),
		implDetailNode("theme.bg-color",
			"背景色覆盖",
			"通过 :root { --bg-color: #ECFEFF; --bg-color-secondary: #FFFFFF; } 控制页面背景色。--bg-color 对应主背景，--bg-color-secondary 对应卡片/面板背景。"),
		implDetailNode("theme.validation",
			"验证方式",
			"1) build 后检查 theme.compiled.css 中对应变量是否出现在文件末尾。2) 浏览器 DevTools → Computed → 查看 --brand-primary 值。3) 运行时 Ctrl+F5 强刷清除 CSS 缓存。"),

		// ── Patterns ────────────────────────────────────────────────────────
		patternNode("atlas-theme-override", "Atlas 3 Theme Override",
			"在 SCSS partial 的 :root 中重定义 --brand-primary 全色阶，通过 CSS 级联覆盖全站样式。无需 !important，无需修改 Atlas 源码。"),
		patternNode("theme-scss-partial", "SCSS Partial 主题分离",
			"将品牌主题放在独立 _brand-theme.scss 文件中，通过 @import 'brand-theme' 引入。与 Atlas 默认文件解耦，升级 Atlas 时不受影响。"),
		patternNode("theme-css-variable-chain", "CSS 变量链调试",
			"当主题不生效时，沿变量链向上排查：.btn-primary → --btn-primary-bg → --brand-primary-600 → :root { --brand-primary }。断在哪一层就是哪一层值不对。"),

		// ── Skills ──────────────────────────────────────────────────────────
		skillNode("theme-branding", "Brand a Mendix Atlas 3 project: create _brand-theme.scss, set :root CSS variables, rebuild"),
		skillNode("theme-debug", "Debug Atlas 3 theme issues: inspect CSS variable chain, check append order, verify compiled CSS"),

		// ── Edges ───────────────────────────────────────────────────────────
		edge("theme", "detail:theme.css-custom-properties", HasSyntax),
		edge("theme", "detail:theme.variable-chain", HasSyntax),
		edge("theme", "detail:theme.custom-variables-entry", HasSyntax),
		edge("theme", "detail:theme.import-partial", HasSyntax),
		edge("theme", "detail:theme.font-import", HasSyntax),
		edge("theme", "detail:theme.not-scss-bottom", HasSyntax),
		edge("theme", "detail:theme.not-important-conflict", HasSyntax),
		edge("theme", "detail:theme.order-module11", HasSyntax),
		edge("theme", "detail:theme.not-scss-vars-only", HasSyntax),
		edge("theme", "detail:theme.color-variants", HasSyntax),
		edge("theme", "detail:theme.bg-color", HasSyntax),
		edge("theme", "detail:theme.validation", HasSyntax),
		edge("theme", "pattern:atlas-theme-override", HasPattern),
		edge("theme", "pattern:theme-scss-partial", HasPattern),
		edge("theme", "pattern:theme-css-variable-chain", HasPattern),
		edge("theme", "skill:theme-branding", HasSkill),
		edge("theme", "skill:theme-debug", HasSkill),
		edge("theme", "page", RelatedTo),
		edge("theme", "widget", RelatedTo),
		edge("theme", "integration", RelatedTo),
	})
}
