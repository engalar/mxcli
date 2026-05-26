// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.4 D1.d: AST → gen-typed EventHandler / Index / IndexedAttribute
// builders. Used by D2 execCreateEntityGen for entities that ship with
// indexes / event handlers, and by D3 execAlterEntityGen for ADD INDEX
// / ADD EVENT HANDLER subcommands.

package executor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// astToEventHandlerGen builds a gen-typed *EventHandler from the AST
// definition. Mirrors the legacy domainmodel.EventHandler construction
// in cmd_entities.go, but uses gen accessors:
//   - Moment / Event are strings on gen (sdk used enum types)
//   - MicroflowQualifiedName replaces sdk's MicroflowName/MicroflowID split
func astToEventHandlerGen(eh *ast.EventHandlerDef) *genDm.EventHandler {
	if eh == nil {
		return nil
	}
	out := genDm.NewEventHandler()
	out.SetMoment(strings.Title(strings.ToLower(eh.Moment))) //nolint:staticcheck // Title is fine for ASCII
	out.SetEvent(strings.Title(strings.ToLower(eh.Event)))
	if eh.Microflow.Module != "" || eh.Microflow.Name != "" {
		out.SetMicroflowQualifiedName(eh.Microflow.String())
	}
	out.SetRaiseErrorOnFalse(eh.RaiseErrorOnFalse)
	out.SetPassEventObject(eh.PassEventObject)
	return out
}

// astToIndexGen builds a gen-typed *Index from an AST Index. Caller must
// supply attrNameToID — gen IndexedAttribute references the attribute by
// ID, not name. Unknown attribute names produce an Index with fewer
// columns; the caller should validate first.
func astToIndexGen(idx *ast.Index, attrNameToID map[string]model.ID) *genDm.Index {
	if idx == nil {
		return nil
	}
	out := genDm.NewIndex()
	for _, col := range idx.Columns {
		id, ok := attrNameToID[col.Name]
		if !ok {
			continue
		}
		ia := genDm.NewIndexedAttribute()
		ia.SetAttributeID(element.ID(id))
		ia.SetAscending(!col.Descending)
		out.AddAttributes(ia)
	}
	return out
}
