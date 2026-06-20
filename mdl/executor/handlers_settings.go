// SPDX-License-Identifier: Apache-2.0

package executor

import "github.com/mendixlabs/mxcli/mdl/ast"

func registerSettingsHandlers(r *Registry) {
	r.Register(&ast.AlterSettingsStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return alterSettings(ctx, stmt.(*ast.AlterSettingsStmt))
	})
	r.Register(&ast.AlterLanguageStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return alterLanguage(ctx, stmt.(*ast.AlterLanguageStmt))
	})
	r.Register(&ast.TranslateStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return translateDocument(ctx, stmt.(*ast.TranslateStmt))
	})
	r.Register(&ast.TranslateMicroflowStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return translateMicroflowStmt(ctx, stmt.(*ast.TranslateMicroflowStmt))
	})
	r.Register(&ast.CreateConfigurationStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return createConfiguration(ctx, stmt.(*ast.CreateConfigurationStmt))
	})
	r.Register(&ast.DropConfigurationStmt{}, func(ctx *ExecContext, stmt ast.Statement) error {
		return dropConfiguration(ctx, stmt.(*ast.DropConfigurationStmt))
	})
}
