// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"github.com/mendixlabs/mxcli/internal/mxgraph"
	"github.com/mendixlabs/mxcli/mdl/types"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// BeginPageBuild initialises the per-page widget-type cache.  Call before building
// any widget BSON for a page, and call EndPageBuild when the page is complete.
func (b *MprBackend) BeginPageBuild() {
	b.widgetTypeCache = make(map[string]*widgetTypeCacheEntry)
}

// EndPageBuild clears the per-page widget-type cache so type-schema $IDs do not
// accidentally leak across different pages.
func (b *MprBackend) EndPageBuild() {
	b.widgetTypeCache = nil
}

// getWidgetTypeCacheEntry returns the cached type entry for widgetID, or nil.
func (b *MprBackend) getWidgetTypeCacheEntry(widgetID string) *widgetTypeCacheEntry {
	if b.widgetTypeCache == nil {
		return nil
	}
	return b.widgetTypeCache[widgetID]
}

// setWidgetTypeCacheEntry stores an entry in the per-page cache (no-op when cache is nil).
func (b *MprBackend) setWidgetTypeCacheEntry(widgetID string, entry *widgetTypeCacheEntry) {
	if b.widgetTypeCache != nil {
		b.widgetTypeCache[widgetID] = entry
	}
}

// New creates a new unconnected MprBackend. Call Connect(path) to open a project.
func New() *MprBackend {
	return &MprBackend{}
}

// NewFromPath opens path for read-write and returns a fully-wired MprBackend.
// Equivalent to Connect on a zero-value backend; useful in tests.
func NewFromPath(path string) (*MprBackend, error) {
	b := &MprBackend{}
	if err := b.Connect(path); err != nil {
		return nil, err
	}
	return b, nil
}

// ---------------------------------------------------------------------------
// ConnectionBackend
// ---------------------------------------------------------------------------

func (b *MprBackend) Connect(path string) error {
	r, err := modelsdkmpr.OpenWithOptions(path, modelsdkmpr.OpenOptions{ReadOnly: false})
	if err != nil {
		return err
	}
	mw := modelsdkmpr.NewWriterWithReader(r)
	b.reader = r
	b.msdkReader = r
	b.msdkWriter = mw
	b.writer = mw
	b.path = path

	// Eagerly create all sub-backends — no lazy init overhead on every call.
	b.modules = newModuleBackend(r)
	b.microflows = newMicroflowBackend(mw)
	b.workflows = newWorkflowBackend(mw)
	b.pages = newPageBackend(mw)
	b.java = newJavaBackend(mw)
	b.domainmodels = newDomainModelBackend(mw)
	b.security = newSecurityBackend(mw)
	b.folders = newFolderBackend(r)
	b.scheduledEvents = newScheduledEventBackend(r)
	b.enumerations = newEnumerationBackend(r)
	b.constants = newConstantBackend(r)
	b.rawUnits = newRawUnitBackend(r, mw)
	b.metadata = newMetadataBackend(r)
	b.mappings = newMappingBackend(r)
	b.subBackendsReady = true
	return nil
}

// initSubBackends is a no-op when subBackendsReady is true, which is set
// in Connect() after all sub-backends are created eagerly.
// Kept as a safeguard so that methods don't NPE when called before Connect().
func (b *MprBackend) initSubBackends() {
	if b.subBackendsReady || b.reader == nil {
		return
	}
	if b.modules == nil {
		b.modules = newModuleBackend(b.reader)
	}
	if b.folders == nil {
		b.folders = newFolderBackend(b.reader)
	}
	if b.scheduledEvents == nil {
		b.scheduledEvents = newScheduledEventBackend(b.reader)
	}
	if b.enumerations == nil {
		b.enumerations = newEnumerationBackend(b.reader)
	}
	if b.constants == nil {
		b.constants = newConstantBackend(b.reader)
	}
	if b.rawUnits == nil && b.msdkWriter != nil {
		if w, ok := b.msdkWriter.(*modelsdkmpr.Writer); ok {
			b.rawUnits = newRawUnitBackend(b.reader, w)
		}
	}
	if b.metadata == nil {
		b.metadata = newMetadataBackend(b.reader)
	}
	if b.mappings == nil {
		b.mappings = newMappingBackend(b.reader)
	}
	if b.microflows == nil && b.msdkWriter != nil {
		if w, ok := b.msdkWriter.(*modelsdkmpr.Writer); ok {
			b.microflows = newMicroflowBackend(w)
		}
	}
	if b.workflows == nil && b.msdkWriter != nil {
		if w, ok := b.msdkWriter.(*modelsdkmpr.Writer); ok {
			b.workflows = newWorkflowBackend(w)
		}
	}
	if b.pages == nil && b.msdkWriter != nil {
		if w, ok := b.msdkWriter.(*modelsdkmpr.Writer); ok {
			b.pages = newPageBackend(w)
		}
	}
	if b.java == nil && b.msdkWriter != nil {
		if w, ok := b.msdkWriter.(*modelsdkmpr.Writer); ok {
			b.java = newJavaBackend(w)
		}
	}
	if b.domainmodels == nil && b.msdkWriter != nil {
		if w, ok := b.msdkWriter.(*modelsdkmpr.Writer); ok {
			b.domainmodels = newDomainModelBackend(w)
		}
	}
	if b.security == nil && b.msdkWriter != nil {
		if w, ok := b.msdkWriter.(*modelsdkmpr.Writer); ok {
			b.security = newSecurityBackend(w)
		}
	}
}

func (b *MprBackend) Disconnect() error {
	if b.reader == nil {
		return nil
	}
	err := b.reader.Close()
	b.reader = nil
	b.msdkReader = nil
	b.msdkWriter = nil
	b.writer = nil
	b.path = ""
	return err
}

func (b *MprBackend) IsConnected() bool { return b.reader != nil }
func (b *MprBackend) Path() string      { return b.path }

// EnableContentCache activates in-memory caching of mxunit file contents.
// Call once after Connect when the backend will be held persistently across
// multiple requests (per-MPR daemon mode). The cache is cleared on any write.
func (b *MprBackend) EnableContentCache() {
	if b.reader != nil {
		b.reader.EnableContentCache()
	}
}

func (b *MprBackend) Version() types.MPRVersion { return types.MPRVersion(b.msdkReader.Version()) }
func (b *MprBackend) ProjectVersion() *types.ProjectVersion {
	return convertProjectVersionFromMsdk(b.msdkReader.ProjectVersion())
}
func (b *MprBackend) GetMendixVersion() (string, error) { return b.msdkReader.GetMendixVersion() }

// GetMxGraph returns the cached mxgraph snapshot from the reader, or nil.
func (b *MprBackend) GetMxGraph() *mxgraph.Graph {
	if b.reader != nil {
		return b.reader.GetMxGraph()
	}
	return nil
}

// Commit is a no-op — the MPR writer auto-commits on each write operation.
func (b *MprBackend) Commit() error { return nil }
