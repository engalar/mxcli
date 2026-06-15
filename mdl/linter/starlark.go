// SPDX-License-Identifier: Apache-2.0

package linter

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	genSec "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// StarlarkRule is a lint rule implemented in Starlark.
type StarlarkRule struct {
	id          string
	name        string
	description string
	severity    Severity
	category    string
	path        string
	ctx         *LintContext
	checkFn     starlark.Callable
	globals     starlark.StringDict
}

// ID returns the rule ID.
func (r *StarlarkRule) ID() string { return r.id }

// Name returns the rule name.
func (r *StarlarkRule) Name() string { return r.name }

// Description returns the rule description.
func (r *StarlarkRule) Description() string { return r.description }

// DefaultSeverity returns the rule severity.
func (r *StarlarkRule) DefaultSeverity() Severity { return r.severity }

// Category returns the rule category.
func (r *StarlarkRule) Category() string { return r.category }

// Check executes the Starlark check function and returns violations.
func (r *StarlarkRule) Check(ctx *LintContext) []Violation {
	r.ctx = ctx

	// Create a new thread for execution
	thread := &starlark.Thread{
		Name: r.id,
		Print: func(_ *starlark.Thread, msg string) {
			fmt.Println(msg)
		},
	}

	// Call the check function
	result, err := starlark.Call(thread, r.checkFn, nil, nil)
	if err != nil {
		return []Violation{{
			RuleID:   r.id,
			Severity: SeverityError,
			Message:  fmt.Sprintf("Starlark rule error: %v", err),
		}}
	}

	// Convert result to violations
	return r.convertViolations(result)
}

// convertViolations converts a Starlark list to Go violations.
func (r *StarlarkRule) convertViolations(result starlark.Value) []Violation {
	var violations []Violation

	list, ok := result.(*starlark.List)
	if !ok {
		return violations
	}

	iter := list.Iterate()
	defer iter.Done()

	var v starlark.Value
	for iter.Next(&v) {
		if viol := r.convertViolation(v); viol != nil {
			violations = append(violations, *viol)
		}
	}

	return violations
}

// convertViolation converts a Starlark struct to a Go Violation.
func (r *StarlarkRule) convertViolation(v starlark.Value) *Violation {
	s, ok := v.(*starlarkstruct.Struct)
	if !ok {
		return nil
	}

	viol := &Violation{
		RuleID:   r.id,
		Severity: r.severity,
	}

	if msg, err := s.Attr("message"); err == nil {
		if str, ok := msg.(starlark.String); ok {
			viol.Message = string(str)
		}
	}

	if loc, err := s.Attr("location"); err == nil {
		if locStruct, ok := loc.(*starlarkstruct.Struct); ok {
			if module, err := locStruct.Attr("module"); err == nil {
				if str, ok := module.(starlark.String); ok {
					viol.Location.Module = string(str)
				}
			}
			if docType, err := locStruct.Attr("document_type"); err == nil {
				if str, ok := docType.(starlark.String); ok {
					viol.Location.DocumentType = string(str)
				}
			}
			if docName, err := locStruct.Attr("document_name"); err == nil {
				if str, ok := docName.(starlark.String); ok {
					viol.Location.DocumentName = string(str)
				}
			}
			if docID, err := locStruct.Attr("document_id"); err == nil {
				if str, ok := docID.(starlark.String); ok {
					viol.Location.DocumentID = string(str)
				}
			}
		}
	}

	if sug, err := s.Attr("suggestion"); err == nil {
		if str, ok := sug.(starlark.String); ok {
			viol.Suggestion = string(str)
		}
	}

	return viol
}

