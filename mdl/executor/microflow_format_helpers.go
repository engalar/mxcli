// SPDX-License-Identifier: Apache-2.0

// Stage 3.2.6.3a: helpers extracted from the deleted
// `cmd_microflows_create.go` and `cmd_microflows_format_action.go`
// (legacy sdk/microflows-typed). These functions are pure (string /
// catalog) — they do not depend on the sdk/microflows types — and are
// consumed by the gen-typed describe / format files
// (cmd_microflows_format_action_gen.go,
// cmd_microflows_format_data_gen.go,
// cmd_microflows_format_external_gen.go,
// cmd_microflows_create_gen.go). Keeping them in a dedicated file
// (rather than re-creating the legacy `cmd_microflows_format_action.go`)
// makes it explicit that there is no remaining sdk-typed implementation.

package executor

import (
	"strings"

	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/mendixlabs/mxcli/model"
	genDm "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
)

// isBuiltinModuleEntity returns true for module names that are
// runtime-provided (System.User, System.Workflow, etc.) and not part
// of the project MPR. Used by validate.go to skip false-positive
// "not found" errors for system references — Studio Pro's `mx check`
// resolves these against the runtime; `mxcli check` cannot.
func isBuiltinModuleEntity(moduleName string) bool {
	return moduleName == "System"
}

// formatEnumSplitCaseValue normalises an enum-split case value for display
// (empty strings become the literal "(empty)" placeholder Mendix uses).
func formatEnumSplitCaseValue(value string) string {
	if value == "" || value == "(empty)" {
		return "(empty)"
	}
	return value
}

// formatEnumSplitCaseValues joins a list of enum-split case values into the
// `value1, value2, ...` form used in describe output.
func formatEnumSplitCaseValues(values []string) string {
	formatted := make([]string, 0, len(values))
	for _, value := range values {
		formatted = append(formatted, formatEnumSplitCaseValue(value))
	}
	return strings.Join(formatted, ", ")
}

// loadRestServices returns all consumed REST services, or nil if no backend.
func loadRestServices(ctx *ExecContext) ([]*model.ConsumedRestService, error) {
	if !ctx.Connected() {
		return nil, nil
	}
	svcs, err := ctx.Backend.ListConsumedRestServices()
	return svcs, err
}

