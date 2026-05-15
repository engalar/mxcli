// SPDX-License-Identifier: Apache-2.0

package mprbackend

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/mendixlabs/mxcli/mdl/backend"
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	"github.com/mendixlabs/mxcli/sdk/mpr"
)

// Stage 3.3.5.E0.create_v3 transitional helpers — converts sdk-typed
// Page / Snippet structs (still emitted by the V3 page builder) to
// gen-typed counterparts via BSON roundtrip so callers can persist
// through the gen-native CreatePageGen / UpdatePageGen / CreateSnippetGen
// surface. To be retired when the V3 builder itself is rewritten to
// emit gen types directly.

// SDKPageToGen serializes a sdk-typed Page to BSON via mpr.SerializePage
// then decodes through the modelsdk codec to produce a gen-typed Page.
func (b *MprBackend) SDKPageToGen(p *backend.Page) (*genPg.Page, error) {
	if p == nil {
		return nil, fmt.Errorf("SDKPageToGen: nil Page")
	}
	if p.TypeName == "" {
		p.TypeName = "Forms$Page"
	}
	contents, err := mpr.SerializePage(p)
	if err != nil {
		return nil, fmt.Errorf("SDKPageToGen: serialize: %w", err)
	}
	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, err := dec.Decode(bson.Raw(contents))
	if err != nil {
		return nil, fmt.Errorf("SDKPageToGen: decode: %w", err)
	}
	page, ok := elem.(*genPg.Page)
	if !ok {
		return nil, fmt.Errorf("SDKPageToGen: decoded element is %T, not *genPg.Page", elem)
	}
	return page, nil
}

// SDKSnippetToGen mirrors SDKPageToGen for Snippet units.
func (b *MprBackend) SDKSnippetToGen(s *backend.Snippet) (*genPg.Snippet, error) {
	if s == nil {
		return nil, fmt.Errorf("SDKSnippetToGen: nil Snippet")
	}
	if s.TypeName == "" {
		s.TypeName = "Forms$Snippet"
	}
	contents, err := mpr.SerializeSnippet(s)
	if err != nil {
		return nil, fmt.Errorf("SDKSnippetToGen: serialize: %w", err)
	}
	dec := codec.NewDecoder(codec.DefaultRegistry)
	elem, err := dec.Decode(bson.Raw(contents))
	if err != nil {
		return nil, fmt.Errorf("SDKSnippetToGen: decode: %w", err)
	}
	snippet, ok := elem.(*genPg.Snippet)
	if !ok {
		return nil, fmt.Errorf("SDKSnippetToGen: decoded element is %T, not *genPg.Snippet", elem)
	}
	return snippet, nil
}
