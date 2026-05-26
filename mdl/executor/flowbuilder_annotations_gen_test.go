// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.b — annotation / terminal event tests.
//
// Verifies the gen-typed counterparts of EndEvent / ErrorEvent
// emission, @caption / @color / @excluded application, and
// Annotation+AnnotationFlow attachment.

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

func TestParseLayoutPos(t *testing.T) {
	cases := []struct {
		in   string
		ok   bool
		x, y int
	}{
		// semicolon format (canonical — Mendix Studio Pro and new mxcli output)
		{"100;200", true, 100, 200},
		{"0;0", true, 0, 0},
		{"-15;25", true, -15, 25},
		// space format (backward compat — old mxcli-generated files)
		{"100 200", true, 100, 200},
		{"0 0", true, 0, 0},
		{"-15 25", true, -15, 25},
		{"", false, 0, 0},
		{"100", false, 0, 0},
		{"abc def", false, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, ok := parseLayoutPos(tc.in)
			if ok != tc.ok {
				t.Fatalf("ok = %t want %t", ok, tc.ok)
			}
			if ok && (got.x != tc.x || got.y != tc.y) {
				t.Fatalf("got (%d, %d), want (%d, %d)", got.x, got.y, tc.x, tc.y)
			}
		})
	}
}

func TestFlowBuilderGenAddEndEventWithReturnNoValue(t *testing.T) {
	fb := &flowBuilderGen{posX: 200, posY: 200, spacing: 160}
	id := fb.addEndEventWithReturn(nil)

	if id == "" {
		t.Fatal("addEndEventWithReturn should return a non-empty ID")
	}
	if !fb.endsWithReturn {
		t.Fatal("endsWithReturn flag should be set")
	}
	if fb.lastReturnEndID != id {
		t.Fatalf("lastReturnEndID = %s, want %s", fb.lastReturnEndID, id)
	}
	if fb.posX != 280 {
		t.Fatalf("posX after EndEvent = %d, want 280 (advance by spacing/2)", fb.posX)
	}
	if len(fb.objects) != 1 {
		t.Fatalf("want 1 object, got %d", len(fb.objects))
	}
	end, ok := fb.objects[0].(*genMf.EndEvent)
	if !ok {
		t.Fatalf("want *genMf.EndEvent, got %T", fb.objects[0])
	}
	if end.RelativeMiddlePoint() != "200;200" {
		t.Fatalf("position = %q, want %q", end.RelativeMiddlePoint(), "200;200")
	}
	if end.Size() != "20;20" {
		t.Fatalf("size = %q, want %q", end.Size(), "20;20")
	}
	if end.ReturnValue() != "" {
		t.Fatalf("ReturnValue = %q, want empty", end.ReturnValue())
	}
}

func TestFlowBuilderGenAddEndEventWithReturnLiteral(t *testing.T) {
	fb := &flowBuilderGen{posX: 100, posY: 100, spacing: 160}
	stmt := &ast.ReturnStmt{
		Value: &ast.LiteralExpr{Kind: ast.LiteralBoolean, Value: true},
	}
	id := fb.addEndEventWithReturn(stmt)

	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	end := fb.objects[0].(*genMf.EndEvent)
	if end.ReturnValue() != "true" {
		t.Fatalf("ReturnValue = %q, want %q", end.ReturnValue(), "true")
	}
}

func TestFlowBuilderGenAddErrorEvent(t *testing.T) {
	fb := &flowBuilderGen{posX: 50, posY: 60, spacing: 200}
	id := fb.addErrorEvent()

	if id == "" {
		t.Fatal("expected non-empty ID")
	}
	if !fb.endsWithReturn {
		t.Fatal("endsWithReturn flag should be set on ErrorEvent")
	}
	if fb.posX != 150 {
		t.Fatalf("posX after ErrorEvent = %d, want 150", fb.posX)
	}
	ev, ok := fb.objects[0].(*genMf.ErrorEvent)
	if !ok {
		t.Fatalf("want *genMf.ErrorEvent, got %T", fb.objects[0])
	}
	if ev.RelativeMiddlePoint() != "50;60" {
		t.Fatalf("position = %q, want %q", ev.RelativeMiddlePoint(), "50;60")
	}
}

func TestFlowBuilderGenAttachAnnotationFindsActivityPosition(t *testing.T) {
	fb := &flowBuilderGen{posX: 200, posY: 200, spacing: 160}
	// Pre-seed a fake action activity at a known position.
	act := genMf.NewActionActivity()
	assignFreshID(act)
	act.SetRelativeMiddlePoint("400;300")
	fb.objects = append(fb.objects, act)

	fb.attachAnnotation("explanation", act.ID())

	if len(fb.objects) != 2 {
		t.Fatalf("want 2 objects (activity + annotation), got %d", len(fb.objects))
	}
	annot, ok := fb.objects[1].(*genMf.Annotation)
	if !ok {
		t.Fatalf("want *genMf.Annotation, got %T", fb.objects[1])
	}
	if annot.Caption() != "explanation" {
		t.Fatalf("caption = %q, want %q", annot.Caption(), "explanation")
	}
	// Annotation positioned 100px above the activity.
	if annot.RelativeMiddlePoint() != "400;200" {
		t.Fatalf("annotation position = %q, want %q", annot.RelativeMiddlePoint(), "400;200")
	}

	if len(fb.annotationFlows) != 1 {
		t.Fatalf("want 1 annotation flow, got %d", len(fb.annotationFlows))
	}
	flow := fb.annotationFlows[0]
	if flow.OriginRefID() != annot.ID() {
		t.Fatalf("flow origin = %s, want %s", flow.OriginRefID(), annot.ID())
	}
	if flow.DestinationRefID() != act.ID() {
		t.Fatalf("flow dest = %s, want %s", flow.DestinationRefID(), act.ID())
	}
}

