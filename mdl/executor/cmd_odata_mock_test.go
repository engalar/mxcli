// SPDX-License-Identifier: Apache-2.0

package executor

import (
	"testing"

	"github.com/mendixlabs/mxcli/mdl/ast"
	"github.com/mendixlabs/mxcli/mdl/backend/mock"
	"github.com/mendixlabs/mxcli/mdl/visitor"
	"github.com/mendixlabs/mxcli/model"
	"github.com/mendixlabs/mxcli/sdk/domainmodel"
)

func TestShowODataClients_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	svc := &model.ConsumedODataService{
		BaseElement:  model.BaseElement{ID: nextID("cos")},
		ContainerID:  mod.ID,
		Name:         "PetStoreClient",
		MetadataUrl:  "https://example.com/$metadata",
		Version:      "1.0",
		ODataVersion: "4.0",
	}
	h := mkHierarchy(mod)
	withContainer(h, svc.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListConsumedODataServicesFunc: func() ([]*model.ConsumedODataService, error) {
			return []*model.ConsumedODataService{svc}, nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, listODataClients(ctx, ""))

	out := buf.String()
	assertContainsStr(t, out, "QualifiedName")
	assertContainsStr(t, out, "MyModule.PetStoreClient")
}

func TestShowODataServices_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	svc := &model.PublishedODataService{
		BaseElement:  model.BaseElement{ID: nextID("pos")},
		ContainerID:  mod.ID,
		Name:         "CatalogService",
		Path:         "/odata/v1",
		Version:      "1.0",
		ODataVersion: "4.0",
	}
	h := mkHierarchy(mod)
	withContainer(h, svc.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListPublishedODataServicesFunc: func() ([]*model.PublishedODataService, error) {
			return []*model.PublishedODataService{svc}, nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, listODataServices(ctx, ""))

	out := buf.String()
	assertContainsStr(t, out, "QualifiedName")
	assertContainsStr(t, out, "MyModule.CatalogService")
}

func TestDescribeODataClient_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	svc := &model.ConsumedODataService{
		BaseElement:  model.BaseElement{ID: nextID("cos")},
		ContainerID:  mod.ID,
		Name:         "PetStoreClient",
		MetadataUrl:  "https://example.com/$metadata",
		Version:      "2.0",
		ODataVersion: "4.0",
	}
	h := mkHierarchy(mod)
	withContainer(h, svc.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListConsumedODataServicesFunc: func() ([]*model.ConsumedODataService, error) {
			return []*model.ConsumedODataService{svc}, nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, describeODataClient(ctx, ast.QualifiedName{Module: "MyModule", Name: "PetStoreClient"}))

	out := buf.String()
	assertContainsStr(t, out, "create odata client")
	assertContainsStr(t, out, "MyModule.PetStoreClient")
	assertContainsStr(t, out, "https://example.com/$metadata")
	assertContainsStr(t, out, "2.0")
}

func TestDescribeODataClient_NotFound(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListConsumedODataServicesFunc: func() ([]*model.ConsumedODataService, error) {
			return nil, nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, describeODataClient(ctx, ast.QualifiedName{Module: "MyModule", Name: "NoSuch"}))
}

