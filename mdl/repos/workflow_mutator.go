// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// WorkflowMutator is the workflow-domain analogue of PageMutator.
// Activities, outcomes, branches, and paths are all addressed by their
// stable model.ID. Commit persists with one Writer call.
type WorkflowMutator interface {
	SetActivityProperty(activityID model.ID, prop string, value any) error
	InsertActivity(parentID model.ID, slot string, activity element.Element) error
	DeleteActivity(activityID model.ID) error
	ReplaceActivity(activityID model.ID, replacement element.Element) error
	Commit() error
}
