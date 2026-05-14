// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.3.g5a — gen-typed ImportFromMapping + ExportToMapping adders.
//
// Two XML/JSON marshaling adders shipped here:
//
//   addImportFromMappingActionGen
//     → genMf.ImportXmlAction
//        ├── XmlDocumentVariableName = source string variable
//        └── ResultHandling
//             ├── OutputVariableName  = result variable
//             └── ImportMappingCall   = mapping reference + cardinality
//
//   addExportToMappingActionGen
//     → genMf.ExportXmlAction
//        ├── MappingQualifiedName        = mapping reference
//        └── MappingArgumentVariableName = source entity variable
//
// gen-vs-legacy schema notes:
//
//   - gen folds the four legacy ResultHandling sub-types
//     (ResultHandlingMapping / ResultHandlingString /
//     ResultHandlingHttpResponse / ResultHandlingNone) into a single
//     `*ResultHandling` element with a top-level `ResultHandlingType`
//     enum on the parent action. For Import the always-mapping case
//     simply attaches an ImportMappingCall sub-element; the discriminator
//     is implicit (the gen describer uses presence of ImportMappingCall
//     to recognise the mapping shape).
//
//   - gen ExportXmlAction has flat `MappingQualifiedName` +
//     `MappingArgumentVariableName` setters — no nested
//     RequestHandling element wrapper like the legacy SDK uses.
//
// Backend-driven cardinality inference (legacy reads import mapping
// metadata from `fb.backend.GetImportMappingByQualifiedName` to flip
// SingleObject false when the JsonStructure root is Array or
// Element[0].MaxOccurs > 1) is **deferred** to commit j (the
// dispatcher commit) where backend resolution is wired in. The
// offline write path defaults to ForceSingleOccurrence=true so the
// most common single-object shape works without metadata.

package executor

import (
	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
)

// addImportFromMappingActionGen emits a `[$Y = ]import from mapping
// Mod.Map($SourceVar);` activity. Builds the nested ResultHandling →
// ImportMappingCall structure that gen requires.
func (fb *flowBuilderGen) addImportFromMappingActionGen(s *ast.ImportFromMappingStmt) element.ID {
	mappingQN := s.Mapping.Module + "." + s.Mapping.Name

	call := genMf.NewImportMappingCall()
	assignFreshID(call)
	call.SetMappingQualifiedName(mappingQN)
	// Default to single-object until backend metadata says otherwise.
	// Backend-driven cardinality inference is wired in commit j; the
	// describer's `formatImportXmlActionGen` emits a single-object
	// surface that matches this default.
	call.SetForceSingleOccurrence(true)

	rh := genMf.NewResultHandling()
	assignFreshID(rh)
	rh.SetOutputVariableName(s.OutputVariable)
	rh.SetImportMappingCall(call)

	action := genMf.NewImportXmlAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetXmlDocumentVariableName(s.SourceVariable)
	action.SetResultHandling(rh)

	return fb.genActivityWrap(action, s.ErrorHandling, s.OutputVariable)
}

// addExportToMappingActionGen emits a `$Y = export to mapping
// Mod.Map($SourceVar);` activity. gen `ExportXmlAction` exposes
// flat setters for the mapping reference + source variable, so no
// nested wrapper element is needed.
func (fb *flowBuilderGen) addExportToMappingActionGen(s *ast.ExportToMappingStmt) element.ID {
	action := genMf.NewExportXmlAction()
	assignFreshID(action)
	action.SetErrorHandlingType(fb.ehTypeGen(s.ErrorHandling))
	action.SetMappingQualifiedName(s.Mapping.Module + "." + s.Mapping.Name)
	action.SetMappingArgumentVariableName(s.SourceVariable)
	return fb.genActivityWrap(action, s.ErrorHandling, "")
}
