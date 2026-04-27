package version

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  Version
	}{
		{"10.6.3", Version{10, 6, 3}},
		{"9.17", Version{9, 17, 0}},
		{"0.0.0", Version{0, 0, 0}},
		{"1.0.0", Version{1, 0, 0}},
		{"11.8.0", Version{11, 8, 0}},
		{"", Version{0, 0, 0}},
	}

	for _, tt := range tests {
		got := Parse(tt.input)
		if got != tt.want {
			t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestVersionString(t *testing.T) {
	v := Version{10, 6, 3}
	if got := v.String(); got != "10.6.3" {
		t.Errorf("String() = %q, want %q", got, "10.6.3")
	}
}

func TestVersionIsZero(t *testing.T) {
	zero := Version{}
	if !zero.IsZero() {
		t.Error("zero Version should be IsZero() == true")
	}
	nonZero := Version{Major: 1}
	if nonZero.IsZero() {
		t.Error("non-zero Version should be IsZero() == false")
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		a, b Version
		want int
	}{
		// equal
		{Version{10, 6, 3}, Version{10, 6, 3}, 0},
		{Version{0, 0, 0}, Version{0, 0, 0}, 0},
		// less
		{Version{9, 17, 0}, Version{10, 6, 3}, -1},
		{Version{10, 5, 9}, Version{10, 6, 3}, -1},
		{Version{10, 6, 2}, Version{10, 6, 3}, -1},
		// greater
		{Version{10, 6, 3}, Version{9, 17, 0}, 1},
		{Version{10, 7, 0}, Version{10, 6, 3}, 1},
		{Version{10, 6, 4}, Version{10, 6, 3}, 1},
	}

	for _, tt := range tests {
		got := tt.a.Compare(tt.b)
		if got != tt.want {
			t.Errorf("(%v).Compare(%v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestPropertyVersionInfo_IsAvailableIn(t *testing.T) {
	tests := []struct {
		name string
		prop PropertyVersionInfo
		v    Version
		want bool
	}{
		{
			name: "always available (empty strings)",
			prop: PropertyVersionInfo{},
			v:    Version{10, 6, 3},
			want: true,
		},
		{
			name: "available — version meets introduced requirement",
			prop: PropertyVersionInfo{Introduced: "10.0.0"},
			v:    Version{10, 6, 3},
			want: true,
		},
		{
			name: "not yet introduced — version is below introduced",
			prop: PropertyVersionInfo{Introduced: "10.6.0"},
			v:    Version{9, 24, 0},
			want: false,
		},
		{
			name: "after deletion — version equals deleted",
			prop: PropertyVersionInfo{Introduced: "9.0.0", Deleted: "11.0.0"},
			v:    Version{11, 0, 0},
			want: false,
		},
		{
			name: "after deletion — version is beyond deleted",
			prop: PropertyVersionInfo{Deleted: "10.0.0"},
			v:    Version{11, 8, 0},
			want: false,
		},
		{
			name: "available — version is before deletion",
			prop: PropertyVersionInfo{Introduced: "9.0.0", Deleted: "11.0.0"},
			v:    Version{10, 24, 0},
			want: true,
		},
		{
			name: "exactly at introduced boundary",
			prop: PropertyVersionInfo{Introduced: "10.6.0"},
			v:    Version{10, 6, 0},
			want: true,
		},
		{
			name: "just before introduced boundary",
			prop: PropertyVersionInfo{Introduced: "10.6.0"},
			v:    Version{10, 5, 9},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.prop.IsAvailableIn(tt.v)
			if got != tt.want {
				t.Errorf("IsAvailableIn(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}
