// SPDX-License-Identifier: Apache-2.0

package executor

import "testing"

func TestResolveHelpPath(t *testing.T) {
	tests := []struct {
		name  string
		words []string
		want  string
	}{
		{"single word exact", []string{"workflow"}, "workflow"},
		{"single word domain-model", []string{"domain-model"}, "domain-model"},
		{"multi-word no match falls through", []string{"user", "task"}, "user.task"},
		{"multi-level path", []string{"workflow", "user", "task"}, "workflow.user-task"},
		{"three-level path", []string{"workflow", "user", "task", "targeting"}, "workflow.user-task.targeting"},
		{"security prefix", []string{"security", "entity", "access"}, "security.entity-access"},
		{"non-existent topic", []string{"nonexistent"}, "nonexistent"},
		{"mixed known and unknown", []string{"workflow", "bogus"}, "workflow.bogus"},
		{"single known nested", []string{"navigation", "create"}, "navigation.create"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveHelpPath(tt.words)
			if got != tt.want {
				t.Errorf("resolveHelpPath(%v) = %q, want %q", tt.words, got, tt.want)
			}
		})
	}
}