// LoadStarlarkRule loads a Starlark rule from a file.
func LoadStarlarkRule(path string) (*StarlarkRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read rule file: %w", err)
	}

	rule := &StarlarkRule{
		path:     path,
		severity: SeverityWarning,
		category: "custom",
	}

	// Build predeclared environment
	predeclared := rule.buildPredeclared()

	// Parse and execute the file
	thread := &starlark.Thread{
		Name: filepath.Base(path),
	}

	globals, err := starlark.ExecFile(thread, path, data, predeclared)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Starlark file: %w", err)
	}

	rule.globals = globals

	// Extract metadata
	if id, ok := globals["RULE_ID"]; ok {
		if str, ok := id.(starlark.String); ok {
			rule.id = string(str)
		}
	}
	if rule.id == "" {
		// Use filename as fallback
		rule.id = strings.TrimSuffix(filepath.Base(path), ".star")
	}

	if name, ok := globals["RULE_NAME"]; ok {
		if str, ok := name.(starlark.String); ok {
			rule.name = string(str)
		}
	}
	if rule.name == "" {
		rule.name = rule.id
	}

	if desc, ok := globals["DESCRIPTION"]; ok {
		if str, ok := desc.(starlark.String); ok {
			rule.description = string(str)
		}
	}

	if sev, ok := globals["SEVERITY"]; ok {
		if str, ok := sev.(starlark.String); ok {
			rule.severity = ParseSeverity(string(str))
		}
	}

	if cat, ok := globals["CATEGORY"]; ok {
		if str, ok := cat.(starlark.String); ok {
			rule.category = string(str)
		}
	}

	// Get the check function
	checkVal, ok := globals["check"]
	if !ok {
		return nil, fmt.Errorf("rule must define a check() function")
	}

	checkFn, ok := checkVal.(starlark.Callable)
	if !ok {
		return nil, fmt.Errorf("check must be a callable function")
	}

	rule.checkFn = checkFn

	return rule, nil
}

// buildPredeclared creates the predeclared environment for Starlark rules.
func (r *StarlarkRule) buildPredeclared() starlark.StringDict {
	return starlark.StringDict{
		// Query functions
		"entities":             starlark.NewBuiltin("entities", r.builtinEntities),
		"microflows":           starlark.NewBuiltin("microflows", r.builtinMicroflows),
		"pages":                starlark.NewBuiltin("pages", r.builtinPages),
		"enumerations":         starlark.NewBuiltin("enumerations", r.builtinEnumerations),
		"widgets":              starlark.NewBuiltin("widgets", r.builtinWidgets),
		"refs_to":              starlark.NewBuiltin("refs_to", r.builtinRefsTo),
		"attributes_for":       starlark.NewBuiltin("attributes_for", r.builtinAttributesFor),
		"permissions":          starlark.NewBuiltin("permissions", r.builtinPermissions),
		"permissions_for":      starlark.NewBuiltin("permissions_for", r.builtinPermissionsFor),
		"snippets":             starlark.NewBuiltin("snippets", r.builtinSnippets),
		"database_connections": starlark.NewBuiltin("database_connections", r.builtinDatabaseConnections),
		"activities_for":       starlark.NewBuiltin("activities_for", r.builtinActivitiesFor),

		// Project-level queries
		"user_roles":       starlark.NewBuiltin("user_roles", r.builtinUserRoles),
		"module_roles":     starlark.NewBuiltin("module_roles", r.builtinModuleRoles),
		"role_mappings":    starlark.NewBuiltin("role_mappings", r.builtinRoleMappings),
		"project_security": starlark.NewBuiltin("project_security", r.builtinProjectSecurity),

		// Violation helpers
		"violation": starlark.NewBuiltin("violation", builtinViolation),
		"location":  starlark.NewBuiltin("location", builtinLocation),

		// String utilities
		"is_pascal_case": starlark.NewBuiltin("is_pascal_case", builtinIsPascalCase),
		"is_camel_case":  starlark.NewBuiltin("is_camel_case", builtinIsCamelCase),
		"matches":        starlark.NewBuiltin("matches", builtinMatches),

		// Struct constructor (from starlarkstruct)
		"struct": starlark.NewBuiltin("struct", starlarkstruct.Make),
	}
}

// builtinEntities returns an iterator over entities.
func (r *StarlarkRule) builtinEntities(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if r.ctx == nil {
		return starlark.NewList(nil), nil
	}

	var entities []starlark.Value
	for _, entity := range r.ctx.Entities() {
		entities = append(entities, entityToStarlark(entity))
	}

	return starlark.NewList(entities), nil
}

// builtinMicroflows returns an iterator over microflows.
func (r *StarlarkRule) builtinMicroflows(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if r.ctx == nil {
		return starlark.NewList(nil), nil
	}

	var microflows []starlark.Value
	for _, mf := range r.ctx.Microflows() {
		microflows = append(microflows, microflowToStarlark(mf))
	}

	return starlark.NewList(microflows), nil
}

