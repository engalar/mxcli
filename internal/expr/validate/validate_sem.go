// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/mendixlabs/mxcli/internal/expr/parse"
	"github.com/mendixlabs/mxcli/internal/expr/scan"
	"github.com/mendixlabs/mxcli/mdl/exprcheck"
)

// IndexReader is the minimal metadata surface required by semantic rules.
// meta.Index and meta.MockIndex both satisfy it.
type IndexReader interface {
	EnumCases(enumQN string) ([]string, bool)
	HasConstant(ref string) bool
	HasEntity(entityQN string) bool
	HasAssociation(assocQN string) bool
	// AssocEndpoints returns the parent and child entity QNs for an association.
	AssocEndpoints(assocQN string) (parent, child string, ok bool)
	// EntityAttrs returns attribute names for an entity (used in fix suggestions).
	EntityAttrs(entityQN string) []string
	// IsEntityComplete reports whether the entity's attribute list is complete.
	// Returns false for entities whose inheritance chain includes protected
	// marketplace module parents — attribute validation should be skipped.
	IsEntityComplete(entityQN string) bool
	// VarEntityQN returns the entity QN for a microflow variable.
	// unitPath is scan.ExprRecord.UnitPath; varName has no leading $.
	// Returns "" when the variable type is unknown (non-entity type, or not tracked).
	VarEntityQN(unitPath, varName string) string
	// VarTypeKind returns the TypeKind of a microflow variable (for SEM-03).
	VarTypeKind(unitPath, varName string) exprcheck.TypeKind
	// MicroflowParamKind returns the TypeKind of a named parameter of a microflow.
	MicroflowParamKind(calleeQN, paramName string) (exprcheck.TypeKind, bool)
	// MicroflowReturnKind returns the TypeKind of a microflow's return value.
	MicroflowReturnKind(mfName string) (exprcheck.TypeKind, bool)
	// HasUserRole reports whether a user role with the given name exists in the project.
	// Used by SEM-08 to validate [%UserRole_Name%] token references.
	HasUserRole(name string) bool
}

// ValidateSemantic applies SEM-04/05/07 rules to a parse result.
// When idx is nil (standalone mode without index), returns nil.
func ValidateSemantic(pr parse.ParseResult, idx IndexReader) []ValidationResult {
	if idx == nil {
		return nil
	}
	var out []ValidationResult
	rec := pr.Record

	out = append(out, checkEnumRefs(rec.Raw, rec, idx)...)
	out = append(out, checkConstantRefs(rec.Raw, rec, idx)...)
	out = append(out, checkPaths(rec.Raw, rec, idx)...)
	out = append(out, checkUserRoleTokens(rec.Raw, rec, idx)...)

	return out
}

// ── SEM-04: enum value references ────────────────────────────────────────────

// enumRefPattern matches Module.Enum.Value triples.
var enumRefPattern = regexp.MustCompile(`\b([A-Z][A-Za-z0-9_]*)\.([A-Z][A-Za-z0-9_]*)\.([A-Z][A-Za-z0-9_][A-Za-z0-9_]*)\b`)

func checkEnumRefs(raw string, rec scan.ExprRecord, idx IndexReader) []ValidationResult {
	var out []ValidationResult
	for _, m := range enumRefPattern.FindAllStringSubmatch(raw, -1) {
		moduleName, enumName, valueName := m[1], m[2], m[3]
		enumQN := moduleName + "." + enumName
		vals, ok := idx.EnumCases(enumQN)
		if !ok {
			continue
		}
		found := false
		for _, v := range vals {
			if v == valueName {
				found = true
				break
			}
		}
		if !found {
			out = append(out, ValidationResult{
				UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType, UnitPath: rec.UnitPath,
				Field: rec.Field, Raw: raw,
				RuleID:   "SEM-04",
				Severity: "ERROR",
				Message:  fmt.Sprintf("Enum value '%s.%s.%s' not found in '%s'.", moduleName, enumName, valueName, enumQN),
				Fix:      fmt.Sprintf("Available values: %s", strings.Join(vals, ", ")),
			})
		}
	}
	return out
}

// ── SEM-05: constant references ───────────────────────────────────────────────

// constantRefPattern matches @Module.Name references.
// Mendix module names are PascalCase (uppercase first char); requiring uppercase
// avoids false positives on email addresses like @mendix.com.
var constantRefPattern = regexp.MustCompile(`@([A-Z][A-Za-z0-9_]*)\.([A-Z][A-Za-z0-9_]*)`)

