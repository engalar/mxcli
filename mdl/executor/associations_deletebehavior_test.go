// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// TestAssociationPreventDeleteHasErrorMessage guards against a Mendix runtime
// crash: SchemeFactory.setDeleteBehavior → None.get at app startup.
//
// A "PREVENT" delete behavior serializes to ChildDeleteBehavior =
// "DeleteMeIfNoReferences". Because that side blocks deletion, the runtime reads
// its error message and does an Option.get on it; a null message throws
// NoSuchElementException: None.get and the runtime never starts. Studio Pro
// always populates a default message for prevent-delete, so mxcli must too.
//
// Non-preventing behaviors (Keep / Cascade) must keep a NULL error message to
// match Studio Pro's BSON exactly.
func TestAssociationPreventDeleteHasErrorMessage(t *testing.T) {
	prevent := astAssociationDeleteBehaviorGen(
		&ast.CreateAssociationStmt{DeleteBehavior: ast.DeleteIfNoReferences},
	).(*genDm.AssociationDeleteBehavior)

	if got := prevent.ChildDeleteBehavior(); got != "DeleteMeIfNoReferences" {
		t.Fatalf("PREVENT child behavior = %q, want DeleteMeIfNoReferences", got)
	}
	if prevent.ChildErrorMessage() == nil {
		t.Fatal("PREVENT (DeleteMeIfNoReferences) must have a non-nil ChildErrorMessage " +
			"— a null message crashes the runtime with SchemeFactory.setDeleteBehavior → None.get")
	}

	// Keep (default) and Cascade must NOT carry an error message (Studio Pro parity).
	for _, tc := range []struct {
		name string
		beh  ast.DeleteBehavior
	}{
		{"keep", ast.DeleteKeepReferences},
		{"cascade", ast.DeleteCascade},
	} {
		d := astAssociationDeleteBehaviorGen(
			&ast.CreateAssociationStmt{DeleteBehavior: tc.beh},
		).(*genDm.AssociationDeleteBehavior)
		if d.ChildErrorMessage() != nil {
			t.Errorf("%s: ChildErrorMessage should be nil (Studio Pro parity), got non-nil", tc.name)
		}
		if d.ParentErrorMessage() != nil {
			t.Errorf("%s: ParentErrorMessage should be nil (Studio Pro parity), got non-nil", tc.name)
		}
	}
}
