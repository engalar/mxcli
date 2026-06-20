// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	mprrepos "github.com/mendixlabs/mxcli/mdl/backend/mpr/repos"
	"github.com/mendixlabs/mxcli/mdl/repos"
)

// PersistentBackend adapters — provide repo accessors used by the linter
// and daemon. Routes through the concrete writer.

func (b *MprBackend) Microflows() repos.MicroflowRepository {
	if b.writer == nil {
		return nil
	}
	return mprrepos.NewMicroflowRepository(b.writer)
}

func (b *MprBackend) Nanoflows() repos.NanoflowRepository {
	if b.writer == nil {
		return nil
	}
	return mprrepos.NewNanoflowRepository(b.writer)
}

func (b *MprBackend) Security() repos.SecurityRepository {
	if b.writer == nil {
		return nil
	}
	return mprrepos.NewSecurityRepository(b.writer)
}

func (b *MprBackend) JavaActions() repos.JavaActionRepository {
	if b.writer == nil {
		return nil
	}
	return mprrepos.NewJavaActionRepository(b.writer)
}

func (b *MprBackend) JavaScriptActions() repos.JavaScriptActionRepository {
	if b.writer == nil {
		return nil
	}
	return mprrepos.NewJavaScriptActionRepository(b.writer)
}

func (b *MprBackend) DomainModels() repos.DomainModelRepository {
	if b.writer == nil {
		return nil
	}
	return mprrepos.NewDomainModelRepository(b.writer)
}

func (b *MprBackend) Workflows() repos.WorkflowRepository {
	if b.writer == nil {
		return nil
	}
	return mprrepos.NewWorkflowRepository(b.writer)
}

func (b *MprBackend) Pages() repos.PageRepository {
	if b.writer == nil {
		return nil
	}
	return mprrepos.NewPageRepository(b.writer)
}

func (b *MprBackend) Layouts() repos.LayoutRepository {
	if b.writer == nil {
		return nil
	}
	return mprrepos.NewLayoutRepository(b.writer)
}

func (b *MprBackend) Snippets() repos.SnippetRepository {
	if b.writer == nil {
		return nil
	}
	return mprrepos.NewSnippetRepository(b.writer)
}
