// SPDX-License-Identifier: Apache-2.0

// Package exprcheck implements a robust recursive-descent parser for
// Mendix microflow expressions, driven by the mined grammar in
// generated/exprgrammar. It produces RobustExpr trees with inline
// Hint diagnostics for hint-emitting consumers (mxcli check, mxcli
// exec).
package exprcheck
