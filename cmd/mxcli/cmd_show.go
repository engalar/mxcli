// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mendixlabs/mxcli/cmd/mxcli/completion"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:     "show <type> [name]",
	Aliases: []string{"list"},
	Short:   "List project elements",
	ValidArgsFunction: completion.ShowValidArgsFunction(comp),
	Long: `List elements from a Mendix project. (Also available as "mxcli list")

Types:
  modules              List all modules
  entities             List all entities
  associations         List all associations
  enumerations         List all enumerations
  microflows           List all microflows
  nanoflows            List all nanoflows
  pages                List all pages
  snippets             List all snippets
  layouts              List all layouts
  constants            List all constants
  workflows            List all workflows
  javaactions          List all java actions
  odataclients         List all consumed OData services
  odataservices        List all published OData services
  businesseventservices  List business event service documents
  businessevents       List individual business event messages
  settings             Show project settings

Multi-word types also accepted: "business event services", "business events", etc.

Example:
  mxcli show -p app.mpr modules
  mxcli show -p app.mpr entities
  mxcli show -p app.mpr entities MyModule
  mxcli show -p app.mpr microflows MyModule
  mxcli show -p app.mpr pages
  mxcli show -p app.mpr workflows
  mxcli show -p app.mpr odataclients
  mxcli show -p app.mpr businesseventservices
  mxcli show -p app.mpr business event services MyModule
  mxcli show -p app.mpr business events
  mxcli show -p app.mpr settings
`,
	// show command uses custom arg parsing via parseShowArgs
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectPath, _ := cmd.Flags().GetString("project")
		if projectPath == "" {
			fmt.Fprintln(os.Stderr, "Error: --project (-p) is required")
			os.Exit(1)
		}

		// Parse type and optional module name from args.
		// Supports single-word ("entities") and multi-word ("business event services") types.
		// Optional module: "entities MyModule", "entities in MyModule",
		//   "business event services MyModule", "business event services in MyModule"
		showKeyword, moduleName := parseShowArgs(args)
		if showKeyword == "" {
			fmt.Fprintf(os.Stderr, "Unknown type: %s\n", strings.Join(args, " "))
			fmt.Fprintln(os.Stderr, "Valid types: modules, entities, associations, enumerations, microflows, nanoflows, pages, snippets, layouts, constants, workflows, javaactions, odataclients, odataservices, businesseventservices, businessevents, settings")
			os.Exit(1)
		}

		var mdlCmd string
		if moduleName != "" {
			mdlCmd = fmt.Sprintf("SHOW %s IN %s", showKeyword, moduleName)
		} else {
			mdlCmd = fmt.Sprintf("SHOW %s", showKeyword)
		}

		if err := executeMDL(projectPath, mdlCmd, cmd.OutOrStdout()); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
			return
		}
	},
}

// showTypeMap maps CLI type strings to MDL SHOW keywords.
// Multi-word entries are checked first (longest match wins).
var showTypeMap = map[string]string{
	// Multi-word types (3 words)
	"BUSINESS EVENT SERVICES": "BUSINESS EVENT SERVICES",
	"BUSINESS EVENT CLIENTS":  "BUSINESS EVENT CLIENTS",
	// Multi-word types (2 words)
	"BUSINESS EVENTS":  "BUSINESS EVENTS",
	"JAVA ACTIONS":     "JAVA ACTIONS",
	"ODATA CLIENTS":    "ODATA CLIENTS",
	"ODATA SERVICES":   "ODATA SERVICES",
	"MODULE ROLES":     "MODULE ROLES",
	"USER ROLES":       "USER ROLES",
	"PROJECT SECURITY": "PROJECT SECURITY",
	// Single-word types (including compressed forms)
	"MODULES":               "MODULES",
	"ENTITIES":              "ENTITIES",
	"ASSOCIATIONS":          "ASSOCIATIONS",
	"ENUMERATIONS":          "ENUMERATIONS",
	"MICROFLOWS":            "MICROFLOWS",
	"NANOFLOWS":             "NANOFLOWS",
	"PAGES":                 "PAGES",
	"SNIPPETS":              "SNIPPETS",
	"LAYOUTS":               "LAYOUTS",
	"CONSTANTS":             "CONSTANTS",
	"WORKFLOWS":             "WORKFLOWS",
	"JAVAACTIONS":           "JAVA ACTIONS",
	"ODATACLIENTS":          "ODATA CLIENTS",
	"ODATASERVICES":         "ODATA SERVICES",
	"BUSINESSEVENTSERVICES": "BUSINESS EVENT SERVICES",
	"BUSINESSEVENTCLIENTS":  "BUSINESS EVENT CLIENTS",
	"BUSINESSEVENTS":        "BUSINESS EVENTS",
	"SETTINGS":              "SETTINGS",
	"MODULEROLES":           "MODULE ROLES",
	"USERROLES":             "USER ROLES",
	"PROJECTSECURITY":       "PROJECT SECURITY",
}

// parseShowArgs parses CLI args into a show keyword and optional module name.
// Tries multi-word type matches first (longest), then single-word.
// Returns ("", "") if no type matched.
func parseShowArgs(args []string) (showKeyword string, moduleName string) {
	upper := make([]string, len(args))
	for i, a := range args {
		upper[i] = strings.ToUpper(a)
	}

	// Try matching 3, 2, then 1 words as the type
	for typeLen := 3; typeLen >= 1; typeLen-- {
		if typeLen > len(args) {
			continue
		}
		candidate := strings.Join(upper[:typeLen], " ")
		if kw, ok := showTypeMap[candidate]; ok {
			// Remaining args after the type are [IN] moduleName
			rest := args[typeLen:]
			for _, r := range rest {
				if strings.EqualFold(r, "in") {
					continue
				}
				moduleName = r
				break
			}
			return kw, moduleName
		}
	}

	return "", ""
}
