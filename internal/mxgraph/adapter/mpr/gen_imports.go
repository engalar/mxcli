package mpr

// Blank-import every gen domain whose BSON types the adapters traverse, so the
// codec descriptor registry is populated for any caller that imports this package.
// Without these, LoadUnit returns elements whose nested Part / ByNameRef children
// fail to decode and Properties() comes back empty (no edges, no derived props).
import (
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/datatypes"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/domainmodels"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/enumerations"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/microflows"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/nanoflows"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/security"
	_ "github.com/mendixlabs/mxcli/modelsdk/gen/workflows"
)
