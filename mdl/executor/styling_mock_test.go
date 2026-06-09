// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
)

// ---------------------------------------------------------------------------
// execShowDesignProperties
// ---------------------------------------------------------------------------

func TestShowDesignProperties_NotConnected(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return false },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execShowDesignProperties(ctx, &ast.ShowDesignPropertiesStmt{})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not connected")
}

func TestShowDesignProperties_NoMprPath(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	ctx.MprPath = ""
	err := execShowDesignProperties(ctx, &ast.ShowDesignPropertiesStmt{})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "project path")
}

// NOTE: execShowDesignProperties happy path requires loadThemeRegistry which
// reads design-properties.json from the filesystem. Would need a temp dir with
// a valid theme structure to test. Tracked separately.

// ---------------------------------------------------------------------------
// execDescribeStyling
// ---------------------------------------------------------------------------

func TestDescribeStyling_NotConnected(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return false },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execDescribeStyling(ctx, &ast.DescribeStylingStmt{
		ContainerType: "page",
		ContainerName: ast.QualifiedName{Module: "Mod", Name: "Home"},
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not connected")
}

// NOTE: execDescribeStyling happy path calls getPageWidgetsFromRaw /
// getSnippetWidgetsFromRaw which use ctx.Backend.GetRawUnit for BSON parsing.
// MockBackend has GetRawUnitFunc but producing valid BSON test data for the
// page widget walker is non-trivial. Tracked separately.

// ---------------------------------------------------------------------------
// execAlterStyling
// ---------------------------------------------------------------------------

func TestAlterStyling_NotConnected(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return false },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	err := execAlterStyling(ctx, &ast.AlterStylingStmt{
		ContainerType: "page",
		ContainerName: ast.QualifiedName{Module: "Mod", Name: "Home"},
		WidgetName:    "container1",
	})
	assertError(t, err)
	assertContainsStr(t, err.Error(), "not connected")
}

// NOTE: execAlterStyling happy path uses walkPageWidgets / walkSnippetWidgets
// (reflection-based applyStylingAssignments on real Page/Snippet structs) +
// ListPages/UpdatePage. The reflection walker needs real page struct data.
// ConnectedForWrite delegates to Connected — cannot differentiate in mock.
// Tracked separately.
