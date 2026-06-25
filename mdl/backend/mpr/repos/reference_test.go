// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/types"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// openWriterAt opens an mmpr.Writer and a BSONScanner on the same fixture path.
// The scanner is backed by a modelsdk/mpr reader.
func openWriterAt(t *testing.T) (*mmpr.Writer, types.BSONScanner) {
	t.Helper()
	dst := openFixture(t)
	mw, err := mmpr.NewWriter(dst)
	if err != nil {
		t.Fatalf("mmpr.NewWriter: %v", err)
	}
	t.Cleanup(func() { _ = mw.Close() })
	scanner, err := mmpr.Open(dst)
	if err != nil {
		t.Fatalf("mmpr.Open: %v", err)
	}
	t.Cleanup(func() { _ = scanner.Close() })
	return mw, scanner
}

func TestReferenceService_ScanRename_NoMatches(t *testing.T) {
	mw, sdkW := openWriterAt(t)
	svc := NewReferenceService(mw, sdkW)

	hits, err := svc.ScanRename("Module.Definitely_Not_Here_xyz", "Module.New_xyz")
	if err != nil {
		t.Fatalf("ScanRename: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("expected 0 hits for non-matching name, got %d", len(hits))
	}
}

func TestReferenceService_UpdateEnumerationRefs_NoMatches(t *testing.T) {
	mw, sdkW := openWriterAt(t)
	svc := NewReferenceService(mw, sdkW)

	if err := svc.UpdateEnumerationRefsInAllDomainModels("Mod.NoSuchEnum_xyz", "Mod.NewEnum_xyz"); err != nil {
		t.Fatalf("UpdateEnumerationRefs: %v", err)
	}
	// No assertion beyond "no error" — fixture has no matching enum,
	// so the scanner returns zero patches and the writer never fires.
}
