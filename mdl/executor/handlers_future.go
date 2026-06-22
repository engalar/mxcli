package executor

import (
	"context"
	"fmt"
	"io"

	"github.com/mendixlabs/mxcli/cmd/mxcli/syntax"
	"github.com/mendixlabs/mxcli/mdl/ast"
)

// execHelpFuture is the ExecContext-free version of execHelp.
func execHelpFuture(ctx context.Context, s *ast.HelpStmt, output io.Writer, format OutputFormat) error {
	if len(s.Topic) > 0 {
		path := syntax.ResolveAlias(resolveHelpPath(s.Topic))
		features := syntax.ByPrefix(path)
		if len(features) == 0 {
			fmt.Fprintf(output, "No syntax help found for: %s\n", path)
			fmt.Fprintln(output, "Use HELP; for a command overview.")
			return nil
		}
		if format == FormatJSON {
			return syntax.WriteJSON(output, features)
		}
		syntax.WriteText(output, features)
		return nil
	}

	fmt.Fprint(output, `MDL Commands:

Connection:
  connect local '<path>'      Connect to local .mpr file
  disconnect                  Disconnect from project
  status                      Show connection status

Domain Model - Enumerations:
  create enumeration Module.Name ...   Create enumeration
  alter  enumeration Module.Name ...   Alter enumeration
  drop   enumeration Module.Name       Drop enumeration

Domain Model - Entities:
  create entity Module.Name ( ... )   Create entity (with attributes and associations)
  alter  entity Module.Name ...       Alter entity
  drop   entity Module.Name            Drop entity

Microflows:
  create microflow Module.Name ( ... )  Create microflow
  drop   microflow Module.Name          Drop microflow

For detailed help: HELP <topic>, e.g. HELP create entity
`)
	return nil
}

// execExitFuture is the ExecContext-free version of execExit.
func execExitFuture(ctx context.Context) error {
	return ErrExit
}
