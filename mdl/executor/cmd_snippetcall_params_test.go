// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"strings"
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	repostesting "github.com/mendixlabs/mxcli/mdl/repos/testing"
	"github.com/mendixlabs/mxcli/mdl/types"
	"github.com/mendixlabs/mxcli/model"
	genPg "github.com/mendixlabs/mxcli/modelsdk/gen/pages"
)

// mkSnippetWithParam creates a gen-typed snippet with one declared parameter.
func mkSnippetWithParam(id, snippetName, paramName string) *genPg.Snippet {
	snp := mkSnippetGen(id, snippetName)
	sp := genPg.NewSnippetParameter()
	sp.SetName(paramName)
	snp.AddParameters(sp)
	return snp
}

// makeSnippetRepoWithQN builds a snippet repo that supports both ListAll and
// FindByQualifiedName for the given snippets. moduleName is used to construct
// qualified names for FindByQualifiedName lookups.
func makeSnippetRepoWithQN(snps []*genPg.Snippet, moduleName string, containerID model.ID) *repostesting.RecordingSnippetRepository {
	byQN := make(map[string]*genPg.Snippet, len(snps))
	for _, s := range snps {
		if s != nil {
			byQN[moduleName+"."+s.Name()] = s
		}
	}
	return &repostesting.RecordingSnippetRepository{
		ListAllFunc:          func() ([]*genPg.Snippet, error) { return snps, nil },
		GetContainerUUIDFunc: func(_ model.ID) (model.ID, error) { return containerID, nil },
		FindByQualifiedNameFunc: func(qn string) (*genPg.Snippet, error) {
			return byQN[qn], nil
		},
	}
}

// newPageBuilder returns a pageBuilder wired to the given mock backend and hierarchy.
func newPageBuilder(mb *mock.MockBackend, h *ContainerHierarchy, moduleName string) *pageBuilder {
	var modID model.ID
	for id, name := range h.moduleNames {
		if name == moduleName {
			modID = id
			break
		}
	}
	cache := &executorCache{hierarchy: h}
	return &pageBuilder{
		backend:          mb,
		moduleID:         modID,
		moduleName:       moduleName,
		widgetScope:      make(map[string]model.ID),
		paramScope:       make(map[string]model.ID),
		paramEntityNames: make(map[string]string),
		execCache:        cache,
		widgetBackend:    mb,
	}
}

// TestSnippetCall_MissingParam_ReturnsError verifies that placing a SNIPPETCALL
// without the required Params yields a validation error (issue #291 — guard).
func TestSnippetCall_MissingParam_ReturnsError(t *testing.T) {
	mod := mkModule("Mod")
	snp := mkSnippetWithParam(string(nextID("snp")), "MySnippet", "Asset")

	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:     func() bool { return true },
		ListModulesFunc:     func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc:     func() ([]*types.FolderInfo, error) { return nil, nil },
		ListSnippetsGenFunc: func() ([]*genPg.Snippet, error) { return []*genPg.Snippet{snp}, nil },
	}

	pb := newPageBuilder(mb, h, "Mod")
	pb.snippetsRepo = makeSnippetRepoWithQN([]*genPg.Snippet{snp}, "Mod", mod.ID)
	w := &ast.WidgetV3{
		Type:       "SNIPPETCALL",
		Name:       "sc1",
		Properties: map[string]any{"Snippet": "Mod.MySnippet"},
	}

	_, err := pb.buildSnippetCallV3(w)
	if err == nil {
		t.Fatal("expected validation error for missing snippet parameter, got nil")
	}
	if !strings.Contains(err.Error(), "Asset") {
		t.Errorf("error should mention missing parameter 'Asset', got: %v", err)
	}
}

