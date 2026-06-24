package testresource

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type ProfileSaver interface {
	Save(Profile) error
}

type ProfileLoader interface {
	Load(name string) (Profile, bool)
}

type Store struct {
	dir string
}

func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

func (s *Store) filePath(name string) string {
	safe := strings.ReplaceAll(name, "/", "_")
	safe = strings.ReplaceAll(safe, " ", "_")
	return filepath.Join(s.dir, safe+".json")
}

func (s *Store) Save(p Profile) error {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return fmt.Errorf("create profile dir: %w", err)
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profile: %w", err)
	}
	return os.WriteFile(s.filePath(p.Name), data, 0644)
}

func (s *Store) Load(name string) (Profile, bool) {
	data, err := os.ReadFile(s.filePath(name))
	if err != nil {
		return Profile{}, false
	}
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return Profile{}, false
	}
	p.Name = name
	return p, true
}

func (s *Store) List() ([]Profile, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read profile dir: %w", err)
	}
	var profiles []Profile
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		if p, ok := s.Load(name); ok {
			profiles = append(profiles, p)
		}
	}
	return profiles, nil
}

type Diff struct {
	Name          string
	HeapDeltaPct  float64
	CPUTimePct    float64
	ReadBytesPct  float64
	WriteBytesPct float64
}

func (s *Store) Compare(baseline, current Profile) (Diff, error) {
	return Diff{
		Name:          baseline.Name,
		HeapDeltaPct:  pctDiff(baseline.HeapDelta, current.HeapDelta),
		CPUTimePct:    pctDiffF(baseline.CPUTimeMs, current.CPUTimeMs),
		ReadBytesPct:  pctDiff(baseline.ReadBytes, current.ReadBytes),
		WriteBytesPct: pctDiff(baseline.WriteBytes, current.WriteBytes),
	}, nil
}

func pctDiff(baseline, current int64) float64 {
	if baseline == 0 && current == 0 {
		return 0
	}
	if baseline == 0 {
		return 100.0
	}
	return math.Round(float64(current-baseline) / float64(baseline) * 100)
}

func pctDiffF(baseline, current float64) float64 {
	if baseline == 0 && current == 0 {
		return 0
	}
	if baseline == 0 {
		return 100.0
	}
	return math.Round((current - baseline) / baseline * 100)
}
