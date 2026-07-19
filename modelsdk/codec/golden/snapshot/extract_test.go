package snapshot

import (
	"os"
	"testing"
)

func TestExtractNanoflowFromMinimal(t *testing.T) {
	if _, err := os.Stat("/tmp/minimal.mpr"); os.IsNotExist(err) {
		t.Skip("/tmp/minimal.mpr not found")
	}
	snaps, err := ExtractFromMPR("/tmp/minimal.mpr", "Microflows$Nanoflow")
	if err != nil {
		t.Fatalf("ExtractFromMPR: %v", err)
	}
	if len(snaps) == 0 {
		t.Fatal("no nanoflow units found")
	}
	for _, s := range snaps {
		t.Logf("Found %s: %d bytes canonical", s.Type, len(s.Canonical))
	}
}
