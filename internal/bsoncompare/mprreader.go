// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import (
	"fmt"
	"hash/fnv"
	"sync"

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
	Raw           []byte   // raw BSON bytes
	ContentHash   uint64   // FNV-1a hash of raw BSON; fast diff skip when golden==actual
}

func ReadAllUnits(mprPath string) ([]UnitDoc, error) {
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
		h := fnv.New64a()
		h.Write(info.Contents)
		out = append(out, UnitDoc{
			QualifiedName: info.QualifiedName,
			UnitType:      info.Type,
			Raw:           info.Contents,
			ContentHash:   h.Sum64(),
		})
	}
	readAllCache.Store(mprPath, &cachedResult{units: out})
	return out, nil
}
