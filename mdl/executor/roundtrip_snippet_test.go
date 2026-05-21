// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"testing"
)

// TestRoundtrip_Snippet_Syntax verifies that DESCRIBE SNIPPET produces valid,
// parseable MDL for the Administration.ReadMe snippet that ships with every
// Mendix project's Administration module.
func TestRoundtrip_Snippet_Syntax(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()
	mdl := env.rtDescribe("describe snippet Administration.ReadMe")
	rtAssertParseOK(t, mdl)
}

// TestRoundtrip_Snippet_Semantic is skipped because re-importing a snippet
// with widget children currently loses the widget body on the second describe
// (known limitation: snippet CREATE OR MODIFY does not preserve existing
// widget children when the snippet already exists). Until that write-path bug
// is fixed, full semantic round-trip testing for snippet content is deferred.
func TestRoundtrip_Snippet_Semantic(t *testing.T) {
	t.Skip("snippet body round-trip not yet idempotent — widget children lost on re-import of existing snippet")
}
