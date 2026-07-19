package snapshot

import (
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type UnitSnapshot struct {
	Type      string          `json:"type"`
	Canonical json.RawMessage `json:"canonical"`
	rawBSON   []byte          `json:"-"`
}

func ToCanonicalJSON(raw []byte) (json.RawMessage, error) {
	var doc bson.D
	if err := bson.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal bson: %w", err)
	}
	extJSON, err := bson.MarshalExtJSON(doc, true, false)
	if err != nil {
		return nil, fmt.Errorf("marshal extjson: %w", err)
	}
	return extJSON, nil
}

func FromCanonicalJSON(data []byte) ([]byte, error) {
	var doc bson.D
	if err := bson.UnmarshalExtJSON(data, true, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal extjson: %w", err)
	}
	raw, err := bson.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshal bson: %w", err)
	}
	return raw, nil
}

func NewUnitSnapshot(unitType string, rawBSON []byte) (*UnitSnapshot, error) {
	canonical, err := ToCanonicalJSON(rawBSON)
	if err != nil {
		return nil, err
	}
	return &UnitSnapshot{
		Type:      unitType,
		Canonical: canonical,
		rawBSON:   rawBSON,
	}, nil
}

func (s *UnitSnapshot) RawBSON() []byte { return s.rawBSON }
