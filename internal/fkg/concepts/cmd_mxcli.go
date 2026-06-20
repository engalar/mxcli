// internal/fkg/concepts/cmd_mxcli.go
package concepts

import (
	"context"
	"fmt"

	"github.com/mendixlabs/mxcli/internal/mxgraph"
)

// mxcli command descriptors are registered via RegisterMXCLICmd() before Build().
// The adapter reads them in Build() to create FKG command nodes.

type cmdDescriptor struct {
	Name   string
	Short  string
	Long   string
	Parent string
	Flags  []string
}

var mxcliCmds []cmdDescriptor

// RegisterMXCLICmd adds an mxcli command to the FKG command registry.
// Called during init() by the CLI layer.
func RegisterMXCLICmd(name, short, long, parent string, flags []string) {
	mxcliCmds = append(mxcliCmds, cmdDescriptor{
		Name: name, Short: short, Long: long, Parent: parent, Flags: flags,
	})
}

func init() { Register(&CmdAdapter{}) }

type CmdAdapter struct{}

func (a *CmdAdapter) Name() string { return "fkg:cmd-mxcli" }
func (a *CmdAdapter) Schema() *mxgraph.GraphSchema {
	return &mxgraph.GraphSchema{
		NodeLabels: []mxgraph.Label{LabelConcept, LabelSyntaxFeature},
		EdgeTypes: []struct {
			Type mxgraph.RelType
			From mxgraph.Label
			To   mxgraph.Label
		}{
			{Specializes, LabelConcept, LabelConcept},
			{HasSyntax, LabelConcept, LabelSyntaxFeature},
		},
	}
}
func (a *CmdAdapter) Watch(_ context.Context, _ mxgraph.EventSink) (func(), error) {
	return func() {}, nil
}
func (a *CmdAdapter) Build(_ context.Context, sink mxgraph.EventSink) error {
	var events []mxgraph.Event
	for _, c := range mxcliCmds {
		cmdID := "cmd:" + c.Name
		events = append(events, conceptNode(cmdID, c.Name, c.Short))

		if c.Long != "" {
			events = append(events, syntaxNode("cmd."+c.Name+".long", c.Long))
			events = append(events, edge(cmdID, "syntax:cmd."+c.Name+".long", HasSyntax))
		}
		for _, f := range c.Flags {
			fid := fmt.Sprintf("cmd.%s.%s", c.Name, f)
			events = append(events, syntaxNode(fid, fmt.Sprintf("Flag %s of %s", f, c.Name)))
			events = append(events, edge(cmdID, "syntax:"+fid, HasSyntax))
		}
		if c.Parent != "" {
			events = append(events, edge(cmdID, "cmd:"+c.Parent, Specializes))
		}
	}
	return sink.Emit(events)
}
