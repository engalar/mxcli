package executor

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
)

// execShowThemeVariables 通过 mxgraph 查询主题变量。
// 语法: SHOW THEME VARIABLES [LIKE pattern] [ATLAS DEFAULTS]

// execShowThemeVariablesFn is the HandlerDeps version of execShowThemeVariables.
func execShowThemeVariablesFn(ctx context.Context, s *ast.ShowThemeVariablesStmt, deps *HandlerDeps) error {
	if deps.Graph == nil {
		return mdlerrors.NewValidationf("graph not built — run REFRESH CATALOG first")
	}

	filter := graphcatalog.ThemeVarFilter{
		ActiveOnly: !s.ShowDefaults,
		Like:       s.LikePattern,
	}
	if s.ShowDefaults {
		filter.Source = "atlas-core-default"
	}

	vars := deps.Graph.ThemeVariables(s.InModule, filter)
	if len(vars) == 0 {
		if s.LikePattern != "" {
			fmt.Fprintf(deps.Output, "No theme variables matching '%s'\n", s.LikePattern)
		} else {
			fmt.Fprintln(deps.Output, "No theme variables found")
		}
		return nil
	}

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
		fmt.Fprintf(deps.Output, "%s (%d):\n", cat, len(items))
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
			fmt.Fprintf(deps.Output, "  %-30s = %-20s [%s]%s\n", v.Name, v.Value, source, mark)
		}
		fmt.Fprintln(deps.Output)
	}
	return nil
}

// execShowDesignPropertiesFromGraph 通过 mxgraph 查询设计属性。
func execShowDesignPropertiesFromGraph(ctx *ExecContext, widgetType string) error {
	deps := execContextToDeps(ctx)
	return execShowDesignPropertiesFromGraphFn(ctx, widgetType, deps)
}

// execShowDesignPropertiesFromGraphFn is the HandlerDeps version of execShowDesignPropertiesFromGraph.
func execShowDesignPropertiesFromGraphFn(ctx context.Context, widgetType string, deps *HandlerDeps) error {
	if deps.Graph == nil {
		return mdlerrors.NewValidationf("graph not built — run REFRESH CATALOG first")
	}

	props := deps.Graph.DesignProperties(widgetType)
	if len(props) == 0 {
		fmt.Fprintf(deps.Output, "No design properties found for %s\n", widgetType)
		return nil
	}

	fmt.Fprintf(deps.Output, "Design Properties for %s:\n", widgetType)
	for _, p := range props {
		fmt.Fprintf(deps.Output, "  %-30s %s", p.Name, p.Type)
		if p.Category != "" {
			fmt.Fprintf(deps.Output, "  [%s]", p.Category)
		}
		fmt.Fprintln(deps.Output)
		if len(p.Options) > 0 {
			fmt.Fprintf(deps.Output, "    Options: %s\n", strings.Join(p.Options, ", "))
		}
		if len(p.ReferencedVars) > 0 {
			fmt.Fprintf(deps.Output, "    References: %s\n", strings.Join(p.ReferencedVars, ", "))
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
