// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// activityBoxWidth and activityBoxHeight are the approximate pixel dimensions of a
// Mendix microflow activity box on the canvas. Two activities overlap when their
// top-left corner positions differ by less than these thresholds.
const activityBoxWidth = 120
const activityBoxHeight = 60

// OverlappingActivitiesRule flags microflow activities whose canvas positions overlap.
//
// The most common cause is writing multiple MDL statements after a single @position
// annotation — e.g. a DECLARE followed immediately by a SET with no second @position.
// The executor auto-places the un-annotated statement only 150px to the right (less
// than one activity width from the next explicitly annotated activity), producing
// overlapping boxes in Studio Pro.
type OverlappingActivitiesRule struct{}

func NewOverlappingActivitiesRule() *OverlappingActivitiesRule {
	return &OverlappingActivitiesRule{}
}

func (r *OverlappingActivitiesRule) ID() string                       { return "MPR008" }
func (r *OverlappingActivitiesRule) Name() string                     { return "OverlappingActivities" }
func (r *OverlappingActivitiesRule) Category() string                 { return "correctness" }
func (r *OverlappingActivitiesRule) DefaultSeverity() linter.Severity { return linter.SeverityWarning }
func (r *OverlappingActivitiesRule) Description() string {
	return "Microflow activities whose canvas positions overlap, typically caused by missing @position annotations in MDL"
}

// activityPos pairs a parsed (x, y) canvas coordinate with the
// activity's caption — both used by the overlap-detection inner loop.
type activityPos struct {
	x, y    int
	caption string
}

func (r *OverlappingActivitiesRule) Check(ctx *linter.LintContext) []linter.Violation {
	reader := ctx.Reader()
	if reader == nil {
		return nil
	}

	var violations []linter.Violation

	for _, mf := range ctx.Microflows() {
		if ctx.IsExcluded(mf.Module) {
			continue
		}

		fullMF, err := reader.GetMicroflowGen(model.ID(mf.ID))
		if err != nil || fullMF == nil {
			continue
		}
		oc, _ := fullMF.ObjectCollection().(*genMf.MicroflowObjectCollection)
		if oc == nil {
			continue
		}

		var activities []activityPos
		var collect func(objects []element.Element)
		collect = func(objects []element.Element) {
			for _, obj := range objects {
				switch act := obj.(type) {
				case *genMf.ActionActivity:
					if x, y, ok := parseActivityPos(act.RelativeMiddlePoint()); ok {
						caption := act.Caption()
						if caption == "" {
							caption = "(unnamed)"
						}
						activities = append(activities, activityPos{x, y, caption})
					}
				case *genMf.LoopedActivity:
					if x, y, ok := parseActivityPos(act.RelativeMiddlePoint()); ok {
						caption := act.Documentation()
						if caption == "" {
							caption = "(loop)"
						}
						activities = append(activities, activityPos{x, y, caption})
					}
					if body, ok := act.ObjectCollection().(*genMf.MicroflowObjectCollection); ok && body != nil {
						collect(body.ObjectsItems())
					}
				case *genMf.ExclusiveSplit:
					if x, y, ok := parseActivityPos(act.RelativeMiddlePoint()); ok {
						activities = append(activities, activityPos{x, y, act.Caption()})
					}
				case *genMf.ExclusiveMerge:
					if x, y, ok := parseActivityPos(act.RelativeMiddlePoint()); ok {
						activities = append(activities, activityPos{x, y, "(merge)"})
					}
				}
			}
		}
		collect(oc.ObjectsItems())

		// Check all pairs for overlapping positions.
		// Skip activities at the origin (0,0) — these are unpositioned/default.
		reported := make(map[string]bool)
		for i := 0; i < len(activities); i++ {
			for j := i + 1; j < len(activities); j++ {
				a, b := activities[i], activities[j]
				if (a.x == 0 && a.y == 0) || (b.x == 0 && b.y == 0) {
					continue
				}
				dx := a.x - b.x
				if dx < 0 {
					dx = -dx
				}
				dy := a.y - b.y
				if dy < 0 {
					dy = -dy
				}
				if dx < activityBoxWidth && dy < activityBoxHeight {
					key := fmt.Sprintf("%d,%d|%d,%d", a.x, a.y, b.x, b.y)
					if reported[key] {
						continue
					}
					reported[key] = true
					violations = append(violations, linter.Violation{
						RuleID:   r.ID(),
						Severity: r.DefaultSeverity(),
						Message: fmt.Sprintf(
							"Activities '%s' (%d,%d) and '%s' (%d,%d) overlap in microflow '%s.%s'. "+
								"Each MDL statement that creates a canvas activity needs its own @position annotation.",
							a.caption, a.x, a.y, b.caption, b.x, b.y,
							mf.Module, mf.Name,
						),
						Location: linter.Location{
							Module:       mf.Module,
							DocumentType: "microflow",
							DocumentName: mf.Name,
							DocumentID:   mf.ID,
						},
						Suggestion: "Add a separate @position(x, y) annotation before each statement. Use 190px spacing between activities.",
					})
				}
			}
		}
	}

	return violations
}

// parseActivityPos parses a RelativeMiddlePoint string. Accepts both the
// canonical semicolon format ("200;200") written by Studio Pro / new mxcli
// and the legacy space-separated format ("200 200") from older mxcli.
func parseActivityPos(s string) (int, int, bool) {
	if s == "" {
		return 0, 0, false
	}
	normalized := strings.ReplaceAll(s, ";", " ")
	var x, y int
	n, err := fmt.Sscanf(normalized, "%d %d", &x, &y)
	if err != nil || n != 2 {
		return 0, 0, false
	}
	return x, y, true
}
