// SPDX-License-Identifier: Apache-2.0

// Package sql provides database connectivity for mxcli.
//
// Import this package for side-effect driver registration.
//
// Build tags:
//   - default:  all three drivers (PostgreSQL + SQL Server + Oracle, ~18 MB)
//   - nooracle: PostgreSQL + SQL Server only (~5 MB saved; omits Oracle driver)
package sql

import (
	_ "github.com/jackc/pgx/v5/stdlib"  // registers "pgx" driver
	_ "github.com/microsoft/go-mssqldb" // registers "sqlserver" driver
)