func TestShowODataClients_FilterByModule(t *testing.T) {
	mod1 := mkModule("Alpha")
	mod2 := mkModule("Beta")
	svc1 := &model.ConsumedODataService{
		BaseElement: model.BaseElement{ID: nextID("cos")},
		ContainerID: mod1.ID,
		Name:        "AlphaSvc",
	}
	svc2 := &model.ConsumedODataService{
		BaseElement: model.BaseElement{ID: nextID("cos")},
		ContainerID: mod2.ID,
		Name:        "BetaSvc",
	}
	h := mkHierarchy(mod1, mod2)
	withContainer(h, svc1.ContainerID, mod1.ID)
	withContainer(h, svc2.ContainerID, mod2.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListConsumedODataServicesFunc: func() ([]*model.ConsumedODataService, error) {
			return []*model.ConsumedODataService{svc1, svc2}, nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, listODataClients(ctx, "Alpha"))

	out := buf.String()
	assertContainsStr(t, out, "Alpha.AlphaSvc")
	assertNotContainsStr(t, out, "Beta.BetaSvc")
}

func TestShowODataServices_FilterByModule(t *testing.T) {
	mod := mkModule("Sales")
	svc := &model.PublishedODataService{
		BaseElement: model.BaseElement{ID: nextID("pos")},
		ContainerID: mod.ID,
		Name:        "SalesSvc",
	}
	h := mkHierarchy(mod)
	withContainer(h, svc.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListPublishedODataServicesFunc: func() ([]*model.PublishedODataService, error) {
			return []*model.PublishedODataService{svc}, nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, listODataServices(ctx, "Sales"))
	assertContainsStr(t, buf.String(), "Sales.SalesSvc")
}

func TestDescribeODataService_Mock(t *testing.T) {
	mod := mkModule("MyModule")
	svc := &model.PublishedODataService{
		BaseElement:  model.BaseElement{ID: nextID("pos")},
		ContainerID:  mod.ID,
		Name:         "CatalogService",
		Path:         "/odata/v1",
		Version:      "1.0",
		ODataVersion: "4.0",
		Namespace:    "MyApp",
	}
	h := mkHierarchy(mod)
	withContainer(h, svc.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListPublishedODataServicesFunc: func() ([]*model.PublishedODataService, error) {
			return []*model.PublishedODataService{svc}, nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, describeODataService(ctx, ast.QualifiedName{Module: "MyModule", Name: "CatalogService"}))

	out := buf.String()
	assertContainsStr(t, out, "create odata service")
	assertContainsStr(t, out, "MyModule.CatalogService")
}

func TestDescribeODataService_NotFound(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListPublishedODataServicesFunc: func() ([]*model.PublishedODataService, error) {
			return nil, nil
		},
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	assertError(t, describeODataService(ctx, ast.QualifiedName{Module: "X", Name: "NoSuch"}))
}

// TestCreateExternalEntity_RejectsNonExistentClient verifies that CREATE EXTERNAL ENTITY
// returns an error when the referenced OData client does not exist (issue #417).
func TestCreateExternalEntity_RejectsNonExistentClient(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	dm := &domainmodel.DomainModel{BaseElement: model.BaseElement{ID: nextID("dm")}, ContainerID: mod.ID}

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{mod}, nil
		},
		GetDomainModelFunc: func(id model.ID) (*domainmodel.DomainModel, error) {
			return dm, nil
		},
		ListConsumedODataServicesFunc: func() ([]*model.ConsumedODataService, error) {
			return nil, nil // no services registered
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	stmt := &ast.CreateExternalEntityStmt{
		Name:       ast.QualifiedName{Module: "MyModule", Name: "FakeEntity"},
		ServiceRef: ast.QualifiedName{Module: "MyModule", Name: "NonExistentClient"},
		EntitySet:  "Products",
	}
	err := execCreateExternalEntity(ctx, stmt)
	assertError(t, err)
	assertContainsStr(t, err.Error(), "odata client not found")
}

func TestCreateExternalEntity_AcceptsExistingClient(t *testing.T) {
	mod := mkModule("MyModule")
	h := mkHierarchy(mod)
	svc := &model.ConsumedODataService{
		BaseElement: model.BaseElement{ID: nextID("cos")},
		ContainerID: mod.ID,
		Name:        "ProductsClient",
	}
	withContainer(h, svc.ContainerID, mod.ID)
	dm := &domainmodel.DomainModel{BaseElement: model.BaseElement{ID: nextID("dm")}, ContainerID: mod.ID}

	var created *domainmodel.Entity
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{mod}, nil
		},
		GetDomainModelFunc: func(id model.ID) (*domainmodel.DomainModel, error) {
			return dm, nil
		},
		ListConsumedODataServicesFunc: func() ([]*model.ConsumedODataService, error) {
			return []*model.ConsumedODataService{svc}, nil
		},
		CreateEntityFunc: func(dmID model.ID, entity *domainmodel.Entity) error {
			created = entity
			return nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	stmt := &ast.CreateExternalEntityStmt{
		Name:       ast.QualifiedName{Module: "MyModule", Name: "Product"},
		ServiceRef: ast.QualifiedName{Module: "MyModule", Name: "ProductsClient"},
		EntitySet:  "Products",
	}
	assertNoError(t, execCreateExternalEntity(ctx, stmt))
	if created == nil {
		t.Error("expected CreateEntity to be called")
	}
}

// TestCreateExternalEntity_AllowCreateChangeLocally_Issue534 verifies that
// AllowCreateChangeLocally is passed through to the new entity.
func TestCreateExternalEntity_AllowCreateChangeLocally_Issue534(t *testing.T) {
	mod := mkModule("TripPin")
	h := mkHierarchy(mod)
	svc := &model.ConsumedODataService{
		BaseElement: model.BaseElement{ID: nextID("cos")},
		ContainerID: mod.ID,
		Name:        "TripPinClient",
	}
	withContainer(h, svc.ContainerID, mod.ID)
	dm := &domainmodel.DomainModel{BaseElement: model.BaseElement{ID: nextID("dm")}, ContainerID: mod.ID}

	var created *domainmodel.Entity
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListModulesFunc: func() ([]*model.Module, error) {
			return []*model.Module{mod}, nil
		},
		GetDomainModelFunc: func(id model.ID) (*domainmodel.DomainModel, error) {
			return dm, nil
		},
		ListConsumedODataServicesFunc: func() ([]*model.ConsumedODataService, error) {
			return []*model.ConsumedODataService{svc}, nil
		},
		CreateEntityFunc: func(dmID model.ID, entity *domainmodel.Entity) error {
			created = entity
			return nil
		},
	}

	ctx, _ := newMockCtx(t, withBackend(mb), withHierarchy(h))
	stmt := &ast.CreateExternalEntityStmt{
		Name:                     ast.QualifiedName{Module: "TripPin", Name: "People"},
		ServiceRef:               ast.QualifiedName{Module: "TripPin", Name: "TripPinClient"},
		EntitySet:                "People",
		AllowCreateChangeLocally: true,
	}
	assertNoError(t, execCreateExternalEntity(ctx, stmt))
	if created == nil {
		t.Fatal("expected CreateEntity to be called")
	}
	if !created.CreateChangeLocally {
		t.Errorf("expected CreateChangeLocally = true, got false")
	}
}

// TestDescribeODataService_ExposeRoundtrip verifies that DESCRIBE ODATA SERVICE
// output for entities with key/filterable/sortable members is valid MDL that
// the parser can re-parse (issue #400).
func TestDescribeODataService_ExposeRoundtrip(t *testing.T) {
	mod := mkModule("MyModule")
	svc := &model.PublishedODataService{
		BaseElement:  model.BaseElement{ID: nextID("pos")},
		ContainerID:  mod.ID,
		Name:         "CatalogService",
		Path:         "/odata/v1",
		Version:      "1.0",
		ODataVersion: "4.0",
		EntityTypes: []*model.PublishedEntityType{
			{
				Entity:      "MyModule.Order",
				ExposedName: "Orders",
				Members: []*model.PublishedMember{
					{Name: "Id", ExposedName: "Id", IsPartOfKey: true},
					{Name: "Name", ExposedName: "Name", Filterable: true, Sortable: true},
				},
			},
		},
		EntitySets: []*model.PublishedEntitySet{
			{
				ExposedName:    "Orders",
				EntityTypeName: "MyModule.Order",
				ReadMode:       "Readable",
				InsertMode:     "NotSupported",
			},
		},
	}
	h := mkHierarchy(mod)
	withContainer(h, svc.ContainerID, mod.ID)

	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
		ListPublishedODataServicesFunc: func() ([]*model.PublishedODataService, error) {
			return []*model.PublishedODataService{svc}, nil
		},
	}

	ctx, buf := newMockCtx(t, withBackend(mb), withHierarchy(h))
	assertNoError(t, describeODataService(ctx, ast.QualifiedName{Module: "MyModule", Name: "CatalogService"}))

	out := buf.String()
	assertContainsStr(t, out, "IsPartOfKey")

	_, errs := visitor.Build(out)
	if len(errs) > 0 {
		t.Errorf("DESCRIBE output failed to parse (roundtrip broken):\n%s\nErrors:", out)
		for _, e := range errs {
			t.Errorf("  %v", e)
		}
	}
}

func TestCreateODataClient_InvalidMetadataURL(t *testing.T) {
	mb := &mock.MockBackend{
		IsConnectedFunc: func() bool { return true },
	}
	ctx, _ := newMockCtx(t, withBackend(mb))
	stmt := &ast.CreateODataClientStmt{
		Name:        ast.QualifiedName{Module: "MyModule", Name: "BadClient"},
		MetadataUrl: "not-a-url",
	}
	err := createODataClient(ctx, stmt)
	assertError(t, err)
	assertContainsStr(t, err.Error(), "MetadataUrl")
}

func TestCreateODataClient_ValidMetadataURLs(t *testing.T) {
	for _, validURL := range []string{
		"https://example.com/odata/$metadata",
		"http://localhost:8080/$metadata",
		"file:///tmp/metadata.xml",
		"./metadata.xml",
		"../service/metadata.xml",
		"/abs/path/metadata.xml",
	} {
		err := validateMetadataURL(validURL)
		if err != nil {
			t.Errorf("expected %q to be valid, got error: %v", validURL, err)
		}
	}
}

func TestValidateMetadataURL_RejectsBarWords(t *testing.T) {
	for _, bad := range []string{"not-a-url", "justword", "no-scheme-no-dots"} {
		err := validateMetadataURL(bad)
		if err == nil {
			t.Errorf("expected %q to be rejected, but got nil error", bad)
		}
	}
}
