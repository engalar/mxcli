// cmd/mxcli/cmd_onto_guide.go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/mendixlabs/mxcli/internal/fkg"
	_ "github.com/mendixlabs/mxcli/internal/fkg/concepts"
	"github.com/spf13/cobra"
)

func init() {
	ontoCmd.AddCommand(ontoGuideCmd)
}

var ontoGuideCmd = &cobra.Command{
	Use:   "guide <concept>",
	Short: "Show implementation guidance for a concept",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		q, err := ontoNew()
		if err != nil {
			return err
		}
		g, ok := q.(fkg.GuidanceQuerier)
		if !ok {
			return fmt.Errorf("querier does not support Guide")
		}
		result, err := g.Guide(args[0])
		if err != nil {
			return err
		}
		if globalJSONFlag {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}
		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "Guidance for: %s — %s\n", result.Concept.Name, result.Concept.Summary)
		fmt.Fprintln(w)

		if len(result.Patterns) > 0 {
			fmt.Fprintln(w, "Patterns:")
			for _, p := range result.Patterns {
				fmt.Fprintf(w, "  %-35s %s\n", p.Name, p.Summary)
			}
			fmt.Fprintln(w)
		}
		if len(result.Steps) > 0 {
			fmt.Fprintln(w, "Implementation steps:")
			for _, s := range result.Steps {
				hint := ""
				if s.SyntaxHint != "" {
					hint = "  → " + s.SyntaxHint
				}
				fmt.Fprintf(w, "  %d. [%s] %s%s\n", s.Order, s.Action, s.Description, hint)
			}
			fmt.Fprintln(w)
		}
		if len(result.SyntaxRefs) > 0 {
			fmt.Fprintln(w, "Syntax references:")
			for _, s := range result.SyntaxRefs {
				fmt.Fprintf(w, "  %-35s %s\n", s.Name, s.Summary)
			}
			fmt.Fprintln(w)
		}
		if len(result.Skills) > 0 {
			fmt.Fprintln(w, "Related skills:")
			for _, s := range result.Skills {
				fmt.Fprintf(w, "  %-35s %s\n", s.Name, s.Summary)
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
