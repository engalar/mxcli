package snapshot

import (
	"encoding/json"
	"fmt"
	"os"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

func WriteSnapshotToFile(snap *UnitSnapshot, path string) error {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func ReadSnapshotFromFile(path string) (*UnitSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	var snap UnitSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}
	// Recover raw BSON from canonical JSON so binary comparison works.
	snap.rawBSON, err = FromCanonicalJSON(snap.Canonical)
	if err != nil {
		return nil, fmt.Errorf("recover bson from canonical: %w", err)
	}
	return &snap, nil
}

func ExtractFromMPR(mprPath, unitType string) ([]*UnitSnapshot, error) {
	r, err := mmpr.Open(mprPath)
	if err != nil {
		return nil, fmt.Errorf("open mpr: %w", err)
	}
	defer r.Close()

	units, err := r.ListUnitsByType(unitType)
	if err != nil {
		return nil, fmt.Errorf("list units: %w", err)
	}

	var results []*UnitSnapshot
	for _, u := range units {
		if u.Type != unitType {
			continue
		}
		raw, err := r.GetRawUnitBytes(u.ID)
		if err != nil {
			continue
		}
		snap, err := NewUnitSnapshot(u.Type, raw)
		if err != nil {
			continue
		}
		results = append(results, snap)
	}
	return results, nil
}

func ExtractFromMPRToFile(mprPath, unitType, outputPath string) error {
	snapshots, err := ExtractFromMPR(mprPath, unitType)
	if err != nil {
		return err
	}
	if len(snapshots) == 0 {
		return fmt.Errorf("no units with type %q found", unitType)
	}
	data, err := json.MarshalIndent(snapshots[0], "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(outputPath, data, 0644)
}
