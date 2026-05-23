// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"fmt"
	"strings"

	"github.com/mendixlabs/mxcli/mdl/model"
)

// ToMDL renders the canonical entity as deterministic MDL text. Output is a
// `create ... entity Module.Name (...)` block followed by zero or more
// `index ...` lines.
func (m *EntityModel) ToMDL() string {
	var sb strings.Builder
	if m.Documentation != "" {
		fmt.Fprintf(&sb, "/**\n * %s\n */\n", m.Documentation)
	}
	if m.Position != nil {
		fmt.Fprintf(&sb, "@Position(%d, %d)\n", m.Position.X, m.Position.Y)
	}
	kindStr := kindToMDL(m.Kind)
	if m.Extends != nil {
		fmt.Fprintf(&sb, "create %s entity %s extends %s (\n", kindStr, m.Name, m.Extends)
	} else {
		fmt.Fprintf(&sb, "create %s entity %s (\n", kindStr, m.Name)
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

func dataTypeToMDL(dt model.DataType) string {
	switch dt.Kind {
	case model.KindString:
		if dt.Length > 0 {
			return fmt.Sprintf("String(%d)", dt.Length)
		}
		return "String"
	case model.KindInteger:
		return "Integer"
	case model.KindLong:
		return "Long"
	case model.KindDecimal:
		if dt.Precision > 0 {
			return fmt.Sprintf("Decimal(%d, %d)", dt.Precision, dt.Scale)
		}
		return "Decimal"
	case model.KindBoolean:
		return "Boolean"
	case model.KindDateTime:
		return "DateTime"
	case model.KindBinary:
		return "Binary"
	case model.KindAutoNumber:
		return "AutoNumber"
	case model.KindEnumRef, model.KindEntityRef, model.KindUnresolvedRef:
		return dt.Ref
	case model.KindListOf:
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
