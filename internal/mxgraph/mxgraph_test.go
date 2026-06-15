package mxgraph

import "testing"

func TestDirectionValues(t *testing.T) {
	if Outbound != 0 || Inbound != 1 || Both != 2 {
		t.Error("unexpected Direction iota values")
	}
}
