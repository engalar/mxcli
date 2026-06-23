// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/mendixlabs/mxcli/mdl/ast"
	mdlerrors "github.com/mendixlabs/mxcli/mdl/errors"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	sqllib "github.com/mendixlabs/mxcli/sql"
)

// ensureSQLManagerFn returns deps.SqlMgr (lazy init) for HandlerDeps.
func ensureSQLManagerFn(deps *HandlerDeps) *sqllib.Manager {
	if deps.SqlMgr == nil {
		deps.SqlMgr = sqllib.NewManager()
	}
	return deps.SqlMgr
}

// getOrAutoConnectFn returns an existing connection or auto-connects using connections.yaml.
func getOrAutoConnectFn(deps *HandlerDeps, alias string) (*sqllib.Connection, error) {
	mgr := ensureSQLManagerFn(deps)
	conn, err := mgr.Get(alias)
	if err == nil {
		return conn, nil
	}
	if acErr := autoConnectFn(deps, alias); acErr != nil {
		return nil, mdlerrors.NewNotFoundMsg("connection", alias, fmt.Sprintf("no connection '%s' (and auto-connect failed: %v)", alias, acErr))
	}
	return mgr.Get(alias)
}

// autoConnectFn resolves a connection alias from env vars or config.
func autoConnectFn(deps *HandlerDeps, alias string) error {
	rc, err := sqllib.ResolveConnection(sqllib.ResolveOptions{Alias: alias})
	if err != nil {
		return fmt.Errorf("cannot resolve connection '%s': %w\nAdd it to .mxcli/connections.yaml or use: sql connect <driver> '<dsn>' as %s", alias, err, alias)
	}
	mgr := ensureSQLManagerFn(deps)
	if err := mgr.Connect(rc.Driver, rc.DSN, alias); err != nil {
		return err
	}
	fmt.Fprintf(deps.Output, "Connected to %s database as '%s' (from config)\n", rc.Driver, alias)
	return nil
}

// execSQLConnectFn handles SQL CONNECT with HandlerDeps.
func execSQLConnectFn(ctx context.Context, s *ast.SQLConnectStmt, deps *HandlerDeps) error {
	if s.DSN == "" && s.Driver == "" {
		return autoConnectFn(deps, s.Alias)
	}
	driver, err := sqllib.ParseDriver(s.Driver)
	if err != nil {
		return err
	}
	mgr := ensureSQLManagerFn(deps)
	if err := mgr.Connect(driver, s.DSN, s.Alias); err != nil {
		return err
	}
	fmt.Fprintf(deps.Output, "Connected to %s database as '%s'\n", driver, s.Alias)
	return nil
}

// execSQLDisconnectFn handles SQL DISCONNECT with HandlerDeps.
func execSQLDisconnectFn(ctx context.Context, s *ast.SQLDisconnectStmt, deps *HandlerDeps) error {
	mgr := ensureSQLManagerFn(deps)
	if err := mgr.Disconnect(s.Alias); err != nil {
		return err
	}
	fmt.Fprintf(deps.Output, "Disconnected '%s'\n", s.Alias)
	return nil
}

// execSQLConnectionsFn handles SQL CONNECTIONS with HandlerDeps.
func execSQLConnectionsFn(ctx context.Context, deps *HandlerDeps) error {
	mgr := ensureSQLManagerFn(deps)
	infos := mgr.List()
	if len(infos) == 0 {
		fmt.Fprintln(deps.Output, "No active sql connections")
		return nil
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Alias < infos[j].Alias
	})
	result := &sqllib.QueryResult{
		Columns: []string{"Alias", "Driver"},
	}
	for _, info := range infos {
		result.Rows = append(result.Rows, []any{info.Alias, string(info.Driver)})
	}
	sqllib.FormatTable(deps.Output, result)
	return nil
}

// execSQLQueryFn handles SQL <alias> <raw-sql> with HandlerDeps.
func execSQLQueryFn(ctx context.Context, s *ast.SQLQueryStmt, deps *HandlerDeps) error {
	conn, err := getOrAutoConnectFn(deps, s.Alias)
	if err != nil {
		return err
	}
	goCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := sqllib.Execute(goCtx, conn, s.Query)
	if err != nil {
		return err
	}
	sqllib.FormatTable(deps.Output, result)
	fmt.Fprintf(deps.Output, "(%d rows)\n", len(result.Rows))
	return nil
}

// execSQLShowTablesFn handles SQL <alias> SHOW TABLES with HandlerDeps.
func execSQLShowTablesFn(ctx context.Context, s *ast.SQLShowTablesStmt, deps *HandlerDeps) error {
	conn, err := getOrAutoConnectFn(deps, s.Alias)
	if err != nil {
		return err
	}
	goCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := sqllib.ShowTables(goCtx, conn)
	if err != nil {
		return err
	}
	sqllib.FormatTable(deps.Output, result)
	fmt.Fprintf(deps.Output, "(%d tables)\n", len(result.Rows))
	return nil
}

