// SPDX-License-Identifier: Apache-2.0

package repos

import (
	"github.com/mendixlabs/mxcli/model"
	genJA "github.com/mendixlabs/mxcli/modelsdk/gen/javaactions"
	genJSA "github.com/mendixlabs/mxcli/modelsdk/gen/javascriptactions"
)

// JavaActionReader queries Java actions. Implementations decode raw BSON
// via codec.Decoder; freshly-decoded gen objects are returned by
// value-pointer (the caller may mutate freely without affecting the cache).
type JavaActionReader interface {
	Get(id model.ID) (*genJA.JavaAction, error)
	List(moduleID model.ID) ([]*genJA.JavaAction, error)
	ListAll() ([]*genJA.JavaAction, error)
	FindByQualifiedName(qn string) (*genJA.JavaAction, error)

	// GetContainerUUID returns the parent container UUID (folder or
	// module ID) of a Java action unit. Codec-decoded gen objects do
	// not carry container linkage, so callers retrieve it from the
	// MPR Unit table by UnitID.
	GetContainerUUID(id model.ID) (model.ID, error)
}

// JavaActionWriter creates/updates/deletes Java actions. Phase A
// implementations stub these; Phase D (write path) fills them in.
type JavaActionWriter interface {
	Create(parentUUID string, containmentName string, ja *genJA.JavaAction) error
	Update(ja *genJA.JavaAction) error
	Delete(id model.ID) error
}

type JavaActionRepository interface {
	JavaActionReader
	JavaActionWriter
}

// JavaScriptActionReader mirrors JavaActionReader for JavaScript actions.
// MDL has no `create javascript action` syntax today, so no writer
// counterpart is required.
type JavaScriptActionReader interface {
	Get(id model.ID) (*genJSA.JavaScriptAction, error)
	List(moduleID model.ID) ([]*genJSA.JavaScriptAction, error)
	ListAll() ([]*genJSA.JavaScriptAction, error)
	FindByQualifiedName(qn string) (*genJSA.JavaScriptAction, error)
	GetContainerUUID(id model.ID) (model.ID, error)
}

type JavaScriptActionRepository interface {
	JavaScriptActionReader
}
