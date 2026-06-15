package mxgraph

import "testing"

func TestDirectionValues(t *testing.T) {
	if Outbound != 0 || Inbound != 1 || Both != 2 {
		t.Error("unexpected Direction iota values")
	}
}

func TestEventTypeValues(t *testing.T) {
	if NodeCreated != 0 || NodeUpdated != 1 || NodeDeleted != 2 || EdgeCreated != 3 || EdgeDeleted != 4 {
		t.Error("unexpected EventType iota values")
	}
}
