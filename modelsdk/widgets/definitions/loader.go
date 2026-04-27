// SPDX-License-Identifier: Apache-2.0

// Package definitions provides embedded widget definition files for the pluggable widget engine.
package definitions

import "embed"

//go:embed *.def.json
var EmbeddedFS embed.FS
