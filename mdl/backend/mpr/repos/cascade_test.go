// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/model"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

func TestCascadeService_DeleteFolder_Empty_OK(t *testing.T) {
	w := openTestWriter(t)
	svc := NewCascadeService(w)

	// Insert a fresh empty folder to delete (avoids touching fixture data).
	folderID := mmpr.GenerateID()
	parent := mmpr.GenerateID()
	if err := w.InsertUnit(folderID, parent, "Folders", "Projects$Folder", []byte{0x05, 0x00, 0x00, 0x00, 0x00}); err != nil {
		// 5-byte BSON empty doc; some encoders won't accept this — skip if so.
		t.Skipf("empty BSON insert rejected by writer (expected for some MPR variants): %v", err)
	}
	if err := svc.DeleteFolder(model.ID(folderID)); err != nil {
		t.Fatalf("DeleteFolder on empty folder failed: %v", err)
	}
}

func TestCascadeService_DeleteFolder_NonEmpty_Refuses(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	svc := NewCascadeService(w)

	// Pick any folder from fixture that has at least one child.
	// MyFirstModule's "Documents" folder typically has children.
	refs, err := r.ListUnitsByType("Projects$Folder")
	if err != nil {
		t.Fatalf("ListUnitsByType: %v", err)
	}
	var nonEmpty model.ID
	for _, ref := range refs {
		if ref.Type != "Projects$Folder" {
			continue
		}
		// crude check: any folder that contains another unit
		blob := mmpr.IDToBsonBinary(ref.ID).Data
		var count int
		if err := w.ConcreteReader().DB().QueryRow(
			"SELECT COUNT(*) FROM Unit WHERE ContainerID = ? AND UnitID != ContainerID",
			blob,
		).Scan(&count); err != nil {
			continue
		}
		if count > 0 {
			nonEmpty = model.ID(ref.ID)
			break
		}
	}
	if nonEmpty == "" {
		t.Skip("no non-empty folder in fixture; cannot verify refusal path")
	}

	err = svc.DeleteFolder(nonEmpty)
	if err == nil {
		t.Fatalf("DeleteFolder on non-empty folder %s should have errored", nonEmpty)
	}
	if !strings.Contains(err.Error(), "not empty") {
		t.Errorf("error %q should mention 'not empty'", err.Error())
	}
}

func TestCascadeService_DeleteModule_RemovesAllChildren(t *testing.T) {
	w := openTestWriter(t)
	r := w.ConcreteReader()
	svc := NewCascadeService(w)

	// Construct a fresh module with a fake child unit, then DeleteModule
	// and assert both the module and the child are gone.
	moduleID := mmpr.GenerateID()
	if err := w.InsertUnit(moduleID, "00000000-0000-0000-0000-000000000001", "Modules", "Projects$ModuleImpl", []byte{0x05, 0x00, 0x00, 0x00, 0x00}); err != nil {
		t.Skipf("InsertUnit rejected empty doc: %v", err)
	}
	childID := mmpr.GenerateID()
	if err := w.InsertUnit(childID, moduleID, "Documents", "Microflows$Microflow", []byte{0x05, 0x00, 0x00, 0x00, 0x00}); err != nil {
		t.Skipf("InsertUnit rejected child: %v", err)
	}

	if err := svc.DeleteModule(model.ID(moduleID)); err != nil {
		t.Fatalf("DeleteModule: %v", err)
	}

	// Module should be gone
	if _, err := r.GetRawUnitBytes(moduleID); err == nil {
		t.Errorf("module %s still readable after DeleteModule", moduleID)
	}
	// Child should be gone
	if _, err := r.GetRawUnitBytes(childID); err == nil {
		t.Errorf("child %s still readable after DeleteModule", childID)
	}
}
