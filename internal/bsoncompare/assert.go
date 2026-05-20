// SPDX-License-Identifier: Apache-2.0
package bsoncompare

import (
	"fmt"
	"strings"
)

type TB interface {
	Helper()
	Errorf(format string, args ...any)
}

type Matcher interface {
	Match(diffs []UnitDiff) error
}

var claimed = map[string]bool{}

func ExpectAdded(qualifiedName string) Matcher {
	return expectAdded{name: qualifiedName}
}

type expectAdded struct{ name string }

func (e expectAdded) Match(diffs []UnitDiff) error {
	for _, d := range diffs {
		if d.QualifiedName == e.name && d.Kind == DiffAdded {
			claimed[e.name] = true
			return nil
		}
	}
	return fmt.Errorf("expected unit %q to be added, but it was not found in diffs", e.name)
}

func ExpectNoOtherChanges() Matcher { return expectNoOtherChanges{} }

type expectNoOtherChanges struct{}

func (expectNoOtherChanges) Match(diffs []UnitDiff) error {
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

func AssertEqual(t TB, aPath, bPath string, opts Options, matchers ...Matcher) {
	t.Helper()
	for k := range claimed {
		delete(claimed, k)
	}
	diffs, err := Compare(aPath, bPath, opts)
	if err != nil {
		t.Errorf("bsoncompare: Compare failed: %v", err)
		return
	}
	var errs []string
	for _, m := range matchers {
		if err := m.Match(diffs); err != nil {
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
