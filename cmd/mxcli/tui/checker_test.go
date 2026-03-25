package tui

import "testing"

func TestParseCheckOutput(t *testing.T) {
	input := `Checking your app for issues...
Checking the version of the mpr file.
The mpr file version is '11.6.4'.
Loading the mpr file.
Checking app for errors...
[error] [CE1613] "The selected association 'MyModule.Priority' no longer exists." at Combo box 'cmbPriority'
[warning] [CW0001] "Unused variable '$var' in microflow" at Microflow 'MyModule.DoSomething'
[error] [CE0463] "Widget definition changed for DataGrid2" at Page 'MyModule.CustomerList'
The app contains: 2 errors.
`

	errors := parseCheckOutput(input)
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
	if errors[0].Location != "Combo box 'cmbPriority'" {
		t.Errorf("unexpected location: %q", errors[0].Location)
	}

	// Second: warning
	if errors[1].Severity != "WARNING" {
		t.Errorf("expected WARNING, got %q", errors[1].Severity)
	}
	if errors[1].Code != "CW0001" {
		t.Errorf("expected CW0001, got %q", errors[1].Code)
	}

	// Third: error
	if errors[2].Code != "CE0463" {
		t.Errorf("expected CE0463, got %q", errors[2].Code)
	}
}

func TestParseCheckOutputEmpty(t *testing.T) {
	errors := parseCheckOutput("Project check passed.\n")
	if len(errors) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errors))
	}
}

func TestParseCheckOutputIgnoresNonMatchingLines(t *testing.T) {
	input := `Checking your app for issues...
Loading the mpr file.
The app contains: 0 errors.
`
	errors := parseCheckOutput(input)
	if len(errors) != 0 {
		t.Fatalf("expected 0 errors from non-matching lines, got %d", len(errors))
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
