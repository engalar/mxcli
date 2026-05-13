// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.5b — gen-typed OData consumer scan.
//
// listExternalActions in cmd_odata.go walks every microflow and
// nanoflow looking for ActionActivity → CallExternalAction objects to
// build the SHOW EXTERNAL ACTIONS table. It does this through
// sdk/microflows.MicroflowObjectCollection / ActionActivity /
// CallExternalAction. This file provides the modelsdk/gen-native
// equivalent that consumes ctx.Microflows / ctx.Nanoflows.
//
// describeExternalEntity, the published-OData-service builders, and
// every other function in cmd_odata.go are unrelated to
// sdk/microflows so they stay in the legacy file.
//
// The dispatch layer (Stage 3.2.6) will route SHOW EXTERNAL ACTIONS
// to listExternalActionsGen and delete the sdk-typed original.

package executor

import (
	"fmt"
	"sort"
	"strings"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// listExternalActionsGen handles SHOW EXTERNAL ACTIONS [IN module]
// using gen-typed microflows / nanoflows. Output table shape is
// identical to listExternalActions: Service / Action / Parameters /
// UsedBy.
//
// The collection-walking pattern mirrors the legacy extractActions
// closure: for every flow, inspect its ObjectCollection items, keep
// the ones whose TypeName is "Microflows$ActionActivity", then look
// at the wrapped Action — if it's a CallExternalAction, record the
// service / action / parameters / caller.
func listExternalActionsGen(ctx *ExecContext, moduleName string) error {
	if ctx == nil {
		return mdlerrors.NewBackend("nil exec context", nil)
	}

	mfs, err := listMicroflowsGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list microflows", err)
	}
	nfs, err := listNanoflowsGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list nanoflows", err)
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	type actionInfo struct {
		service    string
		actionName string
		params     []string
		callers    []string
	}
	actionMap := make(map[string]*actionInfo)

	// extractGen walks an ObjectCollection picking out CallExternalAction
	// activities. Nanoflows reuse MicroflowObjectCollection (gen models
	// share the type) so a single helper covers both flow kinds.
	extractGen := func(oc *genMf.MicroflowObjectCollection, flowModule, flowName string) {
		if oc == nil {
			return
		}
		for _, obj := range oc.ObjectsItems() {
			if obj == nil {
				continue
			}
			act, ok := obj.(*genMf.ActionActivity)
			if !ok || act == nil {
				continue
			}
			cea, ok := act.Action().(*genMf.CallExternalAction)
			if !ok || cea == nil {
				continue
			}

			service := cea.ConsumedODataServiceQualifiedName()
			actionName := cea.Name()
			key := service + "." + actionName

			info, exists := actionMap[key]
			if !exists {
				info = &actionInfo{
					service:    service,
					actionName: actionName,
					params:     externalActionParameterNames(cea),
				}
				actionMap[key] = info
			}
			caller := flowModule + "." + flowName
			seen := false
			for _, c := range info.callers {
				if c == caller {
					seen = true
					break
				}
			}
			if !seen {
				info.callers = append(info.callers, caller)
			}
			// Merge parameter names if a later call site has more.
			if names := externalActionParameterNames(cea); len(names) > len(info.params) {
				info.params = names
			}
		}
	}

	for _, mf := range mfs {
		if mf == nil {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(mf.ID()))
		if moduleName != "" && !strings.EqualFold(modName, moduleName) {
			continue
		}
		oc, _ := mf.ObjectCollection().(*genMf.MicroflowObjectCollection)
		extractGen(oc, modName, mf.Name())
	}
	for _, nf := range nfs {
		if nf == nil {
			continue
		}
		modName := genFlowContainerModule(ctx, h, model.ID(nf.ID()))
		if moduleName != "" && !strings.EqualFold(modName, moduleName) {
			continue
		}
		oc, _ := nf.ObjectCollection().(*genMf.MicroflowObjectCollection)
		extractGen(oc, modName, nf.Name())
	}

	if len(actionMap) == 0 && ctx.Format != FormatJSON {
		fmt.Fprintln(ctx.Output, "No external actions found.")
		return nil
	}

	type row struct {
		service    string
		actionName string
		params     string
		usedBy     string
	}
	var rows []row
	for _, info := range actionMap {
		rows = append(rows, row{
			service:    info.service,
			actionName: info.actionName,
			params:     strings.Join(info.params, ", "),
			usedBy:     strings.Join(info.callers, ", "),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].service != rows[j].service {
			return strings.ToLower(rows[i].service) < strings.ToLower(rows[j].service)
		}
		return strings.ToLower(rows[i].actionName) < strings.ToLower(rows[j].actionName)
	})

	result := &TableResult{
		Columns: []string{"Service", "Action", "Parameters", "UsedBy"},
		Summary: fmt.Sprintf("(%d external actions)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.service, r.actionName, r.params, r.usedBy})
	}
	return writeResult(ctx, result)
}

// externalActionParameterNames pulls the ParameterName values out of
// a CallExternalAction's ParameterMappings list. Mirrors the inline
// loop in the legacy extractActions closure. The mapping items are
// ExternalActionParameterMapping elements; we pick up the name
// either through the typed accessor (when the runtime type matches)
// or via the duck-typed `ParameterName() string` interface (covers
// future/related mapping types without a hard dependency).
func externalActionParameterNames(cea *genMf.CallExternalAction) []string {
	if cea == nil {
		return nil
	}
	items := cea.ParameterMappingsItems()
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if pm, ok := item.(*genMf.ExternalActionParameterMapping); ok {
			out = append(out, pm.ParameterName())
			continue
		}
		// Fallback for any other mapping shape that exposes a
		// ParameterName getter (e.g. legacy/forward-compat elements).
		if pn, ok := item.(interface{ ParameterName() string }); ok {
			out = append(out, pn.ParameterName())
		}
	}
	return out
}
