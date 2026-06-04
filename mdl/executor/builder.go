// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"io"
	"os"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/diaglog"
)

// BackendIface is an alias for backend.FullBackend exported so external
// callers can name the factory function type without importing backend directly.
type BackendIface = backend.FullBackend

// Builder assembles an Executor with a fluent API.
// Call Build() to start, chain option methods, then Create() to get the Executor.
type Builder struct {
	out      io.Writer
	backend  backend.FullBackend
	factory  BackendFactory
	progress io.Writer
	logger   *diaglog.Logger
	format   OutputFormat
	quiet    bool
}

// Build returns a new Builder. Output defaults to os.Stdout.
func Build() *Builder {
	return &Builder{out: os.Stdout}
}

// Out sets the stdout writer (table results, DESCRIBE output, etc.).
func (b *Builder) Out(w io.Writer) *Builder { b.out = w; return b }

// WithBackend installs an already-connected backend (mock or persistent daemon).
// Takes precedence over WithFactory if both are set.
func (b *Builder) WithBackend(be backend.FullBackend) *Builder { b.backend = be; return b }

// WithFactory sets a lazy factory invoked on the first CONNECT statement.
func (b *Builder) WithFactory(f BackendFactory) *Builder { b.factory = f; return b }

// ProgressOut sets the writer for real-time progress messages.
// Defaults to os.Stderr. Wire to a "progress" frame writer in daemon mode.
func (b *Builder) ProgressOut(w io.Writer) *Builder { b.progress = w; return b }

// WithLogger sets the diagnostics logger.
func (b *Builder) WithLogger(l *diaglog.Logger) *Builder { b.logger = l; return b }

// Format sets the output format (FormatTable or FormatJSON).
func (b *Builder) Format(f OutputFormat) *Builder { b.format = f; return b }

// Quiet suppresses connection/status messages.
func (b *Builder) Quiet() *Builder { b.quiet = true; return b }

// Create assembles and returns the configured Executor.
// The caller owns the lifecycle and must call Close().
func (b *Builder) Create() *Executor {
	e := New(b.out)
	if b.backend != nil {
		e.SetBackend(b.backend)
	} else if b.factory != nil {
		e.SetBackendFactory(b.factory)
	}
	if b.progress != nil {
		e.SetProgressOut(b.progress)
	}
	if b.logger != nil {
		e.SetLogger(b.logger)
	}
	if b.format != "" {
		e.SetFormat(b.format)
	}
	e.SetQuiet(b.quiet)
	return e
}
