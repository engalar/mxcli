package tui

import (
	"os"
	"strings"
	"testing"
)

func TestParseCheckJSON(t *testing.T) {
	jsonContent := `{
		"serialization_version": 1,
		"errors": [
			{
				"code": "CE1613",
				"message": "The selected association 'MyModule.Priority' no longer exists.",
				"locations": [
					{
						"module-name": "MyModule",
						"document-name": "Page 'P_ComboBox'",
						"element-name": "Property 'Association' of combo box 'cmbPriority'",
						"element-id": "aaa-111",
						"unit-id": "unit-001"
					}
				]
			},
			{
				"code": "CE0463",
				"message": "Widget definition changed for DataGrid2",
				"locations": [
					{
						"module-name": "MyModule",
						"document-name": "Page 'CustomerList'",
						"element-name": "DataGrid2 widget",
						"element-id": "bbb-222",
						"unit-id": "unit-002"
					}
				]
			}
		],
		"warnings": [
			{
				"code": "CW0001",
				"message": "Unused variable '$var' in microflow",
				"locations": [
					{
						"module-name": "MyModule",
						"document-name": "Microflow 'DoSomething'",
						"element-name": "Variable '$var'",
						"element-id": "ccc-333",
						"unit-id": "unit-003"
					}
				]
			}
		]
	}`

	tmpFile, err := os.CreateTemp("", "mx-check-test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(jsonContent)
	tmpFile.Close()

	errors, err := parseCheckJSON(tmpFile.Name())
	if err != nil {
		t.Fatalf("parseCheckJSON: %v", err)
	}
	if len(errors) != 3 {
		t.Fatalf("expected 3 errors, got %d", len(errors))
	}

	// First error
	if errors[0].Severity != "ERROR" {
		t.Errorf("expected ERROR, got %q", errors[0].Severity)
	}
	if errors[0].Code != "CE1613" {
		t.Errorf("expected CE1613, got %q", errors[0].Code)
	}
	if errors[0].DocumentName != "Page 'P_ComboBox'" {
		t.Errorf("unexpected document: %q", errors[0].DocumentName)
	}
	if errors[0].ElementName == "" {
		t.Error("expected non-empty element name")
	}
	if errors[0].ModuleName != "MyModule" {
		t.Errorf("expected MyModule, got %q", errors[0].ModuleName)
	}
	if errors[0].ElementID != "aaa-111" {
		t.Errorf("expected element-id aaa-111, got %q", errors[0].ElementID)
	}

	// Second error
	if errors[1].Code != "CE0463" {
		t.Errorf("expected CE0463, got %q", errors[1].Code)
	}

	// Third: warning
	if errors[2].Severity != "WARNING" {
		t.Errorf("expected WARNING, got %q", errors[2].Severity)
	}
	if errors[2].Code != "CW0001" {
		t.Errorf("expected CW0001, got %q", errors[2].Code)
	}
}

func TestParseCheckJSONEmpty(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "mx-check-test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(`{"serialization_version": 1}`)
	tmpFile.Close()

	errors, err := parseCheckJSON(tmpFile.Name())
	if err != nil {
		t.Fatalf("parseCheckJSON: %v", err)
	}
	if len(errors) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errors))
	}
}

func TestParseCheckJSONNoLocations(t *testing.T) {
	jsonContent := `{
		"serialization_version": 1,
		"errors": [
			{
				"code": "CE9999",
				"message": "Some error without location",
				"locations": []
			}
		]
	}`

	tmpFile, err := os.CreateTemp("", "mx-check-test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(jsonContent)
	tmpFile.Close()

	errors, err := parseCheckJSON(tmpFile.Name())
	if err != nil {
		t.Fatalf("parseCheckJSON: %v", err)
	}
	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}
	if errors[0].DocumentName != "" {
		t.Errorf("expected empty document name, got %q", errors[0].DocumentName)
	}
}

func TestRenderCheckResultsNilVsEmpty(t *testing.T) {
	// nil = no check has run yet
	result := renderCheckResults(nil, "all")
	if result == "" {
		t.Error("expected non-empty result for nil errors")
	}
	if strings.Contains(result, "passed") {
		t.Error("nil errors should NOT show 'passed' — no check has run yet")
	}

	// empty = check ran, no errors found
	result = renderCheckResults([]CheckError{}, "all")
	if !strings.Contains(result, "passed") {
		t.Error("empty errors should show 'passed'")
	}
}

