// SPDX-License-Identifier: Apache-2.0

// Stage 3.3.2.A1-A5 — gen-typed read paths for Java/JavaScript actions.
//
// Mirrors cmd_javaactions.go and cmd_javascript_actions.go, but reads from
// ctx.JavaActions / ctx.JavaScriptActions (gen-typed) instead of going
// through the legacy backend ListJavaActions / ReadJavaActionByName
// methods that return *sdk/javaactions.JavaAction.
//
// ─── Schema gaps (cf. memory project_gen_schema_gaps) ───
//
// 1. Storage namespace mismatch: Studio Pro emits "CodeActions$X" type
//    names in BSON for JavaAction parameter / return types (Boolean,
//    Integer, String, ConcreteEntity, ListType, EnumerationType, …).
//    The gen registry registers them as "JavaActions$X". Therefore the
//    decoder returns `*element.Base` for legacy fixture data — typed
//    structs (genJA.BooleanType, etc.) only appear when WE create new
//    elements. Strategy: dispatch on `elem.TypeName()` matching BOTH
//    namespaces, and fall back to raw-BSON reads (genJA.ReadBSONString,
//    genJA.DecodeChildElement — thin codec wrappers in the gen package
//    so executor code stays off modelsdk/codec directly) instead of
//    type-asserting to gen concrete types.
//
// 2. Missing gen types (no constructor exists; `*element.Base` is what
//    we get back at decode time):
//      • CodeActions$VoidType (return-only)
//      • CodeActions$LongType
//      • CodeActions$FileDocumentType
//      • CodeActions$StringTemplateParameterType
//      • CodeActions$MicroflowType (gen has only Microflow*ParameterType)
//      • CodeActions$NanoflowType (sibling JavaScriptActions$Nanoflow…
//        is the canonical JS-side spelling)
//    Resolution: TypeName() string switch handles the render. Remove the
//    cases when codegen adds these types (no fix-hint upstream yet).
//
// 3. Dual accessor families on JavaAction itself: the gen JavaAction has
//    BOTH ParametersItems()/ParametersAdd…/ParametersRemove… (legacy
//    "Parameters" BSON key) AND ActionParametersItems()/AddActionParameters
//    (newer "ActionParameters" key); same flip for return type
//    (JavaReturnType vs ActionReturnType). Studio-Pro-emitted fixtures
//    use the LEGACY keys exclusively. Strategy: prefer the legacy
//    accessor when non-empty, fall back to the action one. This is
//    the inverse of plan §3 R1's hypothesis (which assumed the
//    Action* names were canonical) — verified empirically against the
//    expr-checker fixture (2026-05-14).
//
// 4. JavaActionParameter.ParameterType returns a CodeActions$BasicParameterType
//    wrapper that carries the inner type in its "Type" child element.
//    Legacy parser_javaactions.go peels this in parseInnerParameterType.
//    We do the same: when outer == BasicParameterType, decode the "Type"
//    child via genJA.DecodeChildElement and dispatch on the inner TypeName.
//
// 5. MicroflowActionInfo.Icon: gen exposes `IconQualifiedName()` which
//    pulls from the BSON "Icon" key. Legacy stored Icon as a free string;
//    Studio Pro newer versions store as ByNameRef. Both round-trip via
//    the same key, so IconQualifiedName() returns the right value either
//    way. The gen `MicroflowActionInfo` has no `ImageData` accessor
//    (the legacy field for embedded toolbox icons) — read via
//    genJA.ReadBSONString(mai, "ImageData") if needed.

package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genCA "github.com/mendixlabs/mxcli/modelsdk/gen/codeactions"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
)

// readJavaScriptActionSource extracts user code and extra code from the
// project's javascriptsource/<module>/actions/<action>.js file.
// Returns (userCode, extraCode, sourcePath) where sourcePath is the canonical
// relative path even when the file is absent, so callers can show it to the user.
func readJavaScriptActionSource(mprPath, moduleName, actionName string) (userCode, extraCode, sourcePath string) {
	relPath := filepath.Join("javascriptsource", moduleName, "actions", actionName+".js")
	if mprPath == "" {
		return "", "", relPath
	}
	projectRoot := filepath.Dir(mprPath)
	jsPath := filepath.Join(projectRoot, relPath)
	content, err := os.ReadFile(jsPath)
	if err != nil {
		lowerPath := filepath.Join("javascriptsource", strings.ToLower(moduleName), "actions", actionName+".js")
		jsPath = filepath.Join(projectRoot, lowerPath)
		content, err = os.ReadFile(jsPath)
		if err != nil {
			return "", "", relPath
		}
		relPath = lowerPath
	}
	sourcePath = relPath
	source := string(content)

	if beginIdx := strings.Index(source, "// BEGIN USER CODE"); beginIdx != -1 {
		if endIdx := strings.Index(source, "// END USER CODE"); endIdx != -1 && endIdx > beginIdx {
			uc := source[beginIdx+len("// BEGIN USER CODE") : endIdx]
			uc = strings.TrimPrefix(uc, "\n")
			uc = strings.TrimSuffix(uc, "\n")
			uc = strings.TrimRight(uc, " \t")
			userCode = uc
		}
	}
	if beginIdx := strings.Index(source, "// BEGIN EXTRA CODE"); beginIdx != -1 {
		if endIdx := strings.Index(source, "// END EXTRA CODE"); endIdx != -1 && endIdx > beginIdx {
			ec := source[beginIdx+len("// BEGIN EXTRA CODE") : endIdx]
			ec = strings.TrimSpace(ec)
			if ec != "" {
				extraCode = ec
			}
		}
	}
	return userCode, extraCode, sourcePath
}