// builtinPages returns an iterator over pages.
func (r *StarlarkRule) builtinPages(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if r.ctx == nil {
		return starlark.NewList(nil), nil
	}

	var pages []starlark.Value
	for _, page := range r.ctx.Pages() {
		pages = append(pages, pageToStarlark(page))
	}

	return starlark.NewList(pages), nil
}

// builtinEnumerations returns an iterator over enumerations.
func (r *StarlarkRule) builtinEnumerations(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if r.ctx == nil {
		return starlark.NewList(nil), nil
	}

	var enums []starlark.Value
	for _, enum := range r.ctx.Enumerations() {
		enums = append(enums, enumerationToStarlark(enum))
	}

	return starlark.NewList(enums), nil
}

// builtinWidgets returns an iterator over widgets.
func (r *StarlarkRule) builtinWidgets(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if r.ctx == nil {
		return starlark.NewList(nil), nil
	}

	// graphcatalog scopes widgets per page; aggregate across all pages to
	// preserve the previous project-wide widgets() semantics.
	var widgets []starlark.Value
	for _, page := range r.ctx.Pages() {
		for _, widget := range r.ctx.Widgets(page.QualifiedName) {
			widgets = append(widgets, widgetToStarlark(widget))
		}
	}

	return starlark.NewList(widgets), nil
}

// builtinAttributesFor returns the attributes for a given entity.
func (r *StarlarkRule) builtinAttributesFor(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if r.ctx == nil {
		return starlark.NewList(nil), nil
	}

	var entityQualifiedName starlark.String
	if err := starlark.UnpackArgs("attributes_for", args, kwargs,
		"entity_qualified_name", &entityQualifiedName,
	); err != nil {
		return nil, err
	}

	var attrs []starlark.Value
	for _, attr := range r.ctx.Attributes(string(entityQualifiedName)) {
		attrs = append(attrs, attributeToStarlark(attr))
	}

	return starlark.NewList(attrs), nil
}

// builtinRefsTo returns references to a given target name.
func (r *StarlarkRule) builtinRefsTo(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var targetName starlark.String

	if err := starlark.UnpackArgs("refs_to", args, kwargs, "target_name", &targetName); err != nil {
		return nil, err
	}

	if r.ctx == nil {
		return starlark.NewList(nil), nil
	}

	graph := r.ctx.Graph()
	if graph == nil {
		return starlark.NewList(nil), nil
	}
	// graphcatalog.LintReader does not expose traversal; ProjectGraph also
	// implements TraversalReader, so use it when available.
	tr, ok := graph.(graphcatalog.TraversalReader)
	if !ok {
		return starlark.NewList(nil), nil
	}
	var result []starlark.Value
	for _, ref := range tr.References(string(targetName)) {
		result = append(result, refEdgeToStarlark(ref))
	}

	return starlark.NewList(result), nil
}

// builtinPermissions returns all permissions across all element types.
func (r *StarlarkRule) builtinPermissions(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if r.ctx == nil {
		return starlark.NewList(nil), nil
	}

	var result []starlark.Value
	for _, p := range r.ctx.Permissions() {
		result = append(result, permissionNodeToStarlark(p))
	}

	return starlark.NewList(result), nil
}

// builtinPermissionsFor returns the permissions for a given entity.
func (r *StarlarkRule) builtinPermissionsFor(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if r.ctx == nil {
		return starlark.NewList(nil), nil
	}

	var entityQualifiedName starlark.String
	if err := starlark.UnpackArgs("permissions_for", args, kwargs,
		"entity_qualified_name", &entityQualifiedName,
	); err != nil {
		return nil, err
	}

	var perms []starlark.Value
	for _, perm := range r.ctx.Permissions() {
		if perm.EntityName != string(entityQualifiedName) {
			continue
		}
		perms = append(perms, entityPermissionNodeToStarlark(perm))
	}

	return starlark.NewList(perms), nil
}

// builtinSnippets returns an iterator over snippets.
func (r *StarlarkRule) builtinSnippets(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if r.ctx == nil {
		return starlark.NewList(nil), nil
	}

	var snippets []starlark.Value
	for _, s := range r.ctx.Snippets() {
		snippets = append(snippets, snippetToStarlark(s))
	}

	return starlark.NewList(snippets), nil
}

