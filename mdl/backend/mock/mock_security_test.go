// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/model"
)

func TestMockBackend_GetProjectSecurity_DefaultError(t *testing.T) {
	m := &MockBackend{}
	_, err := m.GetProjectSecurity()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "MockBackend.GetProjectSecurity not configured") {
		t.Errorf("expected descriptive error, got: %v", err)
	}
}

func TestMockBackend_GetModuleSecurity_DefaultError(t *testing.T) {
	m := &MockBackend{}
	_, err := m.GetModuleSecurity(model.ID(""))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "MockBackend.GetModuleSecurity not configured") {
		t.Errorf("expected descriptive error, got: %v", err)
	}
}

func TestMockBackend_ListModuleSecurity_DefaultError(t *testing.T) {
	m := &MockBackend{}
	_, err := m.ListModuleSecurity()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "MockBackend.ListModuleSecurity not configured") {
		t.Errorf("expected descriptive error, got: %v", err)
	}
}
