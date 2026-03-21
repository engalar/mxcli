// SPDX-License-Identifier: Apache-2.0

package executor

import "testing"

func TestUnquoteQualifiedName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Module.Entity", "Module.Entity"},
		{`Module."Entity"`, "Module.Entity"},
		{`"Module"."Entity"`, "Module.Entity"},
		{`MaisonElegance."Collection"`, "MaisonElegance.Collection"},
		{`MaisonElegance."FormSubmissionStatus".StatusNew`, "MaisonElegance.FormSubmissionStatus.StatusNew"},
		{"SimpleName", "SimpleName"},
		{`"QuotedOnly"`, "QuotedOnly"},
		{"", ""},
	}
	for _, tt := range tests {
		got := unquoteQualifiedName(tt.input)
		if got != tt.want {
			t.Errorf("unquoteQualifiedName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