// builtinDatabaseConnections returns an iterator over database connections.
func (r *StarlarkRule) builtinDatabaseConnections(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if r.ctx == nil {
		return starlark.NewList(nil), nil
	}

	var connections []starlark.Value
	for _, dc := range r.ctx.DatabaseConnections() {
		connections = append(connections, databaseConnectionToStarlark(dc))
	}

	return starlark.NewList(connections), nil
}

// builtinActivitiesFor returns the activities for a given microflow.
func (r *StarlarkRule) builtinActivitiesFor(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if r.ctx == nil {
		return starlark.NewList(nil), nil
	}

	var microflowQualifiedName starlark.String
	if err := starlark.UnpackArgs("activities_for", args, kwargs,
		"microflow_qualified_name", &microflowQualifiedName,
	); err != nil {
		return nil, err
	}

	// TODO: activities are not modelled as graph nodes in graphcatalog yet.
	// Re-enable when the microflow adapter emits Activity nodes (or expose a
	// deep-reader-backed walk over GetMicroflowGen object collections).
	_ = microflowQualifiedName
	return starlark.NewList(nil), nil
}

// builtinUserRoles returns all user roles from project security.
func (r *StarlarkRule) builtinUserRoles(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if r.ctx == nil {
		return starlark.NewList(nil), nil
	}

	roles := r.ctx.UserRoles()
	var result []starlark.Value
	for _, ur := range roles {
		result = append(result, userRoleToStarlark(ur))
	}

	return starlark.NewList(result), nil
}

// builtinModuleRoles returns all module roles from the catalog.
func (r *StarlarkRule) builtinModuleRoles(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if r.ctx == nil {
		return starlark.NewList(nil), nil
	}

	// graphcatalog has no dedicated module-role listing; derive the distinct
	// set from role mappings.
	seen := make(map[string]bool)
	var result []starlark.Value
	for _, rm := range r.ctx.RoleMappings() {
		if rm.ModuleRole == "" || seen[rm.ModuleRole] {
			continue
		}
		seen[rm.ModuleRole] = true
		result = append(result, moduleRoleStringToStarlark(rm.ModuleRole))
	}

	return starlark.NewList(result), nil
}

// builtinRoleMappings returns all user role to module role mappings.
func (r *StarlarkRule) builtinRoleMappings(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if r.ctx == nil {
		return starlark.NewList(nil), nil
	}

	var result []starlark.Value
	for _, rm := range r.ctx.RoleMappings() {
		result = append(result, roleMappingNodeToStarlark(rm))
	}

	return starlark.NewList(result), nil
}

// builtinProjectSecurity returns project security settings as a Starlark struct.
func (r *StarlarkRule) builtinProjectSecurity(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if r.ctx == nil {
		return starlark.None, nil
	}

	reader := r.ctx.Reader()
	if reader == nil {
		return starlark.None, nil
	}

	ps, err := reader.GetProjectSecurityGen()
	if err != nil || ps == nil {
		return starlark.None, nil
	}

	// Build password_policy sub-struct.
	// schema gap: PasswordPolicySettings() returns element.Element; type-assert to concrete type.
	ppDict := starlark.StringDict{
		"min_length":         starlark.MakeInt(0),
		"require_digit":      starlark.Bool(false),
		"require_mixed_case": starlark.Bool(false),
		"require_symbol":     starlark.Bool(false),
	}
	if pp, ok := ps.PasswordPolicySettings().(*genSec.PasswordPolicySettings); ok && pp != nil {
		ppDict["min_length"] = starlark.MakeInt(int(pp.MinimumLength()))
		ppDict["require_digit"] = starlark.Bool(pp.RequireDigit())
		ppDict["require_mixed_case"] = starlark.Bool(pp.RequireMixedCase())
		ppDict["require_symbol"] = starlark.Bool(pp.RequireSymbol())
	}

	return starlarkstruct.FromStringDict(starlark.String("project_security"), starlark.StringDict{
		"security_level":      starlark.String(ps.SecurityLevel()),
		"enable_demo_users":   starlark.Bool(ps.EnableDemoUsers()),
		"enable_guest_access": starlark.Bool(ps.EnableGuestAccess()),
		"check_security":      starlark.Bool(ps.CheckSecurity()),
		"strict_mode":         starlark.Bool(ps.StrictMode()),
		"anonymous_user_role": starlark.String(ps.GuestUserRoleName()),
		"password_policy":     starlarkstruct.FromStringDict(starlark.String("password_policy"), ppDict),
	}), nil
}

