package widgets

import (
	"testing"
)

func TestGetTemplateFullBSON_ColumnNestedKeyOrder(t *testing.T) {
	counter := 0
	idGen := func() string { counter++; return idForCounter(counter) }

	_, _, propIDs, _, _, err := GetTemplateFullBSON("com.mendix.widget.web.datagrid.Datagrid", idGen, "")
	if err != nil {
		t.Skipf("datagrid template not found: %v", err)
	}
	if propIDs == nil {
		t.Skip("no datagrid template")
	}

	cols, ok := propIDs["columns"]
	if !ok {
		t.Fatal("datagrid template has no `columns` property")
	}
	if len(cols.NestedPropertyIDs) == 0 {
		t.Fatal("columns has no NestedPropertyIDs (object-list schema missing)")
	}
	if len(cols.NestedKeyOrder) != len(cols.NestedPropertyIDs) {
		t.Fatalf("NestedKeyOrder has %d keys, want %d (one per nested property) — order not captured",
			len(cols.NestedKeyOrder), len(cols.NestedPropertyIDs))
	}

	wantPrefix := []string{"showContentAs", "attribute", "content", "dynamicText"}
	for i, want := range wantPrefix {
		if cols.NestedKeyOrder[i] != want {
			t.Errorf("NestedKeyOrder[%d] = %q, want %q (full: %v)", i, cols.NestedKeyOrder[i], want, cols.NestedPropertyIDs)
		}
	}

	for _, k := range cols.NestedKeyOrder {
		if _, ok := cols.NestedPropertyIDs[k]; !ok {
			t.Errorf("NestedKeyOrder key %q is not in NestedPropertyIDs", k)
		}
	}
}
