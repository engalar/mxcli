// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// encoder / decoder are thin wrappers around codec.Encoder /
// codec.Decoder so every Stage 2 repo can share one instance per
// constructor without re-creating zero-value structs ad-hoc.
type encoder struct{ e *codec.Encoder }
type decoder struct{ d *codec.Decoder }

func newEncoder() *encoder { return &encoder{e: &codec.Encoder{}} }
func newDecoder() *decoder { return &decoder{d: codec.NewDecoder(codec.DefaultRegistry)} }

func (e *encoder) Encode(elem element.Element) ([]byte, error) { return e.e.Encode(elem) }

// EncodePage encodes a Page and injects the AllowedModuleRoles BSON field
// (version int32(1)) alongside AllowedRoles. Mendix's CE0557 validator
// checks AllowedModuleRoles, not AllowedRoles (version 3 gen field).
// Without both fields, pages referenced from navigation/buttons/URLs
// get false CE0557 errors.
//
// Uses codec.PatchBSONField instead of raw bson manipulation.
// bson.A is still needed for array construction because the codec
// does not expose a generic versioned-array builder (NewVersionedArray
// only supports version int32(3)).
func (e *encoder) EncodePage(page *genPg.Page) ([]byte, error) {
	contents, err := e.Encode(page)
	if err != nil {
		return nil, err
	}
	roles := page.AllowedRolesQualifiedNames()
	if len(roles) == 0 {
		return contents, nil
	}
	rolesArr := bson.A{int32(1)}
	for _, r := range roles {
		rolesArr = append(rolesArr, r)
	}
	return codec.PatchBSONField(contents, "AllowedModuleRoles", rolesArr)
}

func (d *decoder) Decode(raw []byte) (element.Element, error) {
	return d.d.Decode(bson.Raw(raw))
}
