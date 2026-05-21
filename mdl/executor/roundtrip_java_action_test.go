// SPDX-License-Identifier: Apache-2.0
package executor

import (
	"strings"
	"testing"
)

// TestRoundtrip_JavaAction_Describe verifies that DESCRIBE JAVA ACTION returns
// a well-formed output for a Java action provisioned in the roundtrip project (L1).
//
// L2/L3 are skipped: the describe output omits the required `as $$ ... $$` body,
// so it cannot be re-parsed by visitor.Build or re-imported without modification.
func TestRoundtrip_JavaAction_Describe(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()

	mdl, err := env.describeMDL("describe java action RoundtripModule.ExternalCall")
	if err != nil {
		t.Fatalf("describe java action RoundtripModule.ExternalCall: %v", err)
	}

	if strings.TrimSpace(mdl) == "" {
		t.Fatal("expected non-empty output from describe java action")
	}
	if !strings.Contains(mdl, "create java action ") {
		t.Errorf("expected 'create java action' keyword in output, got:\n%s", mdl)
	}
	if !strings.Contains(mdl, "ExternalCall") {
		t.Errorf("expected action name 'ExternalCall' in output, got:\n%s", mdl)
	}
	if !strings.Contains(mdl, "InputText") {
		t.Errorf("expected parameter 'InputText' in output, got:\n%s", mdl)
	}
	if !strings.Contains(mdl, "String") {
		t.Errorf("expected return type 'String' in output, got:\n%s", mdl)
	}
}

// TestRoundtrip_JavaAction_ShowList verifies that SHOW JAVA ACTIONS lists the
// provisioned Java action in RoundtripModule (L1).
func TestRoundtrip_JavaAction_ShowList(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()

	out, err := env.describeMDL("show java actions in RoundtripModule")
	if err != nil {
		t.Fatalf("show java actions: %v", err)
	}

	if !strings.Contains(out, "ExternalCall") {
		t.Errorf("expected 'ExternalCall' in java action listing, got:\n%s", out)
	}
	if !strings.Contains(out, "RoundtripModule") {
		t.Errorf("expected module name 'RoundtripModule' in listing, got:\n%s", out)
	}
}
