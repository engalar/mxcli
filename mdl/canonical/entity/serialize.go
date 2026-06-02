// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/canonical"
)

// ToMDL renders the canonical entity as deterministic MDL text. Output is a
// `create ... entity Module.Name (...)` block followed by zero or more
// `index ...` lines.
//
// ToMDL is a back-compat alias for ToMDLStatement(false); call ToMDLStatement
// directly when the prefix must be `create or modify` (e.g. DESCRIBE output).
func (m *EntityModel) ToMDL() string {
	return m.ToMDLStatement(false)
}

// ToMDLStatement renders the canonical entity as deterministic MDL text. When
// createOrModify is true the statement begins with `create or modify` so the
// output is idempotent on re-execution (used by DESCRIBE); otherwise it
// begins with `create`. The prefix is injected at the statement line — never
// via post-hoc string substitution — so documentation blocks that happen to
// contain the word "create" are preserved verbatim.
func (m *EntityModel) ToMDLStatement(createOrModify bool) string {
	var sb strings.Builder
	if m.Documentation != "" {
		fmt.Fprintf(&sb, "/**\n * %s\n */\n", m.Documentation)
	}
	if m.Position != nil {
		fmt.Fprintf(&sb, "@Position(%d, %d)\n", m.Position.X, m.Position.Y)
	}
	kindStr := kindToMDL(m.Kind)
	prefix := "create"
	if createOrModify {
		prefix = "create or modify"
	}
	if m.Extends != nil {
		fmt.Fprintf(&sb, "%s %s entity %s extends %s (\n", prefix, kindStr, m.Name, m.Extends)
	} else {
		fmt.Fprintf(&sb, "%s %s entity %s (\n", prefix, kindStr, m.Name)
	}
	for i, attr := range m.Attributes {
		if attr.Documentation != "" {
			fmt.Fprintf(&sb, "  /** %s */\n", attr.Documentation)
		}
		comma := ","
		if i == len(m.Attributes)-1 {
			comma = ""
		}
		fmt.Fprintf(&sb, "  %s: %s%s%s\n", attr.Name, dataTypeToMDL(attr.Type), constraintsToMDL(attr), comma)
	}
	sb.WriteString(")")
	// Note: IndexModel.Name populated via Hydrate may be a UUID-formatted
	// DataStorageGuid (e.g. "d3f9a8b2-1234-...") which contains hyphens and is
	// not a valid MDL IDENTIFIER. This causes DESCRIBE output to be
	// non-re-parseable for entities with named indexes. Tracked as a known gap;
	// full fix requires storing a user-visible index name in EntityModel.
	for _, idx := range m.Indexes {
		cols := make([]string, 0, len(idx.Columns))
		for _, col := range idx.Columns {
			if col.Ascending {
				cols = append(cols, col.Name)
			} else {
				cols = append(cols, col.Name+" desc")
			}
		}
		if idx.Name != "" {
			fmt.Fprintf(&sb, "\nindex %s (%s)", idx.Name, strings.Join(cols, ", "))
		} else {
			fmt.Fprintf(&sb, "\nindex (%s)", strings.Join(cols, ", "))
		}
	}
	for _, eh := range m.EventHandlers {
		paramStr := "()"
		if eh.PassEventObject {
			paramStr = "($currentObject)"
		}
		options := ""
		if eh.RaiseErrorOnFalse && strings.EqualFold(eh.Moment, "Before") {
			options = " raise error"
		}
		fmt.Fprintf(&sb, "\non %s %s call %s%s%s",
			strings.ToLower(eh.Moment), strings.ToLower(eh.Event),
			eh.Microflow, paramStr, options)
	}

	// System members (after event handlers).
	if len(m.SystemMembers) > 0 {
		fmt.Fprintf(&sb, "\nsystem members (%s)", strings.Join(m.SystemMembers, ", "))
	}

	// OQL body for view entities.
	if m.Kind == EntityView && m.OQL != "" {
		sb.WriteString(" as (\n")
		for _, line := range strings.Split(m.OQL, "\n") {
			fmt.Fprintf(&sb, "  %s\n", line)
		}
		sb.WriteString(")")
	}
	return sb.String()
}

func kindToMDL(k EntityKind) string {
	switch k {
	case EntityNonPersistent:
		return "non-persistent"
	case EntityView:
		return "view"
	case EntityExternal:
		return "external"
	default:
		return "persistent"
	}
}

func dataTypeToMDL(dt canonical.DataType) string {
	switch dt.Kind {
	case canonical.KindString:
		if dt.Length > 0 {
			return fmt.Sprintf("String(%d)", dt.Length)
		}
		return "String"
	case canonical.KindInteger:
		return "Integer"
	case canonical.KindLong:
		return "Long"
	case canonical.KindDecimal:
		if dt.Precision > 0 {
			return fmt.Sprintf("Decimal(%d, %d)", dt.Precision, dt.Scale)
		}
		return "Decimal"
	case canonical.KindBoolean:
		return "Boolean"
	case canonical.KindDateTime:
		return "DateTime"
	case canonical.KindBinary:
		return "Binary"
	case canonical.KindAutoNumber:
		return "AutoNumber"
	case canonical.KindEnumRef, canonical.KindEntityRef, canonical.KindUnresolvedRef:
		return dt.Ref
	case canonical.KindListOf:
		return "List of " + dt.Ref
	default:
		return "Unknown"
	}
}

func constraintsToMDL(attr AttributeModel) string {
	var sb strings.Builder
	if attr.NotNull {
		sb.WriteString(" not null")
		if attr.NotNullError != "" {
			fmt.Fprintf(&sb, " error '%s'", strings.ReplaceAll(attr.NotNullError, "'", "''"))
		}
	}
	if attr.Unique {
		sb.WriteString(" unique")
		if attr.UniqueError != "" {
			fmt.Fprintf(&sb, " error '%s'", strings.ReplaceAll(attr.UniqueError, "'", "''"))
		}
	}
	if attr.HasDefault {
		fmt.Fprintf(&sb, " default %s", attr.DefaultValue)
	}
	if attr.Calculated && attr.CalculatedMicroflow != nil {
		fmt.Fprintf(&sb, " calculated by %s", attr.CalculatedMicroflow)
	}
	return sb.String()
}
