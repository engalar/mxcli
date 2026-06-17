// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"log"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// actionBuilderFn builds a client action gen element from an ActionV3 AST node.
type actionBuilderFn func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error)

// actionBuilders maps each action type string to its builder.
var actionBuilders = map[string]actionBuilderFn{
	"save": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		act := genPg.NewSaveChangesClientAction()
		assignFreshID(act)
		act.SetClosePage(action.ClosePage)
		return act, nil
	},
	"cancel": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		act := genPg.NewCancelChangesClientAction()
		assignFreshID(act)
		act.SetClosePage(action.ClosePage)
		return act, nil
	},
	"close": func(pb *pageBuilder, _ *ast.ActionV3) (element.Element, error) {
		act := genPg.NewClosePageClientAction()
		assignFreshID(act)
		return act, nil
	},
	"delete": func(pb *pageBuilder, _ *ast.ActionV3) (element.Element, error) {
		act := genPg.NewDeleteClientAction()
		assignFreshID(act)
		return act, nil
	},
	"create": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		entityID, err := pb.resolveEntity(ast.QualifiedName{
			Module: pb.extractModule(action.Target),
			Name:   pb.extractName(action.Target),
		})
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve entity for create", err)
		}
		_ = entityID

		act := genPg.NewCreateObjectClientAction()
		assignFreshID(act)
		ref := genDm.NewDirectEntityRef()
		assignFreshID(ref)
		ref.SetEntityQualifiedName(action.Target)
		act.SetEntityRef(ref)

		if action.ThenAction != nil && action.ThenAction.Type == "showPage" {
			if _, err := pb.resolvePageRef(action.ThenAction.Target); err != nil {
				log.Printf("warning: then show_page %s not found (will still create action by name)", action.ThenAction.Target)
			}
			ps := genPg.NewPageSettings()
			assignFreshID(ps)
			ps.SetPageQualifiedName(action.ThenAction.Target)
			act.SetPageSettings(ps)
		}
		return act, nil
	},
	"showPage": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		if _, err := pb.resolvePageRef(action.Target); err != nil {
			log.Printf("warning: action show_page %s not found (will still create action by name)", action.Target)
		}

		act := genPg.NewPageClientAction()
		assignFreshID(act)
		ps := genPg.NewPageSettings()
		assignFreshID(ps)
		ps.SetPageQualifiedName(action.Target)
		// ParameterMappings intentionally left empty: Mendix propagates the page
		// parameter from the calling context automatically. Storing PageParameterMapping
		// (Forms$FormCallArgument) objects inside FormSettings (PageSettings) causes
		// LayoutCallArgument constructor failures in Studio Pro when the container type
		// is PageSettings rather than LayoutCall.
		act.SetPageSettings(ps)
		act.SetDisabledDuringExecution(false)
		act.SetNumberOfPagesToClose2("1")
		return act, nil
	},
	"microflow": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		if _, err := pb.resolveMicroflow(action.Target); err != nil {
			log.Printf("warning: action microflow %s not found (will still create action by name)", action.Target)
		}

		act := genPg.NewMicroflowClientAction()
		assignFreshID(act)
		settings := genPg.NewMicroflowSettings()
		assignFreshID(settings)
		settings.SetMicroflowQualifiedName(action.Target)

		for _, arg := range action.Args {
			mm := genPg.NewMicroflowParameterMapping()
			assignFreshID(mm)
			// SP11.6.6: fully-qualified "Module.MicroflowName.ParamName"
			mm.SetParameterQualifiedName(action.Target + "." + arg.Name)

			if strVal, ok := arg.Value.(string); ok {
				// SP11.6.6: use Expression for all values (not Variable sub-object)
				mm.SetExpression(strVal)
			}
			settings.AddParameterMappings(mm)
		}

		act.SetMicroflowSettings(settings)
		act.SetDisabledDuringExecution(false)
		if action.ClosePage {
			setRawBSONField(act, "ClosePage", true)
		}
		return act, nil
	},
	"nanoflow": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		nfID, err := pb.resolveNanoflowByName(action.Target)
		if err != nil {
			return nil, mdlerrors.NewBackend("resolve nanoflow", err)
		}
		_ = nfID

		act := genPg.NewCallNanoflowClientAction()
		assignFreshID(act)
		act.SetNanoflowQualifiedName(action.Target)

		for _, arg := range action.Args {
			nm := genPg.NewNanoflowParameterMapping()
			assignFreshID(nm)
			// Use the fully-qualified "Module.NanoflowName.ParamName" form. A bare
			// param name leaves Mendix unable to resolve the Parameter reference and
			// makes `mx check` crash with "Parameter property ... null". Mirrors the
			// microflow path above.
			// TODO: CE0115 nanoflow arg matching still broken; needs Studio Pro BSON sample
			nm.SetParameterQualifiedName(action.Target + "." + arg.Name)

			if strVal, ok := arg.Value.(string); ok {
				if strings.HasPrefix(strVal, "$") {
					pv := genPg.NewPageVariable()
					assignFreshID(pv)
					pv.SetPageParameterQualifiedName(strVal)
					nm.SetVariable(pv)
				} else {
					nm.SetExpression(strVal)
				}
			}
			act.AddParameterMappings(nm)
		}
		act.SetDisabledDuringExecution(false)
		return act, nil
	},
	"openLink": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		act := genPg.NewOpenLinkClientAction()
		assignFreshID(act)
		act.SetLinkType("Web")
		addr := genPg.NewStaticOrDynamicString()
		assignFreshID(addr)
		addr.SetValue(action.LinkURL)
		act.SetAddress(addr)
		return act, nil
	},
	"signOut": func(pb *pageBuilder, _ *ast.ActionV3) (element.Element, error) {
		act := genPg.NewSignOutClientAction()
		assignFreshID(act)
		return act, nil
	},
	"completeTask": func(pb *pageBuilder, action *ast.ActionV3) (element.Element, error) {
		act := genPg.NewSetTaskOutcomeClientAction()
		assignFreshID(act)
		act.SetClosePage(true)
		act.SetCommit(true)
		act.SetOutcomeValue(action.OutcomeValue)
		return act, nil
	},
}

// ActionBuilders returns the action builder map (exported for tests).
func ActionBuilders() map[string]actionBuilderFn {
	return actionBuilders
}
