// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.f4b — gen-typed addRetrieveActionGen.
//
// Mirrors flowBuilder.addRetrieveAction surface:
//
//   - StartVariable empty → DatabaseRetrieveSource w/ entity QN +
//     optional XPath (from WHERE), Range (from LIMIT/OFFSET), and
//     SortItemList (from SORT BY).
//   - StartVariable set → AssociationRetrieveSource w/ start var +
//     association QN. The legacy reverse-Reference detection (which
//     promotes a child→parent navigation to a DatabaseRetrieveSource
//     with XPath constraint) requires backend-driven domain-model
//     metadata; this gen path delegates that decision to the legacy
//     resolver via the `lookupAssociation` adapter — when offline it
//     defaults to the AssociationRetrieveSource shape and "List of E"
//     output type.
//
// Range mapping:
//
//   - LIMIT 1, no offset → ConstantRange{SingleObject: true}
//     (matches legacy RangeTypeFirst)
//   - any other LIMIT or any OFFSET → CustomRange{Limit, Offset}
//
// Output variable type registration:
//
//   - LIMIT 1 → single entity ("Module.Entity")
//   - else → "List of Module.Entity"
//   - association retrieve (Reference forward) → "Module.Entity"
//     (reverse / ReferenceSet → "List of Module.Entity")
//   - offline default for association retrieve: "List of Module.Assoc"
//     (matches legacy fallback when assocInfo is nil)

package executor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
)

// addRetrieveActionGen emits a `retrieve $V from ...` activity.
func (fb *flowBuilderGen) addRetrieveActionGen(s *ast.RetrieveStmt) element.ID {
	var source element.Element
	if s.StartVariable != "" {
		source = fb.buildAssociationRetrieveSourceGen(s)
	} else {
		source = fb.buildDatabaseRetrieveSourceGen(s)
	}

	action := genMf.NewRetrieveAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetOutputVariableName(s.Variable)
	action.SetRetrieveSource(source)

	return fb.genActivityWrap(action, s.ErrorHandling, s.Variable)
}

// buildDatabaseRetrieveSourceGen constructs a DatabaseRetrieveSource
// from a RetrieveStmt with no StartVariable. Sets entity QN, optional
// XPath (WHERE), Range (LIMIT/OFFSET), and SortItemList (SORT BY).
// Also registers the output variable's type in fb.varTypes.
func (fb *flowBuilderGen) buildDatabaseRetrieveSourceGen(s *ast.RetrieveStmt) *genMf.DatabaseRetrieveSource {
	entityQN := s.Source.Module + "." + s.Source.Name

	src := genMf.NewDatabaseRetrieveSource()
	assignFreshID(src)
	src.SetEntityQualifiedName(entityQN)

	if s.Where != nil {
		src.SetXPathConstraint(retrieveXPathConstraint(s.Where))
	}

	if s.Limit != "" {
		src.SetRange(buildRetrieveRangeGen(s.Limit, s.Offset))
	}

	if len(s.SortColumns) > 0 {
		src.SetSortItemList(buildRetrieveSortItemListGen(s.SortColumns, entityQN))
	}

	if fb.varTypes != nil {
		if s.Limit == "1" {
			fb.varTypes[s.Variable] = entityQN
		} else {
			fb.varTypes[s.Variable] = "List of " + entityQN
		}
	}
	return src
}

