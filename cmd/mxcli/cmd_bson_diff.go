// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/term"
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
	return renderNDSL(doc, 0), nil
}

// renderNDSL converts a bson.D document to NDSL text using "=" field separators.
func renderNDSL(doc bson.D, indent int) string {
	var sb strings.Builder
	renderNDSLDoc(&sb, doc, indent)
	return strings.TrimRight(sb.String(), "\n")
}

func renderNDSLDoc(sb *strings.Builder, doc bson.D, indent int) {
	pad := strings.Repeat("  ", indent)

	typeName := ""
	for _, e := range doc {
		if e.Key == "$Type" {
			typeName, _ = e.Value.(string)
			break
		}
	}
	if typeName != "" {
		sb.WriteString(pad + typeName + "\n")
	}

	renderNDSLFields(sb, doc, indent+1)
}

func renderNDSLFields(sb *strings.Builder, doc bson.D, indent int) {
	type field struct {
		key string
		val any
	}
	var fields []field
	for _, e := range doc {
		if e.Key == "$ID" || e.Key == "$Type" {
			continue
		}
		fields = append(fields, field{e.Key, e.Value})
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].key < fields[j].key
	})
	for _, f := range fields {
		renderNDSLField(sb, f.key, f.val, indent)
	}
}

func renderNDSLField(sb *strings.Builder, key string, val any, indent int) {
	pad := strings.Repeat("  ", indent)

	switch v := val.(type) {
	case nil:
		fmt.Fprintf(sb, "%s%s = null\n", pad, key)
	case bson.Binary:
		fmt.Fprintf(sb, "%s%s = <uuid>\n", pad, key)
	case bson.D:
		typeName := ""
		for _, e := range v {
			if e.Key == "$Type" {
				typeName, _ = e.Value.(string)
				break
			}
		}
		if typeName != "" {
			fmt.Fprintf(sb, "%s%s = %s\n", pad, key, typeName)
		} else {
			fmt.Fprintf(sb, "%s%s =\n", pad, key)
		}
		renderNDSLFields(sb, v, indent+1)
	case bson.A:
		renderNDSLArray(sb, key, v, indent)
	case string:
		fmt.Fprintf(sb, "%s%s = %q\n", pad, key, v)
	case bool:
		fmt.Fprintf(sb, "%s%s = %v\n", pad, key, v)
	default:
		fmt.Fprintf(sb, "%s%s = %v\n", pad, key, v)
	}
}

func renderNDSLArray(sb *strings.Builder, key string, arr bson.A, indent int) {
	pad := strings.Repeat("  ", indent)

	markerStr := ""
	startIdx := 0
	if len(arr) > 0 {
		if marker, ok := arr[0].(int32); ok {
			markerStr = fmt.Sprintf(" [marker=%d]", marker)
			startIdx = 1
		}
	}

	elements := arr[startIdx:]
	if len(elements) == 0 {
		fmt.Fprintf(sb, "%s%s%s = []\n", pad, key, markerStr)
		return
	}

	fmt.Fprintf(sb, "%s%s%s:\n", pad, key, markerStr)
	for _, elem := range elements {
		renderNDSLArrayElement(sb, elem, indent+1)
	}
}

func renderNDSLArrayElement(sb *strings.Builder, elem any, indent int) {
	pad := strings.Repeat("  ", indent)

	switch v := elem.(type) {
	case bson.D:
		typeName := ""
		for _, e := range v {
			if e.Key == "$Type" {
				typeName, _ = e.Value.(string)
				break
			}
		}
		if typeName != "" {
			fmt.Fprintf(sb, "%s- %s\n", pad, typeName)
		} else {
			fmt.Fprintf(sb, "%s-\n", pad)
		}
		renderNDSLFields(sb, v, indent+2)
	case string:
		fmt.Fprintf(sb, "%s- %q\n", pad, v)
	default:
		fmt.Fprintf(sb, "%s- %v\n", pad, elem)
	}
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
