// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"testing"

	"github.com/mendixlabs/mxcli/model"
)

func TestProjectReader_FiltersAppStoreModules(t *testing.T) {
	fakeModules := []*model.Module{
		{Name: "MyModule", FromAppStore: false},
		{Name: "DatabaseConnector", FromAppStore: true, AppStoreGuid: "2888", AppStoreVersion: "7.0.2"},
		{Name: "CommunityCommons", FromAppStore: true, AppStoreGuid: "170", AppStoreVersion: "11.5.0"},
	}

	pr := &ProjectReader{lister: &mockLister{modules: fakeModules}}
	result, err := pr.ListInstalledModules("")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(result))
	}
	if result[0].Name != "DatabaseConnector" {
		t.Errorf("first: got %q, want DatabaseConnector", result[0].Name)
	}
	if result[1].AppStoreVersion != "11.5.0" {
		t.Errorf("version: got %q, want 11.5.0", result[1].AppStoreVersion)
	}
}

type mockLister struct {
	modules []*model.Module
}

func (m *mockLister) ListModules() ([]*model.Module, error) {
	return m.modules, nil
}

func TestProjectReader_EmptyIfNoneFromAppStore(t *testing.T) {
	pr := &ProjectReader{lister: &mockLister{
		modules: []*model.Module{
			{Name: "MyModule", FromAppStore: false},
			{Name: "OtherModule", FromAppStore: false},
		},
	}}
	result, err := pr.ListInstalledModules("")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0, got %d", len(result))
	}
}