// moduleOf extracts the module prefix from a qualified name "Module.Element".
func moduleOf(qualifiedName string) string {
	if i := strings.IndexByte(qualifiedName, '.'); i >= 0 {
		return qualifiedName[:i]
	}
	return ""
}

// entityToStarlark converts a graphcatalog EntityNode to a Starlark struct.
// Fields the in-memory graph does not carry (folder, entity_type, counts) are
// kept as keys with zero values so existing custom rules don't break with an
// AttributeError; richer document data is reached via the deep reader.
func entityToStarlark(e graphcatalog.EntityNode) starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String("entity"), starlark.StringDict{
		"id":                    starlark.String(e.ID),
		"name":                  starlark.String(e.Name),
		"qualified_name":        starlark.String(e.QualifiedName),
		"module_name":           starlark.String(e.Module),
		"folder":                starlark.String(""),
		"entity_type":           starlark.String(""),
		"description":           starlark.String(""),
		"generalization":        starlark.String(""),
		"attribute_count":       starlark.MakeInt(0),
		"access_rule_count":     starlark.MakeInt(0),
		"validation_rule_count": starlark.MakeInt(0),
		"has_event_handlers":    starlark.Bool(false),
		"is_external":           starlark.Bool(e.IsExternal),
	})
}

// microflowToStarlark converts a graphcatalog MicroflowNode to a Starlark struct.
func microflowToStarlark(mf graphcatalog.MicroflowNode) starlark.Value {
	mfType := "Microflow"
	if mf.IsNanoflow {
		mfType = "Nanoflow"
	}
	return starlarkstruct.FromStringDict(starlark.String("microflow"), starlark.StringDict{
		"id":              starlark.String(mf.ID),
		"name":            starlark.String(mf.Name),
		"qualified_name":  starlark.String(mf.QualifiedName),
		"module_name":     starlark.String(mf.Module),
		"folder":          starlark.String(""),
		"microflow_type":  starlark.String(mfType),
		"description":     starlark.String(""),
		"return_type":     starlark.String(mf.ReturnType),
		"parameter_count": starlark.MakeInt(0),
		"activity_count":  starlark.MakeInt(0),
		"complexity":      starlark.MakeInt(0),
	})
}

// pageToStarlark converts a graphcatalog PageNode to a Starlark struct.
func pageToStarlark(p graphcatalog.PageNode) starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String("page"), starlark.StringDict{
		"id":             starlark.String(p.ID),
		"name":           starlark.String(p.Name),
		"qualified_name": starlark.String(p.QualifiedName),
		"module_name":    starlark.String(p.Module),
		"folder":         starlark.String(""),
		"title":          starlark.String(""),
		"url":            starlark.String(""),
		"description":    starlark.String(""),
		"widget_count":   starlark.MakeInt(0),
	})
}

// enumerationToStarlark converts a graphcatalog EnumerationNode to a Starlark struct.
func enumerationToStarlark(e graphcatalog.EnumerationNode) starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String("enumeration"), starlark.StringDict{
		"id":             starlark.String(e.ID),
		"name":           starlark.String(e.Name),
		"qualified_name": starlark.String(e.QualifiedName),
		"module_name":    starlark.String(e.Module),
		"folder":         starlark.String(""),
		"description":    starlark.String(""),
		"value_count":    starlark.MakeInt(0),
	})
}

// widgetToStarlark converts a graphcatalog WidgetNode to a Starlark struct.
// container_type / container_qualified_name / entity_ref / attribute_ref are not
// carried by the graph node and are kept as empty keys for API compatibility.
func widgetToStarlark(w graphcatalog.WidgetNode) starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String("widget"), starlark.StringDict{
		"id":                       starlark.String(w.ID),
		"name":                     starlark.String(w.Name),
		"widget_type":              starlark.String(w.WidgetType),
		"container_id":             starlark.String(w.PageID),
		"container_qualified_name": starlark.String(""),
		"container_type":           starlark.String(""),
		"module_name":              starlark.String(""),
		"entity_ref":               starlark.String(""),
		"attribute_ref":            starlark.String(""),
	})
}

