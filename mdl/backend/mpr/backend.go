// SPDX-License-Identifier: Apache-2.0

// Package mprbackend provides the MprBackend implementation of
// backend.FullBackend. The package name is "mprbackend" (not "mpr") to
// avoid collision with the sdk/mpr package in import blocks.
package mprbackend

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/mdl/backend/unitstore"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/mdl/linter"
	"github.com/mendixlabs/mxcli/mdl/types"
	modelsdkmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

var _ backend.FullBackend = (*MprBackend)(nil)
var _ backend.PageModelBackend = (*MprBackend)(nil)
var _ backend.ImportBufferBackend = (*MprBackend)(nil)
var _ linter.LintReader = (*MprBackend)(nil)

// MprBackend implements backend.FullBackend + backend.BackendFactory
// by delegating to a single modelsdk/mpr Reader/Writer pair and domain-
// specific sub-backends created eagerly in Connect().
type MprBackend struct {
	reader          *modelsdkmpr.Reader
	msdkReader      *modelsdkmpr.Reader // alias of reader; kept for *_compat.go ergonomics
	msdkWriter      modelsdkmpr.UnitWriter
	writer          *modelsdkmpr.Writer // concrete writer, set in Connect()
	path            string
	scriptBuf       *ScriptBuffer
	unitBuf         *unitstore.BufferedUnitStore
	widgetTypeCache map[string]*widgetTypeCacheEntry

	// subBackendsReady is set to true once Connect() finishes creating
	// all sub-backends. Replaces repeated initSubBackends() calls.
	subBackendsReady bool

	// Domain-specific sub-backends. Created eagerly in Connect().
	modules         *moduleBackend
	microflows      *microflowBackend
	workflows       *workflowBackend
	pages           *pageBackend
	java            *javaBackend
	domainmodels    *domainModelBackend
	security        *securityBackend
	folders         *folderBackend
	scheduledEvents *scheduledEventBackend
	enumerations    *enumerationBackend
	constants       *constantBackend
	rawUnits        *rawUnitBackend
	metadata        *metadataBackend
	mappings        *mappingBackend
	settings        *settingsBackend
	navigation      *navigationBackend

	// graph is the in-memory project graph built at startup.
	graph *graphcatalog.ProjectGraph
}

// widgetTypeCacheEntry holds the per-page cached type schema for one widget type.
type widgetTypeCacheEntry struct {
	rawType     bson.D                               // shared CustomWidgets$CustomWidgetType
	rawObject   bson.D                               // template object; deep-cloned per instance
	propTypeIDs map[string]types.PropertyTypeIDEntry // for property-key → TypePointer lookups
}
