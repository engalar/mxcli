// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"strings"
	"testing"
)

// TestRoundtrip_JavaScriptAction_Describe verifies that DESCRIBE JAVASCRIPT ACTION
// returns a well-formed output for an existing JS action in the roundtrip project (L1).
//
// MDL has no `create javascript action` syntax, so JS actions cannot be provisioned
// via seed.mdl. This test targets FeedbackModule.JS_isStrictMode, which is part of the
// default mx create-project template for Mendix 11.6.6 and is present in
// the committed testdata/roundtrip/mprcontents/. If recreate.sh is rerun
// with a Mendix version that no longer includes FeedbackModule by default,
// a self-hosted JS action in RoundtripModule should be added to seed.mdl instead.
//
// L2/L3 are skipped: the describe output is not valid re-importable MDL
// (no `create javascript action` grammar rule exists).
func TestRoundtrip_JavaScriptAction_Describe(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()

	mdl, err := env.describeMDL("describe javascript action FeedbackModule.JS_isStrictMode")
	if err != nil {
		t.Fatalf("describe javascript action FeedbackModule.JS_isStrictMode: %v", err)
	}

	if strings.TrimSpace(mdl) == "" {
		t.Fatal("expected non-empty output from describe javascript action")
	}
	if !strings.Contains(mdl, "create javascript action ") {
		t.Errorf("expected 'create javascript action' keyword in output, got:\n%s", mdl)
	}
	if !strings.Contains(mdl, "JS_isStrictMode") {
		t.Errorf("expected action name 'JS_isStrictMode' in output, got:\n%s", mdl)
	}
}

// TestRoundtrip_JavaScriptAction_ShowList verifies that SHOW JAVASCRIPT ACTIONS
// lists JS actions present in the roundtrip project (L1).
func TestRoundtrip_JavaScriptAction_ShowList(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()

	out, err := env.describeMDL("show javascript actions in FeedbackModule")
	if err != nil {
		t.Fatalf("show javascript actions: %v", err)
	}

	if !strings.Contains(out, "JS_isStrictMode") {
		t.Errorf("expected 'JS_isStrictMode' in javascript action listing, got:\n%s", out)
	}
	if !strings.Contains(out, "FeedbackModule") {
		t.Errorf("expected module 'FeedbackModule' in listing, got:\n%s", out)
	}
}
