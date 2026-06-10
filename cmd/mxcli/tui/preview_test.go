package tui

import "testing"

func TestBuildDescribeCmd(t *testing.T) {
	tests := []struct {
		nodeType      string
		qualifiedName string
		want          string
	}{
		// Security multi-word types must produce correct grammar
		{"modulerole", "Administration.User", "DESCRIBE MODULE ROLE Administration.User"},
		{"modulerole", "MyModule.Admin", "DESCRIBE MODULE ROLE MyModule.Admin"},
		{"userrole", "User", "DESCRIBE USER ROLE 'User'"},
		{"userrole", "Administrator", "DESCRIBE USER ROLE 'Administrator'"},
		{"demouser", "demo_user", "DESCRIBE DEMO USER 'demo_user'"},
		{"demouser", "MerchantUser", "DESCRIBE DEMO USER 'MerchantUser'"},

		// Case-insensitive matching for node type
		{"ModuleRole", "MyModule.Admin", "DESCRIBE MODULE ROLE MyModule.Admin"},
		{"UserRole", "User", "DESCRIBE USER ROLE 'User'"},
		{"DemoUser", "demo_user", "DESCRIBE DEMO USER 'demo_user'"},

		// Virtual root node: show structure overview
		{"systemoverview", "SystemOverview", "SHOW STRUCTURE DEPTH 2"},
		{"SystemOverview", "SystemOverview", "SHOW STRUCTURE DEPTH 2"},

		// Virtual container nodes: no valid DESCRIBE, return empty string
		{"security", "", ""},
		{"category", "", ""},
		{"domainmodel", "", ""},
		{"navigation", "", ""},
		{"projectsecurity", "", ""},
		{"navprofile", "", ""},

		// Image collection types
		{"imagecollection", "Atlas_Core.Web", "DESCRIBE IMAGE COLLECTION Atlas_Core.Web"},
		{"ImageCollection", "MyModule.Images", "DESCRIBE IMAGE COLLECTION MyModule.Images"},

		// Generic types fall through to default
		{"entity", "MyModule.Customer", "DESCRIBE ENTITY MyModule.Customer"},
		{"microflow", "MyModule.DoSomething", "DESCRIBE MICROFLOW MyModule.DoSomething"},
		{"page", "MyModule.Home_Overview", "DESCRIBE PAGE MyModule.Home_Overview"},

		// Layout types fall through to default (case-insensitive)
		{"layout", "Atlas_Core.Atlas_Default", "DESCRIBE LAYOUT Atlas_Core.Atlas_Default"},
		{"layout", "MyModule.MyLayout", "DESCRIBE LAYOUT MyModule.MyLayout"},
		{"Layout", "MyModule.MyLayout", "DESCRIBE LAYOUT MyModule.MyLayout"},
	}

	for _, tc := range tests {
		got := buildDescribeCmd(tc.nodeType, tc.qualifiedName)
		if got != tc.want {
			t.Errorf("buildDescribeCmd(%q, %q)\n  got:  %q\n  want: %q",
				tc.nodeType, tc.qualifiedName, got, tc.want)
		}
	}
}

func TestExtractImagePaths_LowercaseKeywords(t *testing.T) {
	// MDL output uses lowercase keywords since commit f70a74158
	output := `create or modify image collection MyModule.Icons (
    image logo from file '/tmp/mxcli-preview/MyModule.Icons/logo.png',
    image banner from file '/tmp/mxcli-preview/MyModule.Icons/banner.svg'
);`
	paths := extractImagePaths(output)
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
	if paths[0] != "/tmp/mxcli-preview/MyModule.Icons/logo.png" {
		t.Errorf("paths[0] = %q, want /tmp/mxcli-preview/MyModule.Icons/logo.png", paths[0])
	}
	if paths[1] != "/tmp/mxcli-preview/MyModule.Icons/banner.svg" {
		t.Errorf("paths[1] = %q, want /tmp/mxcli-preview/MyModule.Icons/banner.svg", paths[1])
	}
}
