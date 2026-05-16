// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"testing"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
	sdkmpr "github.com/mendixlabs/mxcli/sdk/mpr"
)

// openSDKWriterAt opens both mmpr and sdk/mpr writers on the same fixture
// path (each gets its own SQLite connection). Sufficient for tests that
// don't perform concurrent writes; production code uses the Wrap pattern
// for shared-connection safety.
func openSDKWriterAt(t *testing.T) (*mmpr.Writer, *sdkmpr.Writer) {
	t.Helper()
	dst := copyFixture(t, fixturePath, t.TempDir())
	mw, err := mmpr.NewWriter(dst)
	if err != nil {
		t.Fatalf("mmpr.NewWriter: %v", err)
	}
	t.Cleanup(func() { _ = mw.Close() })
	sdkW, err := sdkmpr.NewWriter(dst)
	if err != nil {
		t.Fatalf("sdkmpr.NewWriter: %v", err)
	}
	t.Cleanup(func() { _ = sdkW.Close() })
	return mw, sdkW
}

func TestReferenceService_ScanRename_NoMatches(t *testing.T) {
	mw, sdkW := openSDKWriterAt(t)
	svc := NewReferenceService(mw, sdkW.Reader())

	hits, err := svc.ScanRename("Module.Definitely_Not_Here_xyz", "Module.New_xyz")
	if err != nil {
		t.Fatalf("ScanRename: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("expected 0 hits for non-matching name, got %d", len(hits))
	}
}

func TestReferenceService_UpdateEnumerationRefs_NoMatches(t *testing.T) {
	mw, sdkW := openSDKWriterAt(t)
	svc := NewReferenceService(mw, sdkW.Reader())

	if err := svc.UpdateEnumerationRefsInAllDomainModels("Mod.NoSuchEnum_xyz", "Mod.NewEnum_xyz"); err != nil {
		t.Fatalf("UpdateEnumerationRefs: %v", err)
	}
	// No assertion beyond "no error" — fixture has no matching enum,
	// so the scanner returns zero patches and the writer never fires.
}
