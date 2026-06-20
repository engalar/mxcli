// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.f4a — gen-typed ListOperationAction adder.
//
// 13 list-operation primitives covered, mirroring the legacy
// flowBuilder.addListOperationAction switch:
//
//   ast.ListOpHead       → *genMf.Head
//   ast.ListOpTail       → *genMf.Tail
//   ast.ListOpFind       → *genMf.Find (binary= attr) | *genMf.FindByExpression
//   ast.ListOpFilter     → *genMf.Filter (binary= attr) | *genMf.FilterByExpression
//   ast.ListOpSort       → *genMf.Sort with SortItemList of SortItem
//   ast.ListOpUnion      → *genMf.Union
//   ast.ListOpIntersect  → *genMf.Intersect
//   ast.ListOpSubtract   → *genMf.Subtract
//   ast.ListOpContains   → *genMf.Contains
//   ast.ListOpEquals     → *genMf.ListEquals
//   ast.ListOpRange      → *genMf.ListRange wrapping *genMf.CustomRange
//
// The Find / Filter `binary= attr` attribute-shortcut path falls back
// to its expression sibling when the condition isn't a bare
// `attr = expr` BinaryExpr — same predicate as legacy
// listAttributeOperation, ported to operate on gen elements.
//
// Output-variable type tracking parity:
//
//   - Filter / Sort / Tail / Union / Intersect / Subtract / Range
//     preserve the input list type (`List of E` → `List of E`).
//   - Head / Find unwrap the list element type (`List of E` → `E`).
//   - Contains / Equals return Boolean — varTypes is intentionally
//     not updated (the existing builder convention).

package executor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// addListOperationActionGen emits a list-operation activity. The
// concrete operation element is dispatched by s.Operation; unknown
// op kinds return "" (empty ID) to mirror legacy behaviour.
func (fb *flowBuilderGen) addListOperationActionGen(s *ast.ListOperationStmt) element.ID {
	op := fb.buildListOperationGen(s)
	if op == nil {
		return ""
	}

	action := genMf.NewListOperationAction()
	assignFreshID(action)
	action.SetOperation(op)
	action.SetOutputVariableName(s.OutputVariable)

	fb.trackListOperationOutputType(s)

	return fb.genActivityWrap(action, nil, "")
}

// buildListOperationGen dispatches on s.Operation and returns the
// concrete gen operation element. Returns nil for unknown kinds.
func (fb *flowBuilderGen) buildListOperationGen(s *ast.ListOperationStmt) element.Element {
	switch s.Operation {
	case ast.ListOpHead:
		op := genMf.NewHead()
		assignFreshID(op)
		op.SetListVariableName(s.InputVariable)
		return op
	case ast.ListOpTail:
		op := genMf.NewTail()
		assignFreshID(op)
		op.SetListVariableName(s.InputVariable)
		return op
	case ast.ListOpFind:
		if op := fb.listAttributeFindGen(s); op != nil {
			return op
		}
		op := genMf.NewFindByExpression()
		assignFreshID(op)
		op.SetListVariableName(s.InputVariable)
		op.SetExpression(fb.exprToString(s.Condition))
		return op
	case ast.ListOpFilter:
		if op := fb.listAttributeFilterGen(s); op != nil {
			return op
		}
		op := genMf.NewFilterByExpression()
		assignFreshID(op)
		op.SetListVariableName(s.InputVariable)
		op.SetExpression(fb.exprToString(s.Condition))
		return op
	case ast.ListOpSort:
		op := genMf.NewSort()
		assignFreshID(op)
		op.SetListVariableName(s.InputVariable)
		op.SetSortItemList(fb.buildSortItemListGen(s))
		return op
	case ast.ListOpUnion:
		op := genMf.NewUnion()
		assignFreshID(op)
		op.SetListVariableName(s.InputVariable)
		op.SetSecondListOrObjectVariableName(s.SecondVariable)
		return op
	case ast.ListOpIntersect:
		op := genMf.NewIntersect()
		assignFreshID(op)
		op.SetListVariableName(s.InputVariable)
		op.SetSecondListOrObjectVariableName(s.SecondVariable)
		return op
	case ast.ListOpSubtract:
		op := genMf.NewSubtract()
		assignFreshID(op)
		op.SetListVariableName(s.InputVariable)
		op.SetSecondListOrObjectVariableName(s.SecondVariable)
		return op
	case ast.ListOpContains:
		op := genMf.NewContains()
		assignFreshID(op)
		op.SetListVariableName(s.InputVariable)
		op.SetSecondListOrObjectVariableName(s.SecondVariable)
		return op
	case ast.ListOpEquals:
		op := genMf.NewListEquals()
		assignFreshID(op)
		op.SetListVariableName(s.InputVariable)
		op.SetSecondListOrObjectVariableName(s.SecondVariable)
		return op
	case ast.ListOpRange:
		op := genMf.NewListRange()
		assignFreshID(op)
		op.SetListVariableName(s.InputVariable)
		cr := genMf.NewCustomRange()
		assignFreshID(cr)
		if s.OffsetExpr != nil {
			cr.SetOffsetExpression(fb.exprToString(s.OffsetExpr))
		}
		if s.LimitExpr != nil {
			cr.SetLimitExpression(fb.exprToString(s.LimitExpr))
		}
		op.SetCustomRange(cr)
		return op
	}
	return nil
}