func TestRenderCheckResultsWithDocLocation(t *testing.T) {
	errors := []CheckError{
		{
			Severity:     "ERROR",
			Code:         "CE1613",
			Message:      "Association no longer exists",
			DocumentName: "Page 'P_ComboBox'",
			ElementName:  "combo box 'cmbPriority'",
			ModuleName:   "MyModule",
		},
	}
	result := renderCheckResults(errors, "all")
	if !strings.Contains(result, "MyModule.P_ComboBox (Page)") {
		t.Errorf("expected qualified doc location, got: %s", result)
	}
	if !strings.Contains(result, "combo box 'cmbPriority'") {
		t.Error("expected element name in rendered output")
	}
}

func TestFormatCheckBadge(t *testing.T) {
	// No check run yet
	badge := formatCheckBadge(nil, false)
	if badge != "" {
		t.Errorf("expected empty badge, got %q", badge)
	}

	// Running
	badge = formatCheckBadge(nil, true)
	if badge == "" {
		t.Error("expected non-empty badge for running state")
	}

	// Pass
	badge = formatCheckBadge([]CheckError{}, false)
	if badge == "" {
		t.Error("expected non-empty badge for pass state")
	}

	// Errors
	errors := []CheckError{
		{Severity: "ERROR", Code: "CE0001"},
		{Severity: "WARNING", Code: "CW0001"},
		{Severity: "ERROR", Code: "CE0002"},
	}
	badge = formatCheckBadge(errors, false)
	if badge == "" {
		t.Error("expected non-empty badge with errors")
	}
}

func TestGroupCheckErrors(t *testing.T) {
	errors := []CheckError{
		{Severity: "ERROR", Code: "CE1613", Message: "Association no longer exists",
			ModuleName: "Mod", DocumentName: "Page 'P1'", ElementName: "combo 'a'", ElementID: "id-1"},
		{Severity: "ERROR", Code: "CE1613", Message: "Association no longer exists",
			ModuleName: "Mod", DocumentName: "Page 'P2'", ElementName: "combo 'b'", ElementID: "id-2"},
		{Severity: "ERROR", Code: "CE0463", Message: "Widget definition changed",
			ModuleName: "Mod", DocumentName: "Page 'P3'", ElementName: "grid", ElementID: "id-3"},
	}

	groups := groupCheckErrors(errors)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	// First group: CE1613 with 2 items
	if groups[0].Code != "CE1613" {
		t.Errorf("expected CE1613, got %q", groups[0].Code)
	}
	if len(groups[0].Items) != 2 {
		t.Errorf("expected 2 items in CE1613 group, got %d", len(groups[0].Items))
	}

	// Second group: CE0463 with 1 item
	if groups[1].Code != "CE0463" {
		t.Errorf("expected CE0463, got %q", groups[1].Code)
	}
	if len(groups[1].Items) != 1 {
		t.Errorf("expected 1 item in CE0463 group, got %d", len(groups[1].Items))
	}
}

func TestGroupCheckErrorsDedup(t *testing.T) {
	// Same code + same element-id should be deduplicated with count
	errors := []CheckError{
		{Severity: "ERROR", Code: "CE1613", Message: "Assoc gone",
			ModuleName: "M", DocumentName: "Page 'P1'", ElementName: "combo 'a'", ElementID: "same-id"},
		{Severity: "ERROR", Code: "CE1613", Message: "Assoc gone",
			ModuleName: "M", DocumentName: "Page 'P1'", ElementName: "combo 'a'", ElementID: "same-id"},
		{Severity: "ERROR", Code: "CE1613", Message: "Assoc gone",
			ModuleName: "M", DocumentName: "Page 'P1'", ElementName: "combo 'a'", ElementID: "same-id"},
	}

	groups := groupCheckErrors(errors)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Items) != 1 {
		t.Fatalf("expected 1 deduplicated item, got %d", len(groups[0].Items))
	}
	if groups[0].Items[0].Count != 3 {
		t.Errorf("expected count 3, got %d", groups[0].Items[0].Count)
	}
}

