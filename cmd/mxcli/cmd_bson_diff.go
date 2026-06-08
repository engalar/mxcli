// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/term"

	bsondebug "github.com/mendixlabs/mxcli/cmd/mxcli/bson"
)

const (
	ansiRed   = "\033[31m"
	ansiGreen = "\033[32m"
	ansiCyan  = "\033[36m"
	ansiReset = "\033[0m"
)

var bsonDiffCmd = &cobra.Command{
	Use:   "diff <file1.mxunit> <file2.mxunit>",
	Short: "Show git-diff style NDSL diff between two .mxunit files",
	Long: `Read two raw .mxunit BSON files, render both as NDSL, and output a
unified diff in git-diff style. No project file required.

Exit codes:
  0  files are identical
  1  files differ
  2  error (file not found, invalid BSON)

Examples:
  mxcli bson diff reference.mxunit generated.mxunit
  mxcli bson diff a.mxunit b.mxunit --no-color | pbcopy
`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		forceColor, _ := cmd.Flags().GetBool("color")
		noColor, _ := cmd.Flags().GetBool("no-color")
		useColor := resolveColor(forceColor, noColor)

		file1, file2 := args[0], args[1]

		text, changed, err := computeBsonDiff(file1, file2, file1, file2)
		if err != nil {
			return err
		}

		if !changed {
			return nil
		}

		if useColor {
			fmt.Fprint(cmd.OutOrStdout(), colorizeUnifiedDiff(text))
		} else {
			fmt.Fprint(cmd.OutOrStdout(), text)
		}
		return nil
	},
}

// readAndRenderMxunit reads a .mxunit file and returns its NDSL representation.
// Fields are rendered as "Key = value" (equals separator) for diff clarity.
func readAndRenderMxunit(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	var doc bson.D
	if err := bson.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("parse BSON in %s: %w", path, err)
	}
	return bsondebug.RenderForDiff(doc, 0), nil
}

// computeBsonDiff renders both files as NDSL and returns a unified diff string.
// changed is false when the files produce identical NDSL output.
func computeBsonDiff(file1, file2, label1, label2 string) (text string, changed bool, err error) {
	ndsl1, err := readAndRenderMxunit(file1)
	if err != nil {
		return "", false, err
	}
	ndsl2, err := readAndRenderMxunit(file2)
	if err != nil {
		return "", false, err
	}

	ud := difflib.UnifiedDiff{
		A:        difflib.SplitLines(ndsl1),
		B:        difflib.SplitLines(ndsl2),
		FromFile: label1,
		ToFile:   label2,
		Context:  3,
	}
	text, err = difflib.GetUnifiedDiffString(ud)
	if err != nil {
		return "", false, fmt.Errorf("diff: %w", err)
	}
	return text, text != "", nil
}

// colorizeUnifiedDiff applies ANSI colors to a unified diff string.
func colorizeUnifiedDiff(diff string) string {
	var sb strings.Builder
	for _, line := range strings.SplitAfter(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "@@"):
			sb.WriteString(ansiCyan + line + ansiReset)
		case strings.HasPrefix(line, "-"):
			sb.WriteString(ansiRed + line + ansiReset)
		case strings.HasPrefix(line, "+"):
			sb.WriteString(ansiGreen + line + ansiReset)
		default:
			sb.WriteString(line)
		}
	}
	return sb.String()
}

// resolveColor determines whether to emit ANSI color codes.
// Priority: --color flag > --no-color flag > NO_COLOR env > terminal auto-detect.
func resolveColor(forceColor, noColor bool) bool {
	if forceColor {
		return true
	}
	if noColor {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func init() {
	bsonDiffCmd.Flags().Bool("color", false, "Force color output")
	bsonDiffCmd.Flags().Bool("no-color", false, "Disable color output")
}
