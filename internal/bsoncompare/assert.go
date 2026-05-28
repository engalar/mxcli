// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// TB is the subset of *testing.T used by AssertEqual.
type TB interface {
	Helper()
	Errorf(format string, args ...any)
}

// Matcher evaluates a slice of UnitDiff and returns an error if its expectation
// is not met. claimed tracks which units have been expected by prior matchers.
type Matcher interface {
	Match(diffs []UnitDiff, claimed map[string]bool) error
}

// ExpectAdded returns a Matcher that passes when the named unit appears as DiffAdded.
func ExpectAdded(qualifiedName string) Matcher {
	return expectAdded{name: qualifiedName}
}

type expectAdded struct{ name string }

func (e expectAdded) Match(diffs []UnitDiff, claimed map[string]bool) error {
	for _, d := range diffs {
		if d.QualifiedName == e.name && d.Kind == DiffAdded {
			claimed[e.name] = true
			return nil
		}
	}
	return fmt.Errorf("expected unit %q to be added, but it was not found in diffs", e.name)
}

// ExpectRemoved returns a Matcher that passes when the named unit appears as DiffRemoved.
func ExpectRemoved(qualifiedName string) Matcher {
	return expectRemoved{name: qualifiedName}
}

type expectRemoved struct{ name string }

func (e expectRemoved) Match(diffs []UnitDiff, claimed map[string]bool) error {
	for _, d := range diffs {
		if d.QualifiedName == e.name && d.Kind == DiffRemoved {
			claimed[e.name] = true
			return nil
		}
	}
	return fmt.Errorf("expected unit %q to be removed, but it was not found in diffs", e.name)
}

// ExpectChanged returns a Matcher that passes when the named unit appears as DiffChanged.
func ExpectChanged(qualifiedName string) Matcher {
	return expectChanged{name: qualifiedName}
}

type expectChanged struct{ name string }

func (e expectChanged) Match(diffs []UnitDiff, claimed map[string]bool) error {
	for _, d := range diffs {
		if d.QualifiedName == e.name && d.Kind == DiffChanged {
			claimed[e.name] = true
			return nil
		}
	}
	return fmt.Errorf("expected unit %q to be changed, but it was not found in diffs", e.name)
}

// WithUnitCheck returns a Matcher that finds the unit with the given
// qualified name in the diff list and runs check against its actual BSON
// document (the bPath / post-mutation side). The unit is claimed on success.
// Fails if the unit is not found in the diffs or check returns an error.
func WithUnitCheck(qualifiedName string, check func(bson.D) error) Matcher {
	return withUnitCheck{name: qualifiedName, check: check}
}

type withUnitCheck struct {
	name  string
	check func(bson.D) error
}

func (w withUnitCheck) Match(diffs []UnitDiff, claimed map[string]bool) error {
	for _, d := range diffs {
		if d.QualifiedName != w.name {
			continue
		}
		if d.ActualDoc == nil {
			return fmt.Errorf("WithUnitCheck %q: unit was removed (no actual doc)", w.name)
		}
		if err := w.check(d.ActualDoc); err != nil {
			return fmt.Errorf("WithUnitCheck %q: %w", w.name, err)
		}
		claimed[w.name] = true
		return nil
	}
	return fmt.Errorf("WithUnitCheck %q: unit not found in diffs", w.name)
}

// ExpectNoOtherChanges returns a Matcher that passes only when all remaining
// unclaimed UnitDiff entries are empty. Must be the last matcher.
func ExpectNoOtherChanges() Matcher { return expectNoOtherChanges{} }

type expectNoOtherChanges struct{}

func (expectNoOtherChanges) Match(diffs []UnitDiff, claimed map[string]bool) error {
	var unexpected []string
	for _, d := range diffs {
		if !claimed[d.QualifiedName] {
			unexpected = append(unexpected, fmt.Sprintf("%s (%s)", d.QualifiedName, d.Kind))
		}
	}
	if len(unexpected) > 0 {
		return fmt.Errorf("unexpected changes:\n  %s", strings.Join(unexpected, "\n  "))
	}
	return nil
}

// AssertEqual compares aPath and bPath MPRs and applies each matcher in order.
// claimed is local per call, so AssertEqual is safe for parallel tests.
func AssertEqual(t TB, aPath, bPath string, opts Options, matchers ...Matcher) {
	t.Helper()
	claimed := make(map[string]bool)
	diffs, err := Compare(aPath, bPath, opts)
	if err != nil {
		t.Errorf("bsoncompare: Compare failed: %v", err)
		return
	}
	var errs []string
	for _, m := range matchers {
		if err := m.Match(diffs, claimed); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		t.Errorf("bsoncompare assertion failed:\n%s\n%s",
			strings.Join(errs, "\n"),
			FormatDiff(diffs),
		)
	}
}
