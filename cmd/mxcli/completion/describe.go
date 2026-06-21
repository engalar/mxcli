// SPDX-License-Identifier: Apache-2.0

package completion

import (
	"github.com/spf13/cobra"
)

// describeTypes are the valid first positional args for `describe`.
var describeTypes = []string{
	"module", "entity", "microflow", "nanoflow", "page", "snippet", "layout",
	"workflow", "enumeration", "constant", "image", "scheduled event",
	"business event service", "json structure", "import mapping", "export mapping",
	"consumed rest service", "published rest service",
	"consumed odata service", "published odata service",
	"database connection", "data transformer",
	"java action", "javascript action",
	"agent", "knowledge base", "consumed mcp service", "model",
	"system overview", "project",
}

// DescribeValidArgsFunction completes the type keyword and the qualified name.
func DescribeValidArgsFunction(comp *Completer) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Phase 1: no args yet — completing the type keyword
		if len(args) == 0 {
			return filterPrefix(describeTypes, toComplete), cobra.ShellCompDirectiveNoFileComp
		}
		// One arg that is NOT a known type → still completing the keyword
		if len(args) == 1 && !isKnownDescribeType(args[0]) {
			return filterPrefix(describeTypes, args[0]), cobra.ShellCompDirectiveNoFileComp
		}
		// Phase 2: completing the qualified name
		return completeProjectEntity(cmd, comp, args[len(args)-1], toComplete)
	}
}

// isKnownDescribeType returns true when word matches a known describe type.
func isKnownDescribeType(word string) bool {
	for _, t := range describeTypes {
		if t == word {
			return true
		}
	}
	return false
}

// ShowValidArgsFunction completes the type keyword and optional module name.
func ShowValidArgsFunction(comp *Completer) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return filterPrefix(showTypes, toComplete), cobra.ShellCompDirectiveNoFileComp
		}
		if len(args) == 1 && !isKnownShowType(args[0]) {
			return filterPrefix(showTypes, args[0]), cobra.ShellCompDirectiveNoFileComp
		}
		return completeModuleName(cmd, comp, toComplete)
	}
}

func isKnownShowType(word string) bool {
	for _, t := range showTypes {
		if t == word {
			return true
		}
	}
	return false
}

// showTypes are the valid type keywords for `show`.
var showTypes = []string{
	"modules", "entities", "microflows", "nanoflows", "pages", "snippets", "layouts",
	"workflows", "enumerations", "constants", "images", "scheduled events",
	"business event services", "json structures", "import mappings", "export mappings",
	"consumed rest services", "published rest services",
	"consumed odata services", "published odata services",
	"database connections", "data transformers",
	"java actions", "javascript actions",
	"agents", "knowledge bases", "consumed mcp services", "models",
	"folders", "module settings",
}

// ── shared helpers ─────────────────────────────────────────────────────

func completeProjectEntity(cmd *cobra.Command, comp *Completer, typeName, toComplete string) ([]string, cobra.ShellCompDirective) {
	mprPath := projectFlag(cmd)
	if mprPath == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if err := comp.EnsureConnected(mprPath); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	switch typeName {
	case "entity":
		return comp.EntitySuggestions(toComplete), cobra.ShellCompDirectiveNoFileComp
	case "microflow":
		return comp.MicroflowSuggestions(toComplete), cobra.ShellCompDirectiveNoFileComp
	case "nanoflow":
		return comp.NanoflowSuggestions(toComplete), cobra.ShellCompDirectiveNoFileComp
	case "page":
		return comp.PageSuggestions(toComplete), cobra.ShellCompDirectiveNoFileComp
	case "layout":
		return comp.LayoutSuggestions(toComplete), cobra.ShellCompDirectiveNoFileComp
	case "module":
		return comp.ModuleSuggestions(toComplete), cobra.ShellCompDirectiveNoFileComp
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeModuleName(cmd *cobra.Command, comp *Completer, toComplete string) ([]string, cobra.ShellCompDirective) {
	mprPath := projectFlag(cmd)
	if mprPath == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if err := comp.EnsureConnected(mprPath); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return comp.ModuleSuggestions(toComplete), cobra.ShellCompDirectiveNoFileComp
}

func filterPrefix(list []string, prefix string) []string {
	if prefix == "" {
		return list
	}
	var out []string
	for _, s := range list {
		if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
			out = append(out, s)
		}
	}
	return out
}

// projectFlag extracts the --project / -p flag value from the command.
// It checks the command itself, its parent chain (Cobra's Flag method
// inherits persistent flags), and falls back to the root's persistent flags.
func projectFlag(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	// Check local + inherited persistent flags
	if f := cmd.Flag("project"); f != nil {
		if v := f.Value.String(); v != "" {
			return v
		}
	}
	if f := cmd.Flag("p"); f != nil {
		if v := f.Value.String(); v != "" {
			return v
		}
	}
	// Check root persistent flags
	if root := cmd.Root(); root != nil {
		if f := root.Flag("project"); f != nil {
			if v := f.Value.String(); v != "" {
				return v
			}
		}
	}
	return ""
}
