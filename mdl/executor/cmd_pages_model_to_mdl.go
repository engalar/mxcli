// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"io"

	"github.com/mendixlabs/mxcli/mdl/pagerender"
	"github.com/mendixlabs/mxcli/mdl/types"
)

// pageModelToMDL renders a PageModel to MDL V3 text and writes it to w.
// The rendering logic lives in the pagerender package so that both this
// executor and the canonical/page layer can render without importing each
// other. This thin wrapper preserves the existing call sites.
func pageModelToMDL(w io.Writer, pm *types.PageModel, modName, pageName string) {
	pagerender.PageModelToMDL(w, pm, modName, pageName)
}

// renderWidget renders a single widget node as MDL text. Thin wrapper over
// pagerender.RenderWidget for existing executor call sites (describe path).
func renderWidget(w io.Writer, node *types.WidgetNode, depth int) {
	pagerender.RenderWidget(w, node, depth)
}
