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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genCA "github.com/mendixlabs/mxcli/modelsdk/gen/codeactions"
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
	return listJavaActionsGenFn(ctx, moduleName, ctx.Deps)
}

// listJavaActionsGenFn is the HandlerDeps version of listJavaActionsGen.
func listJavaActionsGenFn(ctx context.Context, moduleName string, deps *HandlerDeps) error {
	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listJavaActionsWithContainerGenDeps(deps)
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
	return writeResultTo(deps.Output, deps.Format, result)
}

// modelIDFromElementID converts element.ID to model.ID; both are
// "type ID string" aliases but Go treats them as distinct named types.
// Helper lives here so cmd_javaactions_gen.go does not need to leak the
// conversion at every container-resolve site.
func modelIDFromElementID(id element.ID) model.ID {
	return model.ID(id)
}

// ─────────────────────────────────────────────────────────────────────
// A3 — describeJavaActionGen
// ─────────────────────────────────────────────────────────────────────

// describeJavaActionGen handles DESCRIBE JAVA ACTION using gen-typed
// data. Mirrors the legacy describeJavaAction output format byte-for-byte
// where possible; differences are noted inline.
func describeJavaActionGen(ctx *ExecContext, name ast.QualifiedName) error {
	return describeJavaActionGenFn(ctx, name, ctx.Deps)
}

// describeJavaActionGenFn is the HandlerDeps version of describeJavaActionGen.
func describeJavaActionGenFn(ctx context.Context, name ast.QualifiedName, deps *HandlerDeps) error {
	if deps == nil || deps.JavaActionRepo == nil {
		return mdlerrors.NewNotFound("java action", name.Module+"."+name.Name)
	}
	qn := name.Module + "." + name.Name
	ja, err := deps.JavaActionRepo.FindByQualifiedName(qn)
	if err != nil || ja == nil {
		return mdlerrors.NewNotFound("java action", qn)
	}

	var sb strings.Builder

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

	userCode, allImports, extraCode := readJavaActionSource(deps.MprPath, name.Module, name.Name)
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
	fmt.Fprintln(deps.Output, sb.String())

	if el := ja.ExportLevel(); el != "" && el != "Hidden" {
		fmt.Fprintf(deps.Output, "-- export level: %s\n", el)
	}
	if ja.Excluded() {
		fmt.Fprintln(deps.Output, "-- EXCLUDED: true")
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
	return listJavaScriptActionsGenFn(ctx, moduleName, ctx.Deps)
}

// listJavaScriptActionsGenFn is the HandlerDeps version of listJavaScriptActionsGen.
func listJavaScriptActionsGenFn(ctx context.Context, moduleName string, deps *HandlerDeps) error {
	h, err := GetOrBuildHierarchy(deps)
	if err != nil {
		return mdlerrors.NewBackend("build hierarchy", err)
	}

	pairs, err := listJavaScriptActionsWithContainerGenDeps(deps)
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
	return writeResultTo(deps.Output, deps.Format, result)
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
	return describeJavaScriptActionGenFn(ctx, name, ctx.Deps)
}

// describeJavaScriptActionGenFn is the HandlerDeps version of describeJavaScriptActionGen.
func describeJavaScriptActionGenFn(ctx context.Context, name ast.QualifiedName, deps *HandlerDeps) error {
	if deps == nil || deps.JavaScriptActionRepo == nil {
		return mdlerrors.NewNotFound("javascript action", name.Module+"."+name.Name)
	}
	qn := name.Module + "." + name.Name

	pairs, err := listJavaScriptActionsWithContainerGenDeps(deps)
	if err != nil {
		return mdlerrors.NewBackend("list javascript actions", err)
	}
	h, err := GetOrBuildHierarchy(deps)
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

	params := jsa.ParametersItems()
	if len(params) == 0 {
		params = jsa.ActionParametersItems()
	}
	for i, p := range params {
		pp, ok := p.(*genJSA.JavaScriptActionParameter)
		if !ok {
			continue
		}
		if i > 0 {
			sb.WriteString(", ")
		}
		typeElem := pp.ActionParameterType()
		if typeElem != nil {
			if bpt, bptOK := typeElem.(*genCA.BasicParameterType); bptOK {
				typeElem = bpt.Type()
			}
		}
		sb.WriteString(pp.Name())
		sb.WriteString(": ")
		sb.WriteString(formatJavaActionTypeGen(typeElem, typeParams))
		if pp.IsRequired() {
			sb.WriteString(" not null")
		}
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

	userCode, extraCode, sourcePath := readJavaScriptActionSource(deps.MprPath, name.Module, name.Name)
	importsStr := readJavaScriptActionImports(deps.MprPath, name.Module, name.Name)

	sourceExists := false
	if deps.MprPath != "" {
		projectRoot := filepath.Dir(deps.MprPath)
		primary := filepath.Join(projectRoot, "javascriptsource", name.Module, "actions", name.Name+".js")
		if _, err := os.Stat(primary); err == nil {
			sourceExists = true
		} else {
			fallback := filepath.Join(projectRoot, "javascriptsource", strings.ToLower(name.Module), "actions", name.Name+".js")
			if _, err := os.Stat(fallback); err == nil {
				sourceExists = true
			}
		}
	}

	sb.WriteString("\n{")
	if sourceExists {
		sb.WriteString("\nimports $$")
		if importsStr != "" {
			sb.WriteString("\n")
			sb.WriteString(importsStr)
		}
		sb.WriteString("\n$$")

		sb.WriteString("\nextra $$")
		if extraCode != "" {
			sb.WriteString("\n")
			sb.WriteString(extraCode)
		}
		sb.WriteString("\n$$")

		sb.WriteString("\ncode $$")
		if userCode != "" {
			sb.WriteString("\n")
			sb.WriteString(userCode)
		}
		sb.WriteString("\n$$")
	}
	sb.WriteString("\n}")

	sb.WriteString(";")
	fmt.Fprintln(deps.Output, sb.String())

	fmt.Fprintf(deps.Output, "-- source: %s", sourcePath)
	if !sourceExists {
		fmt.Fprintf(deps.Output, " (NOT FOUND)")
	}
	fmt.Fprintln(deps.Output)

	if el := jsa.ExportLevel(); el != "" && el != "Hidden" {
		fmt.Fprintf(deps.Output, "-- export level: %s\n", el)
	}
	if jsa.Excluded() {
		fmt.Fprintln(deps.Output, "-- EXCLUDED: true")
	}
	if rn := jsa.ActionDefaultReturnName(); rn != "" {
		fmt.Fprintf(deps.Output, "-- return NAME: '%s'\n", rn)
	}
	return nil
}

func listJavaActionsGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listJavaActionsFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.JavaActionRepo, moduleName)
}


func listJavaScriptActionsGenDeps(ctx context.Context, deps *HandlerDeps, moduleName string) error {
	return listJavaScriptActionsFuture(ctx, deps.Output, deps.Format, deps.ModuleLister, deps.MetadataReader, deps.FolderManager, deps.JavaScriptActionRepo, moduleName)
}


func describeJavaActionGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeJavaActionGenFuture(ctx, deps.Output, deps.JavaActionRepo, name)
}


func describeJavaScriptActionGenDeps(ctx context.Context, deps *HandlerDeps, name ast.QualifiedName) error {
	return describeJavaScriptActionGenFn(ctx, name, deps)
}

// ── Modules ──


