// SPDX-License-Identifier: Apache-2.0

// Package linter provides an extensible linting framework for Mendix projects.
package linter

import (
	"context"
)

// Severity indicates how serious a violation is.
type Severity int

const (
	SeverityHint Severity = iota
	SeverityInfo
	SeverityWarning
	SeverityError
)

// String returns the string representation of the severity.
func (s Severity) String() string {
	switch s {
	case SeverityHint:
		return "hint"
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}

// Symbol returns the symbol used in text output.
func (s Severity) Symbol() string {
	switch s {
	case SeverityHint:
		return "💡"
	case SeverityInfo:
		return "ℹ"
	case SeverityWarning:
		return "⚠"
	case SeverityError:
		return "✗"
	default:
		return "?"
	}
}

// Violation represents a single lint violation.
type Violation struct {
	RuleID     string
	Severity   Severity
	Message    string
	Location   Location
	Suggestion string
	Extra      any // Extra data for auto-fix (set by rule, used by fixer)
}

// Location identifies where a violation occurred.
type Location struct {
	Module       string // e.g., "Sales"
	DocumentType string // "entity", "microflow", "page"
	DocumentName string // e.g., "Customer"
	DocumentID   string // UUID
}

// QualifiedName returns the full qualified name of the location.
func (l Location) QualifiedName() string {
	if l.Module == "" {
		return l.DocumentName
	}
	return l.Module + "." + l.DocumentName
}

// Rule is the interface that all lint rules must implement.
type Rule interface {
	ID() string
	Name() string
	Description() string
	DefaultSeverity() Severity
	Category() string
	Check(ctx *LintContext) []Violation
}

// RuleConfig holds configuration for a specific rule.
type RuleConfig struct {
	Enabled  bool
	Severity Severity
	Options  map[string]any
}

// Linter orchestrates running lint rules against a project.
type Linter struct {
	ctx        *LintContext
	rules      []Rule
	configs    map[string]RuleConfig
	maxWorkers int
}

// New creates a new Linter with the given context.
func New(ctx *LintContext) *Linter {
	return &Linter{
		ctx:        ctx,
		rules:      []Rule{},
		configs:    make(map[string]RuleConfig),
		maxWorkers: 4,
	}
}

// AddRule adds a rule to the linter.
func (l *Linter) AddRule(rule Rule) {
	l.rules = append(l.rules, rule)
}

// ConfigureRule sets configuration for a specific rule.
func (l *Linter) ConfigureRule(ruleID string, config RuleConfig) {
	l.configs[ruleID] = config
}

// SetMaxWorkers sets the maximum number of parallel workers.
func (l *Linter) SetMaxWorkers(n int) {
	if n > 0 {
		l.maxWorkers = n
	}
}

// Rules returns all registered rules.
func (l *Linter) Rules() []Rule {
	return l.rules
}

// Run executes all enabled rules and returns the violations found.
func (l *Linter) Run(ctx context.Context) ([]Violation, error) {
	var allViolations []Violation

	// Run rules sequentially to avoid SQLite concurrency issues with in-memory db
	for _, rule := range l.rules {
		// Check if rule is enabled
		if config, ok := l.configs[rule.ID()]; ok && !config.Enabled {
			continue
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return allViolations, ctx.Err()
		default:
		}

		// Run the rule
		violations := rule.Check(l.ctx)

		// Apply configured severity if different from default
		if config, ok := l.configs[rule.ID()]; ok {
			for i := range violations {
				violations[i].Severity = config.Severity
			}
		}

		// Collect results
		allViolations = append(allViolations, violations...)
	}

	return allViolations, nil
}

// Summary holds counts of violations by severity.
type Summary struct {
	Errors   int
	Warnings int
	Infos    int
	Hints    int
	Total    int
}

// Summarize counts violations by severity.
func Summarize(violations []Violation) Summary {
	var s Summary
	for _, v := range violations {
		switch v.Severity {
		case SeverityError:
			s.Errors++
		case SeverityWarning:
			s.Warnings++
		case SeverityInfo:
			s.Infos++
		case SeverityHint:
			s.Hints++
		}
	}
	s.Total = len(violations)
	return s
}
