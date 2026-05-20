// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"path/filepath"
	"testing"
)

func TestDocumentFilePath_NoFolder(t *testing.T) {
	got := documentFilePath("/out", "MyModule", "", "MyModule.Customer")
	want := filepath.Join("/out", "MyModule", "MyModule.Customer.mdl")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDocumentFilePath_WithFolder(t *testing.T) {
	got := documentFilePath("/out", "MyModule", "Microflows/ACT", "MyModule.ACT_Foo")
	want := filepath.Join("/out", "MyModule", "Microflows", "ACT", "MyModule.ACT_Foo.mdl")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