// writeJavaScriptActionSource writes or overwrites the JS action source file at
// javascriptsource/<lowercase-module>/actions/<ActionName>.js.
func writeJavaScriptActionSource(mprPath, moduleName, actionName, imports, extraCode, userCode string) error {
	if mprPath == "" {
		return fmt.Errorf("writeJavaScriptActionSource: mprPath is empty")
	}
	projectRoot := filepath.Dir(mprPath)
	modLower := strings.ToLower(moduleName)
	dir := filepath.Join(projectRoot, "javascriptsource", modLower, "actions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create js action dir: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("// This file was generated by Mendix Studio Pro.\n")
	sb.WriteString("//\n")
	sb.WriteString("// WARNING: Only the following code will be retained when actions are regenerated:\n")
	sb.WriteString("// - the import list\n")
	sb.WriteString("// - the code between BEGIN USER CODE and END USER CODE\n")
	sb.WriteString("// - the code between BEGIN EXTRA CODE and END EXTRA CODE\n")
	sb.WriteString("// Other code you write will be lost the next time you deploy the project.\n")
	sb.WriteString("import { Big } from \"big.js\";\n")
	if imports != "" {
		for _, line := range strings.Split(imports, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || line == `import { Big } from "big.js";` {
				continue
			}
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")
	sb.WriteString("// BEGIN EXTRA CODE\n")
	if extraCode != "" {
		sb.WriteString(extraCode)
		if !strings.HasSuffix(extraCode, "\n") {
			sb.WriteString("\n")
		}
	}
	sb.WriteString("// END EXTRA CODE\n")
	sb.WriteString("\n")
	sb.WriteString("export async function ")
	sb.WriteString(actionName)
	sb.WriteString("() {\n")
	sb.WriteString("\t// BEGIN USER CODE\n")
	if userCode != "" {
		for _, line := range strings.Split(userCode, "\n") {
			sb.WriteString("\t")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\t// END USER CODE\n")
	sb.WriteString("}\n")

	jsPath := filepath.Join(dir, actionName+".js")
	return os.WriteFile(jsPath, []byte(sb.String()), 0644)
}

// ─────────────────────────────────────────────────────────────────────
// A1 — listJavaActionsGen
// ─────────────────────────────────────────────────────────────────────

// listJavaActionsGen handles SHOW JAVA ACTIONS using gen-typed JavaAction
// units from listJavaActionsWithContainerGen. Mirrors listJavaActions in
// output shape; only the type source changes.
func listJavaActionsGen(ctx *ExecContext, moduleName string) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listJavaActionsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list java actions", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		folderPath    string
	}
	var rows []row
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(modelIDFromElementID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if moduleName != "" && modName != moduleName {
			continue
		}
		qn := modName + "." + p.Elem.Name()
		folder := h.BuildFolderPath(modelIDFromElementID(p.ContainerID))
		rows = append(rows, row{qn, modName, p.Elem.Name(), folder})
	}
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Folder"},
		Summary: fmt.Sprintf("(%d java actions)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.folderPath})
	}
	return writeResult(ctx, result)
}

// modelIDFromElementID converts element.ID to model.ID; both are
// "type ID string" aliases but Go treats them as distinct named types.
// Helper lives here so cmd_javaactions_gen.go does not need to leak the
// conversion at every container-resolve site.
func modelIDFromElementID(id element.ID) model.ID {
	return model.ID(id)
}

// ─────────────────────────────────────────────────────────────────────
// A2 — formatJavaActionTypeGen + formatJavaActionReturnTypeGen
// ─────────────────────────────────────────────────────────────────────

// formatJavaActionTypeGen dispatches on elem.TypeName() to render the MDL
// type syntax for a gen-typed Java action parameter or return type.
//
// `typeParams` is the action's type-parameter def list (from
// ActionTypeParametersItems / TypeParametersItems) — needed to resolve
// EntityTypeParameterType.TypeParameterRefID and
// ParameterizedEntityType.TypeParameterRefID back to the displayed name
// (gen does not store the resolved name on the use site, only the BY_ID
// pointer).
//
// Accepts both "CodeActions$X" (Studio-Pro-emitted; primary) and
// "JavaActions$X" (gen-emitted; for elements created in this session)
// storage namespaces. See the schema-gap note at the top of this file.
func formatJavaActionTypeGen(elem element.Element, typeParams []element.Element) string {
	if elem == nil {
		return "Object"
	}
	switch elem.TypeName() {
	case "CodeActions$VoidType":
		return "Void"
	case "CodeActions$BooleanType", "JavaActions$BooleanType":
		return "Boolean"
	case "CodeActions$IntegerType", "JavaActions$IntegerType":
		return "Integer"
	case "CodeActions$LongType":
		return "Long"
	case "CodeActions$DecimalType", "JavaActions$DecimalType":
		return "Decimal"
	case "CodeActions$StringType", "JavaActions$StringType":
		return "String"
	case "CodeActions$DateTimeType", "JavaActions$DateTimeType":
		return "DateTime"
	case "CodeActions$FileDocumentType":
		return "FileDocument"
	case "CodeActions$ConcreteEntityType", "CodeActions$EntityType",
		"JavaActions$ConcreteEntityType":
		// Both legacy ("Entity") and gen ("Entity" via ByNameRef on
		// ConcreteEntityType.entity) read the same BSON key.
		if name := genJA.ReadBSONString(elem, "Entity"); name != "" {
			return name
		}
		// Gen-typed ConcreteEntityType exposes the qualified name
		// directly when populated through normal accessors.
		if et, ok := elem.(*genJA.ConcreteEntityType); ok && et.EntityQualifiedName() != "" {
			return et.EntityQualifiedName()
		}
		return "Object"
	case "CodeActions$ListType", "JavaActions$ListType":
		// Gen ListType wraps inner Parameter (sub-Element); legacy
		// ListType has flat "Entity" key. Try both.
		if lt, ok := elem.(*genJA.ListType); ok {
			if inner := lt.Parameter(); inner != nil {
				return "List of " + formatJavaActionTypeGen(inner, typeParams)
			}
		}
		// Raw fallback: try direct "Entity" (Studio-Pro-flat form),
		// then nested "Parameter.Entity".
		if entity := genJA.ReadBSONString(elem, "Entity"); entity != "" {
			return "List of " + entity
		}
		if inner := genJA.DecodeChildElement(elem, "Parameter"); inner != nil {
			return "List of " + formatJavaActionTypeGen(inner, typeParams)
		}
		return "List"
	case "CodeActions$EnumerationType", "JavaActions$EnumerationType":
		if et, ok := elem.(*genJA.EnumerationType); ok && et.EnumerationQualifiedName() != "" {
			return "Enum " + et.EnumerationQualifiedName()
		}
		if name := genJA.ReadBSONString(elem, "Enumeration"); name != "" {
			return "Enum " + name
		}
		return "Enumeration"
	case "CodeActions$EntityTypeParameterType", "JavaActions$EntityTypeParameterType":
		if name := resolveTypeParamNameFromEntityTypeParameterType(elem, typeParams); name != "" {
			return "entity <" + name + ">"
		}
		return "entity <>"
	case "CodeActions$ParameterizedEntityType", "JavaActions$ParameterizedEntityType":
		if name := resolveTypeParamNameFromParameterizedEntityType(elem, typeParams); name != "" {
			return name
		}
		return "T"
	case "CodeActions$TypeParameter", "JavaActions$TypeParameter":
		// Legacy "TypeParameter" used as a use-site has the name in
		// the "TypeParameter" string field; gen TypeParameter is a
		// def (Name() accessor). Try gen first, then raw.
		if tp, ok := elem.(*genJA.TypeParameter); ok && tp.Name() != "" {
			return tp.Name()
		}
		if name := genJA.ReadBSONString(elem, "TypeParameter"); name != "" {
			return name
		}
		return "T"
	case "CodeActions$MicroflowType", "JavaActions$MicroflowJavaActionParameterType",
		"JavaActions$MicroflowParameterType":
		return "Microflow"
	case "CodeActions$NanoflowType",
		"JavaScriptActions$NanoflowJavaScriptActionParameterType":
		return "Nanoflow"
	case "JavaScriptActions$MicroflowJavaScriptActionParameterType":
		return "Microflow"
	case "CodeActions$StringTemplateParameterType":
		if grammar := genJA.ReadBSONString(elem, "Grammar"); grammar != "" {
			return "StringTemplate(" + grammar + ")"
		}
		return "StringTemplate"
	case "CodeActions$BasicParameterType", "JavaActions$BasicParameterType":
		// BasicParameterType wraps the actual type in a "Type" child.
		// Decode the child and dispatch recursively.
		if inner := genJA.DecodeChildElement(elem, "Type"); inner != nil {
			return formatJavaActionTypeGen(inner, typeParams)
		}
		return "Object"
	default:
		// Unknown type — strip the namespace + "Type" suffix as a
		// best-effort display so we never panic on encountering a
		// new schema variant.
		n := elem.TypeName()
		if i := strings.Index(n, "$"); i >= 0 {
			n = n[i+1:]
		}
		n = strings.TrimSuffix(n, "Type")
		if n == "" {
			return "Object"
		}
		return n
	}
}

// formatJavaActionReturnTypeGen renders a return-type element. Identical
// dispatch as formatJavaActionTypeGen but treats nil as "Void" instead of
// "Object" because that's the convention for methods with no return.
func formatJavaActionReturnTypeGen(elem element.Element, typeParams []element.Element) string {
	if elem == nil {
		return "Void"
	}
	return formatJavaActionTypeGen(elem, typeParams)
}

// resolveTypeParamNameFromEntityTypeParameterType reads the
// TypeParameterPointer ID from elem and looks it up in the type-param
// def list. Works for both gen-typed and raw-fallback elements.
func resolveTypeParamNameFromEntityTypeParameterType(elem element.Element, typeParams []element.Element) string {
	if etp, ok := elem.(*genJA.EntityTypeParameterType); ok {
		if name := lookupTypeParamName(etp.TypeParameterRefID(), typeParams); name != "" {
			return name
		}
	}
	// Raw fallback: read the binary ID via the bsonutil helper would
	// require importing private decode logic; instead, fall back to
	// reading "TypeParameter" as a string (older Studio Pro variant)
	// or returning empty.
	if name := genJA.ReadBSONString(elem, "TypeParameter"); name != "" {
		return name
	}
	return ""
}

// resolveTypeParamNameFromParameterizedEntityType is the twin for
// ParameterizedEntityType (use-site referencing a type-param def by ID).
func resolveTypeParamNameFromParameterizedEntityType(elem element.Element, typeParams []element.Element) string {
	if pt, ok := elem.(*genJA.ParameterizedEntityType); ok {
		if name := lookupTypeParamName(pt.TypeParameterRefID(), typeParams); name != "" {
			return name
		}
	}
	if name := genJA.ReadBSONString(elem, "TypeParameter"); name != "" {
		return name
	}
	return ""
}

// lookupTypeParamName scans the def list for a TypeParameter whose ID
// matches `id`; returns its Name(). Empty string if not found or if id
// is empty.
func lookupTypeParamName(id element.ID, typeParams []element.Element) string {
	if id == "" {
		return ""
	}
	for _, tp := range typeParams {
		if tp == nil {
			continue
		}
		if typed, ok := tp.(*genJA.TypeParameter); ok && typed.ID() == id {
			return typed.Name()
		}
		// Unregistered raw element — match on raw $ID + read Name field.
		if tp.ID() == id {
			if name := genJA.ReadBSONString(tp, "Name"); name != "" {
				return name
			}
		}
	}
	return ""
}

// javaActionParametersOf returns the parameter list, preferring the
// legacy ParametersItems (BSON key "Parameters" — what Studio Pro emits
// for existing elements) over the newer ActionParametersItems
// (BSON key "ActionParameters"). See schema-gap note (3) at top of file.
func javaActionParametersOf(ja *genJA.JavaAction) []element.Element {
	if ja == nil {
		return nil
	}
	if items := ja.ParametersItems(); len(items) > 0 {
		return items
	}
	return ja.ActionParametersItems()
}

// javaActionTypeParametersOf is the type-parameter twin of
// javaActionParametersOf. Same precedence rule.
func javaActionTypeParametersOf(ja *genJA.JavaAction) []element.Element {
	if ja == nil {
		return nil
	}
	if items := ja.TypeParametersItems(); len(items) > 0 {
		return items
	}
	return ja.ActionTypeParametersItems()
}

// javaActionReturnTypeElement returns the return-type sub-element,
// preferring JavaReturnType (legacy "JavaReturnType" BSON key) over
// ActionReturnType (newer "ActionReturnType"). See schema-gap note (3).
func javaActionReturnTypeElement(ja *genJA.JavaAction) element.Element {
	if ja == nil {
		return nil
	}
	if rt := ja.JavaReturnType(); rt != nil {
		return rt
	}
	return ja.ActionReturnType()
}

// javaScriptActionReturnTypeElement mirrors javaActionReturnTypeElement for
// JavaScript actions. Studio Pro stores the return type under the legacy
// "JavaReturnType" BSON key (same as Java actions), but genJSA.JavaScriptAction
// only exposes "ActionReturnType" via its typed getter — which is absent in
// Studio-Pro-emitted BSON and therefore always nil. We raw-read the legacy key
// via genJA.DecodeChildElement so that void/Nothing and typed returns are not lost.
func javaScriptActionReturnTypeElement(jsa *genJSA.JavaScriptAction) element.Element {
	if jsa == nil {
		return nil
	}
	// Prefer the legacy JavaReturnType field written by Studio Pro.
	if rt := genJA.DecodeChildElement(jsa, "JavaReturnType"); rt != nil {
		return rt
	}
	// Fall back to ActionReturnType set by our own write path.
	return jsa.ActionReturnType()
}

// javaActionParameterParameterType returns the parameter's type sub-
// element, preferring ParameterType (legacy "ParameterType" BSON key —
// often a BasicParameterType wrapper) over JavaType / ActionParameterType
// (newer keys). The format helper above peels BasicParameterType when
// it gets back an outer wrapper.
func javaActionParameterParameterType(p *genJA.JavaActionParameter) element.Element {
	if p == nil {
		return nil
	}
	if pt := p.ParameterType(); pt != nil {
		return pt
	}
	if pt := p.JavaType(); pt != nil {
		return pt
	}
	return p.ActionParameterType()
}

// javaActionMicroflowActionInfoOf returns the modeler/action info sub-
// element, preferring MicroflowActionInfo (legacy) over ModelerActionInfo
// (newer) for Studio-Pro-fixture compatibility.
func javaActionMicroflowActionInfoOf(ja *genJA.JavaAction) element.Element {
	if ja == nil {
		return nil
	}
	if mai := ja.MicroflowActionInfo(); mai != nil {
		return mai
	}
	return ja.ModelerActionInfo()
}

// ─────────────────────────────────────────────────────────────────────
// A3 — describeJavaActionGen
// ─────────────────────────────────────────────────────────────────────

// describeJavaActionGen handles DESCRIBE JAVA ACTION using gen-typed
// data. Mirrors the legacy describeJavaAction output format byte-for-byte
// where possible; differences are noted inline.
func describeJavaActionGen(ctx *ExecContext, name ast.QualifiedName) error {
	if ctx == nil || ctx.JavaActions == nil {
		return mdlerrors.NewNotFound("java action", name.Module+"."+name.Name)
	}
	qn := name.Module + "." + name.Name
	ja, err := ctx.JavaActions.FindByQualifiedName(qn)
	if err != nil || ja == nil {
		return mdlerrors.NewNotFound("java action", qn)
	}

	var sb strings.Builder

	// Documentation comment (JavaDoc style).
	doc := strings.ReplaceAll(ja.Documentation(), "\r\n", "\n")
	doc = strings.ReplaceAll(doc, "\r", "\n")
	if doc != "" {
		sb.WriteString("/**\n")
		for _, line := range strings.Split(doc, "\n") {
			sb.WriteString(" * ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sb.WriteString(" */\n")
	}

	sb.WriteString("create or modify java action ")
	sb.WriteString(qn)
	sb.WriteString("(")

	typeParams := javaActionTypeParametersOf(ja)
	params := javaActionParametersOf(ja)

	hasDescriptions := false
	for _, p := range params {
		pp, ok := p.(*genJA.JavaActionParameter)
		if !ok {
			continue
		}
		if pp.Description() != "" {
			hasDescriptions = true
			break
		}
	}

	wrote := 0
	for _, p := range params {
		pp, ok := p.(*genJA.JavaActionParameter)
		if !ok {
			continue
		}
		if wrote > 0 {
			sb.WriteString(", ")
		}
		if hasDescriptions {
			sb.WriteString("\n    ")
		}
		sb.WriteString(pp.Name())
		sb.WriteString(": ")
		sb.WriteString(formatJavaActionTypeGen(javaActionParameterParameterType(pp), typeParams))
		if pp.IsRequired() {
			sb.WriteString(" not null")
		}
		if pp.Description() != "" {
			pd := strings.ReplaceAll(pp.Description(), "\r\n", "\n")
			pd = strings.ReplaceAll(pd, "\r", "\n")
			firstLine, _, _ := strings.Cut(pd, "\n")
			sb.WriteString("  -- ")
			sb.WriteString(firstLine)
		}
		wrote++
	}
	if hasDescriptions {
		sb.WriteString("\n")
	}
	sb.WriteString(")")

	rt := javaActionReturnTypeElement(ja)
	if rt != nil {
		sb.WriteString(" returns ")
		sb.WriteString(formatJavaActionReturnTypeGen(rt, typeParams))
	}

	if rn := ja.ActionDefaultReturnName(); rn != "" {
		sb.WriteString("\n-- return NAME: '")
		sb.WriteString(rn)
		sb.WriteString("'")
	}

	if mai := javaActionMicroflowActionInfoOf(ja); mai != nil {
		caption := genJA.ReadBSONString(mai, "Caption")
		category := genJA.ReadBSONString(mai, "Category")
		if caption != "" {
			sb.WriteString("\nexposed as '")
			sb.WriteString(caption)
			sb.WriteString("' in '")
			sb.WriteString(category)
			sb.WriteString("'")
			// Icon: prefer gen IconQualifiedName(), fall back to raw.
			icon := ""
			if typed, ok := mai.(*genJA.MicroflowActionInfo); ok {
				icon = typed.IconQualifiedName()
			}
			if icon == "" {
				icon = genJA.ReadBSONString(mai, "Icon")
			}
			if icon != "" {
				sb.WriteString("\n-- icon: ")
				sb.WriteString(icon)
			}
		}
	}

	userCode, allImports, extraCode := readJavaActionSource(ctx.MprPath, name.Module, name.Name)
	if len(allImports) > 0 {
		sb.WriteString("\nimports $$\n")
		for _, imp := range allImports {
			sb.WriteString("    ")
			sb.WriteString(imp)
			sb.WriteString("\n")
		}
		sb.WriteString("$$")
	}
	if userCode != "" {
		sb.WriteString("\ncode $$\n")
		for _, line := range strings.Split(userCode, "\n") {
			sb.WriteString("    ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sb.WriteString("$$")
	}
	if extraCode != "" {
		sb.WriteString("\nextra $$\n")
		for _, line := range strings.Split(extraCode, "\n") {
			sb.WriteString("    ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sb.WriteString("$$")
	}

	sb.WriteString(";")
	fmt.Fprintln(ctx.Output, sb.String())

	if el := ja.ExportLevel(); el != "" && el != "Hidden" {
		fmt.Fprintf(ctx.Output, "-- export level: %s\n", el)
	}
	if ja.Excluded() {
		fmt.Fprintln(ctx.Output, "-- EXCLUDED: true")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────
// A4 — listJavaScriptActionsGen
// ─────────────────────────────────────────────────────────────────────

// listJavaScriptActionsGen handles SHOW JAVASCRIPT ACTIONS using gen-typed
// JavaScriptAction units. Renders Platform alongside the standard
// qualified-name/module/name/folder columns; unset Platform renders as
// "All" (legacy convention).
func listJavaScriptActionsGen(ctx *ExecContext, moduleName string) error {
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listJavaScriptActionsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list javascript actions", err)
	}

	type row struct {
		qualifiedName string
		module        string
		name          string
		platform      string
		folderPath    string
	}
	var rows []row
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(modelIDFromElementID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if moduleName != "" && modName != moduleName {
			continue
		}
		qn := modName + "." + p.Elem.Name()
		folder := h.BuildFolderPath(modelIDFromElementID(p.ContainerID))
		platform := p.Elem.Platform()
		if platform == "" {
			platform = "All"
		}
		rows = append(rows, row{qn, modName, p.Elem.Name(), platform, folder})
	}
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].qualifiedName) < strings.ToLower(rows[j].qualifiedName)
	})

	result := &TableResult{
		Columns: []string{"Qualified Name", "Module", "Name", "Platform", "Folder"},
		Summary: fmt.Sprintf("(%d javascript actions)", len(rows)),
	}
	for _, r := range rows {
		result.Rows = append(result.Rows, []any{r.qualifiedName, r.module, r.name, r.platform, r.folderPath})
	}
	return writeResult(ctx, result)
}

// ─────────────────────────────────────────────────────────────────────
// A5 — describeJavaScriptActionGen
// ─────────────────────────────────────────────────────────────────────

// describeJavaScriptActionGen handles DESCRIBE JAVASCRIPT ACTION using
// gen-typed data. JavaScriptAction shares the CodeActions$ type tree with
// JavaAction (return / parameter types), so we reuse formatJavaAction…
// helpers. JS-specific bits: no MicroflowActionInfo (uses ModelerActionInfo
// in gen, or none in legacy fixtures), Platform renders as a `PLATFORM`
// clause, and the source body comes from javascriptsource/<module>/actions.
func describeJavaScriptActionGen(ctx *ExecContext, name ast.QualifiedName) error {
	if ctx == nil || ctx.JavaScriptActions == nil {
		return mdlerrors.NewNotFound("javascript action", name.Module+"."+name.Name)
	}
	qn := name.Module + "." + name.Name

	// Container-aware lookup yields both the element and its folder path.
	pairs, err := listJavaScriptActionsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list javascript actions", err)
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}
	var jsa *genJSA.JavaScriptAction
	var folderPath string
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(modelIDFromElementID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName+"."+p.Elem.Name() == qn {
			jsa = p.Elem
			folderPath = h.BuildFolderPath(modelIDFromElementID(p.ContainerID))
			break
		}
	}
	if jsa == nil {
		return mdlerrors.NewNotFound("javascript action", qn)
	}

	var sb strings.Builder

	// Documentation.
	doc := strings.ReplaceAll(jsa.Documentation(), "\r\n", "\n")
	doc = strings.ReplaceAll(doc, "\r", "\n")
	if doc != "" {
		sb.WriteString("/**\n")
		for _, line := range strings.Split(doc, "\n") {
			sb.WriteString(" * ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		sb.WriteString(" */\n")
	}

	sb.WriteString("create or modify javascript action ")
	sb.WriteString(qn)

	// Type parameters list (generics).
	typeParams := jsa.ActionTypeParametersItems()
	if len(typeParams) > 0 {
		names := make([]string, 0, len(typeParams))
		for _, tp := range typeParams {
			if typed, ok := tp.(*genJA.TypeParameter); ok && typed.Name() != "" {
				names = append(names, typed.Name())
			} else if tn := genJA.ReadBSONString(tp, "Name"); tn != "" {
				names = append(names, tn)
			}
		}
		if len(names) > 0 {
			sb.WriteString("<")
			sb.WriteString(strings.Join(names, ", "))
			sb.WriteString(">")
		}
	}
	sb.WriteString("(")

	params := jsa.ActionParametersItems()
	hasDescriptions := false
	for _, p := range params {
		pp, ok := p.(*genJSA.JavaScriptActionParameter)
		if !ok {
			continue
		}
		if pp.Description() != "" {
			hasDescriptions = true
			break
		}
	}

	wrote := 0
	for _, p := range params {
		pp, ok := p.(*genJSA.JavaScriptActionParameter)
		if !ok {
			continue
		}
		if wrote > 0 {
			sb.WriteString(", ")
		}
		if hasDescriptions {
			sb.WriteString("\n    ")
		}
		sb.WriteString(pp.Name())
		sb.WriteString(": ")
		sb.WriteString(formatJavaActionTypeGen(pp.ActionParameterType(), typeParams))
		if pp.IsRequired() {
			sb.WriteString(" not null")
		}
		if pp.Description() != "" {
			pd := strings.ReplaceAll(pp.Description(), "\r\n", "\n")
			pd = strings.ReplaceAll(pd, "\r", "\n")
			firstLine, _, _ := strings.Cut(pd, "\n")
			sb.WriteString("  -- ")
			sb.WriteString(firstLine)
		}
		wrote++
	}
	if hasDescriptions {
		sb.WriteString("\n")
	}
	sb.WriteString(")")

	sb.WriteString("\n  returns ")
	sb.WriteString(formatJavaActionReturnTypeGen(javaScriptActionReturnTypeElement(jsa), typeParams))

	platform := jsa.Platform()
	if platform == "" {
		platform = "All"
	}
	sb.WriteString("\n  PLATFORM '")
	sb.WriteString(platform)
	sb.WriteString("'")

	if folderPath != "" {
		sb.WriteString("\n  folder '")
		sb.WriteString(folderPath)
		sb.WriteString("'")
	}

	// EXPOSED AS clause — JS uses ModelerActionInfo (no MicroflowActionInfo
	// alias in the gen JavaScriptAction surface).
	if mai := jsa.ModelerActionInfo(); mai != nil {
		caption := genJA.ReadBSONString(mai, "Caption")
		category := genJA.ReadBSONString(mai, "Category")
		if caption != "" {
			sb.WriteString("\n  exposed as '")
			sb.WriteString(caption)
			sb.WriteString("' in '")
			sb.WriteString(category)
			sb.WriteString("'")
		}
	}

	// Code body block: { imports $$ $$ extra $$ $$ code $$ $$ }
	userCode, extraCode, _ := readJavaScriptActionSource(ctx.MprPath, name.Module, name.Name)
	importsStr := readJavaScriptActionImports(ctx.MprPath, name.Module, name.Name)

	sb.WriteString("\n{")
	if importsStr != "" {
		sb.WriteString("\nimports $$\n")
		sb.WriteString(importsStr)
		sb.WriteString("\n$$")
	}
	if extraCode != "" {
		sb.WriteString("\nextra $$\n")
		sb.WriteString(extraCode)
		sb.WriteString("\n$$")
	}
	if userCode != "" {
		sb.WriteString("\ncode $$\n")
		sb.WriteString(userCode)
		sb.WriteString("\n$$")
	}
	sb.WriteString("\n}")

	sb.WriteString(";")
	fmt.Fprintln(ctx.Output, sb.String())

	if el := jsa.ExportLevel(); el != "" && el != "Hidden" {
		fmt.Fprintf(ctx.Output, "-- export level: %s\n", el)
	}
	if jsa.Excluded() {
		fmt.Fprintln(ctx.Output, "-- EXCLUDED: true")
	}
	if rn := jsa.ActionDefaultReturnName(); rn != "" {
		fmt.Fprintf(ctx.Output, "-- return NAME: '%s'\n", rn)
	}
	return nil
}

// readJavaScriptActionImports reads all `import ...` lines from the JS
// action source file, joined by newline. Returns "" when the file is
// absent. Used by describe to populate the `imports $$ $$` block.
func readJavaScriptActionImports(mprPath, moduleName, actionName string) string {
	if mprPath == "" {
		return ""
	}
	projectRoot := filepath.Dir(mprPath)
	candidates := []string{
		filepath.Join(projectRoot, "javascriptsource", moduleName, "actions", actionName+".js"),
		filepath.Join(projectRoot, "javascriptsource", strings.ToLower(moduleName), "actions", actionName+".js"),
	}
	for _, jsPath := range candidates {
		content, err := os.ReadFile(jsPath)
		if err != nil {
			continue
		}
		var importLines []string
		for _, line := range strings.Split(string(content), "\n") {
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "import ") {
				importLines = append(importLines, t)
			}
		}
		return strings.Join(importLines, "\n")
	}
	return ""
}

// ─────────────────────────────────────────────────────────────────────
// D3 — execDropJavaActionGen
// ─────────────────────────────────────────────────────────────────────

// execDropJavaActionGen handles DROP JAVA ACTION using gen-typed reads
// from listJavaActionsWithContainerGen and the gen-aware repo path.
// Mirrors execDropJavaAction (cmd_javaactions.go:249) but consumes
// container UUIDs from the cache helper rather than from a sdk-typed
// ContainerID field that gen objects don't carry.
func execDropJavaActionGen(ctx *ExecContext, s *ast.DropJavaActionStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}
	pairs, err := listJavaActionsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list java actions", err)
	}
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName != s.Name.Module || p.Elem.Name() != s.Name.Name {
			continue
		}
		// Phase D Delete is not yet wired through the gen repo (Phase
		// D5 fills the writer); for now route through the legacy
		// backend Delete which targets the same MPR Unit row by ID.
		if err := ctx.Backend.DeleteJavaAction(model.ID(p.Elem.ID())); err != nil {
			return mdlerrors.NewBackend("delete java action", err)
		}
		if err := ctx.Backend.DeleteJavaSourceFile(modName, p.Elem.Name()); err != nil {
			return mdlerrors.NewBackend("delete java source file", err)
		}
		invalidateJavaActionsCache(ctx)
		fmt.Fprintf(ctx.Output, "Dropped java action: %s.%s\n", s.Name.Module, s.Name.Name)
		return nil
	}
	return mdlerrors.NewNotFound("java action", s.Name.Module+"."+s.Name.Name)
}

// ─────────────────────────────────────────────────────────────────────
// A6 — execCreateJavaScriptAction
// ─────────────────────────────────────────────────────────────────────

// execCreateJavaScriptAction handles CREATE [OR MODIFY] JAVASCRIPT ACTION.
// Phase 1: the action must already exist in BSON (Studio Pro creates the
// unit + its parameter/return BSON); this handler updates the Platform
// field and (re)writes the javascriptsource/<module>/actions/<Name>.js
// source file from the supplied imports/extra/code blocks.
func execCreateJavaScriptAction(ctx *ExecContext, s *ast.CreateJavaScriptActionStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}
	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}
	pairs, err := listJavaScriptActionsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list javascript actions", err)
	}
	var existingJSA *genJSA.JavaScriptAction
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(modelIDFromElementID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && p.Elem.Name() == s.Name.Name {
			existingJSA = p.Elem
			break
		}
	}
	if existingJSA == nil {
		qn := s.Name.Module + "." + s.Name.Name
		return mdlerrors.NewNotFoundMsg("javascript action", qn,
			fmt.Sprintf("javascript action %s not found in project — create it in Mendix Studio Pro first", qn))
	}
	if s.Platform != "" {
		existingJSA.SetPlatform(s.Platform)
		if err := ctx.Backend.UpdateJavaScriptActionGen(existingJSA); err != nil {
			return mdlerrors.NewBackend("update javascript action platform", err)
		}
	}
	if s.Imports != "" || s.ExtraCode != "" || s.UserCode != "" {
		if err := writeJavaScriptActionSource(ctx.MprPath, s.Name.Module, s.Name.Name,
			s.Imports, s.ExtraCode, s.UserCode); err != nil {
			return mdlerrors.NewBackend("write javascript source file", err)
		}
		invalidateJavaScriptActionsCache(ctx)
	}
	fmt.Fprintf(ctx.Output, "updated javascript action %s.%s\n", s.Name.Module, s.Name.Name)
	return nil
}