// listAttributeFindGen detects the `attr = expr` shortcut and returns
// a *genMf.Find pre-populated with the attribute / expression. Returns
// nil when the condition isn't a bare binary `=` expression with a
// recognisable left-hand attribute reference — caller falls back to
// FindByExpression.
func (fb *flowBuilderGen) listAttributeFindGen(s *ast.ListOperationStmt) *genMf.Find {
	binary, ok := s.Condition.(*ast.BinaryExpr)
	if !ok || binary.Operator != "=" {
		return nil
	}
	fieldName, ok := listOperationFieldName(binary.Left)
	if !ok {
		return nil
	}
	op := genMf.NewFind()
	assignFreshID(op)
	op.SetListVariableName(s.InputVariable)
	attr, assoc := fb.resolveListOperationMember(s.InputVariable, fieldName)
	if attr != "" {
		op.SetAttributeQualifiedName(attr)
	}
	if assoc != "" {
		op.SetAssociationQualifiedName(assoc)
	}
	op.SetExpression(fb.exprToString(binary.Right))
	return op
}

// listAttributeFilterGen mirrors listAttributeFindGen for the Filter
// operation, returning a *genMf.Filter.
func (fb *flowBuilderGen) listAttributeFilterGen(s *ast.ListOperationStmt) *genMf.Filter {
	binary, ok := s.Condition.(*ast.BinaryExpr)
	if !ok || binary.Operator != "=" {
		return nil
	}
	fieldName, ok := listOperationFieldName(binary.Left)
	if !ok {
		return nil
	}
	op := genMf.NewFilter()
	assignFreshID(op)
	op.SetListVariableName(s.InputVariable)
	attr, assoc := fb.resolveListOperationMember(s.InputVariable, fieldName)
	if attr != "" {
		op.SetAttributeQualifiedName(attr)
	}
	if assoc != "" {
		op.SetAssociationQualifiedName(assoc)
	}
	op.SetExpression(fb.exprToString(binary.Right))
	return op
}

// resolveListOperationMember resolves a list-operation member name
// (attribute vs association) using the same algorithm as
// resolveMemberChangeGen, so list ops follow the same qualification
// rules as change-object members. Stage 3.2.6.4: standalone — the
// legacy `flowBuilder.resolveListOperationMember` adapter is gone.
func (fb *flowBuilderGen) resolveListOperationMember(listVariable, memberName string) (attribute, association string) {
	entityQN := ""
	if fb.varTypes != nil {
		if listType := fb.varTypes[listVariable]; strings.HasPrefix(listType, "List of ") {
			entityQN = strings.TrimPrefix(listType, "List of ")
		}
	}
	resolved := resolveMemberChangeGenStandalone(fb.moduleLister, fb.domainModelReader, memberName, entityQN, nil)
	return resolved.attributeQN, resolved.associationQN
}

// buildSortItemListGen builds a *genMf.SortItemList with one
// *genMf.SortItem per s.SortSpecs entry. Bare attribute names are
// qualified with the input list's element-entity QN; pre-qualified
// dotted names are passed through unchanged.
func (fb *flowBuilderGen) buildSortItemListGen(s *ast.ListOperationStmt) *genMf.SortItemList {
	entityType := ""
	if fb.varTypes != nil {
		listType := fb.varTypes[s.InputVariable]
		if after, ok := strings.CutPrefix(listType, "List of "); ok {
			entityType = after
		}
	}

	itemList := genMf.NewSortItemList()
	assignFreshID(itemList)
	for _, spec := range s.SortSpecs {
		attrPath := spec.Attribute
		if entityType != "" && !strings.Contains(spec.Attribute, ".") {
			attrPath = entityType + "." + spec.Attribute
		}
		order := genMf.SortOrderEnumAscending
		if !spec.Ascending {
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

// trackListOperationOutputType updates fb.varTypes with the result
// type of an emitted operation:
//
//   - Filter / Sort / Tail / Union / Intersect / Subtract / Range
//     preserve the input list type.
//   - Head / Find unwrap the list element type.
//   - Contains / Equals return Boolean — left untracked.
//
// No-op when varTypes is nil or no input/output variable is supplied.
func (fb *flowBuilderGen) trackListOperationOutputType(s *ast.ListOperationStmt) {
	if fb.varTypes == nil || s.OutputVariable == "" || s.InputVariable == "" {
		return
	}
	inputType := fb.varTypes[s.InputVariable]
	switch s.Operation {
	case ast.ListOpFilter, ast.ListOpSort, ast.ListOpTail,
		ast.ListOpUnion, ast.ListOpIntersect, ast.ListOpSubtract, ast.ListOpRange:
		if inputType != "" {
			fb.varTypes[s.OutputVariable] = inputType
		}
	case ast.ListOpHead, ast.ListOpFind:
		if after, ok := strings.CutPrefix(inputType, "List of "); ok {
			fb.varTypes[s.OutputVariable] = after
		}
	}
}
