// SPDX-License-Identifier: Apache-2.0

package versions

import (
	"testing"
)

func TestLoad(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(r.specs) != 3 {
		t.Fatalf("expected 3 specs, got %d", len(r.specs))
	}
	if len(r.index) == 0 {
		t.Fatal("index is empty after load")
	}
}

func TestParseSemVer(t *testing.T) {
	tests := []struct {
		input   string
		want    SemVer
		wantErr bool
	}{
		{"10.18.0", SemVer{10, 18, 0}, false},
		{"11.0.0", SemVer{11, 0, 0}, false},
		{"9.24", SemVer{9, 24, 0}, false},
		{"bad", SemVer{}, true},
	}
	for _, tt := range tests {
		v, err := ParseSemVer(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseSemVer(%q) error=%v, wantErr=%v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && v != tt.want {
			t.Errorf("ParseSemVer(%q) = %v, want %v", tt.input, v, tt.want)
		}
	}
}

func TestSemVerAtLeast(t *testing.T) {
	tests := []struct {
		v, other SemVer
		want     bool
	}{
		{SemVer{10, 18, 0}, SemVer{10, 18, 0}, true},
		{SemVer{11, 0, 0}, SemVer{10, 18, 0}, true},
		{SemVer{10, 17, 0}, SemVer{10, 18, 0}, false},
		{SemVer{10, 18, 1}, SemVer{10, 18, 0}, true},
		{SemVer{10, 18, 0}, SemVer{10, 18, 1}, false},
		{SemVer{9, 0, 0}, SemVer{10, 0, 0}, false},
	}
	for _, tt := range tests {
		got := tt.v.AtLeast(tt.other)
		if got != tt.want {
			t.Errorf("%v.AtLeast(%v) = %v, want %v", tt.v, tt.other, got, tt.want)
		}
	}
}

func TestIsAvailable(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	tests := []struct {
		area, name string
		version    SemVer
		want       bool
	}{
		// View entities require 10.18+
		{"domain_model", "view_entities", SemVer{10, 18, 0}, true},
		{"domain_model", "view_entities", SemVer{10, 17, 0}, false},
		{"domain_model", "view_entities", SemVer{11, 0, 0}, true},
		// Basic entities available in 9.x+
		{"domain_model", "entities", SemVer{9, 0, 0}, true},
		// Page parameters require 11.0+
		{"pages", "page_parameters", SemVer{10, 24, 0}, false},
		{"pages", "page_parameters", SemVer{11, 0, 0}, true},
		// Unknown feature
		{"domain_model", "teleportation", SemVer{11, 0, 0}, false},
	}
	for _, tt := range tests {
		got := r.IsAvailable(tt.area, tt.name, tt.version)
		if got != tt.want {
			t.Errorf("IsAvailable(%s, %s, %v) = %v, want %v",
				tt.area, tt.name, tt.version, got, tt.want)
		}
	}
}

func TestFeaturesForVersion(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	features := r.FeaturesForVersion(SemVer{10, 24, 0})
	if len(features) == 0 {
		t.Fatal("expected features for 10.24.0, got none")
	}

	// Check that view_entities is available at 10.24
	found := false
	for _, f := range features {
		if f.Area == "domain_model" && f.Name == "view_entities" {
			found = true
			if !f.Available {
				t.Error("view_entities should be available at 10.24")
			}
		}
	}
	if !found {
		t.Error("view_entities not found in features list")
	}

	// Check that page_parameters is NOT available at 10.24
	for _, f := range features {
		if f.Area == "pages" && f.Name == "page_parameters" {
			if f.Available {
				t.Error("page_parameters should NOT be available at 10.24")
			}
		}
	}
}

func TestFeaturesAddedSince(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	added := r.FeaturesAddedSince(SemVer{10, 24, 0})
	if len(added) == 0 {
		t.Fatal("expected features added since 10.24.0, got none")
	}

	// page_parameters (11.0+) should be in the list
	found := false
	for _, f := range added {
		if f.Name == "page_parameters" {
			found = true
			break
		}
	}
	if !found {
		t.Error("page_parameters should appear in features added since 10.24")
	}

	// entities (10.0+) should NOT be in the list
	for _, f := range added {
		if f.Name == "entities" && f.Area == "domain_model" {
			t.Error("entities should not appear in features added since 10.24")
		}
	}
}

func TestFeaturesInArea(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	features := r.FeaturesInArea("domain_model", SemVer{10, 24, 0})
	if len(features) == 0 {
		t.Fatal("expected domain_model features, got none")
	}

	for _, f := range features {
		if f.Area != "domain_model" {
			t.Errorf("expected area domain_model, got %s", f.Area)
		}
	}
}

func TestAreas(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	areas := r.Areas()
	if len(areas) == 0 {
		t.Fatal("expected areas, got none")
	}

	// Should contain at least domain_model and microflows
	areaSet := make(map[string]bool)
	for _, a := range areas {
		areaSet[a] = true
	}
	for _, want := range []string{"domain_model", "microflows", "pages", "security"} {
		if !areaSet[want] {
			t.Errorf("expected area %q in list", want)
		}
	}
}

func TestDeprecatedPatterns(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	deps := r.DeprecatedPatterns(SemVer{10, 24, 0})
	if len(deps) == 0 {
		t.Fatal("expected deprecated patterns for 10.24, got none")
	}
	if deps[0].ID != "DEP001" {
		t.Errorf("expected DEP001, got %s", deps[0].ID)
	}
}

func TestUpgradeOpportunities(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	ups := r.UpgradeOpportunities(10, 11)
	if len(ups) == 0 {
		t.Fatal("expected upgrade opportunities from 10 to 11, got none")
	}

	// Should not find upgrades for nonexistent path
	ups = r.UpgradeOpportunities(9, 10)
	if len(ups) != 0 {
		t.Errorf("expected no upgrade opportunities from 9 to 10, got %d", len(ups))
	}
}

func TestDisplayName(t *testing.T) {
	e := FeatureEntry{Name: "view_entities"}
	if got := e.DisplayName(); got != "view entities" {
		t.Errorf("DisplayName() = %q, want %q", got, "view entities")
	}
}
