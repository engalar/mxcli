// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/cmd/mxcli/completion"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze <topic> [name]",
	Short: "Analyze project structure — navigation, page, entity, orphans, flow",
	ValidArgsFunction: completion.AnalyzeValidArgsFunction(comp),
	Long: `Analyze Mendix project structure using the in-memory graph.

Topics:
  navigation          Show navigation profiles, menu tree, and home pages
  page <QN>           Show page data container hierarchy + context variables + widget appearance
  entity <QN>         Show entity data flow: pages, microflows creators/retrievers, permissions
  orphans             Show orphan pages with no navigation or microflow references
  flow [kind [name]]  Transitive data flow: list entry points or show chains from one

Examples:
  mxcli analyze -p app.mpr navigation
  mxcli analyze -p app.mpr page MyModule.MyPage
  mxcli analyze -p app.mpr entity MyModule.MyEntity
  mxcli analyze -p app.mpr orphans
  mxcli analyze -p app.mpr flow                              (all entry points + reachability)
  mxcli analyze -p app.mpr flow navigation Responsive        (chains from navigation)
  mxcli analyze -p app.mpr flow workflow MyWorkflow          (chains from workflow)
  mxcli analyze -p app.mpr flow microflow MyMicroflow        (chains from microflow)
`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath, _ := cmd.Flags().GetString("project")
		if projectPath == "" {
			return fmt.Errorf("--project (-p) is required")
		}

		exec, logger := buildExec("analyze", cmd.OutOrStdout())
		defer logger.Close()
		defer exec.Close()

		connectCmd := fmt.Sprintf("CONNECT LOCAL '%s'", projectPath)
		prog, errs := visitor.Build(connectCmd)
		if len(errs) > 0 {
			for _, err := range errs {
				fmt.Fprintf(cmd.ErrOrStderr(), "Parse error: %v\n", err)
			}
			return fmt.Errorf("connect failed")
		}
		if err := exec.ExecuteProgram(prog); err != nil {
			return fmt.Errorf("connect: %w", err)
		}

		topic := args[0]
		var name string
		if len(args) > 1 {
			name = args[1]
		}

		// Build the graph if not already cached
		pg, err := exec.BuildGraph()
		if err != nil {
			return fmt.Errorf("build graph: %w", err)
		}

		// Create a minimal ExecContext for calling analyze functions.
		ctx := &executor.ExecContext{}
		ctx.Graph = pg
		ctx.Output = cmd.OutOrStdout()

		showPerf, _ := cmd.Flags().GetBool("perf")
		if showPerf {
			pt := executor.NewPerfTimer()
			pt.SetOutput(cmd.OutOrStdout())
			ctx.Perf = pt
		}

		depth, _ := cmd.Flags().GetInt("depth")

		switch topic {
		case "navigation", "nav":
			return executor.AnalyzeNavigation(ctx)
		case "page":
			if name == "" {
				return fmt.Errorf("page QN required: mxcli analyze page MyModule.MyPage")
			}
			return executor.AnalyzePage(ctx, name)
		case "entity":
			if name == "" {
				return fmt.Errorf("entity QN required: mxcli analyze entity MyModule.MyEntity")
			}
			return executor.AnalyzeEntity(ctx, name)
		case "orphans", "orphan":
			return executor.AnalyzeOrphans(ctx)
		case "flow":
			// mxcli analyze flow                      (list all entry points + reachability)
			// mxcli analyze flow navigation Responsive (chains from navigation)
			// mxcli analyze flow workflow MyWF         (chains from workflow)
			// mxcli analyze flow microflow MyMF        (chains from microflow)
			entryKind := name
			entryName := ""
			if len(args) > 2 {
				entryName = strings.Join(args[2:], " ")
			}
			return executor.AnalyzeFlow(ctx, entryKind, entryName, depth)
		default:
			return fmt.Errorf("unknown analyze topic: %q (try: navigation, page <QN>, entity <QN>, orphans, flow)", topic)
		}
	},
}

func init() {
	analyzeCmd.Flags().Int("depth", 0, "Max traversal depth for flow analysis (0=unlimited)")
	analyzeCmd.Flags().Bool("perf", false, "Show performance timing breakdown")
	rootCmd.AddCommand(analyzeCmd)
}
