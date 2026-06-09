// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"encoding/json"
	"fmt"

	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/model"
)

// domainModelELKData is the JSON output schema for the domain model ELK diagram.
type domainModelELKData struct {
	Format          string                         `json:"format"`
	Type            string                         `json:"type"`
	ModuleName      string                         `json:"moduleName"`
	FocusEntity     string                         `json:"focusEntity,omitempty"`
	Entities        []domainModelELKEntity         `json:"entities"`
	Associations    []domainModelELKAssoc          `json:"associations"`
	Generalizations []domainModelELKGeneralization `json:"generalizations"`
	MdlSource       string                         `json:"mdlSource,omitempty"`
	SourceMap       map[string]elkSourceRange      `json:"sourceMap,omitempty"`
}

type domainModelELKEntity struct {
	ID         string                    `json:"id"`
	Name       string                    `json:"name"`
	Category   string                    `json:"category"`
	IsFocus    bool                      `json:"isFocus,omitempty"`
	Attributes []domainModelELKAttribute `json:"attributes"`
	Width      float64                   `json:"width"`
	Height     float64                   `json:"height"`
}

type domainModelELKAttribute struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type domainModelELKAssoc struct {
	ID       string `json:"id"`
	SourceID string `json:"sourceId"`
	TargetID string `json:"targetId"`
	Name     string `json:"name"`
	Type     string `json:"type"` // "reference" or "referenceSet"
}

type domainModelELKGeneralization struct {
	ChildID    string `json:"childId"`
	ParentID   string `json:"parentId"`
	ParentName string `json:"parentName"`
}

// Sizing constants for ELK node dimension calculation.
const (
	elkCharWidth      = 7.5
	elkHeaderHeight   = 28.0
	elkAttrLineHeight = 18.0
	elkHPadding       = 24.0
	elkMinWidth       = 100.0
)

// domainModelELK routes all renders through the gen-typed implementation.
func domainModelELK(ctx *ExecContext, name string) error {
	return domainModelELKGen(ctx, name)
}

// --- helpers ---

// makeGhostEntity creates a minimal entity node for cross-module references.
func makeGhostEntity(id, name string) domainModelELKEntity {
	width := float64(len(name))*elkCharWidth + elkHPadding
	if width < elkMinWidth {
		width = elkMinWidth
	}
	return domainModelELKEntity{
		ID:       id,
		Name:     name,
		Category: "external",
		Width:    width,
		Height:   elkHeaderHeight + elkAttrLineHeight,
	}
}

// addGhostIfNeeded adds a ghost entity if the given ID is not in the included set.
func addGhostIfNeeded(id model.ID, includedIDs map[model.ID]bool, allEntityNames map[model.ID]string, ghosts map[string]*domainModelELKEntity) {
	if includedIDs[id] {
		return
	}
	ghostID := "entity-" + string(id)
	if _, exists := ghosts[ghostID]; exists {
		return
	}
	name := "Unknown"
	if qn, ok := allEntityNames[id]; ok {
		name = qn
	}
	ghost := makeGhostEntity(ghostID, name)
	ghosts[ghostID] = &ghost
}

// emitDomainModelELK marshals and writes the domain model ELK data to output.
func emitDomainModelELK(ctx *ExecContext, data domainModelELKData) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return mdlerrors.NewBackend("marshal json", err)
	}
	fmt.Fprint(ctx.Output, string(out))
	return nil
}

// --- Executor method wrappers for callers not yet migrated ---

// DomainModelELK is the exported Executor method, called from outside the package.
func (e *Executor) DomainModelELK(name string) error {
	return domainModelELK(e.newExecContext(context.Background()), name)
}
