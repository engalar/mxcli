// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteResultTable(t *testing.T) {
	var buf bytes.Buffer
	e := &Executor{output: &buf}

	r := &TableResult{
		Columns: []string{"Name", "Count"},
		Rows: [][]any{
			{"Alice", 10},
			{"Bob", 5},
		},
		Summary: "(2 items)",
	}
	if err := e.writeResult(r); err != nil {
		t.Fatalf("writeResult: %v", err)
	}

	out := buf.String()

	// Header row
	if !strings.Contains(out, "| Name") {
		t.Errorf("expected header with Name, got: %s", out)
	}
	if !strings.Contains(out, "| Count") {
		t.Errorf("expected header with Count, got: %s", out)
	}

	// Data rows
	if !strings.Contains(out, "Alice") {
		t.Errorf("expected Alice in output, got: %s", out)
	}
	if !strings.Contains(out, "Bob") {
		t.Errorf("expected Bob in output, got: %s", out)
	}

	// Separator
	if !strings.Contains(out, "|---") {
		t.Errorf("expected separator row, got: %s", out)
	}

	// Summary
	if !strings.Contains(out, "(2 items)") {
		t.Errorf("expected summary, got: %s", out)
	}
}

func TestWriteResultJSON(t *testing.T) {
	var buf bytes.Buffer
	e := &Executor{output: &buf, format: FormatJSON}

	r := &TableResult{
		Columns: []string{"Name", "Count"},
		Rows: [][]any{
			{"Alice", 10},
			{"Bob", 5},
		},
		Summary: "(2 items)",
	}
	if err := e.writeResult(r); err != nil {
		t.Fatalf("writeResult: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid json: %v\noutput: %s", err, buf.String())
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result))
	}
	if result[0]["Name"] != "Alice" {
		t.Errorf("expected Alice, got %v", result[0]["Name"])
	}
	// JSON numbers decode as float64
	if result[0]["Count"] != float64(10) {
		t.Errorf("expected 10, got %v", result[0]["Count"])
	}
	if result[1]["Name"] != "Bob" {
		t.Errorf("expected Bob, got %v", result[1]["Name"])
	}
}

func TestWriteResultJSONEmpty(t *testing.T) {
	var buf bytes.Buffer
	e := &Executor{output: &buf, format: FormatJSON}

	r := &TableResult{
		Columns: []string{"Name"},
	}
	if err := e.writeResult(r); err != nil {
		t.Fatalf("writeResult: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty array, got %d items", len(result))
	}
}

func TestWriteDescribeJSON(t *testing.T) {
	var buf bytes.Buffer
	ctx := &ExecContext{
		ExecIO: ExecIO{Output: &buf, Format: FormatJSON},
	}

	err := writeDescribeJSON(ctx, "Sales.Customer", "entity", func() error {
		_, err := ctx.Output.Write([]byte("create entity Sales.Customer;\n"))
		return err
	})
	if err != nil {
		t.Fatalf("writeDescribeJSON: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid json: %v\noutput: %s", err, buf.String())
	}

	if result["name"] != "Sales.Customer" {
		t.Errorf("expected name Sales.Customer, got %v", result["name"])
	}
	if result["type"] != "entity" {
		t.Errorf("expected type entity, got %v", result["type"])
	}
	if !strings.Contains(result["mdl"].(string), "create entity") {
		t.Errorf("expected mdl to contain create entity, got %v", result["mdl"])
	}
}

func TestWriteDescribeJSONPassthrough(t *testing.T) {
	var buf bytes.Buffer
	ctx := &ExecContext{
		ExecIO: ExecIO{Output: &buf, Format: FormatTable},
	}

	err := writeDescribeJSON(ctx, "Sales.Customer", "entity", func() error {
		_, err := ctx.Output.Write([]byte("create entity Sales.Customer;\n"))
		return err
	})
	if err != nil {
		t.Fatalf("writeDescribeJSON: %v", err)
	}

	// In table mode, should pass through directly
	out := buf.String()
	if out != "create entity Sales.Customer;\n" {
		t.Errorf("expected passthrough, got: %s", out)
	}
}