func TestGroupCheckErrorsNoElementID(t *testing.T) {
	// Without element-id, fallback to doc+element dedup
	errors := []CheckError{
		{Severity: "ERROR", Code: "CE9999", Message: "Some error",
			ModuleName: "M", DocumentName: "Page 'P1'", ElementName: "widget 'w'"},
		{Severity: "ERROR", Code: "CE9999", Message: "Some error",
			ModuleName: "M", DocumentName: "Page 'P1'", ElementName: "widget 'w'"},
	}

	groups := groupCheckErrors(errors)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Items) != 1 {
		t.Fatalf("expected 1 deduplicated item, got %d", len(groups[0].Items))
	}
	if groups[0].Items[0].Count != 2 {
		t.Errorf("expected count 2, got %d", groups[0].Items[0].Count)
	}
}

func TestRenderCheckResultsGrouped(t *testing.T) {
	errors := []CheckError{
		{Severity: "ERROR", Code: "CE1613", Message: "Association gone",
			ModuleName: "Mod", DocumentName: "Page 'P1'", ElementName: "combo 'a'", ElementID: "id-1"},
		{Severity: "ERROR", Code: "CE1613", Message: "Association gone",
			ModuleName: "Mod", DocumentName: "Page 'P2'", ElementName: "combo 'b'", ElementID: "id-2"},
		{Severity: "ERROR", Code: "CE1613", Message: "Association gone",
			ModuleName: "Mod", DocumentName: "Page 'P2'", ElementName: "combo 'b'", ElementID: "id-2"},
	}

	result := renderCheckResults(errors, "all")

	// Should show grouped code header with message
	if !strings.Contains(result, "CE1613") {
		t.Error("expected CE1613 in output")
	}
	if !strings.Contains(result, "Association gone") {
		t.Error("expected message in group header")
	}

	// Deduplicated: id-2 appears twice, should show (x2)
	if !strings.Contains(result, "(x2)") {
		t.Error("expected (x2) count suffix for deduplicated entry")
	}

	// Doc locations
	if !strings.Contains(result, "Mod.P1 (Page)") {
		t.Errorf("expected Mod.P1 (Page) in output, got: %s", result)
	}
	if !strings.Contains(result, "Mod.P2 (Page)") {
		t.Errorf("expected Mod.P2 (Page) in output, got: %s", result)
	}

	// Element names with > prefix
	if !strings.Contains(result, "> ") {
		t.Error("expected element name with > prefix")
	}

	// Summary should count raw errors (3 total)
	if !strings.Contains(result, "3") {
		t.Error("expected raw count of 3 in summary")
	}
}

func TestRenderCheckAnchors(t *testing.T) {
	errors := []CheckError{
		{Severity: "ERROR", Code: "CE1613", Message: "Association gone",
			ModuleName: "Mod", DocumentName: "Page 'P1'", ElementName: "combo 'a'", ElementID: "id-1"},
		{Severity: "ERROR", Code: "CE1613", Message: "Association gone",
			ModuleName: "Mod", DocumentName: "Page 'P2'", ElementName: "combo 'b'", ElementID: "id-2"},
		{Severity: "WARNING", Code: "CW0001", Message: "Unused variable",
			ModuleName: "Mod", DocumentName: "Microflow 'DoSomething'", ElementName: "variable '$var'", ElementID: "id-3"},
	}

	groups := groupCheckErrors(errors)
	result := renderCheckAnchors(groups, errors)

	// Summary line
	if !strings.Contains(result, "[mxcli:check] errors=2 warnings=1 deprecations=0") {
		t.Errorf("expected summary anchor, got: %s", result)
	}

	// Per-item anchors for CE1613
	if !strings.Contains(result, "[mxcli:check:CE1613] severity=ERROR count=1 doc=Mod.P1 type=Page element=combo 'a'") {
		t.Errorf("expected CE1613 P1 anchor, got: %s", result)
	}
	if !strings.Contains(result, "[mxcli:check:CE1613] severity=ERROR count=1 doc=Mod.P2 type=Page element=combo 'b'") {
		t.Errorf("expected CE1613 P2 anchor, got: %s", result)
	}

	// Per-item anchor for CW0001
	if !strings.Contains(result, "[mxcli:check:CW0001] severity=WARNING count=1 doc=Mod.DoSomething type=Microflow element=variable '$var'") {
		t.Errorf("expected CW0001 anchor, got: %s", result)
	}
}

