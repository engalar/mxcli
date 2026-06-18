// cmd/mxcli/cmd_onto.go
package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/mendixlabs/mxcli/internal/fkg"
	"github.com/mendixlabs/mxcli/internal/fkg/concepts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	rootCmd.AddCommand(ontoCmd)
	ontoCmd.AddCommand(ontoSchemaCmd)
	ontoCmd.AddCommand(ontoExploreCmd)
	ontoCmd.AddCommand(ontoPathCmd)
	ontoCmd.AddCommand(ontoGuideCmd)
	ontoCmd.AddCommand(ontoPlanCmd)
	ontoCmd.AddCommand(ontoOrchestrateCmd)
	ontoExploreCmd.Flags().Int("depth", 2, "Traversal depth (default 2)")

	// Note: command tree registration happens lazily via initMXCLICmds.
	// This ensures all init() functions across the package have completed
	// before we read the command tree.
}

var mxcliCmdsOnce sync.Once

// initMXCLICmds registers the mxcli command tree into FKG.
// Called lazily before the first onto query so all init() functions have run.
func initMXCLICmds() {
	mxcliCmdsOnce.Do(func() {
		registerMXCLICommands(rootCmd, "")
	})
}

// registerMXCLICommands recursively walks the cobra command tree and registers
// each command with FKG so that "onto explore cmd:exec" etc. can discover them.
func registerMXCLICommands(cmd *cobra.Command, parent string) {
	use := cmd.Use
	// Use the first word of Use as the canonical command name
	// (e.g., "exec <file>" → "exec", "show <type> [name]" → "show")
	name := use
	for i, r := range use {
		if r == ' ' || r == '\t' {
			name = use[:i]
			break
		}
	}
	if name == "" {
		return
	}

	flags := []string{}
	func() {
		defer func() { recover() }() // ignore flagset collisions in cobra internals
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
			flags = append(flags, "--"+f.Name)
		})
	}()

	concepts.RegisterMXCLICmd(name, cmd.Short, cmd.Long, parent, flags)

	for _, sub := range cmd.Commands() {
		registerMXCLICommands(sub, name)
	}
}

var ontoCmd = &cobra.Command{
	Use:   "onto",
	Short: "Query the mxcli feature knowledge graph (ontology)",
	Long: `Explore mxcli's capability ontology: concepts, MDL syntax features,
skills, patterns, extensions, and the relationships between them.

Useful for AI-assisted task planning: start from a prior concept ("page",
"microflow", "security") and get implementation guidance or explore what
syntax, skills, and related concepts are available.

Examples:
  mxcli onto schema
  mxcli onto explore page
  mxcli onto explore page --depth 3
  mxcli onto path page security
  mxcli onto guide entity
  mxcli onto plan curriculum:academy-04-pages
  mxcli onto explore page --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return nil
		}
		w := cmd.OutOrStdout()
		fmt.Fprintln(w, `╭─ mxcli onto ───────────────────────────────────────────╮
│                                                          │
│  This is the Feature Knowledge Graph (FKG). It helps AI  │
│  understand which mxcli concepts exist, how they relate, │
│  and what MDL syntax, skills, and patterns are available.│
│                                                          │
│  Commands:                                               │
│    onto schema    —  show all node types, edge types,    │
│                      and root concept nodes              │
│    onto explore   —  traverse neighbours of a concept    │
│    onto path      —  find a connection path between      │
│                      two concept nodes                   │
│    onto guide     —  get implementation guidance for     │
│                      a concept                           │
│    onto plan      —  show curriculum plan for             │
│                      a module                            │
│    onto orchestrate—  ordered implementation plan         │
│                      across multiple concepts             │
│                                                          │
│  Examples:                                               │
│    mxcli onto schema                                     │
│    mxcli onto explore page                               │
│    mxcli onto explore page --depth 3                     │
│    mxcli onto path page security                         │
│    mxcli onto guide entity                               │
│    mxcli onto plan curriculum:academy-04-pages           │
│    mxcli onto orchestrate entity microflow page security │
│    mxcli onto explore page --json                        │
│                                                          │
│  Tip: start with "onto schema" to see all root concepts, │
│  then "onto guide <concept>" for implementation steps.   │
╰──────────────────────────────────────────────────────────╯`)
		return cmd.Help()
	},
}

// ontoNew builds the FKG, ensuring the mxcli command tree is registered first.
func ontoNew() (fkg.Querier, error) {
	initMXCLICmds()
	return fkg.New()
}

var ontoSchemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Show the full ontology skeleton (node types, edge types, root concepts)",
	RunE: func(cmd *cobra.Command, args []string) error {
		q, err := ontoNew()
		if err != nil {
			return err
		}
		result := q.Schema()
		if globalJSONFlag {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}
		w := cmd.OutOrStdout()
		fmt.Fprintln(w, "Node types:")
		for _, nt := range result.NodeTypes {
			fmt.Fprintf(w, "  %-20s %d\n", nt.Label, nt.Count)
		}
		fmt.Fprintln(w, "\nEdge types:")
		for _, et := range result.EdgeTypes {
			fmt.Fprintf(w, "  %-20s %d\n", et.RelType, et.Count)
		}
		fmt.Fprintln(w, "\nRoot concepts:")
		for _, r := range result.Roots {
			fmt.Fprintf(w, "  %-20s — %s\n", r.Name, r.Summary)
		}
		return nil
	},
}

var ontoExploreCmd = &cobra.Command{
	Use:   "explore <id>",
	Short: "Explore the neighbourhood of a concept node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		depth, _ := cmd.Flags().GetInt("depth")
		q, err := ontoNew()
		if err != nil {
			return err
		}
		result, err := q.Explore(args[0], depth)
		if err != nil {
			return err
		}
		if globalJSONFlag {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}
		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "%s: %s [%s]\n", result.Seed.Label, result.Seed.Name, result.Seed.Summary)

		// Group neighbours by label
		byLabel := map[string][]fkg.NodeSummary{}
		for _, n := range result.Nodes {
			byLabel[n.Label] = append(byLabel[n.Label], n)
		}
		for _, label := range []string{"Concept", "SyntaxFeature", "Skill", "Doc"} {
			nodes := byLabel[label]
			if len(nodes) == 0 {
				continue
			}
			fmt.Fprintf(w, "  %s (%d):\n", label, len(nodes))
			for _, n := range nodes {
				fmt.Fprintf(w, "    %-35s %s\n", n.Name, n.Summary)
			}
		}
		return nil
	},
}

var ontoPathCmd = &cobra.Command{
	Use:   "path <from> <to>",
	Short: "Discover structural path schemas between two nodes",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		q, err := ontoNew()
		if err != nil {
			return err
		}
		schemas, err := q.Path(args[0], args[1])
		if err != nil {
			return err
		}
		if globalJSONFlag {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(schemas)
		}
		w := cmd.OutOrStdout()
		if len(schemas) == 0 {
			fmt.Fprintf(w, "No path found between %q and %q\n", args[0], args[1])
			return nil
		}
		fmt.Fprintf(w, "Paths (%d found):\n", len(schemas))
		for i, s := range schemas {
			parts := make([]string, 0, len(s.Steps)*2+1)
			parts = append(parts, args[0])
			for _, step := range s.Steps {
				name := step.NodeName
				if name == "" {
					name = step.NodeID
				}
				parts = append(parts, fmt.Sprintf("─[%s]→", step.RelType), name)
			}
			fmt.Fprintf(w, "  %d. %s\n", i+1, strings.Join(parts, " "))
		}
		return nil
	},
}