func TestFlowBuilderGenAttachAnnotationMissingActivityFallsBackToOrigin(t *testing.T) {
	fb := &flowBuilderGen{posX: 0, posY: 0}
	// Activity ID that doesn't exist in fb.objects — annotation
	// should still attach but at the (0, -100) origin offset.
	fb.attachAnnotation("note", element.ID("missing-activity-id"))

	if len(fb.objects) != 1 {
		t.Fatalf("want 1 object (annotation only), got %d", len(fb.objects))
	}
	annot := fb.objects[0].(*genMf.Annotation)
	if annot.RelativeMiddlePoint() != "0;-100" {
		t.Fatalf("position = %q, want %q", annot.RelativeMiddlePoint(), "0;-100")
	}
}

func TestFlowBuilderGenAttachFreeAnnotation(t *testing.T) {
	fb := &flowBuilderGen{posX: 300, posY: 200}
	fb.attachFreeAnnotation("free")
	if len(fb.objects) != 1 {
		t.Fatalf("want 1 object, got %d", len(fb.objects))
	}
	annot := fb.objects[0].(*genMf.Annotation)
	if annot.Caption() != "free" {
		t.Fatalf("caption = %q, want %q", annot.Caption(), "free")
	}
	// Free annotation positioned 100px above current cursor.
	if annot.RelativeMiddlePoint() != "300;100" {
		t.Fatalf("position = %q, want %q", annot.RelativeMiddlePoint(), "300;100")
	}
	// No annotation flow created for free annotations.
	if len(fb.annotationFlows) != 0 {
		t.Fatalf("free annotation should not create a flow, got %d", len(fb.annotationFlows))
	}
}

func TestFlowBuilderGenApplyAnnotationsActionCaptionColorExcluded(t *testing.T) {
	fb := &flowBuilderGen{}
	act := genMf.NewActionActivity()
	assignFreshID(act)
	act.SetAutoGenerateCaption(true)
	fb.objects = append(fb.objects, act)

	fb.applyAnnotations(act.ID(), &ast.ActivityAnnotations{
		Caption:  "My Caption",
		Color:    "#ff0000",
		Excluded: true,
	})

	if act.Caption() != "My Caption" {
		t.Fatalf("caption = %q, want %q", act.Caption(), "My Caption")
	}
	if act.AutoGenerateCaption() {
		t.Fatal("AutoGenerateCaption should be false after explicit caption")
	}
	if act.BackgroundColor() != "#ff0000" {
		t.Fatalf("color = %q, want %q", act.BackgroundColor(), "#ff0000")
	}
	if !act.Disabled() {
		t.Fatal("Excluded should set Disabled=true")
	}
}

func TestFlowBuilderGenApplyAnnotationsExclusiveSplit(t *testing.T) {
	fb := &flowBuilderGen{}
	split := genMf.NewExclusiveSplit()
	assignFreshID(split)
	fb.objects = append(fb.objects, split)

	fb.applyAnnotations(split.ID(), &ast.ActivityAnnotations{Caption: "Right format?"})

	if split.Caption() != "Right format?" {
		t.Fatalf("caption = %q, want %q", split.Caption(), "Right format?")
	}
}

func TestFlowBuilderGenApplyAnnotationsAttachesAnnotationText(t *testing.T) {
	fb := &flowBuilderGen{}
	act := genMf.NewActionActivity()
	assignFreshID(act)
	act.SetRelativeMiddlePoint("100;100")
	fb.objects = append(fb.objects, act)

	fb.applyAnnotations(act.ID(), &ast.ActivityAnnotations{AnnotationText: "explains"})

	// Should have created an Annotation + AnnotationFlow.
	if len(fb.objects) != 2 {
		t.Fatalf("want 2 objects, got %d", len(fb.objects))
	}
	if len(fb.annotationFlows) != 1 {
		t.Fatalf("want 1 annotation flow, got %d", len(fb.annotationFlows))
	}
}

func TestFlowBuilderGenApplyPendingAnnotationsClearsState(t *testing.T) {
	fb := &flowBuilderGen{}
	act := genMf.NewActionActivity()
	assignFreshID(act)
	fb.objects = append(fb.objects, act)
	fb.pendingAnnotations = &ast.ActivityAnnotations{Caption: "Pending"}

	fb.applyPendingAnnotations(act.ID())

	if act.Caption() != "Pending" {
		t.Fatalf("caption = %q, want %q", act.Caption(), "Pending")
	}
	if fb.pendingAnnotations != nil {
		t.Fatal("pendingAnnotations should be cleared after apply")
	}
}

func TestFlowBuilderGenMergeStatementAnnotationsMerges(t *testing.T) {
	fb := &flowBuilderGen{}
	// First merge sets caption and color.
	fb.mergeStatementAnnotations(&ast.LogStmt{
		Annotations: &ast.ActivityAnnotations{Caption: "First", Color: "#abc"},
	})
	// Second merge adds annotation text but leaves caption alone
	// (later non-empty fields overwrite, empty ones don't).
	fb.mergeStatementAnnotations(&ast.LogStmt{
		Annotations: &ast.ActivityAnnotations{AnnotationText: "details"},
	})

	if fb.pendingAnnotations == nil {
		t.Fatal("pendingAnnotations should be non-nil after merge")
	}
	got := fb.pendingAnnotations
	if got.Caption != "First" || got.Color != "#abc" || got.AnnotationText != "details" {
		t.Fatalf("merge result = %+v", got)
	}
}