func TestRenderCheckAnchorsEmpty(t *testing.T) {
	result := renderCheckAnchors(nil, nil)
	if !strings.Contains(result, "[mxcli:check] errors=0 warnings=0 deprecations=0") {
		t.Errorf("expected zero-count summary, got: %s", result)
	}
}

func TestRenderCheckAnchorsWithDedup(t *testing.T) {
	errors := []CheckError{
		{Severity: "ERROR", Code: "CE1613", Message: "Assoc gone",
			ModuleName: "M", DocumentName: "Page 'P1'", ElementName: "combo 'a'", ElementID: "same-id"},
		{Severity: "ERROR", Code: "CE1613", Message: "Assoc gone",
			ModuleName: "M", DocumentName: "Page 'P1'", ElementName: "combo 'a'", ElementID: "same-id"},
		{Severity: "ERROR", Code: "CE1613", Message: "Assoc gone",
			ModuleName: "M", DocumentName: "Page 'P1'", ElementName: "combo 'a'", ElementID: "same-id"},
	}

	groups := groupCheckErrors(errors)
	result := renderCheckAnchors(groups, errors)

	// Should show count=3 for the deduplicated item
	if !strings.Contains(result, "count=3") {
		t.Errorf("expected count=3 in deduped anchor, got: %s", result)
	}
	// Summary should show raw error count
	if !strings.Contains(result, "errors=3") {
		t.Errorf("expected errors=3 in summary, got: %s", result)
	}
}