// execSQLShowViewsFn handles SQL <alias> SHOW VIEWS with HandlerDeps.
func execSQLShowViewsFn(ctx context.Context, s *ast.SQLShowViewsStmt, deps *HandlerDeps) error {
	conn, err := getOrAutoConnectFn(deps, s.Alias)
	if err != nil {
		return err
	}
	goCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := sqllib.ShowViews(goCtx, conn)
	if err != nil {
		return err
	}
	sqllib.FormatTable(deps.Output, result)
	fmt.Fprintf(deps.Output, "(%d views)\n", len(result.Rows))
	return nil
}

// execSQLShowFunctionsFn handles SQL <alias> SHOW FUNCTIONS with HandlerDeps.
func execSQLShowFunctionsFn(ctx context.Context, s *ast.SQLShowFunctionsStmt, deps *HandlerDeps) error {
	conn, err := getOrAutoConnectFn(deps, s.Alias)
	if err != nil {
		return err
	}
	goCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := sqllib.ShowFunctions(goCtx, conn)
	if err != nil {
		return err
	}
	sqllib.FormatTable(deps.Output, result)
	fmt.Fprintf(deps.Output, "(%d functions)\n", len(result.Rows))
	return nil
}

// execSQLGenerateConnectorFn handles SQL <alias> GENERATE CONNECTOR with HandlerDeps.
func execSQLGenerateConnectorFn(ctx context.Context, s *ast.SQLGenerateConnectorStmt, deps *HandlerDeps) error {
	conn, err := getOrAutoConnectFn(deps, s.Alias)
	if err != nil {
		return err
	}
	goCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cfg := &sqllib.GenerateConfig{
		Conn:   conn,
		Module: s.Module,
		Alias:  s.Alias,
		Tables: s.Tables,
		Views:  s.Views,
	}
	result, err := sqllib.GenerateConnector(goCtx, cfg)
	if err != nil {
		return err
	}
	for _, skip := range result.SkippedCols {
		fmt.Fprintf(deps.Output, "-- warning: skipped unmappable column: %s\n", skip)
	}
	if s.Exec {
		fmt.Fprintf(deps.Output, "Generating connector (%d tables, %d views)...\n",
			result.TableCount, result.ViewCount)
		if err := executeGeneratedMDLFn(deps, result.ExecutableMDL); err != nil {
			return err
		}
		fmt.Fprintf(deps.Output, "\n-- Database Connection definition (configure in Studio Pro with Database Connector module):\n")
		fmt.Fprint(deps.Output, result.ConnectionMDL)
		return nil
	}
	fmt.Fprint(deps.Output, result.MDL)
	fmt.Fprintf(deps.Output, "\n-- Generated: %d tables, %d views\n", result.TableCount, result.ViewCount)
	return nil
}

// executeGeneratedMDLFn parses and executes MDL text via deps.
func executeGeneratedMDLFn(deps *HandlerDeps, mdl string) error {
	prog, errs := visitor.Build(mdl)
	if len(errs) > 0 {
		return mdlerrors.NewBackend("parse generated MDL", fmt.Errorf("%v", errs[0]))
	}
	// Use the Executor's ExecuteProgram through a temporary ExecContext.
	ectx := phase3d2bNewExecContext(context.Background(), deps)
	if ectx.ExecuteProgramFn == nil {
		return mdlerrors.NewBackend("execute generated MDL", fmt.Errorf("ExecuteProgramFn not set — ExecContext was not created via Executor dispatch"))
	}
	return ectx.ExecuteProgramFn(prog)
}

// execSQLDescribeTableFn handles SQL <alias> DESCRIBE <table> with HandlerDeps.
func execSQLDescribeTableFn(ctx context.Context, s *ast.SQLDescribeTableStmt, deps *HandlerDeps) error {
	conn, err := getOrAutoConnectFn(deps, s.Alias)
	if err != nil {
		return err
	}
	goCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := sqllib.DescribeTable(goCtx, conn, s.Table)
	if err != nil {
		return err
	}
	sqllib.FormatTable(deps.Output, result)
	fmt.Fprintf(deps.Output, "(%d columns)\n", len(result.Rows))
	return nil
}

// ensureSQLManager lazily initializes the SQL connection manager.
func ensureSQLManager(ctx *ExecContext) *sqllib.Manager {
	return ensureSQLManagerFn(execContextToDeps(ctx))
}

func getOrAutoConnect(ctx *ExecContext, alias string) (*sqllib.Connection, error) {
	return getOrAutoConnectFn(execContextToDeps(ctx), alias)
}


func autoConnect(ctx *ExecContext, alias string) error {
	return autoConnectFn(execContextToDeps(ctx), alias)
}








func executeGeneratedMDL(ctx *ExecContext, mdl string) error {
	return executeGeneratedMDLFn(execContextToDeps(ctx), mdl)
}