// attributeToStarlark converts a graphcatalog AttributeNode to a Starlark struct.
func attributeToStarlark(a graphcatalog.AttributeNode) starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String("attribute"), starlark.StringDict{
		"id":                    starlark.String(a.ID),
		"name":                  starlark.String(a.Name),
		"entity_id":             starlark.String(""),
		"entity_qualified_name": starlark.String(a.Entity),
		"module_name":           starlark.String(moduleOf(a.Entity)),
		"data_type":             starlark.String(a.DataType),
		"length":                starlark.MakeInt(0),
		"is_unique":             starlark.Bool(false),
		"is_required":           starlark.Bool(false),
		"default_value":         starlark.String(""),
		"is_calculated":         starlark.Bool(false),
		"description":           starlark.String(""),
	})
}

// refEdgeToStarlark converts a graphcatalog RefEdge to a Starlark reference struct.
func refEdgeToStarlark(r graphcatalog.RefEdge) starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String("reference"), starlark.StringDict{
		"source_type": starlark.String(""),
		"source_id":   starlark.String(""),
		"source_name": starlark.String(r.Source),
		"target_type": starlark.String(""),
		"target_id":   starlark.String(""),
		"target_name": starlark.String(r.Target),
		"ref_kind":    starlark.String(r.RefKind),
		"module_name": starlark.String(moduleOf(r.Source)),
	})
}

// permissionNodeToStarlark converts a graphcatalog PermissionNode to a Starlark struct.
func permissionNodeToStarlark(p graphcatalog.PermissionNode) starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String("permission"), starlark.StringDict{
		"module_role_name": starlark.String(p.ModuleRole),
		"element_type":     starlark.String("ENTITY"),
		"element_name":     starlark.String(p.EntityName),
		"member_name":      starlark.String(""),
		"access_type":      starlark.String(p.AccessRights),
		"xpath_constraint": starlark.String(""),
		"is_constrained":   starlark.Bool(false),
		"module_name":      starlark.String(moduleOf(p.EntityName)),
	})
}

// entityPermissionNodeToStarlark converts a PermissionNode to the entity-scoped
// Starlark struct returned by permissions_for.
func entityPermissionNodeToStarlark(p graphcatalog.PermissionNode) starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String("entity_permission"), starlark.StringDict{
		"module_role_name": starlark.String(p.ModuleRole),
		"module_name":      starlark.String(moduleOf(p.EntityName)),
		"entity_name":      starlark.String(p.EntityName),
		"access_type":      starlark.String(p.AccessRights),
		"member_name":      starlark.String(""),
		"xpath_constraint": starlark.String(""),
		"is_constrained":   starlark.Bool(false),
	})
}

// userRoleToStarlark converts a UserRoleInfo to a Starlark struct.
func userRoleToStarlark(ur UserRoleInfo) starlark.Value {
	var moduleRoles []starlark.Value
	for _, mr := range ur.ModuleRoles {
		moduleRoles = append(moduleRoles, starlark.String(mr))
	}
	return starlarkstruct.FromStringDict(starlark.String("user_role"), starlark.StringDict{
		"name":         starlark.String(ur.Name),
		"is_anonymous": starlark.Bool(ur.IsAnonymous),
		"module_roles": starlark.NewList(moduleRoles),
	})
}

// moduleRoleStringToStarlark converts a module-role name (derived from role
// mappings) to a Starlark struct.
func moduleRoleStringToStarlark(name string) starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String("module_role"), starlark.StringDict{
		"name":        starlark.String(name),
		"module_name": starlark.String(moduleOf(name)),
		"description": starlark.String(""),
	})
}

// roleMappingNodeToStarlark converts a graphcatalog RoleMappingNode to a Starlark struct.
func roleMappingNodeToStarlark(rm graphcatalog.RoleMappingNode) starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String("role_mapping"), starlark.StringDict{
		"user_role_name":   starlark.String(rm.UserRole),
		"module_role_name": starlark.String(rm.ModuleRole),
		"module_name":      starlark.String(moduleOf(rm.ModuleRole)),
	})
}

