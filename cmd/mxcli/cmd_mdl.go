package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/normalize"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/spf13/cobra"
)

var mdlCmd = &cobra.Command{
	Use:   "mdl",
	Short: "MDL script utilities",
	Long: `MDL script utilities including normalization for diff comparison.

Subcommands:
  normalize  Normalize an MDL script for noise-free diff comparison`,
}

var mdlNormalizeCmd = &cobra.Command{
	Use:   "normalize [file.mdl | -]",
	Short: "Normalize an MDL script for diff comparison",
	Long: `Normalize an MDL script by removing noise (comments, @position, 
whitespace, keyword casing) and sorting statements by type.

The output is suitable for diff comparison between describe output and
a hand-written MDL script — only semantic differences remain.

Use '-' (or omit argument) to read from stdin.

Examples:
  # Normalize a file, write to stdout
  mxcli mdl normalize describe-output.mdl

  # Normalize from pipe, diff against app.mdl
  mxcli describe entity MyModule.MyEntity | mxcli mdl normalize \\
    | diff <(mxcli mdl normalize app.mdl) -

  # Compare describe output with hand-written MDL
  mxcli mdl normalize describe-snapshot.mdl -o /tmp/a.mdl
  mxcli mdl normalize app.mdl -o /tmp/b.mdl
  diff /tmp/a.mdl /tmp/b.mdl
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outPath, _ := cmd.Flags().GetString("output")
		noSort, _ := cmd.Flags().GetBool("no-sort")
		keepComments, _ := cmd.Flags().GetBool("keep-comments")
		keepPosition, _ := cmd.Flags().GetBool("keep-position")

		fromStdin := len(args) == 0 || args[0] == "-"
		filePath := ""
		if !fromStdin {
			filePath = args[0]
		}

		var data []byte
		var err error
		if fromStdin {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(filePath)
		}
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}

		label := filePath
		if fromStdin {
			label = "<stdin>"
		}

		// Validate syntax
		prog, errs := visitor.Build(string(data))
		if len(errs) > 0 {
			var msgs []string
			for _, e := range errs {
				msgs = append(msgs, e.Error())
			}
			return fmt.Errorf("syntax errors in %s:\n%s", label, strings.Join(msgs, "\n"))
		}
		_ = prog

		opts := normalize.DefaultOptions()
		if noSort {
			opts.SortStatements = false
		}
		if keepComments {
			opts.StripComments = false
		}
		if keepPosition {
			opts.StripPosition = false
		}

		result, err := normalize.Normalize(string(data), opts)
		if err != nil {
			return fmt.Errorf("normalize: %w", err)
		}

		if outPath != "" {
			if err := os.WriteFile(outPath, []byte(result), 0644); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
		} else {
			fmt.Print(result)
		}

		return nil
	},
}

func init() {
	mdlNormalizeCmd.Flags().StringP("output", "o", "", "Write to file instead of stdout")
	mdlNormalizeCmd.Flags().Bool("no-sort", false, "Don't sort statements by type")
	mdlNormalizeCmd.Flags().Bool("keep-comments", false, "Preserve comments")
	mdlNormalizeCmd.Flags().Bool("keep-position", false, "Preserve @Position annotations")

	mdlCmd.AddCommand(mdlNormalizeCmd)
}
