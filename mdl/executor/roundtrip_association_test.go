// SPDX-License-Identifier: Apache-2.0

//go:build integration

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/internal/bsoncompare"
)

const rtAssocQN = "RoundtripModule.Item_Category"

func TestRoundtrip_Association_Syntax(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()
	mdl := env.rtDescribe("describe association " + rtAssocQN)
	rtAssertParseOK(t, mdl)
}

func TestRoundtrip_Association_Semantic(t *testing.T) {
	env := setupRoundtripEnv(t)
	defer env.teardown()
	env.rtAssertSemantic("describe association " + rtAssocQN)
}

func TestRoundtrip_Association_Storage(t *testing.T) {
	env := setupRoundtripEnv(t)
	snap := snapshotMPR(t, env.projectPath)
	mdl := env.rtDescribe("describe association " + rtAssocQN)
	if err := env.executeMDL(mdl); err != nil {
		t.Fatalf("re-import association MDL: %v", err)
	}
	env.teardown()
	bsoncompare.AssertEqual(t, snap, env.projectPath,
		bsoncompare.DefaultOptions(),
		bsoncompare.ExpectNoOtherChanges(),
	)
}
