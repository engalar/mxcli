// SPDX-License-Identifier: Apache-2.0

// Package golden provides BSON golden-test infrastructure.
//
// Each GoldenEntry holds a Studio Pro-generated .bson file and a Builder
// that constructs an equivalent element tree from gen types. The test
// runner (see modelsdk/codec/golden_test.go) encodes the tree and
// compares the output byte-for-byte with the golden file.
//
// Adding a new golden:
//
//  1. Export a .bson from Studio Pro (or from /tmp/minimal.mpr) into
//     testdata/golden/.
//  2. Write a Builder function in this package that constructs the
//     equivalent element tree.
//  3. Register the entry in registry.go.
//  4. Run: go test ./modelsdk/codec/ -run TestGolden

package golden

import "github.com/mendixlabs/mxcli/modelsdk/element"

// GoldenEntry describes one golden BSON test case.
type GoldenEntry struct {
	// Name is the test subtest name (e.g. "Nanoflow").
	Name string

	// Source documents where the golden came from.
	Source string

	// BSON is the raw golden bytes (from testdata/golden/).
	BSON []byte

	// Builder constructs the equivalent element tree. The tree is
	// encoded by the test runner and compared against BSON.
	Builder func() element.Element

	// SkipFields lists BSON paths whose values are not compared.
	// The path format is a dot-separated key traversal with array
	// indices in brackets, e.g. "$.ObjectCollection.Objects[4].$ID".
	// By default all $ID fields are ignored.
	SkipFields []string
}