// snippetToStarlark converts a graphcatalog SnippetNode to a Starlark struct.
func snippetToStarlark(s graphcatalog.SnippetNode) starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String("snippet"), starlark.StringDict{
		"id":             starlark.String(s.ID),
		"name":           starlark.String(s.Name),
		"qualified_name": starlark.String(s.QualifiedName),
		"module_name":    starlark.String(s.Module),
		"folder":         starlark.String(""),
		"widget_count":   starlark.MakeInt(0),
	})
}

// databaseConnectionToStarlark converts a graphcatalog DatabaseConnectionNode to a Starlark struct.
func databaseConnectionToStarlark(dc graphcatalog.DatabaseConnectionNode) starlark.Value {
	return starlarkstruct.FromStringDict(starlark.String("database_connection"), starlark.StringDict{
		"id":             starlark.String(dc.ID),
		"name":           starlark.String(dc.Name),
		"qualified_name": starlark.String(""),
		"module_name":    starlark.String(""),
		"folder":         starlark.String(""),
		"database_type":  starlark.String(dc.DatabaseType),
		"query_count":    starlark.MakeInt(0),
	})
}

// builtinViolation creates a violation struct.
func builtinViolation(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var message starlark.String
	var location starlark.Value = starlark.None
	var suggestion starlark.String

	if err := starlark.UnpackArgs("violation", args, kwargs,
		"message", &message,
		"location?", &location,
		"suggestion?", &suggestion,
	); err != nil {
		return nil, err
	}

	return starlarkstruct.FromStringDict(starlark.String("violation"), starlark.StringDict{
		"message":    message,
		"location":   location,
		"suggestion": suggestion,
	}), nil
}

// builtinLocation creates a location struct.
func builtinLocation(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var module, documentType, documentName, documentID starlark.String

	if err := starlark.UnpackArgs("location", args, kwargs,
		"module", &module,
		"document_type", &documentType,
		"document_name", &documentName,
		"document_id?", &documentID,
	); err != nil {
		return nil, err
	}

	return starlarkstruct.FromStringDict(starlark.String("location"), starlark.StringDict{
		"module":        module,
		"document_type": documentType,
		"document_name": documentName,
		"document_id":   documentID,
	}), nil
}

// builtinIsPascalCase checks if a string is PascalCase.
func builtinIsPascalCase(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s starlark.String
	if err := starlark.UnpackArgs("is_pascal_case", args, kwargs, "s", &s); err != nil {
		return nil, err
	}

	str := string(s)
	if str == "" {
		return starlark.False, nil
	}

	runes := []rune(str)
	if !unicode.IsUpper(runes[0]) {
		return starlark.False, nil
	}

	for _, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return starlark.False, nil
		}
	}

	return starlark.True, nil
}

// builtinIsCamelCase checks if a string is camelCase.
func builtinIsCamelCase(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s starlark.String
	if err := starlark.UnpackArgs("is_camel_case", args, kwargs, "s", &s); err != nil {
		return nil, err
	}

	str := string(s)
	if str == "" {
		return starlark.False, nil
	}

	runes := []rune(str)
	if !unicode.IsLower(runes[0]) {
		return starlark.False, nil
	}

	for _, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return starlark.False, nil
		}
	}

	return starlark.True, nil
}

// builtinMatches checks if a string matches a regex pattern.
func builtinMatches(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s, pattern starlark.String
	if err := starlark.UnpackArgs("matches", args, kwargs, "s", &s, "pattern", &pattern); err != nil {
		return nil, err
	}

	re, err := regexp.Compile(string(pattern))
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	if re.MatchString(string(s)) {
		return starlark.True, nil
	}
	return starlark.False, nil
}

// LoadStarlarkRulesFromDir loads all Starlark rules from a directory.
func LoadStarlarkRulesFromDir(dir string) ([]*StarlarkRule, error) {
	var rules []*StarlarkRule

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return rules, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".star") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		rule, err := LoadStarlarkRule(path)
		if err != nil {
			// Log warning but continue loading other rules
			fmt.Printf("Warning: failed to load rule %s: %v\n", path, err)
			continue
		}

		rules = append(rules, rule)
	}

	return rules, nil
}
