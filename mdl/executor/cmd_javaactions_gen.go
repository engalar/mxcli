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
//    namespaces, and fall back to raw-BSON reads (codec.ReadBSONFieldString,
//    codec.DecodeChild) instead of type-asserting to gen concrete types.
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
//    child via codec.DecodeChild and dispatch on the inner TypeName.
//
// 5. MicroflowActionInfo.Icon: gen exposes `IconQualifiedName()` which
//    pulls from the BSON "Icon" key. Legacy stored Icon as a free string;
//    Studio Pro newer versions store as ByNameRef. Both round-trip via
//    the same key, so IconQualifiedName() returns the right value either
//    way. The gen `MicroflowActionInfo` has no `ImageData` accessor
//    (the legacy field for embedded toolbox icons) — read via
//    codec.ReadBSONFieldString(mai.Raw(), "ImageData") if needed.

package executor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
)

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
		if name := readBSONString(elem, "Entity"); name != "" {
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
		if entity := readBSONString(elem, "Entity"); entity != "" {
			return "List of " + entity
		}
		if inner, err := codec.DecodeChild(elem.Raw(), "Parameter"); err == nil {
			return "List of " + formatJavaActionTypeGen(inner, typeParams)
		}
		return "List"
	case "CodeActions$EnumerationType", "JavaActions$EnumerationType":
		if et, ok := elem.(*genJA.EnumerationType); ok && et.EnumerationQualifiedName() != "" {
			return "Enum " + et.EnumerationQualifiedName()
		}
		if name := readBSONString(elem, "Enumeration"); name != "" {
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
		if name := readBSONString(elem, "TypeParameter"); name != "" {
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
		if grammar := readBSONString(elem, "Grammar"); grammar != "" {
			return "StringTemplate(" + grammar + ")"
		}
		return "StringTemplate"
	case "CodeActions$BasicParameterType", "JavaActions$BasicParameterType":
		// BasicParameterType wraps the actual type in a "Type" child.
		// Decode the child and dispatch recursively.
		if inner, err := codec.DecodeChild(elem.Raw(), "Type"); err == nil && inner != nil {
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
	if name := readBSONString(elem, "TypeParameter"); name != "" {
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
	if name := readBSONString(elem, "TypeParameter"); name != "" {
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
			if name := readBSONString(tp, "Name"); name != "" {
				return name
			}
		}
	}
	return ""
}

// readBSONString is a non-erroring wrapper around codec.ReadBSONFieldString
// that returns "" on missing/error. The polymorphic-dispatch helpers above
// don't care about the distinction between "field absent" and "field
// present but empty"; both render as the no-name fallback path.
func readBSONString(elem element.Element, key string) string {
	if elem == nil {
		return ""
	}
	raw := elem.Raw()
	if raw == nil {
		return ""
	}
	if s, err := codec.ReadBSONFieldString(raw, key); err == nil {
		return s
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

	sb.WriteString("create java action ")
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
		caption := readBSONString(mai, "Caption")
		category := readBSONString(mai, "Category")
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
				icon = readBSONString(mai, "Icon")
			}
			if icon != "" {
				sb.WriteString("\n-- icon: ")
				sb.WriteString(icon)
			}
		}
	}

	if javaCode := readJavaActionUserCode(ctx.MprPath, name.Module, name.Name); javaCode != "" {
		sb.WriteString("\nas $$\n")
		sb.WriteString(javaCode)
		sb.WriteString("\n$$")
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
	jsa, err := ctx.JavaScriptActions.FindByQualifiedName(qn)
	if err != nil || jsa == nil {
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

	sb.WriteString("create javascript action ")
	sb.WriteString(qn)

	// Type parameters list (generics).
	typeParams := jsa.ActionTypeParametersItems()
	if len(typeParams) > 0 {
		names := make([]string, 0, len(typeParams))
		for _, tp := range typeParams {
			if typed, ok := tp.(*genJA.TypeParameter); ok && typed.Name() != "" {
				names = append(names, typed.Name())
			} else if tn := readBSONString(tp, "Name"); tn != "" {
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

	platform := jsa.Platform()
	if platform == "" {
		platform = "All"
	}
	if platform != "All" {
		sb.WriteString("\n  PLATFORM ")
		sb.WriteString(platform)
	}

	if rt := jsa.ActionReturnType(); rt != nil {
		sb.WriteString("\n  returns ")
		sb.WriteString(formatJavaActionReturnTypeGen(rt, typeParams))
	}

	if rn := jsa.ActionDefaultReturnName(); rn != "" {
		sb.WriteString("\n-- return NAME: '")
		sb.WriteString(rn)
		sb.WriteString("'")
	}

	// EXPOSED AS clause — JS uses ModelerActionInfo (no MicroflowActionInfo
	// alias in the gen JavaScriptAction surface).
	if mai := jsa.ModelerActionInfo(); mai != nil {
		caption := readBSONString(mai, "Caption")
		category := readBSONString(mai, "Category")
		if caption != "" {
			sb.WriteString("\n  exposed as '")
			sb.WriteString(caption)
			sb.WriteString("' in '")
			sb.WriteString(category)
			sb.WriteString("'")
			icon := ""
			if typed, ok := mai.(*genJA.MicroflowActionInfo); ok {
				icon = typed.IconQualifiedName()
			}
			if icon == "" {
				icon = readBSONString(mai, "Icon")
			}
			if icon != "" {
				sb.WriteString("\n-- icon: ")
				sb.WriteString(icon)
			}
		}
	}

	userCode, extraCode := readJavaScriptActionSource(ctx.MprPath, name.Module, name.Name)
	if userCode != "" {
		sb.WriteString("\nas $$\n")
		sb.WriteString(userCode)
		sb.WriteString("\n$$")
	}

	sb.WriteString(";")
	fmt.Fprintln(ctx.Output, sb.String())

	if el := jsa.ExportLevel(); el != "" && el != "Hidden" {
		fmt.Fprintf(ctx.Output, "-- export level: %s\n", el)
	}
	if jsa.Excluded() {
		fmt.Fprintln(ctx.Output, "-- EXCLUDED: true")
	}
	if platform == "All" {
		fmt.Fprintln(ctx.Output, "-- PLATFORM: All")
	}
	if extraCode != "" {
		fmt.Fprintln(ctx.Output, "-- EXTRA CODE:")
		for _, line := range strings.Split(extraCode, "\n") {
			fmt.Fprintf(ctx.Output, "-- %s\n", line)
		}
	}
	return nil
}
