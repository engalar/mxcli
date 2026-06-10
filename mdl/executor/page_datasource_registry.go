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

// dataSourceBuilderFn 构建 datasource gen element，返回元素和已解析的 entity 限定名（可为空）。
type dataSourceBuilderFn func(pb *pageBuilder, ds *ast.DataSourceV3) (element.Element, string, error)

// dataSourceBuilders maps each DataSourceV3.Type string to its builder.
var dataSourceBuilders = map[string]dataSourceBuilderFn{
	"parameter": func(pb *pageBuilder, ds *ast.DataSourceV3) (element.Element, string, error) {
		paramName := strings.TrimPrefix(ds.Reference, "$")
		entityID, ok := pb.paramScope[paramName]
		entityName := pb.paramEntityNames[paramName]
		if !ok {
			entityID, ok = pb.paramScope["$"+paramName]
			entityName = pb.paramEntityNames["$"+paramName]
		}
		if !ok {
			return nil, "", mdlerrors.NewNotFound("parameter", ds.Reference)
		}

		if entityName == "" {
			var err error
			entityName, err = pb.getEntityNameByID(entityID)
			if err != nil {
				log.Printf("warning: could not resolve entity name for ID %s: %v", entityID, err)
			}
		}

		dvs := genPg.NewDataViewSource()
		assignFreshID(dvs)
		dvs.SetForceFullObjects(false)
		if pb.isSnippet {
			dvs.SetSnippetParameterQualifiedName(paramName)
		} else {
			// SP11.6.6: use SourceVariable (nested PageVariable) instead of flat PageParameter
			sv := genPg.NewPageVariable()
			assignFreshID(sv)
			sv.SetPageParameterQualifiedName(paramName)
			dvs.SetSourceVariable(sv)
		}
		if entityName != "" {
			// Set entity ref for type awareness
			ref := genDm.NewDirectEntityRef()
			assignFreshID(ref)
			ref.SetEntityQualifiedName(entityName)
			dvs.SetEntityRef(ref)
		}
		return dvs, entityName, nil
	},

	"database": func(pb *pageBuilder, ds *ast.DataSourceV3) (element.Element, string, error) {
		entityID, err := pb.resolveEntity(ast.QualifiedName{
			Module: pb.extractModule(ds.Reference),
			Name:   pb.extractName(ds.Reference),
		})
		if err != nil {
			return nil, "", mdlerrors.NewBackend("resolve entity", err)
		}
		_ = entityID

		// DataView database source → Forms$DataViewSource with EntityRef
		dvs := genPg.NewDataViewSource()
		assignFreshID(dvs)
		dvs.SetEntityPath(ds.Reference)
		ref := genDm.NewDirectEntityRef()
		assignFreshID(ref)
		ref.SetEntityQualifiedName(ds.Reference)
		dvs.SetEntityRef(ref)
		return dvs, ds.Reference, nil
	},

	"microflow": func(pb *pageBuilder, ds *ast.DataSourceV3) (element.Element, string, error) {
		mfID, err := pb.resolveMicroflow(ds.Reference)
		if err != nil {
			return nil, "", mdlerrors.NewBackend("resolve microflow", err)
		}
		_ = mfID

		entityName := pb.getMicroflowReturnEntityName(ds.Reference)

		ms := genPg.NewMicroflowSource()
		assignFreshID(ms)
		settings := genPg.NewMicroflowSettings()
		assignFreshID(settings)
		settings.SetMicroflowQualifiedName(ds.Reference)
		ms.SetMicroflowSettings(settings)
		return ms, entityName, nil
	},

	"nanoflow": func(pb *pageBuilder, ds *ast.DataSourceV3) (element.Element, string, error) {
		return pb.buildNanoflowSourceGen(ds)
	},

	"association": func(pb *pageBuilder, ds *ast.DataSourceV3) (element.Element, string, error) {
		ctxVar := ds.ContextVariable
		if ctxVar == "currentObject" {
			ctxVar = ""
		}

		path := ds.Reference
		destEntity := ""
		if idx := strings.Index(path, "/"); idx >= 0 {
			destEntity = path[idx+1:]
			path = path[:idx]
		} else {
			destEntity = pb.resolveAssociationDestination(path, pb.entityContext)
		}

		as := genPg.NewAssociationSource()
		assignFreshID(as)
		as.SetEntityPath(path + "/" + destEntity)
		if ctxVar != "" {
			// SourceVariable is a PageVariable element in gen
			pv := genPg.NewPageVariable()
			assignFreshID(pv)
			pv.SetPageParameterQualifiedName(ctxVar)
			as.SetSourceVariable(pv)
		}
		return as, destEntity, nil
	},

	"selection": func(pb *pageBuilder, ds *ast.DataSourceV3) (element.Element, string, error) {
		widgetName := ds.Reference
		widgetID, ok := pb.widgetScope[widgetName]
		if !ok {
			return nil, "", mdlerrors.NewNotFound("widget", widgetName)
		}
		_ = widgetID

		entityName := pb.paramEntityNames[widgetName]

		lts := genPg.NewListenTargetSource()
		assignFreshID(lts)
		lts.SetListenTarget(widgetName)
		return lts, entityName, nil
	},
}
