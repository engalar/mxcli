// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.4 — Visualizations migration (gen-typed shared helpers).
//
// This file holds the gen-typed counterparts of the visualization
// label / details / case-label helpers that legacy `cmd_mermaid.go`
// exposes (mermaidActivityLabel, mermaidActionLabel,
// mermaidActivityDetails, mermaidActionDetails, mermaidCaseLabel,
// classifyMicroflowNode, classifyAction). They are shared by the
// gen-typed Mermaid renderer (Stage 3.2.4 commit 4) and the gen-typed
// ELK builders (Stage 3.2.4 commits 1+3).
//
// The labels are intentionally textually identical to legacy output
// so that fixture comparisons across the two implementations diff to
// nothing. Where gen exposes a richer / poorer accessor than legacy
// (e.g. `EntityQualifiedName()` vs `(EntityID + EntityQualifiedName)`)
// we mirror legacy's *display* by reading the qualified name first
// and dropping silently when nothing is available, since the viz
// layer is rendering for humans, not round-tripping for the parser.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/model"
	genDC "github.com/mendixlabs/mxcli/modelsdk/gen/databaseconnector"
	genDT "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	genTx "github.com/mendixlabs/mxcli/modelsdk/gen/texts"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// ────────────────────────────────────────────────────────
// Activity label — mirrors legacy mermaidActivityLabel
// ────────────────────────────────────────────────────────

// mermaidActivityLabelGen returns a short label for a gen microflow
// activity node, mirroring legacy `mermaidActivityLabel`.
func mermaidActivityLabelGen(obj element.Element, entityNames map[model.ID]string) string {
	switch a := obj.(type) {
	case *genMf.ActionActivity:
		return mermaidActionLabelGen(a, entityNames)
	case *genMf.ExclusiveSplit:
		if cond, ok := a.SplitCondition().(*genMf.ExpressionSplitCondition); ok && cond != nil {
			expr := cond.Expression()
			if len(expr) > 40 {
				expr = expr[:37] + "..."
			}
			if expr != "" {
				return sanitizeMermaidLabel(expr)
			}
		}
		return "Split"
	case *genMf.InheritanceSplit:
		return "Inheritance Split"
	case *genMf.LoopedActivity:
		return "Loop"
	case *genMf.StartEvent:
		return "Start"
	case *genMf.EndEvent:
		if rv := a.ReturnValue(); rv != "" {
			return "Return: " + sanitizeMermaidLabel(mermaidTruncate(rv, 30))
		}
		return "End"
	default:
		return "Activity"
	}
}

