// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

func TestRoundtrip_Constant_Syntax(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()
	for _, name := range []string{"RoundtripModule.ApiBaseUrl", "RoundtripModule.MaxRetries"} {
		mdl := env.rtDescribe("describe constant " + name)
		rtAssertParseOK(t, mdl)
	}
}

func TestRoundtrip_Constant_Semantic(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()
	env.rtAssertSemantic("describe constant RoundtripModule.ApiBaseUrl")
	env.rtAssertSemantic("describe constant RoundtripModule.MaxRetries")
}

func TestRoundtrip_Constant_Storage(t *testing.T) {
	env := setupRoundtripEnv(t)
	snap := snapshotMPR(t, env.projectPath)
	for _, name := range []string{"RoundtripModule.ApiBaseUrl", "RoundtripModule.MaxRetries"} {
		mdl := env.rtDescribe("describe constant " + name)
		if err := env.executeMDL(mdl); err != nil {
			t.Fatalf("re-import constant %s: %v", name, err)
		}
	}
	env.teardown()
	bsoncompare.AssertEqual(t, snap, env.projectPath,
		bsoncompare.DefaultOptions(),
		bsoncompare.ExpectNoOtherChanges(),
	)
}