// newGenElementByType allocates a bare *element.Base with the given
// storage TypeName. Used as a stub for gen schema gaps (LongType,
// VoidType, etc.) until codegen ships dedicated constructors. The
// resulting element has no properties wired so it can only carry its
// $Type marker on roundtrip — fine for JavaAction parameter / return
// types where the inner shape is empty (no nested fields).
//
// schema gap: Remove call sites once gen ships New<Name>Type for the
// missing CodeActions$ variants.
func newGenElementByType(name string) element.Element {
	b := &element.Base{}
	b.SetTypeName(name)
	return b
}

// ─────────────────────────────────────────────────────────────────────
// D1 — AST → gen JavaAction param/return type converters
// ─────────────────────────────────────────────────────────────────────

// astDataTypeToJavaActionParamTypeGen mirrors astDataTypeToJavaActionParamType
// (cmd_javaactions.go:455) but returns a gen element.Element instead of an
// sdk-typed CodeActionParameterType. typeParamIDs resolves type-parameter
// names (e.g., "pEntity") to the TypeParameter unit IDs created in the
// same execCreateJavaActionGen call so EntityTypeParameterType can carry
// the BY_ID pointer.
//
// schema gap: gen has no constructor for several CodeActions$ variants
// (LongType, FileDocumentType). Phase D code emits a plain *element.Base
// for those by allocating via element.New for the CodeActions$ namespace
// (Studio Pro's storage convention). The format helpers already dispatch
// on TypeName() for both namespaces (see schema-gap note (2) at top of
// file). Remove these element.New fallbacks when codegen ships them.
func astDataTypeToJavaActionParamTypeGen(dt ast.DataType, typeParamIDs map[string]element.ID) element.Element {
	switch dt.Kind {
	case ast.TypeBoolean:
		return genCA.NewBooleanType()
	case ast.TypeInteger:
		return genCA.NewIntegerType()
	case ast.TypeLong:
		return newGenElementByType("CodeActions$LongType")
	case ast.TypeDecimal:
		return genCA.NewDecimalType()
	case ast.TypeString:
		return genCA.NewStringType()
	case ast.TypeDateTime, ast.TypeDate:
		return genCA.NewDateTimeType()
	case ast.TypeEntityTypeParam:
		etp := genCA.NewEntityTypeParameterType()
		if id, ok := typeParamIDs[dt.TypeParamName]; ok {
			etp.SetTypeParameterID(id)
		}
		return etp
	case ast.TypeEntity, ast.TypeEnumeration:
		// Check first if this is a bare unqualified name that matches a type parameter
		// (e.g. InputObject: T where T is a declared type parameter).
		if dt.EnumRef != nil && dt.EnumRef.Module == "" {
			if id, ok := typeParamIDs[dt.EnumRef.Name]; ok {
				pe := genCA.NewParameterizedEntityType()
				pe.SetTypeParameterID(id)
				return pe
			}
		}
		// TypeEnumeration with a qualified name is treated as entity
		// type here; the visitor cannot distinguish entity from
		// enumeration types for bare qualified names like
		// Module.EntityName (see CLAUDE.md).
		entityName := ""
		if dt.EntityRef != nil {
			entityName = dt.EntityRef.Module + "." + dt.EntityRef.Name
		} else if dt.EnumRef != nil {
			entityName = dt.EnumRef.Module + "." + dt.EnumRef.Name
		}
		et := genCA.NewConcreteEntityType()
		et.SetEntityQualifiedName(entityName)
		return et
	case ast.TypeListOf:
		entityName := ""
		if dt.EntityRef != nil {
			entityName = dt.EntityRef.Module + "." + dt.EntityRef.Name
		}
		inner := genCA.NewConcreteEntityType()
		inner.SetEntityQualifiedName(entityName)
		lt := genCA.NewListType()
		lt.SetParameter(inner)
		return lt
	default:
		return genJA.NewStringType()
	}
}