// TestSnippetCall_WithParam_Succeeds verifies that a SNIPPETCALL with correct
// Params mapping passes validation and produces a SnippetCallWidget with mappings.
func TestSnippetCall_WithParam_Succeeds(t *testing.T) {
	mod := mkModule("Mod")
	snp := mkSnippetWithParam(string(nextID("snp")), "MySnippet", "Asset")

	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:     func() bool { return true },
		ListModulesFunc:     func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc:     func() ([]*types.FolderInfo, error) { return nil, nil },
		ListSnippetsGenFunc: func() ([]*genPg.Snippet, error) { return []*genPg.Snippet{snp}, nil },
	}

	pb := newPageBuilder(mb, h, "Mod")
	pb.snippetsRepo = makeSnippetRepoWithQN([]*genPg.Snippet{snp}, "Mod", mod.ID)
	w := &ast.WidgetV3{
		Type: "SNIPPETCALL",
		Name: "sc1",
		Properties: map[string]any{
			"Snippet": "Mod.MySnippet",
			"Params":  []ast.SnippetCallParam{{ParamName: "Asset", Variable: "$Asset"}},
		},
	}

	sc, err := pb.buildSnippetCallV3(w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sc.ParameterMappings) != 1 {
		t.Fatalf("ParameterMappings: want 1, got %d", len(sc.ParameterMappings))
	}
	if sc.ParameterMappings[0].ParamName != "Asset" {
		t.Errorf("ParamName: want Asset, got %q", sc.ParameterMappings[0].ParamName)
	}
	if sc.ParameterMappings[0].Argument != "$Asset" {
		t.Errorf("Argument: want $Asset, got %q", sc.ParameterMappings[0].Argument)
	}
}

// TestSnippetCall_NoParam_NoSnippetParams_Succeeds verifies that a parameterless
// snippet call against a snippet with no declared parameters works (no regression).
func TestSnippetCall_NoParam_NoSnippetParams_Succeeds(t *testing.T) {
	mod := mkModule("Mod")
	snp := mkSnippetGen(string(nextID("snp")), "Footer") // no parameters declared

	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:     func() bool { return true },
		ListModulesFunc:     func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc:     func() ([]*types.FolderInfo, error) { return nil, nil },
		ListSnippetsGenFunc: func() ([]*genPg.Snippet, error) { return []*genPg.Snippet{snp}, nil },
	}

	pb := newPageBuilder(mb, h, "Mod")
	pb.snippetsRepo = makeSnippetRepoWithQN([]*genPg.Snippet{snp}, "Mod", mod.ID)
	w := &ast.WidgetV3{
		Type:       "SNIPPETCALL",
		Name:       "sc1",
		Properties: map[string]any{"Snippet": "Mod.Footer"},
	}

	sc, err := pb.buildSnippetCallV3(w)
	if err != nil {
		t.Fatalf("unexpected error for parameterless snippet: %v", err)
	}
	if len(sc.ParameterMappings) != 0 {
		t.Errorf("expected empty ParameterMappings, got %d", len(sc.ParameterMappings))
	}
}

// TestSnippetCall_DollarPrefixParam_Succeeds verifies that param names with $
// prefix (as the user might write $Asset: $var) are matched correctly.
func TestSnippetCall_DollarPrefixParam_Succeeds(t *testing.T) {
	mod := mkModule("Mod")
	snp := mkSnippetWithParam(string(nextID("snp")), "MySnippet", "Asset")

	h := mkHierarchy(mod)

	mb := &mock.MockBackend{
		IsConnectedFunc:     func() bool { return true },
		ListModulesFunc:     func() ([]*model.Module, error) { return []*model.Module{mod}, nil },
		ListFoldersFunc:     func() ([]*types.FolderInfo, error) { return nil, nil },
		ListSnippetsGenFunc: func() ([]*genPg.Snippet, error) { return []*genPg.Snippet{snp}, nil },
	}

	pb := newPageBuilder(mb, h, "Mod")
	pb.snippetsRepo = makeSnippetRepoWithQN([]*genPg.Snippet{snp}, "Mod", mod.ID)
	// User writes $Asset: $Asset (dollar-prefixed param name)
	w := &ast.WidgetV3{
		Type: "SNIPPETCALL",
		Name: "sc1",
		Properties: map[string]any{
			"Snippet": "Mod.MySnippet",
			"Params":  []ast.SnippetCallParam{{ParamName: "$Asset", Variable: "$Asset"}},
		},
	}

	sc, err := pb.buildSnippetCallV3(w)
	if err != nil {
		t.Fatalf("unexpected error with $-prefixed param name: %v", err)
	}
	if len(sc.ParameterMappings) != 1 {
		t.Fatalf("ParameterMappings: want 1, got %d", len(sc.ParameterMappings))
	}
	if sc.ParameterMappings[0].ParamName != "Asset" {
		t.Errorf("ParamName: want Asset (stripped), got %q", sc.ParameterMappings[0].ParamName)
	}
}