// buildAssociationRetrieveSourceGen constructs an
// AssociationRetrieveSource. The legacy reverse-Reference detection
// (which would expand a child→parent traversal into a DB source with
// XPath) needs backend-driven association metadata; we attempt the
// same lookup here via the legacy adapter and fall back to the bare
// association source when no metadata is available.
func (fb *flowBuilderGen) buildAssociationRetrieveSourceGen(s *ast.RetrieveStmt) element.Element {
	assocQN := s.Source.Module + "." + s.Source.Name

	// Backend-driven reverse-Reference promotion (when metadata
	// available). Stage 3.2.6.4: standalone helpers replaced the
	// legacy `flowBuilder` adapter — both `lookupAssociationGen` and
	// `entityIsSubtypeOfGen` are pure-read and live in
	// flowbuilder_assoc_lookup_gen.go.
	if fb.backend != nil {
		assocInfo := lookupAssociationGen(fb.backend, s.Source.Module, s.Source.Name)
		startVarType := ""
		if fb.varTypes != nil {
			startVarType = fb.varTypes[s.StartVariable]
		}
		startsFromChildSide := assocInfo != nil &&
			assocInfo.childEntityQN != "" &&
			entityIsSubtypeOfGen(fb.backend, startVarType, assocInfo.childEntityQN)

		if assocInfo != nil &&
			assocInfo.Type == domainmodel.AssociationTypeReference &&
			assocInfo.Owner != "" &&
			assocInfo.parentPersistable &&
			assocInfo.childEntityQN != "" &&
			startsFromChildSide &&
			(assocInfo.Owner != domainmodel.AssociationOwnerBoth) {
			// Reverse traversal: child → parent (one-to-many).
			src := genMf.NewDatabaseRetrieveSource()
			assignFreshID(src)
			src.SetEntityQualifiedName(assocInfo.parentEntityQN)
			src.SetXPathConstraint("[" + assocQN + " = $" + s.StartVariable + "]")
			if fb.varTypes != nil {
				fb.varTypes[s.Variable] = "List of " + assocInfo.parentEntityQN
			}
			return src
		}

		// Forward traversal — register the produced output type
		// based on the association's kind.
		if fb.varTypes != nil && assocInfo != nil {
			otherEntity := assocInfo.childEntityQN
			if startsFromChildSide {
				otherEntity = assocInfo.parentEntityQN
			}
			switch assocInfo.Type {
			case domainmodel.AssociationTypeReference:
				if startsFromChildSide {
					fb.varTypes[s.Variable] = "List of " + otherEntity
				} else {
					fb.varTypes[s.Variable] = otherEntity
				}
			case domainmodel.AssociationTypeReferenceSet:
				if otherEntity != "" {
					fb.varTypes[s.Variable] = "List of " + otherEntity
				} else {
					fb.varTypes[s.Variable] = "List of " + assocQN
				}
			default:
				fb.varTypes[s.Variable] = "List of " + assocQN
			}
		}
	}

	// Default association source (offline path / forward traversal).
	src := genMf.NewAssociationRetrieveSource()
	assignFreshID(src)
	src.SetStartVariableName(s.StartVariable)
	src.SetAssociationQualifiedName(assocQN)

	// Offline fallback: register output as "List of <assoc QN>" if
	// the backend branch above didn't already set it.
	if fb.varTypes != nil {
		if _, set := fb.varTypes[s.Variable]; !set {
			fb.varTypes[s.Variable] = "List of " + assocQN
		}
	}
	return src
}

// buildRetrieveRangeGen returns a *genMf.ConstantRange (when LIMIT
// is "1" with no offset, mirroring legacy RangeTypeFirst) or a
// *genMf.CustomRange wrapping the limit/offset expressions.
func buildRetrieveRangeGen(limit, offset string) element.Element {
	if limit == "1" && offset == "" {
		cr := genMf.NewConstantRange()
		assignFreshID(cr)
		cr.SetSingleObject(true)
		return cr
	}
	cr := genMf.NewCustomRange()
	assignFreshID(cr)
	if limit != "" {
		cr.SetLimitExpression(limit)
	}
	if offset != "" {
		cr.SetOffsetExpression(offset)
	}
	return cr
}

// buildRetrieveSortItemListGen builds a SortItemList from
// RetrieveStmt.SortColumns. Bare attributes are qualified with the
// retrieve's entity QN; pre-qualified `Module.Entity.Attribute`
// names pass through. Ascending vs descending decided
// case-insensitively from col.Order.
func buildRetrieveSortItemListGen(columns []ast.SortColumnDef, entityQN string) *genMf.SortItemList {
	itemList := genMf.NewSortItemList()
	assignFreshID(itemList)
	for _, col := range columns {
		attrPath := col.Attribute
		if !strings.Contains(attrPath, ".") {
			attrPath = entityQN + "." + col.Attribute
		}
		order := genMf.SortOrderEnumAscending
		if strings.EqualFold(col.Order, "desc") {
			order = genMf.SortOrderEnumDescending
		}
		item := genMf.NewSortItem()
		assignFreshID(item)
		item.SetAttributePath(attrPath)
		item.SetSortOrder(order)
		itemList.AddItems(item)
	}
	return itemList
}
