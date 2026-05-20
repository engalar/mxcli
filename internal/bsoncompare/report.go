// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import (
	"fmt"
	"strings"
)

func FormatDiff(diffs []UnitDiff) string {
	if len(diffs) == 0 {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "bsoncompare: %d unit(s) differ\n", len(diffs))
	for _, ud := range diffs {
		fmt.Fprintf(&sb, "\n[%s] %s (%s)\n", strings.ToUpper(string(ud.Kind)), ud.QualifiedName, ud.UnitType)
		for _, fd := range ud.Fields {
			switch fd.Kind {
			case DiffChanged:
				fmt.Fprintf(&sb, "  ~ %s\n      - %s\n      + %s\n", fd.Path, fd.Golden, fd.Actual)
			case DiffAdded:
				fmt.Fprintf(&sb, "  + %s = %s\n", fd.Path, fd.Actual)
			case DiffRemoved:
				fmt.Fprintf(&sb, "  - %s = %s\n", fd.Path, fd.Golden)
			case DiffWarning:
				fmt.Fprintf(&sb, "  ? %s (unknown ref)\n", fd.Path)
			}
		}
	}
	return sb.String()
}
