// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.b — gen-typed annotation handling and terminal events.
//
// This file is the gen-typed counterpart of
// `cmd_microflows_builder_annotations.go`. It provides the helpers
// the per-statement adders (next commits) call to:
//
//   - emit `EndEvent` (return) and `ErrorEvent` (raise error) terminal
//     events on the gen object collection;
//   - attach `Annotation` + `AnnotationFlow` pairs above an emitted
//     activity for the @annotation directive;
//   - apply pending @caption / @color / @excluded annotations to the
//     freshly-emitted activity.
//
// The pure-AST helpers `getStatementAnnotations` / `stmtOwnAnchor`
// (legacy file) carry no SDK type references and are reused as-is.
// They live in the legacy file because Stage 3.2.6 will move them out
// once `cmd_microflows_builder_annotations.go` is deleted.

package executor

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// mergeStatementAnnotations extracts annotations from a statement and
// merges them into pendingAnnotations. Identical algorithm to legacy
// flowBuilder.mergeStatementAnnotations.
func (fb *flowBuilderGen) mergeStatementAnnotations(stmt ast.MicroflowStatement) {
	ann := getStatementAnnotations(stmt)
	if ann == nil {
		return
	}
	if fb.pendingAnnotations == nil {
		fb.pendingAnnotations = &ast.ActivityAnnotations{}
	}
	if ann.Position != nil {
		fb.pendingAnnotations.Position = ann.Position
	}
	if ann.Caption != "" {
		fb.pendingAnnotations.Caption = ann.Caption
	}
	if ann.Color != "" {
		fb.pendingAnnotations.Color = ann.Color
	}
	if ann.AnnotationText != "" {
		fb.pendingAnnotations.AnnotationText = ann.AnnotationText
	}
	if len(ann.FreeAnnotations) > 0 {
		fb.pendingAnnotations.FreeAnnotations = append(fb.pendingAnnotations.FreeAnnotations, ann.FreeAnnotations...)
	}
	if ann.Anchor != nil {
		fb.pendingAnnotations.Anchor = ann.Anchor
	}
	if ann.TrueBranchAnchor != nil {
		fb.pendingAnnotations.TrueBranchAnchor = ann.TrueBranchAnchor
	}
	if ann.FalseBranchAnchor != nil {
		fb.pendingAnnotations.FalseBranchAnchor = ann.FalseBranchAnchor
	}
	if ann.IteratorAnchor != nil {
		fb.pendingAnnotations.IteratorAnchor = ann.IteratorAnchor
	}
	if ann.BodyTailAnchor != nil {
		fb.pendingAnnotations.BodyTailAnchor = ann.BodyTailAnchor
	}
}

// applyAnnotations applies the @caption / @color / @excluded /
// @annotation parts of `ann` to the activity identified by activityID.
//
// @position is applied at activity-creation time (in the per-statement
// adders), so this method only handles caption/color/excluded mutation
// and the optional @annotation attachment.
//
// Activity discovery iterates fb.objects looking for the matching ID.
// We dispatch to the four gen activity kinds the legacy path
// recognised (ActionActivity / ExclusiveSplit / InheritanceSplit /
// LoopedActivity) and silently no-op for everything else, matching
// legacy behaviour.
func (fb *flowBuilderGen) applyAnnotations(activityID element.ID, ann *ast.ActivityAnnotations) {
	if ann == nil {
		return
	}

	if ann.Caption != "" || ann.Color != "" || ann.Excluded {
		for _, obj := range fb.objects {
			if obj == nil || obj.ID() != activityID {
				continue
			}
			switch a := obj.(type) {
			case *genMf.ActionActivity:
				if ann.Caption != "" {
					a.SetCaption(ann.Caption)
					a.SetAutoGenerateCaption(false)
				}
				if ann.Color != "" {
					a.SetBackgroundColor(ann.Color)
				}
				if ann.Excluded {
					a.SetDisabled(true)
				}
			case *genMf.ExclusiveSplit:
				if ann.Caption != "" {
					a.SetCaption(ann.Caption)
				}
			case *genMf.InheritanceSplit:
				if ann.Caption != "" {
					a.SetCaption(ann.Caption)
				}
				// LoopedActivity has no Caption setter in the gen schema —
				// the legacy SDK type carried one but the BSON didn't
				// surface it in any roundtrip fixture, so dropping the
				// branch here is a no-op for known examples and matches
				// what the gen describer reads back.
			}
			break
		}
	}

	if ann.AnnotationText != "" {
		fb.attachAnnotation(ann.AnnotationText, activityID)
	}
}

// applyPendingAnnotations applies (and clears) any annotations queued
// by mergeStatementAnnotations for the freshly-emitted activity.
func (fb *flowBuilderGen) applyPendingAnnotations(activityID element.ID) {
	if activityID == "" || fb.pendingAnnotations == nil {
		return
	}
	fb.applyAnnotations(activityID, fb.pendingAnnotations)
	fb.pendingAnnotations = nil
}

