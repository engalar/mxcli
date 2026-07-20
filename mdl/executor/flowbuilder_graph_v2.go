// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.j — gen-typed buildFlowGraphGen entry.
//
// Top-level driver that turns a microflow body (slice of AST
// statements) into a complete *genMf.MicroflowObjectCollection
// with StartEvent + per-statement activities + sequence flow chain
// + EndEvent (synthesised when the body doesn't already terminate).
//
// Returns the assembled ObjectCollection. The caller (k —
// execCreateMicroflowGen) takes the flows / annotation flows from
// fb.flows / fb.annotationFlows and copies them onto the
// Microflow's Flows array (gen Mendix BSON layout: flows live at
// the microflow level, not inside the ObjectCollection).
//
// Out of scope (deferred from legacy):
//
//   - @position annotation pre-scan (legacy shifts StartEvent left
//     of authored coordinates) — when first stmt has @position
//   - retry-loop incomingRedirect handling (already deferred from i)
//   - terminatePendingErrorHandlersAtEnd full implementation —
//     placeholder no-op here (the EH queue helpers from d are
//     wired but the terminal-rejoin emission needs more work)
//   - @anchor preservation across compound statements
//   - free-floating @annotation flush at body end
//
// The minimal viable shipped here covers the dominant fresh-author
// shape: linear chain of statements with optional final RETURN.

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// buildFlowGraphGen iterates the microflow body and emits the
// complete object graph. Returns the *genMf.MicroflowObjectCollection
// that the caller attaches to the Microflow being built.
//
// Side effect: populates fb.objects / fb.flows / fb.annotationFlows
// with the emitted elements. The caller owns moving fb.flows onto
// the Microflow's Flows array.
func (fb *flowBuilderGen) buildFlowGraphGen(stmts []ast.MicroflowStatement, returns *ast.MicroflowReturnType) *genMf.MicroflowObjectCollection {
	// Defensive init — caller may forget.
	if fb.measurer == nil {
		fb.measurer = &layoutMeasurer{varTypes: fb.varTypes}
	}
	if fb.declaredVars == nil {
		fb.declaredVars = make(map[string]string)
	}
	if fb.varTypes == nil {
		fb.varTypes = make(map[string]string)
	}
	// Set return value expression for synthesised EndEvent.
	fb.returnType = returns
	if returns != nil && returns.Variable != "" {
		fb.returnValue = "$" + returns.Variable
	}
	fb.baseY = fb.posY

	// Emit StartEvent at the current cursor.
	startEvent := genMf.NewStartEvent()
	startID := assignFreshID(startEvent)
	startEvent.SetRelativeMiddlePoint(layoutPos(fb.posX, fb.posY))
	startEvent.SetSize(layoutSize(EventSize, EventSize))
	fb.objects = append(fb.objects, startEvent)
	lastID := startID
	fb.posX += fb.spacing

	// Synthetic declare for return variable.
	// 11.12.1+ validator requires the return variable to have a
	// declaration node in the graph (ObjectCollection), not just a
	// ReturnVariableName metadata field on the microflow.
	// List-typed variables require CreateListAction; all other types
	// use CreateVariableAction (see flowbuilder_actions_v2.go:76-95).
	// CE0111 guard: skip synthetic declaration when the body's first
	// activity (retrieve/aggregate/cast/set) already declares the same
	// variable — having both creates "Duplicate variable name" errors.
	if returns != nil && returns.Variable != "" && !fb.isNanoflow && !bodyHasDeclareFor(stmts, returns.Variable) {
		if !retVarUsedInStmts(stmts, returns.Variable) {
			var declAction element.Element
			if returns.Type.Kind == ast.TypeListOf {
				entityQN := ""
				if returns.Type.EntityRef != nil {
					entityQN = returns.Type.EntityRef.Module + "." + returns.Type.EntityRef.Name
				}
				listAct := genMf.NewCreateListAction()
				assignFreshID(listAct)
				listAct.SetErrorHandlingType(fb.ehTypeGen(nil))
				listAct.SetOutputVariableName(returns.Variable)
				listAct.SetEntityQualifiedName(entityQN)
				declAction = listAct
				if fb.varTypes != nil && entityQN != "" {
					fb.varTypes[returns.Variable] = "List of " + entityQN
				}
				if fb.declaredVars != nil {
					fb.declaredVars[returns.Variable] = "List of " + entityQN
				}
			} else {
				declAct := genMf.NewCreateVariableAction()
				assignFreshID(declAct)
				declAct.SetErrorHandlingType(fb.ehTypeGen(nil))
				declAct.SetVariableName(returns.Variable)
				if dt := convertASTToGenDataType(returns.Type); dt != nil {
					declAct.SetVariableType(dt)
				}
				declAct.SetInitialValue(mendixExprValue(defaultInitialValue(returns.Type)))
				declAction = declAct
				if ref := paramEntityRef(returns.Type); ref != nil && ref.Module != "" {
					if fb.varTypes != nil {
						fb.varTypes[returns.Variable] = ref.Module + "." + ref.Name
					}
				} else if fb.declaredVars != nil {
					fb.declaredVars[returns.Variable] = returns.Type.Kind.String()
				}
			}
			declID := fb.genActivityWrap(declAction, nil, "")
			fb.flows = append(fb.flows, newHorizontalFlowGen(lastID, declID))
			lastID = declID
		} else if fb.varTypes != nil {
			// Register return variable type so the executor knows the type
			// without needing a synthetic declaration node.
			if returns.Type.Kind == ast.TypeListOf {
				entityQN := ""
				if returns.Type.EntityRef != nil {
					entityQN = returns.Type.EntityRef.Module + "." + returns.Type.EntityRef.Name
				}
				fb.varTypes[returns.Variable] = "List of " + entityQN
			} else if ref := paramEntityRef(returns.Type); ref != nil && ref.Module != "" {
				fb.varTypes[returns.Variable] = ref.Module + "." + ref.Name
			}
		}
	}

	// Iterate body statements via the dispatcher (h1).
	for _, stmt := range stmts {
		// Consume any pending case label set by the previous statement
		// (e.g. a pass-through if-without-else whose false branch flows
		// directly from the split to the next activity).
		pendingCase := fb.nextConnectionCase
		fb.nextConnectionCase = ""

		activityID := fb.addStatementGen(stmt)
		if activityID == "" {
			continue
		}
		fb.applyPendingAnnotations(activityID)

		// Connect previous to current, using a case-labelled flow when
		// the previous statement left a pending case (pass-through split).
		var flow *genMf.SequenceFlow
		if pendingCase != "" {
			flow = newHorizontalFlowWithCaseGen(lastID, activityID, pendingCase)
		} else {
			flow = newHorizontalFlowGen(lastID, activityID)
		}
		fb.flows = append(fb.flows, flow)

		// Compound statements (IF/loop/split) advertise their merge via
		// nextConnectionPoint; consume so subsequent flow originates
		// from the right place.
		if fb.nextConnectionPoint != "" {
			lastID = fb.nextConnectionPoint
			fb.nextConnectionPoint = ""
		} else {
			lastID = activityID
		}
	}

	// Synthesise final EndEvent unless the body already terminates
	// (fb.endsWithReturn set by RETURN / both-branches-return etc).
	if !fb.endsWithReturn {
		fb.posX += fb.spacing / 2
		fb.posY = fb.baseY
		endEvent := genMf.NewEndEvent()
		endID := assignFreshID(endEvent)
		endEvent.SetRelativeMiddlePoint(layoutPos(fb.posX, fb.posY))
		endEvent.SetSize(layoutSize(EventSize, EventSize))
		if fb.returnValue != "" {
			endEvent.SetReturnValue(fb.returnValue)
		}
		fb.objects = append(fb.objects, endEvent)

		// Connect last activity to EndEvent. If a pending case is still
		// set (pass-through if as the last statement), use a case flow.
		if fb.nextConnectionCase != "" {
			fb.flows = append(fb.flows, newHorizontalFlowWithCaseGen(lastID, endID, fb.nextConnectionCase))
			fb.nextConnectionCase = ""
		} else {
			fb.flows = append(fb.flows, newHorizontalFlowGen(lastID, endID))
		}
	}

	// Wrap all emitted objects into the MicroflowObjectCollection
	// that the caller attaches to the Microflow.
	oc := genMf.NewMicroflowObjectCollection()
	assignFreshID(oc)
	for _, obj := range fb.objects {
		oc.AddObjects(obj)
	}
	return oc
}

