// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

type UnitDoc struct {
	QualifiedName string
	UnitType      string
	Doc           bson.D
}

func ReadAllUnits(mprPath string) ([]UnitDoc, error) {
	r, err := mmpr.OpenWithOptions(mprPath, mmpr.OpenOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("bsoncompare: open %s: %w", mprPath, err)
	}
	defer r.Close()

	infos, err := r.ListRawUnits("")
	if err != nil {
		return nil, fmt.Errorf("bsoncompare: list units: %w", err)
	}

	out := make([]UnitDoc, 0, len(infos))
	for _, info := range infos {
		var doc bson.D
		if err := bson.Unmarshal(info.Contents, &doc); err != nil {
			continue
		}
		out = append(out, UnitDoc{
			QualifiedName: info.QualifiedName,
			UnitType:      info.Type,
			Doc:           doc,
		})
	}
	return out, nil
}
