// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFormatOQLTable(t *testing.T) {
	t.Parallel()
	result := &OQLResult{
		Columns: []string{"Name", "Age", "City"},
		Rows: [][]any{
			{"Alice", "30", "Amsterdam"},
			{"Bob", "25", "Berlin"},
			{nil, "42", "Copenhagen"},
		},
	}

	var buf bytes.Buffer
	FormatOQLTable(&buf, result)
	output := buf.String()

	// Check header
	if !strings.Contains(output, "| Name") {
		t.Errorf("missing Name header in:\n%s", output)
	}
	if !strings.Contains(output, "| Age") {
		t.Errorf("missing Age header in:\n%s", output)
	}
	if !strings.Contains(output, "| City") {
		t.Errorf("missing City header in:\n%s", output)
	}

	// Check separator
	if !strings.Contains(output, "|---") {
		t.Errorf("missing separator in:\n%s", output)
	}

	// Check data
	if !strings.Contains(output, "Alice") {
		t.Errorf("missing Alice in:\n%s", output)
	}
	if !strings.Contains(output, "NULL") {
		t.Errorf("nil should be displayed as NULL in:\n%s", output)
	}
}

func TestFormatOQLTableEmpty(t *testing.T) {
	t.Parallel()
	result := &OQLResult{
		Columns: []string{"Name"},
		Rows:    nil,
	}

	var buf bytes.Buffer
	FormatOQLTable(&buf, result)
	output := buf.String()

	// Should have header and separator but no data rows
	if !strings.Contains(output, "Name") {
		t.Errorf("missing header in:\n%s", output)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines (header + separator), got %d:\n%s", len(lines), output)
	}
}

func TestFormatOQLTableNoColumns(t *testing.T) {
	t.Parallel()
	result := &OQLResult{}

	var buf bytes.Buffer
	FormatOQLTable(&buf, result)
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty result, got %q", buf.String())
	}
}

func TestFormatOQLJSON(t *testing.T) {
	t.Parallel()
	result := &OQLResult{
		Columns: []string{"Name", "Count"},
		Rows: [][]any{
			{"Alice", "10"},
			{"Bob", nil},
		},
	}

	var buf bytes.Buffer
	if err := FormatOQLJSON(&buf, result); err != nil {
		t.Fatalf("FormatOQLJSON: %v", err)
	}

	var parsed []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}

	if len(parsed) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(parsed))
	}
	if parsed[0]["Name"] != "Alice" {
		t.Errorf("first row Name: got %v, want Alice", parsed[0]["Name"])
	}
	if parsed[1]["Count"] != nil {
		t.Errorf("second row Count should be nil, got %v", parsed[1]["Count"])
	}
}

func TestFormatOQLJSONEmpty(t *testing.T) {
	t.Parallel()
	result := &OQLResult{
		Columns: []string{"Name"},
		Rows:    nil,
	}

	var buf bytes.Buffer
	if err := FormatOQLJSON(&buf, result); err != nil {
		t.Fatalf("FormatOQLJSON: %v", err)
	}

	var parsed []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed) != 0 {
		t.Errorf("expected empty array, got %d objects", len(parsed))
	}
}

func TestExecuteOQL_Success(t *testing.T) {
	t.Parallel()
	expectedAuth := m2eeAuthHeader("testpass")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("X-M2EE-Authentication") != expectedAuth {
			t.Errorf("wrong auth header: got %q, want %q", r.Header.Get("X-M2EE-Authentication"), expectedAuth)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("wrong content type: %s", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["action"] != "preview_execute_oql" {
			t.Errorf("wrong action: %v", body["action"])
		}

		// Return success response matching M2EE feedback envelope format
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":0,"feedback":{"data":[{"Name":"Alice","Age":"30"},{"Name":"Bob","Age":"25"}]}}`))
	}))
	defer server.Close()

	// Parse server URL to get host and port
	host, port := parseTestServerAddr(t, server.URL)

	opts := OQLOptions{
		Host:   host,
		Port:   port,
		Token:  "testpass",
		Direct: true,
	}

	result, err := ExecuteOQL(opts, "SELECT Name, Age FROM Test.Person")
	if err != nil {
		t.Fatalf("ExecuteOQL: %v", err)
	}

	if len(result.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(result.Columns))
	}
	if len(result.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result.Rows))
	}
	if result.Rows[0][0] != "Alice" {
		t.Errorf("first row Name: got %v, want Alice", result.Rows[0][0])
	}
}

func TestExecuteOQL_OQLError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"result": 1,
			"cause":  "Entity 'NonExistent.Foo' is unknown",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)

	opts := OQLOptions{
		Host:   host,
		Port:   port,
		Token:  "testpass",
		Direct: true,
	}

	_, err := ExecuteOQL(opts, "SELECT * FROM NonExistent.Foo")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Entity 'NonExistent.Foo' is unknown") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecuteOQL_AuthFailure(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)

	opts := OQLOptions{
		Host:   host,
		Port:   port,
		Token:  "wrongpass",
		Direct: true,
	}

	_, err := ExecuteOQL(opts, "SELECT 1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecuteOQL_EmptyResult(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":0,"feedback":{"data":[]}}`))
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)

	opts := OQLOptions{
		Host:   host,
		Port:   port,
		Token:  "testpass",
		Direct: true,
	}

	result, err := ExecuteOQL(opts, "SELECT Name FROM Test.Empty")
	if err != nil {
		t.Fatalf("ExecuteOQL: %v", err)
	}
	if len(result.Columns) != 0 {
		t.Errorf("expected 0 columns, got %d", len(result.Columns))
	}
	if len(result.Rows) != 0 {
		t.Errorf("expected 0 rows, got %d", len(result.Rows))
	}
}

func TestExecuteOQL_ColumnOrder(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return JSON with specific key order — using raw JSON to preserve order
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":0,"feedback":{"data":[{"Zebra":"z","Alpha":"a","Middle":"m"}]}}`))
	}))
	defer server.Close()

	host, port := parseTestServerAddr(t, server.URL)

	opts := OQLOptions{
		Host:   host,
		Port:   port,
		Token:  "testpass",
		Direct: true,
	}

	result, err := ExecuteOQL(opts, "SELECT Zebra, Alpha, Middle FROM Test.T")
	if err != nil {
		t.Fatalf("ExecuteOQL: %v", err)
	}

	// Column order should match JSON key order: Zebra, Alpha, Middle
	expected := []string{"Zebra", "Alpha", "Middle"}
	if len(result.Columns) != len(expected) {
		t.Fatalf("expected %d columns, got %d", len(expected), len(result.Columns))
	}
	for i, want := range expected {
		if result.Columns[i] != want {
			t.Errorf("column %d: got %q, want %q", i, result.Columns[i], want)
		}
	}
}

// parseTestServerAddr extracts host and port from an httptest server URL.
func parseTestServerAddr(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	// rawURL is like "http://127.0.0.1:PORT"
	addr := strings.TrimPrefix(rawURL, "http://")
	idx := strings.LastIndexByte(addr, ':')
	if idx < 0 {
		t.Fatalf("no port in test server URL: %s", rawURL)
	}
	host := addr[:idx]
	var port int
	if _, err := fmt.Sscanf(addr[idx+1:], "%d", &port); err != nil {
		t.Fatalf("parsing port from test server URL %s: %v", rawURL, err)
	}
	return host, port
}
