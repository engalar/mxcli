// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/mdl/linter/rules"
)

// ExecLintFn is the HandlerDeps version of execLint.
func ExecLintFn(ctx context.Context, s *ast.LintStmt, deps *HandlerDeps) error {
	if deps.ConnectionManager == nil || !deps.ConnectionManager.IsConnected() {
		return mdlerrors.NewNotConnected()
	}
	if s.ShowRules {
		return listLintRulesFn(ctx, deps)
	}
	if deps.Graph == nil {
		fmt.Fprintln(deps.Output, "Building project graph for linting...")
		if err := buildGraphFromDeps(deps.Output, deps.Quiet, deps.MprPath, &deps.Graph); err != nil {
			return mdlerrors.NewBackend("build project graph", err)
		}
	}
	var lintReader linter.LintReader
	if lr, ok := deps.ConnectionManager.(linter.LintReader); ok {
		lintReader = lr
	}
	lintCtx := linter.NewLintContext(deps.Graph, lintReader)
	projectDir := filepath.Dir(deps.MprPath)
	configPath := linter.FindConfigFile(projectDir)
	config, err := linter.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(deps.Output, "Warning: failed to load lint config: %v\n", err)
		config = linter.DefaultConfig()
	}
	if len(config.ExcludeModules) > 0 {
		lintCtx.SetExcludedModules(config.ExcludeModules)
	}
	lint := linter.New(lintCtx)
	lint.AddRule(rules.NewNamingConventionRule())
	lint.AddRule(rules.NewEmptyMicroflowRule())
	lint.AddRule(rules.NewDomainModelSizeRule())
	lint.AddRule(rules.NewValidationFeedbackRule())
	lint.AddRule(rules.NewImageSourceRule())
	lint.AddRule(rules.NewMissingTranslationsRule())
	lint.AddRule(rules.NewDataGrid2ColumnRule())
	rulesDir := filepath.Join(projectDir, ".claude", "lint-rules")
	starlarkRules, err := linter.LoadStarlarkRulesFromDir(rulesDir)
	if err != nil {
		fmt.Fprintf(deps.Output, "Warning: failed to load custom rules: %v\n", err)
	}
	for _, rule := range starlarkRules {
		lint.AddRule(rule)
	}
	config.ApplyConfig(lint)
	if s.Target != nil && s.ModuleOnly {
		lintCtx.SetExcludedModules(nil)
		fmt.Fprintf(deps.Output, "Linting module: %s\n", s.Target.Module)
	}
	violations, err := lint.Run(context.Background())
	if err != nil {
		return mdlerrors.NewBackend("lint", err)
	}
	if s.Target != nil && s.ModuleOnly {
		filtered := make([]linter.Violation, 0)
		for _, v := range violations {
			if v.Location.Module == s.Target.Module {
				filtered = append(filtered, v)
			}
		}
		violations = filtered
	}
	var format linter.OutputFormat
	switch s.Format {
	case ast.LintFormatJSON:
		format = linter.OutputFormatJSON
	case ast.LintFormatSARIF:
		format = linter.OutputFormatSARIF
	default:
		format = linter.OutputFormatText
	}
	formatter := linter.GetFormatter(format, false)
	return formatter.Format(violations, deps.Output)
}

// listLintRulesFn is the HandlerDeps version of listLintRules.
func listLintRulesFn(ctx context.Context, deps *HandlerDeps) error {
	fmt.Fprintln(deps.Output, "Built-in rules:")
	fmt.Fprintln(deps.Output)
	lint := linter.New(nil)
	lint.AddRule(rules.NewNamingConventionRule())
	lint.AddRule(rules.NewEmptyMicroflowRule())
	lint.AddRule(rules.NewDomainModelSizeRule())
	lint.AddRule(rules.NewValidationFeedbackRule())
	lint.AddRule(rules.NewImageSourceRule())
	lint.AddRule(rules.NewMissingTranslationsRule())
	lint.AddRule(rules.NewDataGrid2ColumnRule())
	lint.AddRule(rules.NewBrokenMFParamRefRule())
	for _, rule := range lint.Rules() {
		fmt.Fprintf(deps.Output, "  %s (%s)\n", rule.ID(), rule.Name())
		fmt.Fprintf(deps.Output, "    %s\n", rule.Description())
		fmt.Fprintf(deps.Output, "    Category: %s, Default Severity: %s\n", rule.Category(), rule.DefaultSeverity())
		fmt.Fprintln(deps.Output)
	}
	if deps.MprPath != "" {
		projectDir := filepath.Dir(deps.MprPath)
		rulesDir := filepath.Join(projectDir, ".claude", "lint-rules")
		starlarkRules, err := linter.LoadStarlarkRulesFromDir(rulesDir)
		if err == nil && len(starlarkRules) > 0 {
			fmt.Fprintln(deps.Output, "Custom rules (from .claude/lint-rules/):")
			fmt.Fprintln(deps.Output)
			for _, rule := range starlarkRules {
				fmt.Fprintf(deps.Output, "  %s (%s)\n", rule.ID(), rule.Name())
				fmt.Fprintf(deps.Output, "    %s\n", rule.Description())
				fmt.Fprintf(deps.Output, "    Category: %s, Default Severity: %s\n", rule.Category(), rule.DefaultSeverity())
				fmt.Fprintln(deps.Output)
			}
		}
	}
	return nil
}

// --- Executor method wrappers for backward compatibility ---
