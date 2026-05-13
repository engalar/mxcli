// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// PageMutator performs localized edits on a page/snippet/layout unit
// without re-encoding the whole element on every call. Obtain via
// PageRepository.OpenForMutation; persist with Commit.
//
// Lifecycle: OpenForMutation → N edits → Commit (single InsertUnit /
// WriteTransaction.WriteUnit at the end). Mutators are not safe for
// concurrent use.
type PageMutator interface {
	SetWidgetProperty(widgetID model.ID, prop string, value any) error
	InsertWidget(parentID model.ID, slot string, widget element.Element) error
	DeleteWidget(widgetID model.ID) error
	ReplaceWidget(widgetID model.ID, replacement element.Element) error
	SetLayout(layoutQN string) error
	Commit() error
}
