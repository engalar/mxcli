// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	sqllib "github.com/mendixlabs/mxcli/sql"
	"github.com/spf13/cobra"
)

var sqlCmd = &cobra.Command{
	Use:   "sql [query]",
	Short: "Execute a SQL query against an external database",
	Long: `Execute a SQL query against an external database (PostgreSQL, Oracle, SQL Server).

Connection credentials are resolved in order:
  1. --dsn flag
  2. Environment variable MXCLI_SQL_<ALIAS>_DSN
  3. Environment variable MXCLI_SQL_<DRIVER>_DSN
  4. .mxcli/connections.yaml file

The driver is auto-detected from the DSN scheme (postgres://, oracle://,
sqlserver://) unless --driver is specified explicitly.

Config file format (.mxcli/connections.yaml):

  # Map format (recommended):
  connections:
    mydb:
      driver: postgres
      dsn: postgres://user:pass@localhost:5432/db
    ora:
      driver: oracle
      dsn: oracle://system:pass@host:1521/service

  # List format (also supported):
  connections:
    - alias: mydb
      driver: postgres
      dsn: postgres://user:pass@localhost:5432/db

Examples:
  # Query with explicit DSN (driver auto-detected from scheme)
  mxcli sql --dsn 'postgres://user:pass@localhost/db' "SELECT 1"

  # Query with explicit driver and DSN
  mxcli sql --driver postgres --dsn 'postgres://user:pass@localhost/db' "SELECT 1"

  # Query using alias (DSN resolved from env or config)
  mxcli sql --alias mydb "SELECT * FROM users LIMIT 10"

  # List tables
  mxcli sql --alias mydb --tables

  # Describe a table
  mxcli sql --alias mydb --describe users

  # JSON output
  mxcli sql --alias mydb --json "SELECT * FROM orders"
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		driverName, _ := cmd.Flags().GetString("driver")
		dsn, _ := cmd.Flags().GetString("dsn")
		alias, _ := cmd.Flags().GetString("alias")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		tables, _ := cmd.Flags().GetBool("tables")
		views, _ := cmd.Flags().GetBool("views")
		functions, _ := cmd.Flags().GetBool("functions")
		describe, _ := cmd.Flags().GetString("describe")

		if alias == "" {
			alias = "default"
		}

		var driver sqllib.DriverName
		if driverName != "" {
			var err error
			driver, err = sqllib.ParseDriver(driverName)
			if err != nil {
				return fmt.Errorf("parsing driver: %w", err)
			}
		}

		resolved, err := sqllib.ResolveConnection(sqllib.ResolveOptions{
			DSN:    dsn,
			Alias:  alias,
			Driver: driver,
		})
		if err != nil {
			return fmt.Errorf("resolving connection: %w", err)
		}

		if resolved.Driver == "" {
			resolved.Driver = sqllib.DriverPostgres
		}

		mgr := sqllib.NewManager()
		defer mgr.CloseAll()

		if err := mgr.Connect(resolved.Driver, resolved.DSN, alias); err != nil {
			return fmt.Errorf("connecting: %w", err)
		}

		conn, _ := mgr.Get(alias)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var result *sqllib.QueryResult
		switch {
		case tables:
			result, err = sqllib.ShowTables(ctx, conn)
		case views:
			result, err = sqllib.ShowViews(ctx, conn)
		case functions:
			result, err = sqllib.ShowFunctions(ctx, conn)
		case describe != "":
			result, err = sqllib.DescribeTable(ctx, conn, describe)
		default:
			if len(args) == 0 {
				return fmt.Errorf("query argument required (or use --tables / --views / --functions / --describe)")
			}
			result, err = sqllib.Execute(ctx, conn, args[0])
		}

		if err != nil {
			return fmt.Errorf("query failed: %w", err)
		}

		if jsonOutput {
			if err := sqllib.FormatJSON(os.Stdout, result); err != nil {
				return fmt.Errorf("formatting JSON: %w", err)
			}
		} else {
			sqllib.FormatTable(os.Stdout, result)
		}

		fmt.Fprintf(os.Stderr, "(%d rows)\n", len(result.Rows))
		return nil
	},
}

func init() {
	sqlCmd.Flags().String("driver", "", "Database driver (postgres, oracle, sqlserver). Auto-detected from DSN scheme if not set.")
	sqlCmd.Flags().String("dsn", "", "Database connection string")
	sqlCmd.Flags().String("alias", "", "Connection alias for DSN resolution")
	sqlCmd.Flags().BoolP("json", "j", false, "Output as JSON array")
	sqlCmd.Flags().Bool("tables", false, "List tables")
	sqlCmd.Flags().Bool("views", false, "List views")
	sqlCmd.Flags().Bool("functions", false, "List functions and procedures")
	sqlCmd.Flags().String("describe", "", "Describe the named table's or view's columns")
}
