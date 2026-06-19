// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/executor"
	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check <file1> [<file2> ...]",
	Short: "Check MDL scripts for errors without executing them",
	Long: `Check one or more MDL script files for syntax errors and optionally validate references.

Without --references, only performs syntax and static validation (no project required).
With -p and --references, also validates cross-file references against a project.

Reference validation is smart: it automatically skips references to objects
that are created within the script itself.

Output includes structured rule IDs (MDL prefix) for each validation issue.

Examples:
  # Check a single file
  mxcli check script.mdl

  # Check multiple files
  mxcli check file1.mdl file2.mdl file3.mdl

  # Check with reference validation against a project
  mxcli check script.mdl -p app.mpr --references

  # JSON output
  mxcli check script.mdl --format json
`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath, _ := cmd.Flags().GetString("project")
		checkRefs, _ := cmd.Flags().GetBool("references")
		format := resolveFormat(cmd, "text")
		isStructured := format != "" && format != "text"

		out := cmd.OutOrStdout()
		errOut := cmd.ErrOrStderr()

		outputFormat := linter.OutputFormat(format)
		formatter := linter.GetFormatter(outputFormat, !isStructured)

		var aggregateErr error
		multiFile := len(args) > 1
		overallStart := time.Now()
		overallStmts := 0

		for fi, filePath := range args {
			fileStart := time.Now()

			if !isStructured && multiFile {
				fmt.Fprintf(out, "\n--- %s ---\n", filePath)
			}

			// Read the file
			readStart := time.Now()
			content, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("reading file %s: %w", filePath, err)
			}
			readTime := time.Since(readStart)

			// Parse
			parseStart := time.Now()
			if !isStructured && !multiFile {
				fmt.Fprintf(out, "Checking syntax: %s\n", filePath)
			}
			prog, errs := visitor.Build(string(content))
			parseTime := time.Since(parseStart)

			if len(errs) > 0 {
				if isStructured {
					var parseViolations []linter.Violation
					for _, parseErr := range errs {
						parseViolations = append(parseViolations, linter.Violation{
							RuleID:   "MDL-SYNTAX",
							Severity: linter.SeverityError,
							Message:  parseErr.Error(),
						})
					}
					formatter.Format(parseViolations, errOut)
				} else {
					fmt.Fprintf(errOut, "Syntax errors found:\n")
					for _, parseErr := range errs {
						fmt.Fprintf(errOut, "  - %v\n", parseErr)
					}
					src := string(content)
					if (strings.Contains(src, "IMPORT") || strings.Contains(src, "import")) &&
						(strings.Contains(src, "QUERY") || strings.Contains(src, "query")) &&
						strings.Contains(src, "$") && !strings.Contains(src, "$$") {
						fmt.Fprintf(errOut, "\nHint: SQL queries in IMPORT statements should use dollar-quoting ($$...$$) instead of single quotes.\n")
						fmt.Fprintf(errOut, "  Example: IMPORT FROM alias QUERY $$SELECT * FROM table$$ INTO Module.Entity MAP (...)\n")
					}
				}
				if aggregateErr == nil {
					aggregateErr = fmt.Errorf("check failed for %d file(s)", len(args)-fi)
				}
				continue
			}
			stmtCount := len(prog.Statements)
			overallStmts += stmtCount

			// Validate statements
			var violations []linter.Violation
			for _, stmt := range prog.Statements {
				if enumStmt, ok := stmt.(*ast.CreateEnumerationStmt); ok {
					violations = append(violations, executor.ValidateEnumeration(enumStmt)...)
				}
				if entityStmt, ok := stmt.(*ast.CreateEntityStmt); ok {
					violations = append(violations, executor.ValidateEntity(entityStmt)...)
				}
				if mfStmt, ok := stmt.(*ast.CreateMicroflowStmt); ok {
					violations = append(violations, executor.ValidateMicroflow(mfStmt)...)
				}
				if viewStmt, ok := stmt.(*ast.CreateViewEntityStmt); ok {
					if viewStmt.Query.RawQuery != "" {
						violations = append(violations, executor.ValidateOQLSyntax(viewStmt.Query.RawQuery)...)
						violations = append(violations, executor.ValidateOQLTypes(viewStmt.Query.RawQuery, viewStmt.Attributes)...)
					}
				}
			}

			validateStart := time.Now()
			violations = append(violations, executor.CheckScriptDuplicates(prog)...)
			validateTime := time.Since(validateStart)

			if isStructured {
				formatter.Format(violations, errOut)
			} else if len(violations) > 0 {
				fmt.Fprintln(errOut)
				formatter.Format(violations, errOut)
			}

			hasFatal := false
			if len(violations) > 0 {
				summary := linter.Summarize(violations)
				if summary.Errors > 0 {
					hasFatal = true
				}
			}

			// Reference checking
			if checkRefs && !hasFatal {
				if projectPath == "" {
					return fmt.Errorf("--project (-p) is required for reference checking")
				}
				if !isStructured && !multiFile {
					fmt.Fprintf(out, "\nValidating references against: %s\n", projectPath)
				}
				exec, logger := buildExec("check", out)
				defer logger.Close()
				defer exec.Close()

				connectProg, _ := visitor.Build(fmt.Sprintf("CONNECT LOCAL '%s'", projectPath))
				for _, stmt := range connectProg.Statements {
					if err := exec.Execute(stmt); err != nil {
						return fmt.Errorf("connecting to project: %w", err)
					}
				}

				validationErrors := exec.ValidateProgram(prog)
				validationErrors = append(validationErrors, exec.CheckProjectConflicts(prog)...)

				if len(validationErrors) > 0 {
					if isStructured {
						var refViolations []linter.Violation
						for _, vErr := range validationErrors {
							refViolations = append(refViolations, linter.Violation{
								RuleID:   "MDL-REF",
								Severity: linter.SeverityError,
								Message:  vErr.Error(),
							})
						}
						formatter.Format(refViolations, errOut)
					} else {
						fmt.Fprintf(errOut, "Reference errors:\n")
						for _, vErr := range validationErrors {
							fmt.Fprintf(errOut, "  %v\n", vErr)
						}
					}
					hasFatal = true
				}
			}

			if !isStructured {
				if hasFatal {
					fmt.Fprintf(errOut, "  ✗ failed\n")
				} else {
					fmt.Fprintf(out, "  ✓ Check passed! (%d statements)\n", stmtCount)
				}
			}

			if hasFatal {
				if aggregateErr == nil {
					aggregateErr = fmt.Errorf("check failed for %d file(s)", len(args)-fi)
				}
			}

			if !isStructured {
				elapsed := time.Since(fileStart)
				fmt.Fprintf(out, "  ─ Performance: read %v, parse %v, validate %v, total %v ─\n",
					roundDuration(readTime), roundDuration(parseTime),
					roundDuration(validateTime), roundDuration(elapsed))
			}
		}

		if !isStructured && multiFile {
			overallTime := time.Since(overallStart)
			if aggregateErr == nil {
				fmt.Fprintf(out, "\n✓ All checks passed! (%d statements in %d files, %s)\n",
					overallStmts, len(args), roundDuration(overallTime))
			} else {
				fmt.Fprintf(errOut, "\n✗ %d files checked, some failed (%s)\n",
					len(args), roundDuration(overallTime))
			}
		}
		return aggregateErr
	},
}

func roundDuration(d time.Duration) time.Duration {
	return d.Round(time.Microsecond)
}

func accessGapDesc(gt executor.GapType) string {
	switch gt {
	case executor.GapEntityRead:
		return "read"
	case executor.GapMFExecute:
		return "execute"
	default:
		return "access"
	}
}
