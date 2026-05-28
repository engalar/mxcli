// SPDX-License-Identifier: Apache-2.0

package mprrepos

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/mendixlabs/mxcli/modelsdk/codec"
	"github.com/mendixlabs/mxcli/modelsdk/element"
)

// encoder / decoder are thin wrappers around codec.Encoder /
// codec.Decoder so every Stage 2 repo can share one instance per
// constructor without re-creating zero-value structs ad-hoc.
type encoder struct{ e *codec.Encoder }
type decoder struct{ d *codec.Decoder }

func newEncoder() *encoder { return &encoder{e: &codec.Encoder{}} }
func newDecoder() *decoder { return &decoder{d: codec.NewDecoder(codec.DefaultRegistry)} }

func (e *encoder) Encode(elem element.Element) ([]byte, error) { return e.e.Encode(elem) }

func (d *decoder) Decode(raw []byte) (element.Element, error) {
	return d.d.Decode(bson.Raw(raw))
}
