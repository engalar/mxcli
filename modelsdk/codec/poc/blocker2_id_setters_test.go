//go:build poc
// SPDX-License-Identifier: Apache-2.0

package poc_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/modelsdk/element"
	genMf "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	mmpr "github.com/mendixlabs/mxcli/modelsdk/mpr"
)

// newID is the canonical way to mint an Element ID for these PoC tests.
// mmpr.GenerateID returns a UUID string; element.ID is `type ID string`.
func newID() element.ID {
	return element.ID(mmpr.GenerateID())
}

// TestBlocker2_GenTypeIDSetters confirms that freshly-constructed gen
// objects expose the public setters the executor will need before calling
// repo.Create(). It also documents what is NOT on the API surface
// (SetContainerID — the parent UUID is passed to InsertUnit at write time,
// not stored on the gen object).
//
// PoC validation: see docs/superpowers/specs/2026-05-13-modelsdk-native-architecture-design.md Section 9 Blocker 2.
func TestBlocker2_GenTypeIDSetters(t *testing.T) {
	mf := genMf.NewMicroflow()
	if mf == nil {
		t.Fatal("NewMicroflow() returned nil")
	}

	// ID — element.Base provides SetID(ID) / ID() ID
	id := newID()
	mf.SetID(id)
	if got := mf.ID(); got != id {
		t.Errorf("after SetID: got %q, want %q", got, id)
	}

	// Name — codegen-emitted property setter
	mf.SetName("PoCFlow")
	if got := mf.Name(); got != "PoCFlow" {
		t.Errorf("after SetName: got %q, want PoCFlow", got)
	}

	// Container — element.Base provides SetContainer(Element) / Container() Element.
	// Note: this is an Element pointer, NOT a UUID. The container's UUID is
	// passed to mmpr.Writer.InsertUnit(unitID, containerID, ...) at write
	// time. There is intentionally no SetContainerID(ID) on the gen API.
	parent := genMf.NewMicroflow()
	parent.SetID(newID())
	mf.SetContainer(parent)
	if got := mf.Container(); got != element.Element(parent) {
		t.Errorf("after SetContainer: got %v, want parent", got)
	}

	// TypeName — Base provides SetTypeName / TypeName, used by the encoder
	// to write the $Type field. NewMicroflow() should pre-populate this.
	if got := mf.TypeName(); got == "" {
		t.Error("TypeName() empty — NewMicroflow should pre-set $Type")
	}
}
