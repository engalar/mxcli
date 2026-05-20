// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/model"
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

func TestClassifyModules(t *testing.T) {
	mods := []*model.Module{
		{Name: "MyFirstModule", FromAppStore: false},
		{Name: "Administration", FromAppStore: true, AppStoreVersion: "3.4.0"},
		{Name: "AtlasCore", FromAppStore: true, AppStoreVersion: "4.0.0"},
	}
	regular, marketplace := classifyModules(mods)
	if len(regular) != 1 || regular[0].Name != "MyFirstModule" {
		t.Errorf("regular modules: got %v", regular)
	}
	if len(marketplace) != 2 {
		t.Errorf("marketplace modules: got %v", marketplace)
	}
}

func TestMarketplaceFileContent(t *testing.T) {
	mods := []*model.Module{
		{Name: "Administration", FromAppStore: true, AppStoreVersion: "3.4.0"},
		{Name: "AtlasCore", FromAppStore: true, AppStoreVersion: "4.0.0"},
	}
	got := marketplaceFileContent(mods)
	if !strings.Contains(got, "Administration") || !strings.Contains(got, "3.4.0") {
		t.Errorf("missing module in output: %q", got)
	}
	if !strings.Contains(got, "AtlasCore") {
		t.Errorf("missing AtlasCore: %q", got)
	}
}