// addEndEventWithReturn emits an EndEvent with the supplied return
// expression and sets the terminator markers on the builder. Returns
// the new EndEvent's ID so the caller can wire its inbound flow.
//
// Layout: the EndEvent's RelativeMiddlePoint is the current
// (posX, posY); Size is a uniform EventSize square. posX is advanced
// by spacing/2 to leave room for downstream fan-in geometry.
func (fb *flowBuilderGen) addEndEventWithReturn(s *ast.ReturnStmt) element.ID {
	retVal := ""
	if s != nil && s.Value != nil {
		retVal = fb.exprToString(s.Value)
	}

	endEvent := genMf.NewEndEvent()
	id := assignFreshID(endEvent)
	endEvent.SetRelativeMiddlePoint(layoutPos(fb.posX, fb.posY))
	endEvent.SetSize(layoutSize(EventSize, EventSize))
	if retVal != "" {
		endEvent.SetReturnValue(retVal)
	}

	fb.objects = append(fb.objects, endEvent)
	fb.endsWithReturn = true
	fb.lastReturnEndID = id
	fb.posX += fb.spacing / 2
	return id
}

// addErrorEvent emits an ErrorEvent terminator. Used by RAISE ERROR
// statements inside custom error handlers. Returns the new event's ID.
func (fb *flowBuilderGen) addErrorEvent() element.ID {
	errorEvent := genMf.NewErrorEvent()
	id := assignFreshID(errorEvent)
	errorEvent.SetRelativeMiddlePoint(layoutPos(fb.posX, fb.posY))
	errorEvent.SetSize(layoutSize(EventSize, EventSize))

	fb.objects = append(fb.objects, errorEvent)
	fb.endsWithReturn = true
	fb.posX += fb.spacing / 2
	return id
}

// attachAnnotation creates a free Annotation positioned above the
// activity identified by activityID and connects them with an
// AnnotationFlow. Position lookup uses parseLayoutPos to read the
// activity's RelativeMiddlePoint string back into integers.
func (fb *flowBuilderGen) attachAnnotation(text string, activityID element.ID) {
	actX, actY := 0, 0
	for _, obj := range fb.objects {
		if obj == nil || obj.ID() != activityID {
			continue
		}
		if pos, ok := genElementMiddlePoint(obj); ok {
			actX, actY = pos.x, pos.y
		}
		break
	}

	annotation := genMf.NewAnnotation()
	annID := assignFreshID(annotation)
	annotation.SetRelativeMiddlePoint(layoutPos(actX, actY-100))
	annotation.SetSize(layoutSize(200, 50))
	annotation.SetCaption(text)
	fb.objects = append(fb.objects, annotation)

	flow := genMf.NewAnnotationFlow()
	assignFreshID(flow)
	flow.SetOriginID(annID)
	flow.SetDestinationID(activityID)
	fb.annotationFlows = append(fb.annotationFlows, flow)
}

// attachFreeAnnotation creates a free-floating Annotation not connected
// to any activity. Positioned above the builder's current cursor.
func (fb *flowBuilderGen) attachFreeAnnotation(text string) {
	annotation := genMf.NewAnnotation()
	assignFreshID(annotation)
	annotation.SetRelativeMiddlePoint(layoutPos(fb.posX, fb.posY-100))
	annotation.SetSize(layoutSize(200, 50))
	annotation.SetCaption(text)
	fb.objects = append(fb.objects, annotation)
}

// genLayoutPoint is a parsed (x, y) pair recovered from a gen
// element's RelativeMiddlePoint string.
type genLayoutPoint struct {
	x, y int
}

// parseLayoutPos parses a RelativeMiddlePoint string. Accepts both the
// canonical semicolon format written by Mendix Studio Pro and by mxcli
// ("200;200") and the legacy space-separated format written by older
// mxcli versions ("200 200"). Returns ok=false on any parse error.
func parseLayoutPos(s string) (genLayoutPoint, bool) {
	if s == "" {
		return genLayoutPoint{}, false
	}
	// Normalise: treat semicolons as spaces so Sscanf handles both formats.
	normalized := strings.ReplaceAll(s, ";", " ")
	var p genLayoutPoint
	n, err := fmt.Sscanf(normalized, "%d %d", &p.x, &p.y)
	if err != nil || n != 2 {
		return genLayoutPoint{}, false
	}
	return p, true
}

// genElementMiddlePoint extracts the RelativeMiddlePoint of any gen
// element that exposes it. Returns ok=false for elements that don't
// (sequence flows, parameters, etc.). The dispatch enumerates the gen
// kinds the builder actually emits — extending the set is safe but
// not required until new emitters land.
func genElementMiddlePoint(e element.Element) (genLayoutPoint, bool) {
	switch a := e.(type) {
	case *genMf.ActionActivity:
		return parseLayoutPos(a.RelativeMiddlePoint())
	case *genMf.StartEvent:
		return parseLayoutPos(a.RelativeMiddlePoint())
	case *genMf.EndEvent:
		return parseLayoutPos(a.RelativeMiddlePoint())
	case *genMf.ErrorEvent:
		return parseLayoutPos(a.RelativeMiddlePoint())
	case *genMf.ExclusiveSplit:
		return parseLayoutPos(a.RelativeMiddlePoint())
	case *genMf.InheritanceSplit:
		return parseLayoutPos(a.RelativeMiddlePoint())
	case *genMf.ExclusiveMerge:
		return parseLayoutPos(a.RelativeMiddlePoint())
	case *genMf.LoopedActivity:
		return parseLayoutPos(a.RelativeMiddlePoint())
	case *genMf.Annotation:
		return parseLayoutPos(a.RelativeMiddlePoint())
	case *genMf.BreakEvent:
		return parseLayoutPos(a.RelativeMiddlePoint())
	case *genMf.ContinueEvent:
		return parseLayoutPos(a.RelativeMiddlePoint())
	}
	return genLayoutPoint{}, false
}