func checkConstantRefs(raw string, rec scan.ExprRecord, idx IndexReader) []ValidationResult {
	var out []ValidationResult
	for _, m := range constantRefPattern.FindAllString(raw, -1) {
		if !idx.HasConstant(m) {
			out = append(out, ValidationResult{
				UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType, UnitPath: rec.UnitPath,
				Field: rec.Field, Raw: raw,
				RuleID:   "SEM-05",
				Severity: "ERROR",
				Message:  fmt.Sprintf("Constant '%s' not found in project.", m),
				Fix:      "Check the constant name and module — it may have been renamed or the module changed.",
			})
		}
	}
	return out
}

// ── SEM-07: path validation ───────────────────────────────────────────────────
//
// Mendix navigation paths follow an alternating grammar:
//
//   $var / Module.Assoc / Module.PeerEntity / attrName
//          ─────────── ─────────────────── ─────────
//          position 1   position 2 (opt)    position 3
//          Association  Peer Entity         Attribute
//
// The peer-entity qualifier (position 2) is optional; when omitted the attribute
// follows the association directly.  For multi-hop paths the pattern repeats:
//   / Module.Assoc2 / Module.Entity2 / attrName
//
// In XPath constraints, Module.Name can also appear as a standalone filter:
//   [Module.Assoc = $var]
//
// Rules applied:
//   - Module.Name at association position → must exist as an association
//   - Module.Name at entity position → must be the parent or child of the preceding assoc
//   - bare attrName at attribute position → must exist on the resolved entity
//
// "System.*" paths are skipped (built-in, not in user domain model).

// pathRe finds navigation path sequences in expression text.
//
// Two alternatives (joined with |):
//
//	Case A — $var-anchored path:   $var / seg1 [/ seg2 ...]
//	  Segments can be Module.Name (association/entity) or bare identifiers
//	  (attributes).  Requires at least one segment after the anchor so that
//	  bare "$var" references (comparisons, assignments) are not matched.
//
//	Case B — unanchored Module.Name path:   Module.Name [/ seg ...]
//	  Used for standalone association filters in XPath ([Module.Assoc = $v])
//	  and mid-path Module.Name references without a leading $var.
var pathRe = regexp.MustCompile(
	// Case A: $var / seg+  (any segment type, at least one)
	`(?:\$[A-Za-z_]\w*|\[%[A-Za-z_]\w*(?:\(\))?\s*%\])` +
		`(?:/[A-Za-z][A-Za-z0-9_.]*(?:\([^)]*\))?)+` +
		`|` +
		// Case B: Module.Name path (no anchor required)
		`[A-Z][A-Za-z0-9_]*\.[A-Za-z][A-Za-z0-9_]*` +
		`(?:/[A-Za-z][A-Za-z0-9_.]*(?:\([^)]*\))?)*`,
)