// escapeExpressionValue escapes raw control characters inside string literals
// of a Mendix expression value so it can be safely embedded in MDL output.
// The lexer's STRING_LITERAL rule forbids raw \r and \n inside single-quoted
// strings. Only characters inside '...' regions are escaped; characters
// outside string literals (structural whitespace) are preserved as-is.
func escapeExpressionValue(v string) string {
	if !strings.ContainsAny(v, "\n\r\t") {
		return v
	}
	var b strings.Builder
	b.Grow(len(v) + 32)
	inString := false
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c == '\'' {
			// Check for escaped quote ('') inside string
			if inString && i+1 < len(v) && v[i+1] == '\'' {
				b.WriteString("''")
				i++
				continue
			}
			inString = !inString
			b.WriteByte(c)
			continue
		}
		if inString {
			switch c {
			case '\n':
				b.WriteString(`\n`)
			case '\r':
				b.WriteString(`\r`)
			case '\t':
				b.WriteString(`\t`)
			default:
				b.WriteByte(c)
			}
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// formatWebServiceReference returns a bare-qualified or quoted reference.
func formatWebServiceReference(ref string) string {
	if isBareQualifiedReference(ref) {
		return ref
	}
	return mdlQuote(ref)
}

func isBareQualifiedReference(ref string) bool {
	if ref == "" {
		return false
	}
	for _, part := range strings.Split(ref, ".") {
		if !isBareIdentifier(part) {
			return false
		}
	}
	return true
}

func isBareIdentifier(part string) bool {
	if part == "" {
		return false
	}
	for i, r := range part {
		if i == 0 {
			if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
				return false
			}
			continue
		}
		if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// parseReverseAssociationXPath inspects a single-predicate XPath constraint
// of the form `[Module.AssocName = $Var]` and returns the association QN +
// driving variable. Used by `database retrieve` formatting to detect when an
// XPath constraint is just a reverse-association traversal that can be
// rewritten as `from $Var/Module.AssocName`.
func parseReverseAssociationXPath(raw string) (string, string, bool) {
	parts, ok := splitTopLevelXPathPredicates(raw)
	if !ok || len(parts) != 1 {
		return "", "", false
	}

	condition := strings.TrimSpace(parts[0])
	if strings.ContainsAny(condition, "<>!") || strings.Count(condition, "=") != 1 {
		return "", "", false
	}

	sides := strings.SplitN(condition, "=", 2)
	assocName := strings.TrimSpace(sides[0])
	startVar := strings.TrimSpace(sides[1])
	if !isQualifiedAssociationName(assocName) || !strings.HasPrefix(startVar, "$") {
		return "", "", false
	}

	startVar = strings.TrimPrefix(startVar, "$")
	if !isSimpleMendixName(startVar) {
		return "", "", false
	}
	return assocName, startVar, true
}

func isQualifiedAssociationName(name string) bool {
	parts := strings.Split(name, ".")
	return len(parts) == 2 && isSimpleMendixName(parts[0]) && isSimpleMendixName(parts[1])
}

func isSimpleMendixName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
				continue
			}
			return false
		}
		if r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

// databaseRetrieveMatchesAssociationTarget returns true when the entity
// supplied to `from` is the OWNER (parent) of the named association — the
// case where a reverse-association rewrite is valid.
func databaseRetrieveMatchesAssociationTarget(ctx *ExecContext, entityName, assocQualifiedName string) bool {
	moduleName, assocName, ok := strings.Cut(assocQualifiedName, ".")
	if !ok {
		return false
	}

	mod, err := ctx.ModuleLister.GetModuleByName(moduleName)
	if err != nil || mod == nil {
		return false
	}
	dm, err := getDomainModelGenCached(ctx, mod.ID)
	if err != nil || dm == nil {
		return false
	}

	entityNames := make(map[model.ID]string)
	for _, entityElem := range dm.EntitiesItems() {
		entity, ok := entityElem.(*genDm.Entity)
		if !ok {
			continue
		}
		entityNames[model.ID(entity.ID())] = moduleName + "." + entity.Name()
	}
	for _, assocElem := range dm.AssociationsItems() {
		assoc, ok := assocElem.(*genDm.Association)
		if !ok {
			continue
		}
		if assoc.Name() == assocName {
			return entityNames[model.ID(assoc.ParentRefID())] == entityName
		}
	}
	return false
}

// splitTopLevelXPathPredicates splits a `[...][...]` chain into its
// top-level predicates, respecting quotes and nested brackets. Returns
// (nil, false) on any structural parse error.
func splitTopLevelXPathPredicates(raw string) ([]string, bool) {
	var parts []string
	input := strings.TrimSpace(raw)
	if input == "" {
		return nil, false
	}

	i := 0
	for i < len(input) {
		for i < len(input) && (input[i] == ' ' || input[i] == '\t' || input[i] == '\r' || input[i] == '\n') {
			i++
		}
		if i >= len(input) {
			break
		}
		if input[i] != '[' {
			return nil, false
		}

		start := i + 1
		depth := 1
		var quote byte
		for i = start; i < len(input); i++ {
			ch := input[i]
			if quote != 0 {
				if ch == quote {
					quote = 0
				}
				continue
			}
			switch ch {
			case '\'', '"':
				quote = ch
			case '[':
				depth++
			case ']':
				depth--
				if depth == 0 {
					part := strings.TrimSpace(input[start:i])
					parts = append(parts, part)
					i++
					goto nextPredicate
				}
			}
		}
		return nil, false

	nextPredicate:
	}

	if len(parts) == 0 {
		return nil, false
	}

	return parts, true
}

// enrichXPathConstraintForDescribe enriches the raw BSON XPathConstraint string for
// DESCRIBE output. String-literal comparisons against enum attributes are replaced with
// qualified enum value references (e.g. Status = 'Open' → Status = Module.OrderStatus.Open).
// Falls back to the original string on any parse failure.
func enrichXPathConstraintForDescribe(ctx *ExecContext, entityQN, constraint string) string {
	if entityQN == "" || constraint == "" {
		return constraint
	}
	enumAttrs := buildEntityEnumAttrMap(ctx, entityQN)
	if len(enumAttrs) == 0 {
		return constraint
	}
	expr, ok := visitor.ParseXPathConstraint(constraint)
	if !ok || expr == nil {
		return constraint
	}
	enriched := enrichXPathExprWithEnums(expr, enumAttrs)
	return "[" + xpathExprToMDLString(enriched) + "]"
}
