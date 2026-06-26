// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import (
	"fmt"
	"hash/fnv"
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
	// Raw is the raw BSON bytes. Preferred over Doc — avoids full decode
	// for unchanged units and ID-map building.
	Raw bson.Raw
	// Doc is lazily decoded from Raw on first call to Decode(). Nil until
	// first decode. Mutated only during the single-threaded Compare call.
	Doc         bson.D
	ContentHash uint64 // FNV-1a hash of raw BSON; fast diff skip when golden==actual
}

// Decode ensures Doc is populated from Raw. Safe to call multiple times.
func (u *UnitDoc) Decode() {
	if u.Doc != nil {
		return
	}
	// bson.Unmarshal into a zero bson.D appends elements.
	var doc bson.D
	if err := bson.Unmarshal(u.Raw, &doc); err != nil {
		return
	}
	u.Doc = doc
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
		h := fnv.New64a()
		h.Write(info.Contents)
		out = append(out, UnitDoc{
			QualifiedName: info.QualifiedName,
			UnitType:      info.Type,
			Raw:           bson.Raw(info.Contents),
			ContentHash:   h.Sum64(),
		})
	}
	readAllCache.Store(mprPath, &cachedResult{units: out})
	return out, nil
}