// mermaidActionLabelGen mirrors legacy mermaidActionLabel for the
// inner Action of a gen ActionActivity.
func mermaidActionLabelGen(a *genMf.ActionActivity, entityNames map[model.ID]string) string {
	if a.Action() == nil {
		return "Action"
	}

	switch act := a.Action().(type) {
	case *genMf.CreateObjectAction:
		entityName := mermaidResolveEntityNameGen(act.EntityQualifiedName(), entityNames)
		return "Create " + sanitizeMermaidLabel(entityName)
	case *genMf.ChangeObjectAction:
		if v := act.ChangeVariableName(); v != "" {
			return "change $" + sanitizeMermaidLabel(v)
		}
		return "Change Object"
	case *genMf.CommitAction:
		if v := act.CommitVariableName(); v != "" {
			return "commit $" + sanitizeMermaidLabel(v)
		}
		return "Commit"
	case *genMf.DeleteAction:
		if v := act.DeleteVariableName(); v != "" {
			return "delete $" + sanitizeMermaidLabel(v)
		}
		return "Delete"
	case *genMf.RollbackAction:
		if v := act.RollbackVariableName(); v != "" {
			return "rollback $" + sanitizeMermaidLabel(v)
		}
		return "Rollback"
	case *genMf.CreateVariableAction:
		if v := act.VariableName(); v != "" {
			return "declare $" + sanitizeMermaidLabel(v)
		}
		return "Declare Variable"
	case *genMf.ChangeVariableAction:
		if v := act.ChangeVariableName(); v != "" {
			return "set $" + sanitizeMermaidLabel(v)
		}
		return "Set Variable"
	case *genMf.RetrieveAction:
		if act.RetrieveSource() != nil {
			switch src := act.RetrieveSource().(type) {
			case *genMf.DatabaseRetrieveSource:
				entityName := mermaidResolveEntityNameGen(src.EntityQualifiedName(), entityNames)
				return "Retrieve " + sanitizeMermaidLabel(entityName)
			case *genMf.AssociationRetrieveSource:
				_ = src
				return "Retrieve by Association"
			}
		}
		return "Retrieve"
	case *genMf.MicroflowCallAction:
		if mc, ok := act.MicroflowCall().(*genMf.MicroflowCall); ok && mc != nil {
			if qn := mc.MicroflowQualifiedName(); qn != "" {
				return "Call " + sanitizeMermaidLabel(mermaidTruncate(qn, 30))
			}
		}
		return "Call Microflow"
	case *genMf.NanoflowCallAction:
		if nc, ok := act.NanoflowCall().(*genMf.NanoflowCall); ok && nc != nil {
			if qn := nc.NanoflowQualifiedName(); qn != "" {
				return "Call " + sanitizeMermaidLabel(mermaidTruncate(qn, 30))
			}
		}
		return "Call Nanoflow"
	case *genMf.JavaActionCallAction:
		if qn := act.JavaActionQualifiedName(); qn != "" {
			return "Call " + sanitizeMermaidLabel(mermaidTruncate(qn, 30))
		}
		return "Call Java Action"
	case *genMf.RestCallAction:
		_ = act
		return "rest Call"
	case *genDC.ExecuteDatabaseQueryAction:
		if q := act.DynamicQuery(); q != "" {
			return "DB Query " + sanitizeMermaidLabel(mermaidTruncate(q, 30))
		}
		if q := act.QueryQualifiedName(); q != "" {
			return "DB Query " + sanitizeMermaidLabel(mermaidTruncate(q, 30))
		}
		return "Execute DB Query"
	case *genMf.CallExternalAction:
		if n := act.Name(); n != "" {
			return "Call External " + sanitizeMermaidLabel(mermaidTruncate(n, 25))
		}
		return "Call External"
	case *genMf.ShowPageAction:
		if pageName := showPageActionPageNameGen(act); pageName != "" {
			return "Show " + sanitizeMermaidLabel(mermaidTruncate(pageName, 30))
		}
		return "Show Page"
	case *genMf.CloseFormAction:
		_ = act
		return "Close Page"
	case *genMf.ShowMessageAction:
		_ = act
		return "Show Message"
	case *genMf.ValidationFeedbackAction:
		_ = act
		return "Validation Feedback"
	case *genMf.LogMessageAction:
		_ = act
		return "Log Message"
	case *genMf.AggregateListAction:
		if v := act.OutputVariableName(); v != "" {
			return "Aggregate $" + sanitizeMermaidLabel(v)
		}
		return "Aggregate List"
	case *genMf.ListOperationAction:
		_ = act
		return "List Operation"
	case *genMf.CastAction:
		_ = act
		return "Cast Object"
	default:
		return "Action"
	}
}

// ────────────────────────────────────────────────────────
// Activity details — mirrors legacy mermaidActivityDetails
// ────────────────────────────────────────────────────────

func mermaidActivityDetailsGen(obj element.Element, entityNames map[model.ID]string) []string {
	switch a := obj.(type) {
	case *genMf.ActionActivity:
		return mermaidActionDetailsGen(a, entityNames)
	case *genMf.ExclusiveSplit:
		var lines []string
		if cap := a.Caption(); cap != "" {
			lines = append(lines, "Caption: "+cap)
		}
		if cond, ok := a.SplitCondition().(*genMf.ExpressionSplitCondition); ok && cond != nil && cond.Expression() != "" {
			lines = append(lines, "Condition: "+cond.Expression())
		}
		return lines
	case *genMf.LoopedActivity:
		var lines []string
		switch ls := a.LoopSource().(type) {
		case *genMf.IterableList:
			if v := ls.ListVariableName(); v != "" {
				lines = append(lines, "List: $"+v)
			}
			if v := ls.VariableName(); v != "" {
				lines = append(lines, "Iterator: $"+v)
			}
		case *genMf.WhileLoopCondition:
			if e := ls.WhileExpression(); e != "" {
				lines = append(lines, "While: "+e)
			}
		}
		return lines
	case *genMf.EndEvent:
		if rv := a.ReturnValue(); rv != "" {
			return []string{"Return: " + rv}
		}
	}
	return nil
}

