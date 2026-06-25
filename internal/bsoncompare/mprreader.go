// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import (
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/v2/bson"

	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// readAllCache caches ReadAllUnits results by mprPath to avoid re-reading
// the same MPR in tests that compare the same project multiple times.
var readAllCache sync.Map // mprPath → *cachedResult

type cachedResult struct {
	units []UnitDoc
	err   error
}

type UnitDoc struct {
	QualifiedName string
	UnitType      string
	Doc           bson.D
}

func ReadAllUnits(mprPath string) ([]UnitDoc, error) {
	// Fast path: cached result.
	if cached, ok := readAllCache.Load(mprPath); ok {
		r := cached.(*cachedResult)
		return r.units, r.err
	}

	r, err := mmpr.OpenWithOptions(mprPath, mmpr.OpenOptions{ReadOnly: true})
	if err != nil {
		readAllCache.Store(mprPath, &cachedResult{err: err})
		return nil, fmt.Errorf("bsoncompare: open %s: %w", mprPath, err)
	}
	defer r.Close()

	infos, err := r.ListRawUnits("")
	if err != nil {
		readAllCache.Store(mprPath, &cachedResult{err: err})
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
	readAllCache.Store(mprPath, &cachedResult{units: out})
	return out, nil
}
