package graphcatalog_test

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/graphcatalog"
	"github.com/mendixlabs/mxcli/mdl/graphcatalog/mock"
)

// TestInterfaceCompliance is a compile-time check that MockProjectGraph satisfies
// both LintReader and TraversalReader.
func TestInterfaceCompliance(t *testing.T) {
	var _ graphcatalog.LintReader = (*mock.MockProjectGraph)(nil)
	var _ graphcatalog.TraversalReader = (*mock.MockProjectGraph)(nil)
}