// mermaidActionDetailsGen mirrors legacy mermaidActionDetails.
func mermaidActionDetailsGen(a *genMf.ActionActivity, entityNames map[model.ID]string) []string {
	if a.Action() == nil {
		return nil
	}

	var lines []string

	switch act := a.Action().(type) {
	case *genMf.CreateObjectAction:
		entityName := mermaidResolveEntityNameGen(act.EntityQualifiedName(), entityNames)
		lines = append(lines, "Entity: "+entityName)
		if v := act.OutputVariableName(); v != "" {
			lines = append(lines, "Output: $"+v)
		}
		if c := act.Commit(); c != "" && c != "No" {
			lines = append(lines, "Commit: "+c)
		}
		for _, item := range act.ItemsItems() {
			mc, ok := item.(*genMf.MemberChange)
			if !ok || mc == nil {
				continue
			}
			name := mermaidMemberNameGen(mc)
			if name != "" && mc.Value() != "" {
				lines = append(lines, name+" = "+mermaidTruncate(mc.Value(), 50))
			}
		}

	case *genMf.ChangeObjectAction:
		if v := act.ChangeVariableName(); v != "" {
			lines = append(lines, "Variable: $"+v)
		}
		if c := act.Commit(); c != "" && c != "No" {
			lines = append(lines, "Commit: "+c)
		}
		for _, item := range act.ItemsItems() {
			mc, ok := item.(*genMf.MemberChange)
			if !ok || mc == nil {
				continue
			}
			name := mermaidMemberNameGen(mc)
			if name != "" && mc.Value() != "" {
				lines = append(lines, name+" = "+mermaidTruncate(mc.Value(), 50))
			}
		}

	case *genMf.CommitAction:
		if v := act.CommitVariableName(); v != "" {
			lines = append(lines, "Variable: $"+v)
		}
		if act.WithEvents() {
			lines = append(lines, "With events: true")
		}

	case *genMf.DeleteAction:
		if v := act.DeleteVariableName(); v != "" {
			lines = append(lines, "Variable: $"+v)
		}

	case *genMf.RollbackAction:
		if v := act.RollbackVariableName(); v != "" {
			lines = append(lines, "Variable: $"+v)
		}

	case *genMf.RetrieveAction:
		if v := act.OutputVariableName(); v != "" {
			lines = append(lines, "Output: $"+v)
		}
		if act.RetrieveSource() != nil {
			switch src := act.RetrieveSource().(type) {
			case *genMf.DatabaseRetrieveSource:
				entityName := mermaidResolveEntityNameGen(src.EntityQualifiedName(), entityNames)
				lines = append(lines, "From: "+entityName)
				if x := src.XPathConstraint(); x != "" {
					lines = append(lines, "Where: "+mermaidTruncate(x, 60))
				}
				if r := src.Range(); r != nil {
					lines = append(lines, formatRetrieveRangeGen(r)...)
				}
				if sl, ok := src.SortItemList().(*genMf.SortItemList); ok && sl != nil {
					for _, s := range sl.ItemsItems() {
						if so, ok := s.(*genMf.SortItem); ok && so != nil {
							attr := so.AttributePath()
							if attr != "" {
								lines = append(lines, "Sort: "+attr+" "+so.SortOrder())
							}
						}
					}
				}
			case *genMf.AssociationRetrieveSource:
				if v := src.StartVariableName(); v != "" {
					lines = append(lines, "From: $"+v)
				}
				if v := src.AssociationQualifiedName(); v != "" {
					lines = append(lines, "Via: "+v)
				}
			}
		}

	case *genMf.MicroflowCallAction:
		if mc, ok := act.MicroflowCall().(*genMf.MicroflowCall); ok && mc != nil {
			if qn := mc.MicroflowQualifiedName(); qn != "" {
				lines = append(lines, "Microflow: "+qn)
			}
			for _, pm := range mc.ParameterMappingsItems() {
				if pmm, ok := pm.(*genMf.MicroflowCallParameterMapping); ok && pmm != nil {
					param := lastDotSegment(pmm.ParameterQualifiedName())
					arg := pmm.Argument()
					lines = append(lines, param+" = "+mermaidTruncate(arg, 50))
				}
			}
		}
		if v := act.OutputVariableName(); v != "" {
			lines = append(lines, "Result: $"+v)
		}

	case *genMf.NanoflowCallAction:
		if nc, ok := act.NanoflowCall().(*genMf.NanoflowCall); ok && nc != nil {
			if qn := nc.NanoflowQualifiedName(); qn != "" {
				lines = append(lines, "Nanoflow: "+qn)
			}
			for _, pm := range nc.ParameterMappingsItems() {
				if pmm, ok := pm.(*genMf.NanoflowCallParameterMapping); ok && pmm != nil {
					param := lastDotSegment(pmm.ParameterQualifiedName())
					arg := pmm.Argument()
					lines = append(lines, param+" = "+mermaidTruncate(arg, 50))
				}
			}
		}
		if v := act.OutputVariableName(); v != "" {
			lines = append(lines, "Result: $"+v)
		}

	case *genMf.ShowPageAction:
		if pageName := showPageActionPageNameGen(act); pageName != "" {
			lines = append(lines, "Page: "+pageName)
		}
		if ps, ok := act.PageSettings().(*genPg.PageSettings); ok && ps != nil {
			if loc := ps.Location(); loc != "" {
				lines = append(lines, "Location: "+loc)
			}
		}
		// Parameter mappings live on PageSettings via ParameterMappings
		// list; legacy emitted them via PageParameterMappings (similar
		// shape). Skip when not present.

	case *genMf.ShowMessageAction:
		if t := act.Type(); t != "" {
			lines = append(lines, "Type: "+t)
		}
		if msg := mermaidTextPreviewGen(act.Template()); msg != "" {
			lines = append(lines, "Message: "+mermaidTruncate(msg, 60))
		}

	case *genMf.ValidationFeedbackAction:
		if v := act.ObjectVariableName(); v != "" {
			target := "$" + v
			memberQN := act.AttributeQualifiedName()
			if memberQN == "" {
				memberQN = act.AssociationQualifiedName()
			}
			if memberQN != "" {
				target += "." + lastDotSegment(memberQN)
			}
			lines = append(lines, "Target: "+target)
		}
		if msg := mermaidTextPreviewGen(act.FeedbackTemplate()); msg != "" {
			lines = append(lines, "Message: "+mermaidTruncate(msg, 60))
		}

	case *genMf.LogMessageAction:
		if l := act.Level(); l != "" {
			lines = append(lines, "Level: "+l)
		}
		if n := act.Node(); n != "" {
			lines = append(lines, "Node: "+n)
		}
		if msg := mermaidTextPreviewGen(act.MessageTemplate()); msg != "" {
			lines = append(lines, "Message: "+mermaidTruncate(msg, 60))
		}

	case *genMf.AggregateListAction:
		if v := act.InputListVariableName(); v != "" {
			lines = append(lines, "List: $"+v)
		}
		if fn := act.AggregateFunction(); fn != "" {
			full := fn
			if attr := act.AttributeQualifiedName(); attr != "" {
				full += " on " + attr
			}
			lines = append(lines, "Function: "+full)
		}
		if v := act.OutputVariableName(); v != "" {
			lines = append(lines, "Output: $"+v)
		}

	case *genMf.CreateVariableAction:
		if v := act.VariableName(); v != "" {
			lines = append(lines, "Variable: $"+v)
		}
		if v := act.InitialValue(); v != "" {
			lines = append(lines, "Value: "+mermaidTruncate(v, 60))
		}

	case *genMf.ChangeVariableAction:
		if v := act.ChangeVariableName(); v != "" {
			lines = append(lines, "Variable: $"+v)
		}
		if v := act.Value(); v != "" {
			lines = append(lines, "Value: "+mermaidTruncate(v, 60))
		}

	case *genMf.JavaActionCallAction:
		if qn := act.JavaActionQualifiedName(); qn != "" {
			lines = append(lines, "Java Action: "+qn)
		}
		if v := act.OutputVariableName(); v != "" {
			lines = append(lines, "Result: $"+v)
		}

	case *genMf.RestCallAction:
		if hc := act.HttpConfiguration(); hc != nil {
			if hcRich, ok := hc.(*genMf.HttpConfiguration); ok && hcRich != nil {
				method := hcRich.HttpMethod()
				if method == "" {
					method = hcRich.NewHttpMethod()
				}
				url := hcRich.CustomLocation()
				if method != "" {
					if url != "" {
						lines = append(lines, method+" "+mermaidTruncate(url, 50))
					} else {
						lines = append(lines, "Method: "+method)
					}
				}
			}
		}
		// Output variable is nested under the response handling; we
		// surface it best-effort (skip if not a simple binary/file).

	case *genDC.ExecuteDatabaseQueryAction:
		if q := act.QueryQualifiedName(); q != "" {
			lines = append(lines, "Query: "+q)
		}
		if q := act.DynamicQuery(); q != "" {
			lines = append(lines, "Dynamic: "+mermaidTruncate(q, 50))
		}
		if v := act.OutputVariableName(); v != "" {
			lines = append(lines, "Output: $"+v)
		}

	case *genMf.CallExternalAction:
		if s := act.ConsumedODataServiceQualifiedName(); s != "" {
			lines = append(lines, "Service: "+s)
		}
		if n := act.Name(); n != "" {
			lines = append(lines, "Action: "+n)
		}
		if v := act.VariableName(); v != "" {
			lines = append(lines, "Result: $"+v)
		}

	case *genMf.CloseFormAction:
		if n := act.NumberOfPagesToClose(); n != "" {
			lines = append(lines, "Pages: "+n)
		}

	case *genMf.ListOperationAction:
		if v := act.OutputVariableName(); v != "" {
			lines = append(lines, "Output: $"+v)
		}
	}

	return lines
}

