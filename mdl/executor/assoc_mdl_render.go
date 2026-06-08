// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// assocMDLSpec holds the resolved fields needed to render an association as MDL.
type assocMDLSpec struct {
	module         string
	name           string
	fromQN         string // FROM entity (ParentPointer) qualified name
	toQN           string // TO entity (ChildPointer) qualified name
	documentation  string
	assocType      string // "Reference" | "ReferenceSet"
	owner          string // "Default" | "Both"
	deleteBehavior string // MDL delete-behavior keyword
}

// renderAssocMDL renders an association spec as deterministic MDL text.
// Uses "create or modify association" for idempotent re-import semantics,
// and "delete_behavior" (underscore form) matching the canonical MDL syntax.
func renderAssocMDL(spec assocMDLSpec) string {
	var sb strings.Builder
	if spec.documentation != "" {
		fmt.Fprintf(&sb, "/**\n * %s\n */\n", spec.documentation)
	}
	fmt.Fprintf(&sb, "create or modify association %s.%s\n", spec.module, spec.name)
	fmt.Fprintf(&sb, "from %s to %s\n", spec.fromQN, spec.toQN)
	fmt.Fprintf(&sb, "type %s\n", spec.assocType)
	fmt.Fprintf(&sb, "owner %s\n", spec.owner)
	fmt.Fprintf(&sb, "delete_behavior %s", spec.deleteBehavior)
	return sb.String()
}

// assocSpecFromAST builds a spec from a parsed CREATE ASSOCIATION statement.
func assocSpecFromAST(s *ast.CreateAssociationStmt) assocMDLSpec {
	return assocMDLSpec{
		module:         s.Name.Module,
		name:           s.Name.Name,
		fromQN:         s.Parent.Module + "." + s.Parent.Name,
		toQN:           s.Child.Module + "." + s.Child.Name,
		documentation:  s.Documentation,
		assocType:      astAssocTypeStr(s.Type),
		owner:          astOwnerStr(s.Owner),
		deleteBehavior: astDeleteBehaviorStr(s.DeleteBehavior),
	}
}

func astAssocTypeStr(t ast.AssociationType) string {
	if t == ast.AssocReferenceSet {
		return "ReferenceSet"
	}
	return "Reference"
}

func astOwnerStr(o ast.OwnerType) string {
	if o == ast.OwnerBoth {
		return "Both"
	}
	return "Default"
}

func astDeleteBehaviorStr(d ast.DeleteBehavior) string {
	switch d {
	case ast.DeleteCascade:
		return "DELETE_AND_REFERENCES"
	case ast.DeleteIfNoReferences:
		return "DELETE_IF_NO_REFERENCES"
	default:
		return "DELETE_BUT_KEEP_REFERENCES"
	}
}

// assocSpecFromGen builds a spec from a gen-typed Association read from a project.
// entityNames maps entity ID strings to qualified names.
func assocSpecFromGen(moduleName string, a *genDm.Association, entityNames map[string]string) assocMDLSpec {
	fromQN := entityNames[string(a.ParentRefID())]
	if fromQN == "" {
		fromQN = string(a.ParentRefID())
	}
	toQN := entityNames[string(a.ChildRefID())]
	if toQN == "" {
		toQN = string(a.ChildRefID())
	}

	assocType := "Reference"
	if a.Type() == "ReferenceSet" {
		assocType = "ReferenceSet"
	}

	owner := "Default"
	if a.Owner() == "Both" {
		owner = "Both"
	}

	deleteBehavior := "DELETE_BUT_KEEP_REFERENCES"
	if dbe, ok := a.DeleteBehavior().(*genDm.AssociationDeleteBehavior); ok && dbe != nil {
		deleteBehavior = genAssocDeleteBehaviorToMDL(dbe.ChildDeleteBehavior())
	}

	return assocMDLSpec{
		module:         moduleName,
		name:           a.Name(),
		fromQN:         fromQN,
		toQN:           toQN,
		documentation:  a.Documentation(),
		assocType:      assocType,
		owner:          owner,
		deleteBehavior: deleteBehavior,
	}
}

// genAssocDeleteBehaviorToMDL maps a gen ChildDeleteBehavior string to its MDL keyword.
func genAssocDeleteBehaviorToMDL(child string) string {
	switch child {
	case "DeleteMeAndReferences":
		return "DELETE_AND_REFERENCES"
	case "DeleteMeIfNoReferences", "DeleteIfNoReferences":
		return "DELETE_IF_NO_REFERENCES"
	default: // "DeleteMeButKeepReferences" and any unknown value
		return "DELETE_BUT_KEEP_REFERENCES"
	}
}
