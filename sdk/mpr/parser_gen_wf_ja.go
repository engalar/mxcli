// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genWf "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"

	"go.mongodb.org/mongo-driver/bson"
)

// workflowDecoder is a package-level codec.Decoder for workflows gen types.
var workflowDecoder = codec.NewDecoder(codec.DefaultRegistry)

// javaActionDecoder is a package-level codec.Decoder for javaactions gen types.
var javaActionDecoder = codec.NewDecoder(codec.DefaultRegistry)

// parseWorkflowGen parses a Workflows$Workflow BSON document into a gen-typed Workflow.
func (r *Reader) parseWorkflowGen(unitID, _ string, contents []byte) (*genWf.Workflow, error) {
	contents, err := r.resolveContents(unitID, contents)
	if err != nil {
		return nil, err
	}
	elem, err := workflowDecoder.Decode(bson.Raw(contents))
	if err != nil {
		return nil, fmt.Errorf("decode workflow %s: %w", unitID, err)
	}
	wf, ok := elem.(*genWf.Workflow)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a Workflow (got %T)", unitID, elem)
	}
	wf.SetID(element.ID(unitID))
	return wf, nil
}

// parseJavaActionGen parses a JavaActions$JavaAction BSON document into a gen-typed JavaAction.
func (r *Reader) parseJavaActionGen(unitID, _ string, contents []byte) (*genJA.JavaAction, error) {
	contents, err := r.resolveContents(unitID, contents)
	if err != nil {
		return nil, err
	}
	elem, err := javaActionDecoder.Decode(bson.Raw(contents))
	if err != nil {
		return nil, fmt.Errorf("decode java action %s: %w", unitID, err)
	}
	ja, ok := elem.(*genJA.JavaAction)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a JavaAction (got %T)", unitID, elem)
	}
	ja.SetID(element.ID(unitID))
	return ja, nil
}

// ListWorkflowsGen returns all workflows as gen-typed Workflow values.
// ContainerID is NOT set on the returned values; callers that need it should join
// against Reader.ListUnitsByType("Workflows$Workflow") to get the ContainerID
// column from the SQLite unit index (same pattern as DomainModel in project_tree.go).
func (r *Reader) ListWorkflowsGen() ([]*genWf.Workflow, error) {
	units, err := r.listUnitsByType("Workflows$Workflow")
	if err != nil {
		return nil, err
	}
	var result []*genWf.Workflow
	for _, u := range units {
		wf, err := r.parseWorkflowGen(u.ID, u.ContainerID, u.Contents)
		if err != nil {
			return nil, fmt.Errorf("failed to parse workflow %s: %w", u.ID, err)
		}
		result = append(result, wf)
	}
	return result, nil
}

// ListJavaActionsGen returns all Java actions as gen-typed JavaAction values.
// ContainerID is NOT set on the returned values; callers that need it should join
// against Reader.ListUnitsByType("JavaActions$JavaAction") to get the ContainerID
// column from the SQLite unit index (same pattern as DomainModel in project_tree.go).
// Note: virtual System module Java actions are NOT included (unlike ListJavaActions).
func (r *Reader) ListJavaActionsGen() ([]*genJA.JavaAction, error) {
	units, err := r.listUnitsByType("JavaActions$JavaAction")
	if err != nil {
		return nil, err
	}
	var result []*genJA.JavaAction
	for _, u := range units {
		ja, err := r.parseJavaActionGen(u.ID, u.ContainerID, u.Contents)
		if err != nil {
			return nil, fmt.Errorf("failed to parse java action %s: %w", u.ID, err)
		}
		result = append(result, ja)
	}
	return result, nil
}
