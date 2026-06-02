// SPDX-License-Identifier: Apache-2.0

package archtest

import "github.com/mendixlabs/mxcli/mdl/model"

// CodecComplete verifies that every Required gen TypeName is registered in
// the registry returned by BuildRegistry, and that both LiftFn and HydrateFn
// are non-nil. The Package argument is ignored — this rule inspects runtime
// registry state, not source files.
type CodecComplete struct {
	BuildRegistry func() *model.DefaultRegistry
	Required      []string
	Hint          string
}

func (r CodecComplete) Name() string { return "CodecComplete" }

func (r CodecComplete) Check(_ Package) []Violation {
	reg := r.BuildRegistry()
	var violations []Violation
	for _, name := range r.Required {
		codec, ok := reg.Lookup(name)
		if !ok {
			violations = append(violations, Violation{
				File:    "codec registration",
				Message: Sprintf("gen type %q not registered", name),
				Hint:    r.Hint,
			})
			continue
		}
		if codec.LiftFn == nil {
			violations = append(violations, Violation{
				File:    "codec registration",
				Message: Sprintf("%q: LiftFn is nil", name),
				Hint:    r.Hint,
			})
		}
		if codec.HydrateFn == nil {
			violations = append(violations, Violation{
				File:    "codec registration",
				Message: Sprintf("%q: HydrateFn is nil", name),
				Hint:    r.Hint,
			})
		}
	}
	return violations
}
