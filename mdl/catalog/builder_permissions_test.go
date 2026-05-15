// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"testing"

	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

func TestEntityAccessFromMemberRightsGen(t *testing.T) {
	tests := []struct {
		name      string
		rule      *genDm.AccessRule
		wantRead  bool
		wantWrite bool
	}{
		{
			name:      "no member accesses, default None",
			rule:      ruleWithDefault("None"),
			wantRead:  false,
			wantWrite: false,
		},
		{
			name:      "no member accesses, default ReadOnly",
			rule:      ruleWithDefault("ReadOnly"),
			wantRead:  true,
			wantWrite: false,
		},
		{
			name:      "no member accesses, default ReadWrite",
			rule:      ruleWithDefault("ReadWrite"),
			wantRead:  true,
			wantWrite: true,
		},
		{
			name:      "explicit member ReadOnly",
			rule:      ruleWithMembers("ReadOnly"),
			wantRead:  true,
			wantWrite: false,
		},
		{
			name:      "explicit member ReadWrite",
			rule:      ruleWithMembers("ReadWrite"),
			wantRead:  true,
			wantWrite: true,
		},
		{
			name:      "mixed members — one ReadOnly one None",
			rule:      ruleWithMembers("ReadOnly", "None"),
			wantRead:  true,
			wantWrite: false,
		},
		{
			name:      "mixed members — one ReadOnly one ReadWrite",
			rule:      ruleWithMembers("ReadOnly", "ReadWrite"),
			wantRead:  true,
			wantWrite: true,
		},
		{
			name:      "all members None",
			rule:      ruleWithMembers("None"),
			wantRead:  false,
			wantWrite: false,
		},
		{
			name:      "empty member accesses falls through to default",
			rule:      ruleWithDefault("ReadOnly"),
			wantRead:  true,
			wantWrite: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRead, gotWrite := entityAccessFromMemberRightsGen(tt.rule)
			if gotRead != tt.wantRead {
				t.Errorf("hasRead = %v, want %v", gotRead, tt.wantRead)
			}
			if gotWrite != tt.wantWrite {
				t.Errorf("hasWrite = %v, want %v", gotWrite, tt.wantWrite)
			}
		})
	}
}

func ruleWithDefault(defaultRights string) *genDm.AccessRule {
	rule := genDm.NewAccessRule()
	rule.SetDefaultMemberAccessRights(defaultRights)
	return rule
}

func ruleWithMembers(rights ...string) *genDm.AccessRule {
	rule := genDm.NewAccessRule()
	for i, right := range rights {
		member := genDm.NewMemberAccess()
		member.SetAttributeQualifiedName("Module.Entity.Attr")
		if i > 0 {
			member.SetAttributeQualifiedName("Module.Entity.Attr" + string(rune('A'+i)))
		}
		member.SetAccessRights(right)
		rule.AddMemberAccesses(member)
	}
	return rule
}