// ────────────────────────────────────────────────────────
// Case-value label — mirrors legacy mermaidCaseLabel
// ────────────────────────────────────────────────────────

// mermaidCaseLabelGen extracts a display label from the CaseValue of a
// gen SequenceFlow. Gen does not model BooleanCase / ExpressionCase as
// distinct types — booleans are EnumerationCase with literal "true" /
// "false", and expression-based cases were rewritten upstream — so the
// switch is narrower than legacy. This keeps fixture diffs aligned for
// the common cases (boolean splits and enumeration splits).
func mermaidCaseLabelGen(cv element.Element) string {
	if cv == nil {
		return ""
	}
	switch c := cv.(type) {
	case *genMf.NoCase:
		_ = c
		return ""
	case *genMf.EnumerationCase:
		return sanitizeMermaidLabel(c.Value())
	}
	return ""
}

// ────────────────────────────────────────────────────────
// Node classification — mirrors legacy classifyMicroflowNode
// ────────────────────────────────────────────────────────

// classifyMicroflowNodeGen returns (nodeType, category) for a gen
// activity, mirroring legacy `classifyMicroflowNode`.
func classifyMicroflowNodeGen(obj element.Element) (nodeType, category string) {
	switch a := obj.(type) {
	case *genMf.StartEvent:
		_ = a
		return "start", "event"
	case *genMf.EndEvent:
		_ = a
		return "end", "event"
	case *genMf.ContinueEvent:
		_ = a
		return "continue", "event"
	case *genMf.BreakEvent:
		_ = a
		return "break", "event"
	case *genMf.ErrorEvent:
		_ = a
		return "error", "event"
	case *genMf.ExclusiveSplit:
		_ = a
		return "split", "controlflow"
	case *genMf.InheritanceSplit:
		_ = a
		return "split", "controlflow"
	case *genMf.ExclusiveMerge:
		_ = a
		return "merge", "controlflow"
	case *genMf.LoopedActivity:
		_ = a
		return "loop", "loop"
	case *genMf.ActionActivity:
		return "action", classifyActionGen(a)
	default:
		return "action", "variable"
	}
}

