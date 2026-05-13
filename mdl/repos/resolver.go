// SPDX-License-Identifier: Apache-2.0

package repos

import "github.com/mendixlabs/mxcli/model"

// QualifiedNameResolver answers "what kind of element is this name?"
// without recursing through domain repositories — it queries the
// underlying SQLite catalog directly via *mmpr.Reader.
type QualifiedNameResolver interface {
	ModuleNameByID(id model.ID) (string, error)
	ResolveQualifiedName(qn string) (id model.ID, kind string, err error)
}
