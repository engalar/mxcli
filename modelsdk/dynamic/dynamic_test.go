package dynamic_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/dynamic"
)

func TestPropertyKindConstantsAreDistinct(t *testing.T) {
	kinds := []dynamic.PropertyKind{
		dynamic.KindString,
		dynamic.KindBool,
		dynamic.KindInt32,
		dynamic.KindFloat64,
		dynamic.KindPart,
		dynamic.KindPartList,
		dynamic.KindByID,
		dynamic.KindStringList,
		dynamic.KindBinary,
		dynamic.KindUnknown,
	}
	seen := map[dynamic.PropertyKind]bool{}
	for _, k := range kinds {
		if seen[k] {
			t.Errorf("duplicate kind value: %v", k)
		}
		seen[k] = true
	}
}
