// SPDX-License-Identifier: Apache-2.0

package association

import (
	"fmt"
	"strings"
)

// ToMDL renders the canonical association as deterministic MDL text.
func (m *AssociationModel) ToMDL() string {
	var sb strings.Builder
	if m.Documentation != "" {
		fmt.Fprintf(&sb, "/**\n * %s\n */\n", m.Documentation)
	}
	fmt.Fprintf(&sb, "create association %s\n", m.Name)
	fmt.Fprintf(&sb, "from %s to %s\n", m.From, m.To)
	fmt.Fprintf(&sb, "type %s\n", assocTypeStr(m.Type))
	fmt.Fprintf(&sb, "owner %s\n", ownerStr(m.Owner))
	fmt.Fprintf(&sb, "delete behavior %s", deleteBehaviorStr(m.DeleteBehavior))
	return sb.String()
}

func assocTypeStr(t AssocType) string {
	if t == AssocReferenceSet {
		return "ReferenceSet"
	}
	return "Reference"
}

func ownerStr(o OwnerType) string {
	if o == OwnerBoth {
		return "Both"
	}
	return "Default"
}

func deleteBehaviorStr(d DeleteBehaviorType) string {
	switch d {
	case DeleteCascade:
		return "DeleteMeAndReferences"
	case DeleteBoth:
		return "DeleteBoth"
	case DeleteKeepParentDeleteChild:
		return "KeepParentDeleteChild"
	case DeleteKeepChildDeleteParent:
		return "KeepChildDeleteParent"
	case DeleteIfNoReferences:
		return "DeleteIfNoReferences"
	default:
		return "DeleteMeButKeepReferences"
	}
}