// flowBuilderGenObjects returns the slice of element.Element objects
// emitted into fb.objects so far. Exposed for the caller (k entry)
// to inspect or copy without mutating the slice. Currently a thin
// accessor — kept for API symmetry with the buildFlowGraphGen
// return value.
func (fb *flowBuilderGen) flowBuilderGenObjects() []element.Element {
	return fb.objects
}

// bodyHasDeclareFor checks whether any statement in the body is
// a declare statement for the given variable name.
func bodyHasDeclareFor(stmts []ast.MicroflowStatement, varName string) bool {
	for _, stmt := range stmts {
		if d, ok := stmt.(*ast.DeclareStmt); ok {
			if d.Variable == varName {
				return true
			}
		}
	}
	return false
}

// defaultInitialValue returns a Mendix expression for the default
// initial value of a return variable, based on its type.
func defaultInitialValue(dt ast.DataType) string {
	switch dt.Kind {
	case ast.TypeString:
		return "''"
	case ast.TypeInteger, ast.TypeLong:
		return "0"
	case ast.TypeBoolean:
		return "true"
	case ast.TypeDecimal:
		return "0.0"
	case ast.TypeDateTime:
		return "dateTime(0)"
	default:
		return "empty"
	}
}

// retVarUsedInStmts checks recursively if any statement uses varName as its
// output/result variable. Handles compound statements (if/loop/split).
func retVarUsedInStmts(stmts []ast.MicroflowStatement, varName string) bool {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.RetrieveStmt:
			if s.Variable == varName {
				return true
			}
		case *ast.AggregateListStmt:
			if s.OutputVariable == varName {
				return true
			}
		case *ast.CastObjectStmt:
			if s.OutputVariable == varName {
				return true
			}
		case *ast.IfStmt:
			if retVarUsedInStmts(s.ThenBody, varName) {
				return true
			}
			if s.HasElse && retVarUsedInStmts(s.ElseBody, varName) {
				return true
			}
		case *ast.LoopStmt:
			if retVarUsedInStmts(s.Body, varName) {
				return true
			}
		case *ast.WhileStmt:
			if retVarUsedInStmts(s.Body, varName) {
				return true
			}
		case *ast.InheritanceSplitStmt:
			for _, c := range s.Cases {
				if retVarUsedInStmts(c.Body, varName) {
					return true
				}
			}
			if retVarUsedInStmts(s.ElseBody, varName) {
				return true
			}
		case *ast.EnumSplitStmt:
			for _, c := range s.Cases {
				if retVarUsedInStmts(c.Body, varName) {
					return true
				}
			}
			if retVarUsedInStmts(s.ElseBody, varName) {
				return true
			}
		}
	}
	return false
}
