// SPDX-License-Identifier: Apache-2.0

package backend

import "github.com/mendixlabs/mxcli/mdl/repos"

// PersistentBackend extends FullBackend with the repo-provider methods exposed
// by MprBackend. The daemon's noOpConnectBackend embeds this interface so that
// executor duck-type checks (microflowsRepoProvider etc.) succeed without
// importing the concrete *mprbackend.MprBackend type.
type PersistentBackend interface {
	FullBackend
	Microflows() repos.MicroflowRepository
	Nanoflows() repos.NanoflowRepository
	Security() repos.SecurityRepository
	JavaActions() repos.JavaActionRepository
	JavaScriptActions() repos.JavaScriptActionRepository
	DomainModels() repos.DomainModelRepository
	Workflows() repos.WorkflowRepository
	Pages() repos.PageRepository
	Layouts() repos.LayoutRepository
	Snippets() repos.SnippetRepository
}
