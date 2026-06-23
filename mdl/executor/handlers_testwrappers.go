// SPDX-License-Identifier: Apache-2.0
// Auto-generated wrappers for test compatibility
package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

func execCreateAssociation(ctx *ExecContext, s *ast.CreateAssociationStmt) error {
    return ExecCreateAssociation(ctx, s)
}

func execCreateEnumeration(ctx *ExecContext, s *ast.CreateEnumerationStmt) error {
    return ExecCreateEnumeration(ctx, s)
}

func execDropEnumeration(ctx *ExecContext, s *ast.DropEnumerationStmt) error {
    return ExecDropEnumeration(ctx, s)
}

func execDropAssociation(ctx *ExecContext, s *ast.DropAssociationStmt) error {
    return ExecDropAssociationGen(ctx, s)
}
