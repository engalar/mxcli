// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/term"

	bsondebug "github.com/mendixlabs/mxcli/cmd/mxcli/bson"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

const (
	ansiRed   = "\033[31m"
	ansiGreen = "\033[32m"
	ansiCyan  = "\033[36m"
	ansiReset = "\033[0m"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare project state, scripts, and BSON files",
	Long: `Compare MDL scripts against project state, local changes against git,
and BSON objects from Mendix project files.

Subcommands:
  script        Compare an MDL script against the current project state
  local         Compare local changes against git (alias: diff-local)
  bson-compare  Compare two BSON objects for differences
  bson-diff     Show git-diff style NDSL diff between two .mxunit files`,
}

var diffScriptCmd = &cobra.Command{
	Use:   "script <script.mdl>",
	Short: "Compare an MDL script against the current project state",
	Long: `Compare an MDL script file against the current state of a Mendix project.

Shows the differences between what the script would create/modify and what
currently exists in the project.

Output Formats:
  unified  - Traditional unified diff format (default)
  side     - Side-by-side comparison
  struct   - Structural changes summary

Examples:
  # Unified diff (default)
  mxcli diff script -p app.mpr changes.mdl

  # Side-by-side diff
  mxcli diff script -p app.mpr changes.mdl --format side

  # Structural diff
  mxcli diff script -p app.mpr changes.mdl --format struct

  # With color output
  mxcli diff script -p app.mpr changes.mdl --color
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		projectPath, _ := cmd.Flags().GetString("project")
		format, _ := cmd.Flags().GetString("format")
		useColor, _ := cmd.Flags().GetBool("color")
		width, _ := cmd.Flags().GetInt("width")

		if projectPath == "" {
			return fmt.Errorf("--project (-p) is required")
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}

		prog, errs := visitor.Build(string(content))
		if len(errs) > 0 {
			fmt.Fprintf(os.Stderr, "Syntax errors found:\n")
			for _, err := range errs {
				fmt.Fprintf(os.Stderr, "  - %v\n", err)
			}
			return fmt.Errorf("syntax errors in script")
		}

		exec, logger := buildExec("subcommand", cmd.OutOrStdout())
		defer logger.Close()
		defer exec.Close()

		connectProg, _ := visitor.Build(fmt.Sprintf("CONNECT LOCAL '%s'", projectPath))
		for _, stmt := range connectProg.Statements {
			if err := exec.Execute(stmt); err != nil {
				return fmt.Errorf("connecting: %w", err)
			}
		}

		opts := executor.DiffOptions{
			Format:   executor.DiffFormat(format),
			UseColor: useColor,
			Width:    width,
		}

		if err := exec.DiffProgram(prog, opts); err != nil {
			return fmt.Errorf("diff: %w", err)
		}
		return nil
	},
}

func runDiffLocal(cmd *cobra.Command, args []string) error {
	projectPath, _ := cmd.Flags().GetString("project")
	ref, _ := cmd.Flags().GetString("ref")
	format, _ := cmd.Flags().GetString("format")
	useColor, _ := cmd.Flags().GetBool("color")
	width, _ := cmd.Flags().GetInt("width")

	if projectPath == "" {
		return fmt.Errorf("--project (-p) is required")
	}

	if ref == "" {
		ref = "HEAD"
	}

	exec, logger := buildExec("subcommand", cmd.OutOrStdout())
	defer logger.Close()
	defer exec.Close()

	connectProg, _ := visitor.Build(fmt.Sprintf("CONNECT LOCAL '%s'", projectPath))
	for _, stmt := range connectProg.Statements {
		if err := exec.Execute(stmt); err != nil {
			return fmt.Errorf("connecting: %w", err)
		}
	}

	opts := executor.DiffOptions{
		Format:   executor.DiffFormat(format),
		UseColor: useColor,
		Width:    width,
	}

	if err := exec.DiffLocal(ref, opts); err != nil {
		return fmt.Errorf("diff-local: %w", err)
	}
	return nil
}

var diffLocalCmd = &cobra.Command{
	Use:   "diff-local",
	Short: "Compare local changes against git",
	Long: `Compare local (uncommitted) changes in mxunit files against a git reference.

This command finds modified mxunit files in the mprcontents/ folder and shows
the differences as MDL. Only works with MPR v2 format (Mendix 10.18+).

The --ref flag accepts any git ref or range (e.g., HEAD, main, main..feature-branch).

Examples:
  # Show uncommitted changes vs HEAD
  mxcli diff-local -p app.mpr

  # Compare against a specific commit
  mxcli diff-local -p app.mpr --ref HEAD~1

  # Compare against a branch
  mxcli diff-local -p app.mpr --ref main

  # Compare two arbitrary revisions (git range syntax)
  mxcli diff-local -p app.mpr --ref main..feature-branch

  # Three-dot range (changes since common ancestor)
  mxcli diff-local -p app.mpr --ref main...feature-branch

  # With structural format
  mxcli diff-local -p app.mpr --format struct --color
`,
	RunE: runDiffLocal,
}

var diffLocalSubCmd = &cobra.Command{
	Use:   "local",
	Short: "Compare local changes against git",
	Long: `Compare local (uncommitted) changes in mxunit files against a git reference.

This command finds modified mxunit files in the mprcontents/ folder and shows
the differences as MDL. Only works with MPR v2 format (Mendix 10.18+).

Examples:
  mxcli diff local -p app.mpr
  mxcli diff local -p app.mpr --ref main
`,
	RunE: runDiffLocal,
}

var diffBsonCompareCmd = &cobra.Command{
	Use:   "bson-compare [name1] [name2]",
	Short: "Compare two BSON objects for differences",
	Long: `Compare two BSON objects from Mendix projects and display a structured diff.

Supports same-project and cross-project comparison. By default, structural
and layout fields ($ID, PersistentId, RelativeMiddlePoint, Size) are skipped.

Examples:
  # Compare two workflows in the same project
  mxcli diff bson-compare -p app.mpr --type workflow WF1 WF2

  # Compare same workflow across two MPR files
  mxcli diff bson-compare -p app.mpr -p2 other.mpr --type workflow MyWorkflow

  # Include structural fields in comparison
  mxcli diff bson-compare -p app.mpr --type workflow --all WF1 WF2
`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runBsonCompare,
}

func runBsonCompare(cmd *cobra.Command, args []string) error {
	projectPath, _ := cmd.Flags().GetString("project")
	secondProject, _ := cmd.Flags().GetString("p2")
	objectType, _ := cmd.Flags().GetString("type")
	includeAll, _ := cmd.Flags().GetBool("all")

	if projectPath == "" {
		return fmt.Errorf("--project (-p) is required")
	}

	reader1, err := mmpr.Open(projectPath)
	if err != nil {
		return fmt.Errorf("opening project: %w", err)
	}
	defer reader1.Close()

	var leftName, rightName string
	var reader2 *mmpr.Reader

	switch len(args) {
	case 2:
		leftName = args[0]
		rightName = args[1]
		if secondProject != "" {
			reader2, err = mmpr.Open(secondProject)
			if err != nil {
				return fmt.Errorf("opening second project: %w", err)
			}
			defer reader2.Close()
		}
	case 1:
		if secondProject == "" {
			return fmt.Errorf("provide two names, or one name with -p2 for cross-MPR comparison")
		}
		leftName = args[0]
		rightName = args[0]
		reader2, err = mmpr.Open(secondProject)
		if err != nil {
			return fmt.Errorf("opening second project: %w", err)
		}
		defer reader2.Close()
	}

	rightReader := reader1
	if reader2 != nil {
		rightReader = reader2
	}

	leftUnit, err := reader1.GetRawUnitByName(objectType, leftName)
	if err != nil {
		return fmt.Errorf("getting %s: %w", leftName, err)
	}

	rightUnit, err := rightReader.GetRawUnitByName(objectType, rightName)
	if err != nil {
		return fmt.Errorf("getting %s: %w", rightName, err)
	}

	format, _ := cmd.Flags().GetString("format")
	if format == "ndsl" {
		var leftDocD, rightDocD bson.D
		if err := bson.Unmarshal(leftUnit.Contents, &leftDocD); err != nil {
			return fmt.Errorf("parsing BSON for %s: %w", leftName, err)
		}
		if err := bson.Unmarshal(rightUnit.Contents, &rightDocD); err != nil {
			return fmt.Errorf("parsing BSON for %s: %w", rightName, err)
		}
		fmt.Printf("=== LEFT: %s ===\n%s\n\n=== RIGHT: %s ===\n%s\n",
			leftName, bsondebug.Render(leftDocD, 0),
			rightName, bsondebug.Render(rightDocD, 0))
		return nil
	}

	var leftDoc, rightDoc bson.M
	if err := bson.Unmarshal(leftUnit.Contents, &leftDoc); err != nil {
		return fmt.Errorf("parsing BSON for %s: %w", leftName, err)
	}
	if err := bson.Unmarshal(rightUnit.Contents, &rightDoc); err != nil {
		return fmt.Errorf("parsing BSON for %s: %w", rightName, err)
	}

	opts := bsondebug.CompareOptions{IncludeAll: includeAll}
	diffs := bsondebug.Compare(leftDoc, rightDoc, opts)

	typeName := leftDoc["$Type"]
	if typeName != nil {
		fmt.Println(typeName)
	}
	if leftName == rightName {
		fmt.Printf("  Comparing: %s (across two MPRs)\n\n", leftName)
	} else {
		fmt.Printf("  Comparing: %s vs %s\n\n", leftName, rightName)
	}

	fmt.Println(bsondebug.FormatDiffs(diffs))
	return nil
}

var diffBsonDiffCmd = &cobra.Command{
	Use:   "bson-diff <file1.mxunit> <file2.mxunit>",
	Short: "Show git-diff style NDSL diff between two .mxunit files",
	Long: `Read two raw .mxunit BSON files, render both as NDSL, and output a
unified diff in git-diff style. No project file required.

Exit codes:
  0  files are identical
  1  files differ
  2  error (file not found, invalid BSON)

Examples:
  mxcli diff bson-diff reference.mxunit generated.mxunit
  mxcli diff bson-diff a.mxunit b.mxunit --no-color
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
	diffCmd.AddCommand(diffScriptCmd)
	diffCmd.AddCommand(diffLocalSubCmd)
	diffCmd.AddCommand(diffBsonCompareCmd)
	diffCmd.AddCommand(diffBsonDiffCmd)

	diffScriptCmd.Flags().StringP("format", "f", "unified", "Output format: unified, side, struct")
	diffScriptCmd.Flags().BoolP("color", "", false, "Use colored output")
	diffScriptCmd.Flags().IntP("width", "w", 120, "Terminal width for side-by-side format")

	diffLocalSubCmd.Flags().StringP("ref", "r", "HEAD", "Git ref or range (e.g., HEAD, main, main..feature)")
	diffLocalSubCmd.Flags().StringP("format", "f", "unified", "Output format: unified, side, struct")
	diffLocalSubCmd.Flags().BoolP("color", "", false, "Use colored output")
	diffLocalSubCmd.Flags().IntP("width", "w", 120, "Terminal width for side-by-side format")

	diffLocalCmd.Flags().StringP("ref", "r", "HEAD", "Git ref or range (e.g., HEAD, main, main..feature)")
	diffLocalCmd.Flags().StringP("format", "f", "unified", "Output format: unified, side, struct")
	diffLocalCmd.Flags().BoolP("color", "", false, "Use colored output")
	diffLocalCmd.Flags().IntP("width", "w", 120, "Terminal width for side-by-side format")

	diffBsonCompareCmd.Flags().StringP("project", "p", "", "Path to first MPR project (required)")
	diffBsonCompareCmd.Flags().String("p2", "", "Path to second MPR project (for cross-MPR comparison)")
	diffBsonCompareCmd.Flags().String("type", "workflow", "Object type: workflow, page, microflow, nanoflow, enumeration, snippet, layout")
	diffBsonCompareCmd.Flags().Bool("all", false, "Include structural/layout fields ($ID, PersistentId, etc.)")
	diffBsonCompareCmd.Flags().String("format", "diff", "Output format: diff, ndsl")

	diffBsonDiffCmd.Flags().Bool("color", false, "Force color output")
	diffBsonDiffCmd.Flags().Bool("no-color", false, "Disable color output")
}
