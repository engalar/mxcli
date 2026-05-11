// SPDX-License-Identifier: Apache-2.0

package hints

import (
	"encoding/json"
	"strings"
	"testing"
)

var sample = Hint{
	Code:     "E001",
	Slug:     "enum-string-mismatch",
	Severity: SeverityError,
	Where: Location{
		File: "fraud.mdl", Line: 36, Column: 21,
		Microflow: "FraudDetection.SUB_UpdateAlertStatus",
		Context:   "IF condition",
	},
	YouWrote: "IF $Alert/Status = 'NewAlert' THEN ...",
	Problem:  "Comparing an Enumeration attribute against a string literal.",
	Fix:      "IF $Alert/Status = FraudDetection.AlertStatus.NewAlert THEN ...",
	Reference: &Reference{
		Enum:       "FraudDetection.AlertStatus",
		EnumValues: []string{"NewAlert", "Validated"},
	},
}

func TestFormat_Text(t *testing.T) {
	out := FormatText(sample)
	must := []string{
		"HINT [E001 enum-string-mismatch] error",
		"WHERE:",
		"fraud.mdl line 36",
		"FraudDetection.SUB_UpdateAlertStatus",
		"YOU WROTE:",
		"IF $Alert/Status = 'NewAlert'",
		"PROBLEM:",
		"FIX:",
		"FraudDetection.AlertStatus.NewAlert",
		"LEGAL VALUES",
		"NewAlert, Validated",
	}
	for _, m := range must {
		if !strings.Contains(out, m) {
			t.Errorf("text output missing %q\n--- output ---\n%s", m, out)
		}
	}
}

func TestFormat_JSON(t *testing.T) {
	out := FormatJSON(sample)
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if got["code"] != "E001" || got["slug"] != "enum-string-mismatch" {
		t.Errorf("missing code/slug: %v", got)
	}
	if got["severity"] != "error" {
		t.Errorf("severity = %v", got["severity"])
	}
	if got["fix"] == nil || got["you_wrote"] == nil || got["problem"] == nil {
		t.Errorf("missing AI-facing fields: %v", got)
	}
}