// classifyActionGen returns the category string for coloring an
// ActionActivity node, mirroring legacy `classifyAction`.
func classifyActionGen(a *genMf.ActionActivity) string {
	if a.Action() == nil {
		return "variable"
	}

	switch a.Action().(type) {
	case *genMf.CreateObjectAction, *genMf.ChangeObjectAction,
		*genMf.CommitAction, *genMf.DeleteAction,
		*genMf.RollbackAction:
		return "object"
	case *genMf.RetrieveAction:
		return "retrieve"
	case *genMf.MicroflowCallAction, *genMf.JavaActionCallAction,
		*genMf.CallExternalAction, *genMf.RestCallAction:
		return "call"
	case *genMf.ShowPageAction, *genMf.CloseFormAction,
		*genMf.ShowMessageAction:
		return "navigation"
	case *genMf.CreateVariableAction, *genMf.ChangeVariableAction,
		*genMf.AggregateListAction, *genMf.ListOperationAction,
		*genMf.CastAction:
		return "variable"
	case *genMf.ValidationFeedbackAction:
		return "validation"
	case *genMf.LogMessageAction:
		return "log"
	default:
		return "variable"
	}
}

// ────────────────────────────────────────────────────────
// DataType display — mirrors legacy formatDataTypeDisplay
// ────────────────────────────────────────────────────────

