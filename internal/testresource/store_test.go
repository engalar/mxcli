package testresource

import (
	"testing"
)

func TestStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	p := Profile{
		Name:       "TestSomething",
		HeapDelta:  42,
		CPUTimeMs:  10.5,
		ReadBytes:  1000,
		WriteBytes: 500,
	}

	if err := s.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, ok := s.Load("TestSomething")
	if !ok {
		t.Fatal("Load returned false")
	}
	if loaded.HeapDelta != 42 {
		t.Errorf("HeapDelta = %d, want 42", loaded.HeapDelta)
	}
}

func TestStore_Compare_DetectsDelta(t *testing.T) {
	baseline := Profile{Name: "T", HeapDelta: 1000, ReadBytes: 50000}
	current := Profile{Name: "T", HeapDelta: 1500, ReadBytes: 60000}

	diff, err := (&Store{}).Compare(baseline, current)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if diff.HeapDeltaPct < 49 || diff.HeapDeltaPct > 51 {
		t.Errorf("HeapDeltaPct = %.1f, want ~50", diff.HeapDeltaPct)
	}
}

func TestStore_List(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	s.Save(Profile{Name: "A", HeapDelta: 1})
	s.Save(Profile{Name: "B", HeapDelta: 2})

	profiles, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(profiles) != 2 {
		t.Errorf("List returned %d profiles, want 2", len(profiles))
	}
}