// astDataTypeToJavaActionReturnTypeGen converts an AST DataType to a gen
// element.Element for use as a Java action return type. Twin of
// astDataTypeToJavaActionParamTypeGen with TypeVoid → CodeActions$VoidType
// fallback (gen has no NewVoidType()).
//
// "void" sometimes parses as a bare TypeEnumeration with a single-part
// EnumRef name (e.g., {Module: "", Name: "void"}); we detect that
// pattern and emit VoidType to match the legacy converter behaviour.
func astDataTypeToJavaActionReturnTypeGen(dt ast.DataType, typeParamIDs map[string]element.ID) element.Element {
	if dt.Kind == ast.TypeVoid {
		// schema gap: no NewVoidType() in gen. Allocate a bare
		// *element.Base with the CodeActions$VoidType storage
		// namespace. Remove when gen ships NewVoidType().
		return newGenElementByType("CodeActions$VoidType")
	}
	// Detect "void" parsed as a bare TypeEnumeration / TypeEntity name.
	if dt.Kind == ast.TypeEntity || dt.Kind == ast.TypeEnumeration {
		entityName := ""
		if dt.EntityRef != nil {
			entityName = dt.EntityRef.Module + "." + dt.EntityRef.Name
		} else if dt.EnumRef != nil {
			entityName = dt.EnumRef.Module + "." + dt.EnumRef.Name
		}
		if strings.EqualFold(strings.TrimPrefix(entityName, "."), "void") {
			return newGenElementByType("CodeActions$VoidType")
		}
	}
	return astDataTypeToJavaActionParamTypeGen(dt, typeParamIDs)
}

