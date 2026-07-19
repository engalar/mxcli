package memory

import "testing"

func TestNewMemoryMPR(t *testing.T) {
	mpr, err := New("11.12.1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer mpr.Close()
	if mpr.DB == nil {
		t.Fatal("DB is nil")
	}
	if mpr.Reader == nil {
		t.Fatal("Reader is nil")
	}
	if mpr.Writer == nil {
		t.Fatal("Writer is nil")
	}
}

func TestMemoryMPR_WriteAndRead(t *testing.T) {
	mpr, err := New("11.12.1")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer mpr.Close()

	unitID := "00000000-0000-0000-0000-000000000001"
	contents := []byte("hello bson")

	err = mpr.Writer.InsertUnit(unitID, unitID, "Documents", "Microflows$Test", contents)
	if err != nil {
		t.Fatalf("InsertUnit: %v", err)
	}
	got, err := mpr.Reader.GetRawUnitBytes(unitID)
	if err != nil {
		t.Fatalf("GetRawUnitBytes: %v", err)
	}
	if string(got) != string(contents) {
		t.Fatalf("got %q, want %q", string(got), string(contents))
	}
}
