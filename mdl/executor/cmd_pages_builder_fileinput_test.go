// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// TestBuildFileManagerV3_Basic verifies that a fileinput widget builds to a
// Forms$FileManager element with name, label, and the file-specific properties
// (AllowedExtensions, MaxFileSize) populated from the MDL widget properties.
func TestBuildFileManagerV3_Basic(t *testing.T) {
	pb := &pageBuilder{
		widgetScope: map[string]model.ID{},
	}

	w := &ast.WidgetV3{
		Type: "fileinput",
		Name: "FileInput1",
		Properties: map[string]any{
			"allowedExtensions": "xlsx,xls",
			"label":             "Upload Excel File",
			"maxFileSize":       5,
		},
	}

	elem, err := pb.buildFileManagerV3(w)
	if err != nil {
		t.Fatalf("buildFileManagerV3 returned error: %v", err)
	}
	if elem == nil {
		t.Fatal("buildFileManagerV3 returned nil element")
	}
	if elem.TypeName() != "Forms$FileManager" {
		t.Errorf("TypeName = %q, want Forms$FileManager", elem.TypeName())
	}

	fm, ok := elem.(*genPg.FileManager)
	if !ok {
		t.Fatalf("element = %T, want *genPg.FileManager", elem)
	}
	if fm.Name() != "FileInput1" {
		t.Errorf("Name = %q, want FileInput1", fm.Name())
	}
	if fm.AllowedExtensions() != "xlsx,xls" {
		t.Errorf("AllowedExtensions = %q, want xlsx,xls", fm.AllowedExtensions())
	}
	if fm.MaxFileSize() != 5 {
		t.Errorf("MaxFileSize = %d, want 5", fm.MaxFileSize())
	}
	if fm.Label() == nil {
		t.Error("Label must be set when label property is provided")
	}
}

// TestBuildFileManagerV3_DuplicateName verifies that registering the same widget
// name twice returns an error.
func TestBuildFileManagerV3_DuplicateName(t *testing.T) {
	pb := &pageBuilder{
		widgetScope: map[string]model.ID{},
	}
	w := &ast.WidgetV3{Type: "fileinput", Name: "Dup", Properties: map[string]any{}}

	if _, err := pb.buildFileManagerV3(w); err != nil {
		t.Fatalf("first build: %v", err)
	}
	if _, err := pb.buildFileManagerV3(w); err == nil {
		t.Error("second build with duplicate name must return an error")
	}
}
