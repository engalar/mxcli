// Package modelsdk provides auto-generated, type-safe access to Mendix MPR files.
//
// modelsdk is designed as a replacement for the hand-written sdk/ package.
// It coexists with sdk/ during migration — no existing code is modified.
//
// # Choosing between modelsdk/ and sdk/
//
// Use modelsdk/ for new code that needs:
//   - Dirty tracking (know what changed before writing)
//   - BSON roundtrip safety (unimplemented fields survive read/write)
//   - Coverage beyond the 5-10 domains in sdk/ (53 domains here)
//
// Use sdk/ for existing code that already works — migrate incrementally.
//
// # Architecture
//
// The package has three layers:
//
//   - element.Base: identity, raw BSON bytes, dirty bitmap with container chain propagation
//   - property.Primitive/Part/PartList/Enum/ByNameRef: lazy-decoded fields with per-bit dirty tracking
//   - codec.Decoder/Encoder: type registry dispatch with three-branch encoding (clean/child-dirty/self-dirty)
//
// # Generated code
//
// The modelsdk/gen/ subdirectory contains auto-generated types for all 53 Mendix
// metamodel domains. To regenerate after updating the TypeScript SDK reference:
//
//	npm install mendixmodelsdk --prefix reference/mendixmodelsdk
//	go run ./cmd/modelsdk-codegen
//
// Each generated domain package contains:
//   - types.go: struct definitions, getters/setters, InitFromRaw, registry init()
//   - enums.go: enumeration type aliases and constants
//   - refs.go: cross-domain reference metadata registration
//   - version.go: per-property introduced/deleted/public version info
package modelsdk
