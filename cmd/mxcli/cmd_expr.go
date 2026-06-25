// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/repair"
	"github.com/mendixlabs/mxcli/internal/expr/report"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/internal/expr/typecheck"
	"github.com/mendixlabs/mxcli/internal/expr/validate"
	"github.com/spf13/cobra"
)

// ── flags shared across expr sub-commands ────────────────────────────────────

var exprFilterType string
var exprFormat string
var exprSeverity string
var exprSummary bool

// ── root: mxcli expr ─────────────────────────────────────────────────────────

var exprCmd = &cobra.Command{
	Use:   "expr",
	Short: "Mendix expression miner, parser, validator and repair suggester",
	Long: `Collect, parse, validate and repair Mendix expressions from mprcontents/ directories.

Sub-commands:
  scan      – collect raw expressions from BSON .mxunit files
  parse     – parse each expression with the exprcheck parser
  validate  – apply SYN/SEM validation rules
  repair    – suggest ranked fixes for validation issues
  report    – full pipeline → HTML/JSON/text summary

Examples:
  mxcli expr scan   mendix-app/mprcontents --summary
  mxcli expr report mendix-app/mprcontents --format html > report.html
  mxcli expr validate -p app.mpr --severity ERROR`,
}

// ── mxcli expr scan ──────────────────────────────────────────────────────────

var exprScanCmd = &cobra.Command{
	Use:   "scan <mprcontents>...",
	Short: "Collect expression strings from BSON .mxunit files",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := collectRecords(args)
		if err != nil {
			return err
		}
		if exprSummary {
			printExprSummary(records)
			return nil
		}
		enc := json.NewEncoder(os.Stdout)
		for _, r := range records {
			if err := enc.Encode(r); err != nil {
				return err
			}
		}
		return nil
	},
}

// ── mxcli expr parse ─────────────────────────────────────────────────────────

var exprParseCmd = &cobra.Command{
	Use:   "parse <mprcontents>...",
	Short: "Parse collected expressions with the exprcheck parser",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := collectRecords(args)
		if err != nil {
			return err
		}
		results := parse.BatchParse(records)
		enc := json.NewEncoder(os.Stdout)
		for _, r := range results {
			if err := enc.Encode(r); err != nil {
				return err
			}
		}
		return nil
	},
}

// ── mxcli expr validate ───────────────────────────────────────────────────────

var exprValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Apply SYN/SEM validation rules to expressions in a project",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		mprPath, _ := cmd.Root().PersistentFlags().GetString("project")
		if mprPath == "" {
			return fmt.Errorf("requires -p project.mpr")
		}
		return runExprValidate(scan.MprContentsPath(mprPath))
	},
}

func runExprValidate(mprContentsPath string) error {
	records, err := scan.ScanMprcontents(mprContentsPath, scan.Options{FilterType: exprFilterType})
	if err != nil {
		return err
	}
	parsed := parse.BatchParse(records)
	checker := typecheck.NewChecker(nil) // nil index: structural checks skipped without index
	var issues []validate.ValidationResult
	for _, pr := range parsed {
		issues = append(issues, validate.ValidateSyntax(pr)...)
		issues = append(issues, checker.Check(pr)...)
		issues = append(issues, checker.CheckStructural(pr)...)
	}
	out, err := report.Render(issues, report.Options{Format: exprFormat, Severity: exprSeverity})
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(out)
	return err
}



// ── mxcli expr repair ────────────────────────────────────────────────────────

var exprRepairCmd = &cobra.Command{
	Use:   "repair <mprcontents>...",
	Short: "Suggest ranked repairs for validation issues",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := collectRecords(args)
		if err != nil {
			return err
		}
		parsed := parse.BatchParse(records)
		enc := json.NewEncoder(os.Stdout)
		for _, pr := range parsed {
			for _, issue := range validate.ValidateSyntax(pr) {
				if exprSeverity != "" && issue.Severity != exprSeverity {
					continue
				}
				if sugs := repair.Suggest(issue); len(sugs) > 0 {
					if err := enc.Encode(sugs); err != nil {
						return err
					}
				}
			}
		}
		return nil
	},
}

// ── mxcli expr report ────────────────────────────────────────────────────────

var exprReportCmd = &cobra.Command{
	Use:   "report <mprcontents>...",
	Short: "Full pipeline: scan → parse → validate → HTML/JSON/text report",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		records, err := collectRecords(args)
		if err != nil {
			return err
		}
		parsed := parse.BatchParse(records)
		var issues []validate.ValidationResult
		for _, pr := range parsed {
			issues = append(issues, validate.ValidateSyntax(pr)...)
		}
		out, err := report.Render(issues, report.Options{Format: exprFormat, Severity: exprSeverity})
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(out)
		return err
	},
}

// ── helpers ──────────────────────────────────────────────────────────────────

func collectRecords(paths []string) ([]scan.ExprRecord, error) {
	var all []scan.ExprRecord
	for _, p := range paths {
		recs, err := scan.ScanMprcontents(p, scan.Options{FilterType: exprFilterType})
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", p, err)
		}
		all = append(all, recs...)
	}
	return all, nil
}

func printExprSummary(recs []scan.ExprRecord) {
	counts := map[string]int{}
	for _, r := range recs {
		counts[r.UnitType]++
	}
	fmt.Printf("Total: %d expressions\n\n", len(recs))
	// Print in descending order (simple selection sort for small maps)
	type kv struct {
		k string
		v int
	}
	var sorted []kv
	for k, v := range counts {
		sorted = append(sorted, kv{k, v})
	}
	for i := range sorted {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].v > sorted[i].v {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	for _, item := range sorted {
		fmt.Printf("  %5d  %s\n", item.v, item.k)
	}
}

// ── init: register with rootCmd ───────────────────────────────────────────────

func init() {
	// shared flags for all expr sub-commands
	for _, cmd := range []*cobra.Command{exprScanCmd, exprParseCmd, exprValidateCmd, exprRepairCmd, exprReportCmd} {
		cmd.Flags().StringVar(&exprFilterType, "filter", "", "Filter by unit_type substring (case-insensitive)")
		cmd.Flags().StringVar(&exprSeverity, "severity", "", "Filter by severity: ERROR | WARNING | INFO")
	}
	for _, cmd := range []*cobra.Command{exprValidateCmd, exprReportCmd} {
		cmd.Flags().StringVar(&exprFormat, "format", "json", "Output format: json | html | text")
	}
	exprScanCmd.Flags().BoolVar(&exprSummary, "summary", false, "Print human-readable stats instead of JSONL")

	exprCmd.AddCommand(exprScanCmd)
	exprCmd.AddCommand(exprParseCmd)
	exprCmd.AddCommand(exprValidateCmd)
	exprCmd.AddCommand(exprRepairCmd)
	exprCmd.AddCommand(exprReportCmd)
	rootCmd.AddCommand(exprCmd)
}
