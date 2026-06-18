// cmd/mxcli/cmd_onto_orchestrate.go
package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/internal/fkg"
	_ "github.com/mendixlabs/mxcli/internal/fkg/concepts"
	"github.com/spf13/cobra"
)

func init() {
	ontoCmd.AddCommand(ontoOrchestrateCmd)
}

var ontoOrchestrateCmd = &cobra.Command{
	Use:   "orchestrate <concepts...>",
	Short: "Ordered implementation plan across multiple concepts",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		q, err := ontoNew()
		if err != nil {
			return err
		}
		o, ok := q.(fkg.Orchestrator)
		if !ok {
			return fmt.Errorf("querier does not support Orchestrate")
		}
		result, err := o.Orchestrate(args)
		if err != nil {
			return err
		}
		if globalJSONFlag {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}
		w := cmd.OutOrStdout()
		fmt.Fprintln(w, "Implementation order:")
		for _, s := range result.Steps {
			deps := ""
			if len(s.DependsOn) > 0 {
				deps = "  ← depends on: " + strings.Join(s.DependsOn, ", ")
			}
			fmt.Fprintf(w, "  %d. %-20s%s\n", s.Order, s.Concept.Name, deps)
		}
		fmt.Fprintln(w)
		for _, s := range result.Steps {
			fmt.Fprintf(w, "  %s:\n", s.Concept.Name)
			if len(s.Patterns) > 0 {
				fmt.Fprintln(w, "    Patterns:")
				for _, p := range s.Patterns {
					fmt.Fprintf(w, "      %-35s %s\n", p.Name, p.Summary)
				}
			}
			if len(s.Skills) > 0 {
				fmt.Fprintln(w, "    Skills:")
				for _, sk := range s.Skills {
					fmt.Fprintf(w, "      %-35s %s\n", sk.Name, sk.Summary)
				}
			}
			fmt.Fprintln(w)
		}
		return nil
	},
}