// formatDataTypeDisplayGen mirrors legacy formatDataTypeDisplay for
// gen DataType elements. Unknown concrete types fall back to their
// gen TypeName so the output stays informative.
func formatDataTypeDisplayGen(dt element.Element) string {
	if dt == nil {
		return ""
	}
	switch t := dt.(type) {
	case *genDT.BooleanType:
		_ = t
		return "Boolean"
	case *genDT.IntegerType:
		_ = t
		return "Integer"
	case *genDT.FloatType:
		_ = t
		// gen package merged Long into Float at the metamodel level
		// (legacy LongType has no direct equivalent in gen datatypes);
		// surface as "Float" to match what shipped in 11.x reflection.
		return "Float"
	case *genDT.DecimalType:
		_ = t
		return "Decimal"
	case *genDT.StringType:
		_ = t
		return "String"
	case *genDT.DateTimeType:
		_ = t
		return "DateTime"
	case *genDT.ObjectType:
		return shortName(t.EntityQualifiedName())
	case *genDT.ListType:
		return "List<" + shortName(t.EntityQualifiedName()) + ">"
	case *genDT.EnumerationType:
		return shortName(t.EnumerationQualifiedName())
	case *genDT.VoidType:
		_ = t
		return "Void"
	case *genDT.BinaryType:
		_ = t
		return "Binary"
	case *genDT.EmptyType:
		_ = t
		return ""
	default:
		return dt.TypeName()
	}
}

// ────────────────────────────────────────────────────────
// Small private helpers
// ────────────────────────────────────────────────────────

// mermaidResolveEntityNameGen resolves an entity name from the
// qualified name (preferred) or falls back to "Entity" when blank.
// Gen actions don't expose the bare EntityID source the legacy lookup
// table indexed, so this is a simpler one-shot resolution.
func mermaidResolveEntityNameGen(qualifiedName string, _ map[model.ID]string) string {
	if qualifiedName != "" {
		return qualifiedName
	}
	return "Entity"
}

// mermaidMemberNameGen extracts a short member name from a gen
// MemberChange — last dot segment of the attribute or association
// qualified name. Mirrors legacy `mermaidMemberName`.
func mermaidMemberNameGen(mc *genMf.MemberChange) string {
	name := mc.AttributeQualifiedName()
	if name == "" {
		name = mc.AssociationQualifiedName()
	}
	if name == "" {
		return ""
	}
	return lastDotSegment(name)
}

// mermaidTextPreviewGen extracts the first non-empty translation from
// a gen Text element. Tries en_US first, then any language.
func mermaidTextPreviewGen(t element.Element) string {
	text, ok := t.(*genTx.Text)
	if !ok || text == nil {
		return ""
	}
	for _, item := range text.TranslationsItems() {
		tr, ok := item.(*genTx.Translation)
		if !ok || tr == nil {
			continue
		}
		if tr.LanguageCode() == "en_US" {
			if v := strings.TrimSpace(tr.Text()); v != "" {
				return v
			}
		}
	}
	for _, item := range text.TranslationsItems() {
		tr, ok := item.(*genTx.Translation)
		if !ok || tr == nil {
			continue
		}
		if v := strings.TrimSpace(tr.Text()); v != "" {
			return v
		}
	}
	return ""
}

// formatRetrieveRangeGen renders a gen RetrieveRange element as the
// legacy "Range: <type>" line. Gen `ConstantRange` exposes only
// `SingleObject()`; the rich Limit/Offset surface that legacy used to
// emit lives on the SDK type and is not present in the gen surface,
// so we degrade gracefully to a single descriptor.
func formatRetrieveRangeGen(r element.Element) []string {
	rng, ok := r.(*genMf.ConstantRange)
	if !ok || rng == nil {
		return nil
	}
	if rng.SingleObject() {
		return []string{"Range: first"}
	}
	return []string{"Range: all"}
}

// showPageActionPageNameGen extracts the resolved page qualified name
// from a gen ShowPageAction's PageSettings (the only carrier in the
// gen surface). Returns "" when nothing resolves.
func showPageActionPageNameGen(a *genMf.ShowPageAction) string {
	if ps, ok := a.PageSettings().(*genPg.PageSettings); ok && ps != nil {
		if qn := strings.TrimSpace(ps.PageQualifiedName()); qn != "" {
			return qn
		}
	}
	return ""
}

// fmtBlankPlaceholder is unused — declaring `_ = fmt.Sprintf` keeps
// the `fmt` import live for future extensions of the helpers above.
var _ = fmt.Sprintf