func TestParseCheckJSONWithDeprecations(t *testing.T) {
	jsonContent := `{
		"serialization_version": 1,
		"errors": [
			{"code": "CE0001", "message": "Some error",
			 "locations": [{"module-name": "M", "document-name": "Page 'P1'", "element-name": "w", "element-id": "e1", "unit-id": "u1"}]}
		],
		"warnings": [
			{"code": "CW0001", "message": "Some warning",
			 "locations": [{"module-name": "M", "document-name": "Microflow 'MF1'", "element-name": "v", "element-id": "e2", "unit-id": "u2"}]}
		],
		"deprecations": [
			{"code": "CD0001", "message": "Deprecated feature used",
			 "locations": [{"module-name": "M", "document-name": "Page 'P2'", "element-name": "old widget", "element-id": "e3", "unit-id": "u3"}]}
		]
	}`

	tmpFile, err := os.CreateTemp("", "mx-check-test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(jsonContent)
	tmpFile.Close()

	errors, err := parseCheckJSON(tmpFile.Name())
	if err != nil {
		t.Fatalf("parseCheckJSON: %v", err)
	}
	if len(errors) != 3 {
		t.Fatalf("expected 3 diagnostics, got %d", len(errors))
	}
	if errors[0].Severity != "ERROR" {
		t.Errorf("expected ERROR, got %q", errors[0].Severity)
	}
	if errors[1].Severity != "WARNING" {
		t.Errorf("expected WARNING, got %q", errors[1].Severity)
	}
	if errors[2].Severity != "DEPRECATION" {
		t.Errorf("expected DEPRECATION, got %q", errors[2].Severity)
	}
	if errors[2].Code != "CD0001" {
		t.Errorf("expected CD0001, got %q", errors[2].Code)
	}
}

func TestRenderCheckResultsFilter(t *testing.T) {
	errors := []CheckError{
		{Severity: "ERROR", Code: "CE0001", Message: "Error msg",
			ModuleName: "M", DocumentName: "Page 'P1'", ElementName: "w1", ElementID: "e1"},
		{Severity: "WARNING", Code: "CW0001", Message: "Warning msg",
			ModuleName: "M", DocumentName: "Microflow 'MF1'", ElementName: "v1", ElementID: "e2"},
		{Severity: "DEPRECATION", Code: "CD0001", Message: "Deprecation msg",
			ModuleName: "M", DocumentName: "Page 'P2'", ElementName: "w2", ElementID: "e3"},
	}

	// Filter by error: should show CE0001 but not CW0001 or CD0001
	result := renderCheckResults(errors, "error")
	if !strings.Contains(result, "CE0001") {
		t.Error("expected CE0001 in error-filtered output")
	}
	if strings.Contains(result, "CW0001") {
		t.Error("CW0001 should be hidden in error filter")
	}
	if strings.Contains(result, "CD0001") {
		t.Error("CD0001 should be hidden in error filter")
	}

	// Filter by warning
	result = renderCheckResults(errors, "warning")
	if strings.Contains(result, "CE0001") {
		t.Error("CE0001 should be hidden in warning filter")
	}
	if !strings.Contains(result, "CW0001") {
		t.Error("expected CW0001 in warning-filtered output")
	}

	// Filter by deprecation
	result = renderCheckResults(errors, "deprecation")
	if !strings.Contains(result, "CD0001") {
		t.Error("expected CD0001 in deprecation-filtered output")
	}
	if strings.Contains(result, "CE0001") {
		t.Error("CE0001 should be hidden in deprecation filter")
	}

	// All filter shows everything
	result = renderCheckResults(errors, "all")
	if !strings.Contains(result, "CE0001") || !strings.Contains(result, "CW0001") || !strings.Contains(result, "CD0001") {
		t.Error("all filter should show all codes")
	}
}

func TestRenderCheckFilterTitle(t *testing.T) {
	errors := []CheckError{
		{Severity: "ERROR", Code: "CE0001"},
		{Severity: "ERROR", Code: "CE0002"},
		{Severity: "WARNING", Code: "CW0001"},
		{Severity: "DEPRECATION", Code: "CD0001"},
	}

	title := renderCheckFilterTitle(errors, "all")
	if !strings.Contains(title, "[All: 2E 1W 1D]") {
		t.Errorf("expected [All: 2E 1W 1D] in title, got %q", title)
	}

	title = renderCheckFilterTitle(errors, "error")
	if !strings.Contains(title, "[Errors: 2]") {
		t.Errorf("expected [Errors: 2] in title, got %q", title)
	}

	title = renderCheckFilterTitle(errors, "warning")
	if !strings.Contains(title, "[Warnings: 1]") {
		t.Errorf("expected [Warnings: 1] in title, got %q", title)
	}

	title = renderCheckFilterTitle(errors, "deprecation")
	if !strings.Contains(title, "[Deprecations: 1]") {
		t.Errorf("expected [Deprecations: 1] in title, got %q", title)
	}

	// nil returns plain title
	title = renderCheckFilterTitle(nil, "all")
	if title != "mx check" {
		t.Errorf("expected plain 'mx check' for nil errors, got %q", title)
	}
}

func TestNextCheckFilter(t *testing.T) {
	if nextCheckFilter("all") != "error" {
		t.Error("all -> error")
	}
	if nextCheckFilter("error") != "warning" {
		t.Error("error -> warning")
	}
	if nextCheckFilter("warning") != "deprecation" {
		t.Error("warning -> deprecation")
	}
	if nextCheckFilter("deprecation") != "all" {
		t.Error("deprecation -> all")
	}
	if nextCheckFilter("unknown") != "all" {
		t.Error("unknown -> all")
	}
}

func TestCountBySeverity(t *testing.T) {
	errors := []CheckError{
		{Severity: "ERROR"},
		{Severity: "ERROR"},
		{Severity: "WARNING"},
		{Severity: "DEPRECATION"},
		{Severity: "DEPRECATION"},
		{Severity: "DEPRECATION"},
	}
	ec, wc, dc := countBySeverity(errors)
	if ec != 2 {
		t.Errorf("expected 2 errors, got %d", ec)
	}
	if wc != 1 {
		t.Errorf("expected 1 warning, got %d", wc)
	}
	if dc != 3 {
		t.Errorf("expected 3 deprecations, got %d", dc)
	}
}

func TestExtractCheckNavLocations(t *testing.T) {
	errors := []CheckError{
		{Severity: "ERROR", Code: "CE1613", Message: "Assoc gone",
			ModuleName: "Mod", DocumentName: "Page 'P1'", ElementName: "combo 'a'"},
		{Severity: "ERROR", Code: "CE1613", Message: "Assoc gone",
			ModuleName: "Mod", DocumentName: "Page 'P1'", ElementName: "combo 'b'"},
		{Severity: "ERROR", Code: "CE0463", Message: "Widget changed",
			ModuleName: "Mod", DocumentName: "Page 'P2'", ElementName: "grid"},
		{Severity: "WARNING", Code: "CW0001", Message: "Unused",
			ModuleName: "Mod", DocumentName: "Microflow 'MF1'", ElementName: "var"},
		{Severity: "ERROR", Code: "CE9999", Message: "No location"},
	}

	locs := extractCheckNavLocations(errors)
	// 3 unique docs (P1, P2, MF1) — CE9999 has no DocumentName
	if len(locs) != 3 {
		t.Fatalf("expected 3 nav locations, got %d", len(locs))
	}
	if locs[0].DocumentName != "Page 'P1'" {
		t.Errorf("expected Page 'P1', got %q", locs[0].DocumentName)
	}
	if locs[0].Code != "CE1613" {
		t.Errorf("expected CE1613, got %q", locs[0].Code)
	}
	if locs[1].DocumentName != "Page 'P2'" {
		t.Errorf("expected Page 'P2', got %q", locs[1].DocumentName)
	}
	if locs[2].DocumentName != "Microflow 'MF1'" {
		t.Errorf("expected Microflow 'MF1', got %q", locs[2].DocumentName)
	}
}

func TestExtractCheckNavLocationsEmpty(t *testing.T) {
	locs := extractCheckNavLocations(nil)
	if len(locs) != 0 {
		t.Errorf("expected 0 locations for nil, got %d", len(locs))
	}
	locs = extractCheckNavLocations([]CheckError{})
	if len(locs) != 0 {
		t.Errorf("expected 0 locations for empty, got %d", len(locs))
	}
}

func TestDocNameToQualifiedName(t *testing.T) {
	tests := []struct {
		mod, doc, want string
	}{
		{"MyModule", "Page 'P_ComboBox'", "MyModule.P_ComboBox"},
		{"MyModule", "Microflow 'DoSomething'", "MyModule.DoSomething"},
		{"", "Page 'Orphan'", "Orphan"},
		{"Mod", "PlainName", "Mod.PlainName"},
		{"", "PlainName", "PlainName"},
	}
	for _, tt := range tests {
		got := docNameToQualifiedName(tt.mod, tt.doc)
		if got != tt.want {
			t.Errorf("docNameToQualifiedName(%q, %q) = %q, want %q", tt.mod, tt.doc, got, tt.want)
		}
	}
}

func TestFilterCheckErrors(t *testing.T) {
	errors := []CheckError{
		{Severity: "ERROR", Code: "CE0001"},
		{Severity: "WARNING", Code: "CW0001"},
		{Severity: "DEPRECATION", Code: "CD0001"},
	}

	filtered := filterCheckErrors(errors, "all")
	if len(filtered) != 3 {
		t.Errorf("all filter: expected 3, got %d", len(filtered))
	}

	filtered = filterCheckErrors(errors, "error")
	if len(filtered) != 1 || filtered[0].Code != "CE0001" {
		t.Errorf("error filter: expected CE0001, got %v", filtered)
	}

	filtered = filterCheckErrors(errors, "warning")
	if len(filtered) != 1 || filtered[0].Code != "CW0001" {
		t.Errorf("warning filter: expected CW0001, got %v", filtered)
	}

	filtered = filterCheckErrors(errors, "deprecation")
	if len(filtered) != 1 || filtered[0].Code != "CD0001" {
		t.Errorf("deprecation filter: expected CD0001, got %v", filtered)
	}
}

func TestFormatCheckBadgeWithDeprecations(t *testing.T) {
	errors := []CheckError{
		{Severity: "ERROR", Code: "CE0001"},
		{Severity: "WARNING", Code: "CW0001"},
		{Severity: "DEPRECATION", Code: "CD0001"},
	}
	badge := formatCheckBadge(errors, false)
	if !strings.Contains(badge, "1E") {
		t.Errorf("expected 1E in badge, got %q", badge)
	}
	if !strings.Contains(badge, "1W") {
		t.Errorf("expected 1W in badge, got %q", badge)
	}
	if !strings.Contains(badge, "1D") {
		t.Errorf("expected 1D in badge, got %q", badge)
	}
}
