package executor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
)

// execShowThemeVariables 通过 mxgraph 查询主题变量。
// 语法: SHOW THEME VARIABLES [LIKE pattern] [ATLAS DEFAULTS]
func execShowThemeVariables(ctx *ExecContext, s *ast.ShowThemeVariablesStmt) error {
	if ctx.Graph == nil {
		return mdlerrors.NewValidationf("graph not built — run REFRESH CATALOG first")
	}

	filter := graphcatalog.ThemeVarFilter{
		ActiveOnly: !s.ShowDefaults,
		Like:       s.LikePattern,
	}
	if s.ShowDefaults {
		filter.Source = "atlas-core-default"
	}

	vars := ctx.Graph.ThemeVariables(s.InModule, filter)
	if len(vars) == 0 {
		if s.LikePattern != "" {
			fmt.Fprintf(ctx.Output, "No theme variables matching '%s'\n", s.LikePattern)
		} else {
			fmt.Fprintln(ctx.Output, "No theme variables found")
		}
		return nil
	}

	// 按分类分组输出
	byCategory := make(map[string][]graphcatalog.ThemeVariableNode)
	for _, v := range vars {
		cat := v.Category
		if cat == "" {
			cat = "other"
		}
		byCategory[cat] = append(byCategory[cat], v)
	}

	for _, cat := range sortedKeys(byCategory) {
		items := byCategory[cat]
		fmt.Fprintf(ctx.Output, "%s (%d):\n", cat, len(items))
		for _, v := range items {
			source := v.Source
			if v.Module != "" {
				source = v.Module
			}
			mark := ""
			if !v.IsActive {
				mark = " (commented)"
			}
			if v.IsDefault {
				mark = " !default"
			}
			fmt.Fprintf(ctx.Output, "  %-30s = %-20s [%s]%s\n", v.Name, v.Value, source, mark)
		}
		fmt.Fprintln(ctx.Output)
	}
	return nil
}

// execShowDesignPropertiesFromGraph 通过 mxgraph 查询设计属性。
func execShowDesignPropertiesFromGraph(ctx *ExecContext, widgetType string) error {
	if ctx.Graph == nil {
		return mdlerrors.NewValidationf("graph not built — run REFRESH CATALOG first")
	}

	props := ctx.Graph.DesignProperties(widgetType)
	if len(props) == 0 {
		fmt.Fprintf(ctx.Output, "No design properties found for %s\n", widgetType)
		return nil
	}

	fmt.Fprintf(ctx.Output, "Design Properties for %s:\n", widgetType)
	for _, p := range props {
		fmt.Fprintf(ctx.Output, "  %-30s %s", p.Name, p.Type)
		if p.Category != "" {
			fmt.Fprintf(ctx.Output, "  [%s]", p.Category)
		}
		fmt.Fprintln(ctx.Output)
		if len(p.Options) > 0 {
			fmt.Fprintf(ctx.Output, "    Options: %s\n", strings.Join(p.Options, ", "))
		}
		if len(p.ReferencedVars) > 0 {
			fmt.Fprintf(ctx.Output, "    References: %s\n", strings.Join(p.ReferencedVars, ", "))
		}
	}
	return nil
}

func sortedKeys(m map[string][]graphcatalog.ThemeVariableNode) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