// ─────────────────────────────────────────────────────────────────────
// D2 — execCreateJavaActionGen
// ─────────────────────────────────────────────────────────────────────

// execCreateJavaActionGen handles CREATE JAVA ACTION via the gen-typed
// write path. Mirrors execCreateJavaAction (cmd_javaactions.go:285) but:
//
//   - Existence check goes through listJavaActionsWithContainerGen
//     (cache-aware, gen-typed) instead of ctx.Backend.ListJavaActions
//     (sdk-typed).
//   - The JavaAction object is built via genJA.New* constructors and
//     SetX accessors. Parameter / return types come from the D1
//     converters (astDataTypeToJavaActionParamTypeGen /
//     astDataTypeToJavaActionReturnTypeGen).
//   - Persistence routes through ctx.Backend.{CreateJavaActionGen,
//     UpdateJavaActionGen} (the gen-aware sibling of CreateJavaAction).
//   - Java source file IO routes through ctx.Backend.WriteJavaSourceFileGen.
//     This currently returns "not implemented (Phase D6)"; we tolerate
//     that error so the BSON unit is still created — D6 will replace
//     the stub with a real generator and source files will start
//     materialising automatically.
//
// schema gap: per the dual-accessor note (3) at top of file, Studio
// Pro fixtures populate the legacy Parameters/ReturnType BSON keys.
// For NEW elements emitted by the gen path, we use the canonical
// ActionParameters / ActionReturnType setters; downstream readers
// already dispatch on both flavours via javaActionParametersOf /
// javaActionReturnTypeElement (no schema-gap mismatch on roundtrip).
func execCreateJavaActionGen(ctx *ExecContext, s *ast.CreateJavaActionStmt) error {
	if !ctx.ConnectedForWrite() {
		return mdlerrors.NewNotConnectedWrite()
	}

	h, err := getHierarchy(ctx)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	// Find the module (no auto-create — keeps parity with legacy
	// execCreateJavaAction behaviour: missing module → NotFound).
	modules, err := ctx.Backend.ListModules()
	if err != nil {
		return mdlerrors.NewBackend("get modules", err)
	}
	var (
		containerID model.ID
		moduleName  string
	)
	for _, mod := range modules {
		if mod.Name == s.Name.Module {
			containerID = mod.ID
			moduleName = mod.Name
			break
		}
	}
	if containerID == "" {
		return mdlerrors.NewNotFound("module", s.Name.Module)
	}

	// Existence check via the cache helper. The helper resolves each
	// JavaAction's container UUID so we can match by module name.
	pairs, err := listJavaActionsWithContainerGen(ctx)
	if err != nil {
		return mdlerrors.NewBackend("list java actions", err)
	}
	var existingJA *genJA.JavaAction
	for _, p := range pairs {
		if p.Elem == nil {
			continue
		}
		modID := h.FindModuleID(model.ID(p.ContainerID))
		modName := h.GetModuleName(modID)
		if modName == s.Name.Module && p.Elem.Name() == s.Name.Name {
			if !s.CreateOrModify {
				return mdlerrors.NewAlreadyExists("java action", s.Name.Module+"."+s.Name.Name)
			}
			existingJA = p.Elem
			break
		}
	}

	// Build the gen JavaAction.
	ja := genJA.NewJavaAction()
	if existingJA != nil {
		// Reuse existing element ID so refs and dirty bits don't
		// shift across replace.
		ja.SetID(existingJA.ID())
	} else {
		ja.SetID(element.ID(types.GenerateID()))
	}
	ja.SetName(s.Name.Name)
	ja.SetDocumentation(s.Documentation)
	ja.SetExportLevel("Public")
	// Mendix Studio Pro always writes Excluded and ActionDefaultReturnName.
	// Omitting them causes MprTool to crash when sorting the project tree.
	ja.SetExcluded(false)
	ja.SetActionDefaultReturnName("ReturnValueName")

	// Build type parameter definitions using the old TypeParameters list (not ActionTypeParameters).
	typeParamIDs := make(map[string]element.ID, len(s.TypeParameters))
	for _, tpName := range s.TypeParameters {
		tp := genJA.NewTypeParameter()
		tp.SetID(element.ID(types.GenerateID()))
		tp.SetName(tpName)
		ja.AddTypeParameters(tp)
		typeParamIDs[tpName] = tp.ID()
	}
	// Always write TypeParameters: [2] even when empty — Mendix expects this field.
	ja.ForceWriteTypeParameters()

	// Set up type-param name lookup for bare-name parameter types
	// (mirrors legacy isTypeParamRef helper).
	typeParamNames := make(map[string]bool, len(s.TypeParameters))
	for _, tpName := range s.TypeParameters {
		typeParamNames[tpName] = true
	}

	// Convert parameters using the old Mendix-native format:
	// Parameters (not ActionParameters), ParameterType with CodeActions$BasicParameterType
	// wrapper (not ActionParameterType with bare type), version marker 2 on the list.
	// This matches what Mendix Studio Pro writes and avoids an MprTool crash
	// (InvalidCastException) when opening the project.
	for _, param := range s.Parameters {
		jaParam := genJA.NewJavaActionParameter()
		jaParam.SetID(element.ID(types.GenerateID()))
		jaParam.SetCategory("")
		jaParam.SetDescription("")
		jaParam.SetName(param.Name)
		jaParam.SetIsRequired(param.IsRequired)

		var innerType element.Element
		switch {
		case param.Type.Kind == ast.TypeEntityTypeParam:
			innerType = astDataTypeToJavaActionParamTypeGen(param.Type, typeParamIDs)
		case isTypeParamRef(param.Type, typeParamNames):
			tpName := getTypeParamRefName(param.Type)
			pet := genCA.NewParameterizedEntityType()
			if id, ok := typeParamIDs[tpName]; ok {
				pet.SetTypeParameterID(id)
			}
			innerType = pet
		default:
			innerType = astDataTypeToJavaActionParamTypeGen(param.Type, typeParamIDs)
		}
		// Wrap in CodeActions$BasicParameterType (Mendix native format).
		bpt := genCA.NewBasicParameterType()
		bpt.SetID(element.ID(types.GenerateID()))
		bpt.SetType(innerType)
		jaParam.SetParameterType(bpt)
		ja.AddParameters(jaParam)
	}

	// Convert return type using the old JavaReturnType field (not ActionReturnType).
	if isTypeParamRef(s.ReturnType, typeParamNames) {
		tpName := getTypeParamRefName(s.ReturnType)
		pet := genCA.NewParameterizedEntityType()
		if id, ok := typeParamIDs[tpName]; ok {
			pet.SetTypeParameterID(id)
		}
		ja.SetJavaReturnType(pet)
	} else {
		ja.SetJavaReturnType(astDataTypeToJavaActionReturnTypeGen(s.ReturnType, typeParamIDs))
	}

	// MicroflowActionInfo when EXPOSED AS clause is present (use CodeActions$ type, old format).
	if s.ExposedCaption != "" {
		mai := genCA.NewMicroflowActionInfo()
		mai.SetCaption(s.ExposedCaption)
		mai.SetCategory(s.ExposedCategory)
		ja.SetMicroflowActionInfo(mai)
	}

	// Persist via the gen-aware backend siblings.
	if existingJA != nil {
		if err := ctx.Backend.UpdateJavaActionGen(ja); err != nil {
			return mdlerrors.NewBackend("update java action", err)
		}
	} else {
		if err := ctx.Backend.CreateJavaActionGen(string(containerID), "Documents", ja); err != nil {
			return mdlerrors.NewBackend("create java action", err)
		}
	}

	// Java source file. ctx.Backend.WriteJavaSourceFileGen currently
	// returns "not implemented (Phase D6)"; tolerate that so the BSON
	// unit is still created. D6 fills in the real generator and source
	// files start materialising automatically.
	//
	// TODO(Stage 3.3.2.D6): drop the "not implemented" fallback once
	// writeJavaSourceFileViaPathGen lands a real generator.
	if s.JavaCode != "" || len(s.Imports) > 0 || s.ExtraCode != "" {
		// Guard: code block must contain only the method body, not the
		// executeAction signature. The skeleton already wraps the body
		// in `public X executeAction() throws Exception { ... }`.
		if strings.Contains(s.JavaCode, "executeAction") {
			return mdlerrors.NewValidationf(
				"code block must contain only the method body, not the method signature\n" +
					"  hint: write only the statements inside executeAction(), e.g.:\n" +
					"    code $$\n" +
					"      return this.Input.trim();\n" +
					"    $$")
		}
		params := make([]*genJA.JavaActionParameter, 0, len(ja.ActionParametersItems()))
		for _, p := range ja.ActionParametersItems() {
			if pp, ok := p.(*genJA.JavaActionParameter); ok {
				params = append(params, pp)
			}
		}
		if werr := ctx.Backend.WriteJavaSourceFileGen(moduleName, s.Name.Name, s.JavaCode, params, ja.ActionReturnType(), s.Imports, s.ExtraCode); werr != nil {
			if !strings.Contains(werr.Error(), "not implemented") {
				return mdlerrors.NewBackend("write java source file", werr)
			}
			// Fall through: BSON unit still created; source file
			// writer is Phase D6's responsibility.
		} else if len(s.Imports) > 0 {
			fmt.Fprintf(ctx.Output, "note: %d import(s) merged into %s.java\n", len(s.Imports), s.Name.Name)
		}
	}

	invalidateJavaActionsCache(ctx)
	invalidateHierarchy(ctx)

	if existingJA != nil {
		fmt.Fprintf(ctx.Output, "Modified java action: %s.%s\n", s.Name.Module, s.Name.Name)
	} else {
		fmt.Fprintf(ctx.Output, "Created java action: %s.%s\n", s.Name.Module, s.Name.Name)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────
// E2 — helpers extracted from the deleted cmd_javaactions.go
// ─────────────────────────────────────────────────────────────────────

// readJavaActionUserCode reads the Java source file and extracts the
// user code section between BEGIN USER CODE / END USER CODE markers.
// Mirror of the helper that lived in the now-deleted cmd_javaactions.go.
func readJavaActionUserCode(mprPath, moduleName, actionName string) string {
	if mprPath == "" {
		return ""
	}
	projectRoot := filepath.Dir(mprPath)
	moduleNameLower := strings.ToLower(moduleName)
	javaPath := filepath.Join(projectRoot, "javasource", moduleNameLower, "actions", actionName+".java")
	content, err := os.ReadFile(javaPath)
	if err != nil {
		return ""
	}
	source := string(content)
	beginMarker := "// begin user CODE"
	endMarker := "// end user CODE"
	beginIdx := strings.Index(source, beginMarker)
	endIdx := strings.Index(source, endMarker)
	if beginIdx == -1 || endIdx == -1 || endIdx <= beginIdx {
		return ""
	}
	userCode := source[beginIdx+len(beginMarker) : endIdx]
	userCode = strings.TrimPrefix(userCode, "\n")
	userCode = strings.TrimSuffix(userCode, "\n")
	userCode = strings.TrimRight(userCode, " \t")
	return userCode
}

// readJavaActionSource reads all three user-editable sections from a .java file:
// userCode (BEGIN/END USER CODE), imports (all import lines), extraCode
// (BEGIN/END EXTRA CODE). Returns empty values when the file is absent or
// markers are not found. Uses uppercase markers that match what
// generateJavaSourceGen writes.
func readJavaActionSource(mprPath, moduleName, actionName string) (userCode string, imports []string, extraCode string) {
	if mprPath == "" {
		return
	}
	projectRoot := filepath.Dir(mprPath)
	moduleNameLower := strings.ToLower(moduleName)
	javaPath := filepath.Join(projectRoot, "javasource", moduleNameLower, "actions", actionName+".java")
	content, err := os.ReadFile(javaPath)
	if err != nil {
		return
	}
	src := string(content)

	if bi := strings.Index(src, "// BEGIN USER CODE"); bi != -1 {
		if ei := strings.Index(src, "// END USER CODE"); ei != -1 && ei > bi {
			uc := src[bi+len("// BEGIN USER CODE") : ei]
			uc = strings.TrimPrefix(uc, "\n")
			uc = strings.TrimSuffix(uc, "\n")
			userCode = strings.TrimRight(uc, " \t")
		}
	}

	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "import ") && strings.HasSuffix(t, ";") {
			imports = append(imports, t)
		}
	}

	if bi := strings.Index(src, "// BEGIN EXTRA CODE"); bi != -1 {
		if ei := strings.Index(src, "// END EXTRA CODE"); ei != -1 && ei > bi {
			ec := src[bi+len("// BEGIN EXTRA CODE") : ei]
			ec = strings.TrimSpace(ec)
			if ec != "" {
				extraCode = ec
			}
		}
	}
	return
}

// isTypeParamRef checks whether an ast.DataType refers to a type
// parameter by name. Used by the AST→gen converters when resolving
// EntityTypeParameterType references against an action's type parameter list.
func isTypeParamRef(dt ast.DataType, typeParamNames map[string]bool) bool {
	name := getTypeParamRefName(dt)
	return name != "" && typeParamNames[name]
}

// getTypeParamRefName extracts the name from a DataType that could be a
// type parameter reference. Returns empty string if not a type-param ref.
func getTypeParamRefName(dt ast.DataType) string {
	switch dt.Kind {
	case ast.TypeEnumeration:
		if dt.EnumRef != nil && dt.EnumRef.Module == "" {
			return dt.EnumRef.Name
		}
		if dt.EnumRef != nil {
			return dt.EnumRef.Module + "." + dt.EnumRef.Name
		}
	case ast.TypeEntity:
		if dt.EntityRef != nil && dt.EntityRef.Module == "" {
			return dt.EntityRef.Name
		}
		if dt.EntityRef != nil {
			return dt.EntityRef.Module + "." + dt.EntityRef.Name
		}
	}
	return ""
}
