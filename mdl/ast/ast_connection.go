// SPDX-License-Identifier: Apache-2.0

package ast

// ============================================================================
// Connection Statements
// ============================================================================

// ConnectStmt represents: CONNECT LOCAL 'path' or CONNECT TO FILESYSTEM 'path'
type ConnectStmt struct {
	Path string
}

func (s *ConnectStmt) isStatement()     {}
func (s *ConnectStmt) TypeName() string { return "Connect" }

// DisconnectStmt represents: DISCONNECT
type DisconnectStmt struct{}

func (s *DisconnectStmt) isStatement()     {}
func (s *DisconnectStmt) TypeName() string { return "Disconnect" }

// StatusStmt represents: STATUS or SHOW STATUS
type StatusStmt struct{}

func (s *StatusStmt) isStatement()     {}
func (s *StatusStmt) TypeName() string { return "Status" }
