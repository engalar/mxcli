// SPDX-License-Identifier: Apache-2.0

//go:build !nooracle

package sql

import (
	_ "github.com/sijms/go-ora/v2" // registers "oracle" driver
)
