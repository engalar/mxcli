// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"io"
	"os"

	"github.com/mendixlabs/mxcli/mdl/backend"
)

// BackendIface is an alias for backward compatibility.
type BackendIface = backend.FullBackend

// Builder assembles an Executor with a fluent API.
type Builder struct {
	out     io.Writer
	backend backend.FullBackend
	factory BackendFactory
	quiet   bool
}

func Build() *Builder {
	return &Builder{out: os.Stdout}
}

func (b *Builder) Out(w io.Writer) *Builder                     { b.out = w; return b }
func (b *Builder) WithBackend(be backend.FullBackend) *Builder   { b.backend = be; return b }
func (b *Builder) WithFactory(f BackendFactory) *Builder         { b.factory = f; return b }
func (b *Builder) Quiet() *Builder                               { b.quiet = true; return b }

func (b *Builder) Create() *Executor {
	e := New(b.out)
	if b.backend != nil {
		e.SetBackend(b.backend)
	} else if b.factory != nil {
		e.SetBackendFactory(b.factory)
	}
	e.SetQuiet(b.quiet)
	return e
}
