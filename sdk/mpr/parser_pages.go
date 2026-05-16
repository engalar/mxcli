// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"fmt"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"

	"go.mongodb.org/mongo-driver/bson"
)

// pagesDecoder is a package-level codec.Decoder for pages gen types.
var pagesDecoder = codec.NewDecoder(codec.DefaultRegistry)

// parsePageGen parses a Forms$Page BSON document into a gen-typed Page.
func (r *Reader) parsePageGen(unitID, _ string, contents []byte) (*genPg.Page, error) {
	contents, err := r.resolveContents(unitID, contents)
	if err != nil {
		return nil, err
	}
	elem, err := pagesDecoder.Decode(bson.Raw(contents))
	if err != nil {
		return nil, fmt.Errorf("decode page %s: %w", unitID, err)
	}
	pg, ok := elem.(*genPg.Page)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a Page (got %T)", unitID, elem)
	}
	pg.SetID(element.ID(unitID))
	return pg, nil
}

// parseLayoutGen parses a Forms$Layout BSON document into a gen-typed Layout.
func (r *Reader) parseLayoutGen(unitID, _ string, contents []byte) (*genPg.Layout, error) {
	contents, err := r.resolveContents(unitID, contents)
	if err != nil {
		return nil, err
	}
	elem, err := pagesDecoder.Decode(bson.Raw(contents))
	if err != nil {
		return nil, fmt.Errorf("decode layout %s: %w", unitID, err)
	}
	ly, ok := elem.(*genPg.Layout)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a Layout (got %T)", unitID, elem)
	}
	ly.SetID(element.ID(unitID))
	return ly, nil
}

// parseSnippetGen parses a Forms$Snippet BSON document into a gen-typed Snippet.
func (r *Reader) parseSnippetGen(unitID, _ string, contents []byte) (*genPg.Snippet, error) {
	contents, err := r.resolveContents(unitID, contents)
	if err != nil {
		return nil, err
	}
	elem, err := pagesDecoder.Decode(bson.Raw(contents))
	if err != nil {
		return nil, fmt.Errorf("decode snippet %s: %w", unitID, err)
	}
	sn, ok := elem.(*genPg.Snippet)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a Snippet (got %T)", unitID, elem)
	}
	sn.SetID(element.ID(unitID))
	return sn, nil
}

// parseBuildingBlockGen parses a Forms$BuildingBlock BSON document into a gen-typed BuildingBlock.
func (r *Reader) parseBuildingBlockGen(unitID, _ string, contents []byte) (*genPg.BuildingBlock, error) {
	contents, err := r.resolveContents(unitID, contents)
	if err != nil {
		return nil, err
	}
	elem, err := pagesDecoder.Decode(bson.Raw(contents))
	if err != nil {
		return nil, fmt.Errorf("decode building block %s: %w", unitID, err)
	}
	bb, ok := elem.(*genPg.BuildingBlock)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a BuildingBlock (got %T)", unitID, elem)
	}
	bb.SetID(element.ID(unitID))
	return bb, nil
}

// parsePageTemplateGen parses a Forms$PageTemplate BSON document into a gen-typed PageTemplate.
func (r *Reader) parsePageTemplateGen(unitID, _ string, contents []byte) (*genPg.PageTemplate, error) {
	contents, err := r.resolveContents(unitID, contents)
	if err != nil {
		return nil, err
	}
	elem, err := pagesDecoder.Decode(bson.Raw(contents))
	if err != nil {
		return nil, fmt.Errorf("decode page template %s: %w", unitID, err)
	}
	pt, ok := elem.(*genPg.PageTemplate)
	if !ok {
		return nil, fmt.Errorf("unit %s is not a PageTemplate (got %T)", unitID, elem)
	}
	pt.SetID(element.ID(unitID))
	return pt, nil
}
