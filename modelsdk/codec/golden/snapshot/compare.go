package snapshot

import (
	"fmt"

	"github.com/mendixlabs/mxcli/modelsdk/codec/golden"
)

type CompareMode int

const (
	CompareStructure CompareMode = iota
	CompareBinary
)

type CompareResult struct {
	Mode  CompareMode
	Diffs []golden.Diff
}

func CompareCanonical(gotBSON []byte, g *UnitSnapshot, mode CompareMode) (*CompareResult, error) {
	switch mode {
	case CompareStructure:
		return compareStructure(gotBSON, g)
	case CompareBinary:
		return compareBinary(gotBSON, g)
	default:
		return nil, fmt.Errorf("unknown compare mode: %d", mode)
	}
}

func compareStructure(gotBSON []byte, g *UnitSnapshot) (*CompareResult, error) {
	gotCanonical, err := ToCanonicalJSON(gotBSON)
	if err != nil {
		return nil, err
	}
	gotRaw, err := FromCanonicalJSON(gotCanonical)
	if err != nil {
		return nil, err
	}
	goldenRaw, err := FromCanonicalJSON(g.Canonical)
	if err != nil {
		return nil, err
	}
	diffs := golden.CompareBSON(gotRaw, goldenRaw, nil)
	return &CompareResult{Mode: CompareStructure, Diffs: diffs}, nil
}

func compareBinary(gotBSON []byte, g *UnitSnapshot) (*CompareResult, error) {
	diffs := golden.CompareBSON(gotBSON, g.RawBSON(), nil)
	return &CompareResult{Mode: CompareBinary, Diffs: diffs}, nil
}
