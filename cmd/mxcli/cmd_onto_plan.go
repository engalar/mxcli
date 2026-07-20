// cmd/mxcli/cmd_onto_plan.go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/mendixlabs/mxcli/internal/fkg"
	_ "github.com/mendixlabs/mxcli/internal/fkg/concepts"
	"github.com/spf13/cobra"
)

func init() {
	ontoCmd.AddCommand(ontoPlanCmd)
}

var ontoPlanCmd = &cobra.Command{
	Use:   "plan <module-id>",
	Short: "Show curriculum plan for a learning module",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		q, err := ontoNew()
		if err != nil {
			return err
		}
		c, ok := q.(fkg.CurriculumQuerier)
		if !ok {
			return fmt.Errorf("querier does not support Plan")
		}
		result, err := c.Plan(args[0])
		if err != nil {
			return err
		}
		if config.JSONOutput {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}
		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "Curriculum plan: %s\n", result.Module.Name)
		fmt.Fprintln(w)

		if len(result.Prerequisites) > 0 {
			fmt.Fprintln(w, "Prerequisites:")
			for _, p := range result.Prerequisites {
				fmt.Fprintf(w, "  %-35s %s\n", p.Name, p.Summary)
			}
			fmt.Fprintln(w)
		}
		if len(result.Concepts) > 0 {
			fmt.Fprintln(w, "Concepts taught:")
			for _, c := range result.Concepts {
				fmt.Fprintf(w, "  %-35s %s\n", c.Name, c.Summary)
			}
			fmt.Fprintln(w)
		}
		if len(result.Skills) > 0 {
			fmt.Fprintln(w, "Skills taught:")
			for _, s := range result.Skills {
				fmt.Fprintf(w, "  %-35s %s\n", s.Name, s.Summary)
			}
			fmt.Fprintln(w)
		}
		if len(result.Patterns) > 0 {
			fmt.Fprintln(w, "Patterns taught:")
			for _, p := range result.Patterns {
				fmt.Fprintf(w, "  %-35s %s\n", p.Name, p.Summary)
			}
			fmt.Fprintln(w)
		}
		if len(result.Extensions) > 0 {
			fmt.Fprintln(w, "Extensions:")
			for _, e := range result.Extensions {
				fmt.Fprintf(w, "  %-35s %s\n", e.Name, e.Summary)
			}
			fmt.Fprintln(w)
		}
		return nil
	},
}
