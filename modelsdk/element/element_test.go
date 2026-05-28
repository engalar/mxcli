package element

import (
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestBase_SetID_ID(t *testing.T) {
	var b Base
	if got := b.ID(); got != "" {
		t.Fatalf("expected empty ID, got %q", got)
	}
	b.SetID("abc-123")
	if got := b.ID(); got != "abc-123" {
		t.Fatalf("expected %q, got %q", "abc-123", got)
	}
}

func TestBase_SetTypeName_TypeName(t *testing.T) {
	var b Base
	if got := b.TypeName(); got != "" {
		t.Fatalf("expected empty TypeName, got %q", got)
	}
	b.SetTypeName("DomainModels$DomainModel")
	if got := b.TypeName(); got != "DomainModels$DomainModel" {
		t.Fatalf("expected %q, got %q", "DomainModels$DomainModel", got)
	}
}

func TestBase_IsDirty(t *testing.T) {
	var b Base
	if b.IsDirty() {
		t.Fatal("expected IsDirty false on zero-value Base")
	}
	b.MarkDirty(0)
	if !b.IsDirty() {
		t.Fatal("expected IsDirty true after MarkDirty(0)")
	}

	var b2 Base
	b2.MarkDirty(63)
	if !b2.IsDirty() {
		t.Fatal("expected IsDirty true after MarkDirty(63)")
	}
	bits := b2.DirtyBits()
	if len(bits) < 1 || bits[0]&(1<<63) == 0 {
		t.Fatalf("unexpected dirty bits: bit 63 not set")
	}
}

func TestMarkDirtyBubbles(t *testing.T) {
	var parent, child Base
	child.SetContainer(&parent)

	child.MarkDirty(0)

	if !child.IsDirty() {
		t.Fatal("child should be dirty after MarkDirty(0)")
	}
	if !parent.childDirty {
		t.Fatal("parent should have childDirty set after child.MarkDirty(0)")
	}
}

func TestMarkChildDirtyIdempotent(t *testing.T) {
	var b Base
	b.MarkChildDirty()
	b.MarkChildDirty()
	if !b.childDirty {
		t.Fatal("expected childDirty to be true")
	}
	// dirty bitmap should be empty — childDirty is a separate flag
	for _, w := range b.dirty {
		if w != 0 {
			t.Fatalf("expected empty dirty bitmap, got non-zero word")
		}
	}
}

func TestDeepBubble(t *testing.T) {
	var root, mid, leaf Base
	mid.SetContainer(&root)
	leaf.SetContainer(&mid)

	leaf.MarkDirty(0)

	if !mid.childDirty {
		t.Fatal("mid should have childDirty set")
	}
	if !root.childDirty {
		t.Fatal("root should have childDirty set")
	}
}

func TestBase_SetRaw_Raw(t *testing.T) {
	var b Base
	if b.Raw() != nil {
		t.Fatal("expected nil Raw on zero-value Base")
	}
	raw := bson.Raw([]byte{5, 0, 0, 0, 0}) // minimal valid BSON doc
	b.SetRaw(raw)
	if got := b.Raw(); string(got) != string(raw) {
		t.Fatalf("Raw mismatch: got %v, want %v", got, raw)
	}
}