// segRe splits a path string into individual segments.
var segRe = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_.]*(?:\([^)]*\))?`)

// qualRe checks whether a segment is Module.Name qualified.
var qualRe = regexp.MustCompile(`^([A-Z][A-Za-z0-9_]*)\.([A-Za-z][A-Za-z0-9_]*)$`)

type pathSeg struct {
	raw    string
	module string // "" if unqualified
	name   string
}

func parseSeg(s string) pathSeg {
	// Strip predicates like [reversed()]
	if i := strings.IndexByte(s, '['); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	if m := qualRe.FindStringSubmatch(s); m != nil {
		return pathSeg{raw: s, module: m[1], name: m[2]}
	}
	return pathSeg{raw: s, name: s}
}

func (s pathSeg) isQualified() bool { return s.module != "" }
func (s pathSeg) qn() string        { return s.module + "." + s.name }
func (s pathSeg) isSystem() bool    { return s.module == "System" }
func (s pathSeg) isAnchor() bool {
	return strings.HasPrefix(s.raw, "$") || strings.HasPrefix(s.raw, "[%")
}

// sorted returns a sorted copy of ss (for deterministic Fix messages).
func sorted(ss []string) []string {
	cp := make([]string, len(ss))
	copy(cp, ss)
	sort.Strings(cp)
	return cp
}

// checkPaths validates Module.Assoc / Module.Entity / attr navigation paths.
func checkPaths(raw string, rec scan.ExprRecord, idx IndexReader) []ValidationResult {
	// For XPath constraints, strip the outer [ ... ] so the pathRe doesn't
	// accidentally consume the '[' as part of a segment.
	content := strings.TrimSpace(raw)
	if strings.HasPrefix(content, "[") && strings.HasSuffix(content, "]") {
		content = content[1 : len(content)-1]
	}

	// Strip string literals and constant references (@Module.Name) so that
	// SEM-05-owned tokens are not also validated by SEM-07.
	content = stripStringLiterals(content)
	content = constantRefPattern.ReplaceAllString(content, " ")

	var out []ValidationResult
	seen := map[string]bool{} // deduplicate identical errors

	for _, match := range pathRe.FindAllString(content, -1) {
		parts := strings.Split(match, "/")
		segs := make([]pathSeg, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			segs = append(segs, parseSeg(p))
		}
		out = append(out, walkPath(segs, raw, rec, idx, seen)...)
	}
	return out
}

// walkPath applies position-based validation to one path segment sequence.
//
// Mendix navigation grammar (after optional $var anchor):
//
//	Module.Assoc  [/ Module.PeerEntity]  [/ attrName]
//	└─ position 0 ┘  └─ position 1    ┘   └─ leaf  ┘
//	   Association      Entity qualifier     Attribute
//
// For multi-hop paths the Assoc→Entity→Assoc→... pattern repeats.
//
// Classification priority for each Module.Name segment:
//  1. Skip if System module.
//  2. Skip if it's a known enum QN (handled by SEM-04).
//  3. If it's a known entity AND previous context set currentEntity:
//     → entity qualifier; validate it matches the expected peer entity.
//  4. If it's a known association → advance entity context to peer entity.
//  5. If it's a known entity with no preceding assoc context → skip
//     (entity reference in non-navigation context, e.g. type argument).
//  6. If none of the above → flag SEM-07 (unknown reference).
func walkPath(segs []pathSeg, raw string, rec scan.ExprRecord, idx IndexReader, seen map[string]bool) []ValidationResult {
	var out []ValidationResult

	if len(segs) == 0 {
		return nil
	}

	// currentEntity: entity we're currently "on" after traversing an association.
	// Empty means unknown (e.g. source is a $var whose type we don't track).
	currentEntity := ""
	// prevAssoc{Parent,Child}: endpoints of the most recently traversed association,
	// used to validate the optional peer-entity qualifier that may follow.
	prevAssocParent := ""
	prevAssocChild := ""
	prevWasAssoc := false

	// Drop leading anchor ($var or [%Token%]) and resolve entity type when possible.
	if segs[0].isAnchor() {
		anchor := segs[0]
		segs = segs[1:]
		// For $var anchors, look up the microflow variable's entity type so that
		// subsequent bare attribute names can be validated against it.
		if strings.HasPrefix(anchor.raw, "$") {
			varName := anchor.raw[1:]
			if entityQN := idx.VarEntityQN(rec.UnitPath, varName); entityQN != "" {
				currentEntity = entityQN
			}
		}
	}
	if len(segs) == 0 {
		return nil
	}
	if segs[0].isSystem() {
		return nil
	}

	for _, seg := range segs {
		if !seg.isQualified() {
			// Bare name: leaf attribute.  Validate only when we know the entity.
			if currentEntity != "" {
				// Mendix system attributes present on every persistent entity;
				// they are not stored in the user domain model.
				// Also skip validation for entities with incomplete attribute sets
				// (protected marketplace module parents not readable from BSON).
				attrs := idx.EntityAttrs(currentEntity)
				if len(attrs) > 0 && !isMendixSystemAttr(seg.name) &&
					idx.IsEntityComplete(currentEntity) && !containsStr(attrs, seg.name) {
					key := currentEntity + "." + seg.name
					if !seen[key] {
						seen[key] = true
						out = append(out, ValidationResult{
							UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType, UnitPath: rec.UnitPath,
							Field: rec.Field, Raw: raw,
							RuleID:   "SEM-07",
							Severity: "WARNING",
							Message:  fmt.Sprintf("Attribute '%s' not found on entity '%s'.", seg.name, currentEntity),
							Fix:      fmt.Sprintf("Available attributes: %s", strings.Join(sorted(attrs), ", ")),
						})
					}
				}
			}
			break
		}

		qn := seg.qn()

		// 1. System module — skip.
		if seg.isSystem() {
			currentEntity = ""
			prevWasAssoc = false
			continue
		}

		// 2. Enum QN — not a navigation step (handled by SEM-04).
		if _, isEnum := idx.EnumCases(qn); isEnum {
			currentEntity = ""
			prevWasAssoc = false
			continue
		}

		// 3. Known entity appearing right after an association → peer-entity qualifier.
		//    Mendix allows traversal in both directions, so the qualifier must be
		//    EITHER endpoint of the preceding association, not just the one we
		//    inferred from direction.
		if idx.HasEntity(qn) && prevWasAssoc {
			if qn != prevAssocParent && qn != prevAssocChild {
				key := "entity-mismatch:" + qn + "@" + prevAssocParent + "+" + prevAssocChild
				if !seen[key] {
					seen[key] = true
					out = append(out, ValidationResult{
						UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType, UnitPath: rec.UnitPath,
						Field: rec.Field, Raw: raw,
						RuleID:   "SEM-07",
						Severity: "WARNING",
						Message:  fmt.Sprintf("Entity qualifier '%s' is not an endpoint of the preceding association (expected '%s' or '%s').", qn, prevAssocParent, prevAssocChild),
						Fix:      fmt.Sprintf("Valid entities here: %s, %s", prevAssocParent, prevAssocChild),
					})
				}
			}
			// Accept whatever the qualifier says as the new current entity.
			currentEntity = qn
			prevWasAssoc = false
			continue
		}

		// 4. Known entity in non-navigation context (e.g. first segment, type arg).
		if idx.HasEntity(qn) {
			currentEntity = qn
			prevWasAssoc = false
			continue
		}

		// 5. Known association → advance entity context.
		if parent, child, ok := idx.AssocEndpoints(qn); ok {
			// Infer direction from currentEntity; if unknown, default to child as peer.
			if currentEntity == "" || currentEntity == parent {
				currentEntity = child
			} else {
				currentEntity = parent
			}
			prevAssocParent = parent
			prevAssocChild = child
			prevWasAssoc = true
			continue
		}

		// 6. Unknown — flag SEM-07.
		key := "unknown:" + qn
		if !seen[key] {
			seen[key] = true
			out = append(out, ValidationResult{
				UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType, UnitPath: rec.UnitPath,
				Field: rec.Field, Raw: raw,
				RuleID:   "SEM-07",
				Severity: "WARNING",
				Message:  fmt.Sprintf("'%s' not found as entity or association in domain model.", qn),
				Fix:      "Verify the module and name — it may be an entity, association, or the module may differ.",
			})
		}
		currentEntity = ""
		prevWasAssoc = false
	}

	return out
}

// assocSuggestionsForModule returns association names in a given module.
// Since IndexReader doesn't expose a list-all-assocs method, we build this
// from the known association QNs via a module-prefix scan.
// (This is only called on error paths, so the O(n) scan is acceptable.)
func assocSuggestionsForModule(module string, idx IndexReader) []string {
	// IndexReader doesn't expose a full list; suggestions are best-effort.
	// Callers with a *meta.Index get real data via AssocEndpoints; for MockIndex
	// this returns nil which is fine for tests.
	_ = module
	_ = idx
	return nil
}

// stripStringLiterals replaces single-quoted string content with spaces so that
// path patterns inside string literals are not matched.
func stripStringLiterals(s string) string {
	var b strings.Builder
	inStr := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\'' {
			inStr = !inStr
			b.WriteByte(c)
			continue
		}
		if inStr {
			b.WriteByte(' ')
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}

func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// isMendixSystemAttr reports whether name is a Mendix built-in attribute that
// is present on every persistent entity but not stored in the user domain model.
func isMendixSystemAttr(name string) bool {
	switch name {
	case "id", "createdDate", "changedDate", "owner", "changedBy":
		return true
	}
	return false
}

// ── SEM-08: UserRole token validation ────────────────────────────────────────

// userRoleTokenRe matches [%UserRole_Name%] patterns and captures the role name.
var userRoleTokenRe = regexp.MustCompile(`\[%UserRole_(\w+)%\]`)

func checkUserRoleTokens(raw string, rec scan.ExprRecord, idx IndexReader) []ValidationResult {
	var out []ValidationResult
	for _, m := range userRoleTokenRe.FindAllStringSubmatch(raw, -1) {
		roleName := m[1]
		if !idx.HasUserRole(roleName) {
			out = append(out, ValidationResult{
				UnitID: rec.UnitID, Project: rec.Project, UnitType: rec.UnitType, UnitPath: rec.UnitPath,
				Field: rec.Field, Raw: raw,
				RuleID:   "SEM-08",
				Severity: "ERROR",
				Message:  fmt.Sprintf("User role '%s' does not exist in this project.", roleName),
				Fix:      "Check the role name — it may have been renamed or removed in project security settings.",
			})
		}
	}
	return out
}
