// SPDX-License-Identifier: Apache-2.0

// Example: Creating a Page using MDL (Mendix Definition Language)
//
// The modern API for creating pages is MDL. Use the mxcli CLI:
//
//	mxcli exec create_page.mdl -p path/to/app.mpr
//
// Where create_page.mdl contains:
//
//	create page MyModule.CustomerEdit (
//	  Title: 'Edit Customer',
//	  Layout: Atlas_Core.Atlas_Default
//	) {
//	  DATAVIEW dv1 (DataSource: ENTITY MyModule.Customer) {
//	    TEXTBOX tbName  (Label: 'Name',  Attribute: Name)
//	    TEXTBOX tbEmail (Label: 'Email', Attribute: Email)
//	    DATEPICKER dpBirth (Label: 'Birth Date', Attribute: BirthDate)
//	    CHECKBOX cbActive  (Label: 'Is Active',  Attribute: IsActive)
//
//	    FOOTER {
//	      ACTIONBUTTON btnSave   (Caption: 'Save',   Style: Primary, Action: SAVE  CLOSEPAGE)
//	      ACTIONBUTTON btnCancel (Caption: 'Cancel', Style: Default, Action: CANCEL CLOSEPAGE)
//	    }
//	  }
//	};
//
// For programmatic use, embed the MDL executor:
//
//	import "github.com/mendixlabs/mxcli/mdl/executor"
//
// See the executor package documentation, the MDL syntax reference at
// docs/01-project/MDL_QUICK_REFERENCE.md, and the page-builder skills under
// .claude/skills/ (create-page.md, alter-page.md, master-detail-pages.md).
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "This example has moved to MDL syntax.")
	fmt.Fprintln(os.Stderr, "See the comment at the top of this file for the modern API.")
	os.Exit(1)
}
