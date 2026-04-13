// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mendixlabs/mxcli/cmd/mxcli/syntax"
	"github.com/spf13/cobra"
)

// topicAliases maps legacy topic names and plurals to registry paths.
var topicAliases = map[string]string{
	// Domain model aliases
	"keywords":        "domain-model.keywords",
	"reserved":        "domain-model.keywords",
	"types":           "domain-model.types",
	"datatypes":       "domain-model.types",
	"data-types":      "domain-model.types",
	"delete":          "domain-model.association.delete-behavior",
	"delete_behavior": "domain-model.association.delete-behavior",
	"delete-behavior": "domain-model.association.delete-behavior",
	"entity":          "domain-model.entity",
	"entities":        "domain-model.entity",
	"enumeration":     "domain-model.enumeration",
	"enum":            "domain-model.enumeration",
	"enumerations":    "domain-model.enumeration",
	"constant":        "domain-model.constant",
	"constants":       "domain-model.constant",
	"association":     "domain-model.association",
	"associations":    "domain-model.association",
	// Plural aliases
	"microflows":      "microflow",
	"pages":           "page",
	"snippets":        "snippet",
	"fragments":       "fragment",
	"workflows":       "workflow",
	// Variant aliases
	"nav":             "navigation",
	"project-settings": "settings",
	"rest-client":     "rest",
	"rest-clients":    "rest",
	"integrations":    "integration",
	"services":        "integration",
	"contract":        "integration",
	"contracts":       "integration",
	"javaaction":      "java-action",
	"java_action":     "java-action",
	"java-actions":    "java-action",
	"javaactions":     "java-action",
	"businessevents":  "business-events",
	"business_events": "business-events",
	"be":              "business-events",
	"xpath-constraints": "xpath",
	"external-sql":    "sql",
	"validation":      "errors",
}

var syntaxCmd = &cobra.Command{
	Use:   "syntax [topic [subtopic...]]",
	Short: "Show MDL syntax reference",
	Long: `Show MDL syntax reference from the feature registry.

Use --json for machine-readable output (optimized for LLM consumption).
Drill down with multiple arguments: mxcli syntax workflow user-task targeting

Top-level topics:
  domain-model    - Entities, associations, enumerations, constants, keywords, types
  microflow       - Microflow/nanoflow creation and activities
  page            - Pages, snippets, fragments, widgets
  security        - Roles, access control, demo users
  workflow        - Workflows, user tasks, decisions, parallel splits
  navigation      - Navigation profiles, menus, home pages
  settings        - Project settings
  integration     - OData, REST, SQL, OQL, XPath, Java actions, business events
  errors          - Common validation errors and fixes
  structure       - SHOW STRUCTURE command
  move            - MOVE command for relocating documents
  search          - Full-text SEARCH command

Examples:
  mxcli syntax --json                          # Full index (LLM: cache this)
  mxcli syntax workflow --json                 # All workflow features
  mxcli syntax workflow user-task targeting     # Drill down to targeting
  mxcli syntax security entity-access           # Entity access rules
  mxcli syntax entity                           # Legacy alias → domain-model.entity
`,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag, _ := cmd.Flags().GetBool("json")

		// No args: show full index (JSON) or help text
		if len(args) == 0 {
			if jsonFlag {
				syntax.WriteJSON(os.Stdout, syntax.All())
				return
			}
			cmd.Help()
			return
		}

		// Build registry path from args
		path := strings.ToLower(strings.Join(args, "."))

		// Apply aliases
		if alias, ok := topicAliases[path]; ok {
			path = alias
		}

		// Query registry
		if syntax.HasPrefix(path) {
			features := syntax.ByPrefix(path)
			if jsonFlag {
				syntax.WriteJSON(os.Stdout, features)
			} else {
				syntax.WriteText(os.Stdout, features)
			}
			return
		}

		fmt.Printf("Unknown topic: %s\n\n", path)
		cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(syntaxCmd)
}
