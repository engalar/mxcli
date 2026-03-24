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

		// Types with no MDL DESCRIBE command return empty (NDSL-only)
		{"imagecollection", "Atlas_Core.Web", ""},
		{"ImageCollection", "MyModule.Images", ""},

		// Generic types fall through to default
		{"entity", "MyModule.Customer", "DESCRIBE ENTITY MyModule.Customer"},
		{"microflow", "MyModule.DoSomething", "DESCRIBE MICROFLOW MyModule.DoSomething"},
		{"page", "MyModule.Home_Overview", "DESCRIBE PAGE MyModule.Home_Overview"},
	}

	for _, tc := range tests {
		got := buildDescribeCmd(tc.nodeType, tc.qualifiedName)
		if got != tc.want {
			t.Errorf("buildDescribeCmd(%q, %q)\n  got:  %q\n  want: %q",
				tc.nodeType, tc.qualifiedName, got, tc.want)
		}
	}
}
