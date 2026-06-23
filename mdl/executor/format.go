// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// OutputFormat controls how command results are rendered.
type OutputFormat string

const (
	// FormatTable renders results as a pipe-delimited markdown table (default).
	FormatTable OutputFormat = "table"
	// FormatJSON renders results as a JSON array of objects.
	FormatJSON OutputFormat = "json"
)

// TableResult holds structured tabular data that can be rendered in multiple formats.
type TableResult struct {
	Columns []string // column headers
	Rows    [][]any  // row data (one slice per row, matching Columns order)
	Summary string   // optional summary line, e.g. "(42 entities)"
}

// writeResult renders a TableResult to ctx.Output in the current format.
func writeResult(ctx *ExecContext, r *TableResult) error {
	return writeResultTo(ctx.Output, ctx.Format, r)
}

// writeResultTo renders a TableResult to the given writer in the given format.
func writeResultTo(output io.Writer, format OutputFormat, r *TableResult) error {
	if format == FormatJSON {
		return writeResultJSONTo(output, r)
	}
	writeResultTableTo(output, r)
	return nil
}

// writeResultTable renders a TableResult as a pipe-delimited markdown table.
func writeResultTable(ctx *ExecContext, r *TableResult) {
	writeResultTableTo(ctx.Output, r)
}

// writeResultTableTo renders a TableResult as a pipe-delimited markdown table.
func writeResultTableTo(output io.Writer, r *TableResult) {
	if len(r.Columns) == 0 {
		return
	}

	widths := make([]int, len(r.Columns))
	for i, col := range r.Columns {
		widths[i] = len(col)
	}
	for _, row := range r.Rows {
		for i, val := range row {
			if i >= len(widths) {
				break
			}
			s := formatCellValue(val)
			if len(s) > widths[i] {
				widths[i] = len(s)
			}
		}
	}

	fmt.Fprint(output, "|")
	for i, col := range r.Columns {
		fmt.Fprintf(output, " %-*s |", widths[i], col)
	}
	fmt.Fprintln(output)

	fmt.Fprint(output, "|")
	for _, w := range widths {
		fmt.Fprintf(output, "-%s-|", strings.Repeat("-", w))
	}
	fmt.Fprintln(output)

	for _, row := range r.Rows {
		fmt.Fprint(output, "|")
		for i := range r.Columns {
			var s string
			if i < len(row) {
				s = formatCellValue(row[i])
			}
			fmt.Fprintf(output, " %-*s |", widths[i], s)
		}
		fmt.Fprintln(output)
	}

	if r.Summary != "" {
		fmt.Fprintf(output, "\n%s\n", r.Summary)
	}
}

// writeResultJSON renders a TableResult as a JSON array of objects.
func writeResultJSON(ctx *ExecContext, r *TableResult) error {
	return writeResultJSONTo(ctx.Output, r)
}

// writeResultJSONTo renders a TableResult as a JSON array of objects.
func writeResultJSONTo(output io.Writer, r *TableResult) error {
	objects := make([]map[string]any, 0, len(r.Rows))
	for _, row := range r.Rows {
		obj := make(map[string]any, len(r.Columns))
		for i, col := range r.Columns {
			if i < len(row) {
				obj[col] = row[i]
			}
		}
		objects = append(objects, obj)
	}

	enc := json.NewEncoder(output)
	enc.SetIndent("", "  ")
	return enc.Encode(objects)
}

// writeDescribeJSON wraps a describe handler's output in a JSON envelope.
// In table/text mode it calls fn directly. In JSON mode it captures fn's output
// and wraps it as {"name": ..., "type": ..., "mdl": ...}.
func writeDescribeJSON(ctx context.Context, name, objectType string, deps *HandlerDeps, fn func() error) error {
	if deps.Format != FormatJSON {
		return fn()
	}

	// Capture fn's text output by temporarily swapping deps.Output.
	var buf bytes.Buffer
	origOutput := deps.Output
	deps.Output = &buf

	err := fn()

	deps.Output = origOutput
	if err != nil {
		return err
	}

	result := map[string]any{
		"name": name,
		"type": objectType,
		"mdl":  buf.String(),
	}
	enc := json.NewEncoder(deps.Output)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// writeDescribeJSONFuture is the ExecContext-free version of writeDescribeJSON.
// In JSON mode it captures fn's text output and wraps it as
// {"name": ..., "type": ..., "mdl": ...}.  In text mode it calls fn directly.
func writeDescribeJSONFuture(output io.Writer, format OutputFormat, name, objectType string, fn func(io.Writer) error) error {
	if format != FormatJSON {
		return fn(output)
	}

	var buf bytes.Buffer
	err := fn(&buf)
	if err != nil {
		return err
	}

	result := map[string]any{
		"name": name,
		"type": objectType,
		"mdl":  buf.String(),
	}
	enc := json.NewEncoder(output)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// ----------------------------------------------------------------------------
// Executor method wrappers (for callers in unmigrated files)
// ----------------------------------------------------------------------------

func (e *Executor) writeResult(r *TableResult) error {
	return writeResult(e.newExecContext(context.Background()), r)
}

// formatCellValue formats a value for table cell display.
func formatCellValue(val any) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}
