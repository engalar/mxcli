// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mendixlabs/mxcli/internal/marketplace/domain"
)

type mockContentRepo struct {
	searchFn func(ctx context.Context, query string, limit int) ([]*domain.Content, error)
	getFn    func(ctx context.Context, id domain.ContentID) (*domain.Content, error)
}

func (m *mockContentRepo) Search(ctx context.Context, query string, limit int) ([]*domain.Content, error) {
	return m.searchFn(ctx, query, limit)
}

func (m *mockContentRepo) Get(ctx context.Context, id domain.ContentID) (*domain.Content, error) {
	return m.getFn(ctx, id)
}

type mockVersionRepo struct {
	getVersionsFn func(ctx context.Context, id domain.ContentID) ([]*domain.Version, error)
}

func (m *mockVersionRepo) GetVersions(ctx context.Context, id domain.ContentID) ([]*domain.Version, error) {
	return m.getVersionsFn(ctx, id)
}

type mockDownloadRepo struct {
	downloadFn func(ctx context.Context, version *domain.Version, destPath string) error
}

func (m *mockDownloadRepo) DownloadVersion(ctx context.Context, version *domain.Version, destPath string) error {
	return m.downloadFn(ctx, version, destPath)
}

type mockModuleLister struct {
	listFn func(projectPath string) ([]domain.InstalledModule, error)
}

func (m *mockModuleLister) ListInstalledModules(projectPath string) ([]domain.InstalledModule, error) {
	return m.listFn(projectPath)
}

type mockModuleInstaller struct {
	installFn func(ctx context.Context, mpkPath, projectPath string) error
}

func (m *mockModuleInstaller) InstallModule(ctx context.Context, mpkPath, projectPath string) error {
	return m.installFn(ctx, mpkPath, projectPath)
}

func TestService_Search_DelegatesToRepo(t *testing.T) {
	svc := NewService(
		&mockContentRepo{
			searchFn: func(_ context.Context, q string, _ int) ([]*domain.Content, error) {
				if q != "database" {
					return nil, errors.New("wrong query")
				}
				return []*domain.Content{{ContentID: 2888, Publisher: "Mendix"}}, nil
			},
		},
		nil, nil, nil, nil,
	)
	results, err := svc.Search(context.Background(), "database", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ContentID != 2888 {
		t.Errorf("unexpected: %+v", results)
	}
}

func TestService_Get_DelegatesToRepo(t *testing.T) {
	svc := NewService(
		&mockContentRepo{
			getFn: func(_ context.Context, id domain.ContentID) (*domain.Content, error) {
				return &domain.Content{ContentID: id, Publisher: "Mendix"}, nil
			},
		},
		nil, nil, nil, nil,
	)
	c, err := svc.Get(context.Background(), 170)
	if err != nil {
		t.Fatal(err)
	}
	if c.ContentID != 170 {
		t.Errorf("got %d, want 170", c.ContentID)
	}
}

func TestService_GetVersions_Delegates(t *testing.T) {
	svc := NewService(nil,
		&mockVersionRepo{
			getVersionsFn: func(_ context.Context, id domain.ContentID) ([]*domain.Version, error) {
				return []*domain.Version{{VersionNumber: "11.5.0"}}, nil
			},
		},
		nil, nil, nil,
	)
	versions, err := svc.GetVersions(context.Background(), 170)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 || versions[0].VersionNumber != "11.5.0" {
		t.Errorf("unexpected: %+v", versions)
	}
}

func TestService_Download_SelectsLatestVersion(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	svc := NewService(nil,
		&mockVersionRepo{
			getVersionsFn: func(_ context.Context, _ domain.ContentID) ([]*domain.Version, error) {
				return []*domain.Version{
					{VersionNumber: "1.0.0", PublicationDate: yesterday},
					{VersionNumber: "2.0.0", PublicationDate: now},
				}, nil
			},
		},
		&mockDownloadRepo{
			downloadFn: func(_ context.Context, v *domain.Version, dest string) error {
				if v.VersionNumber != "2.0.0" {
					return errors.New("expected latest version 2.0.0")
				}
				return os.WriteFile(dest, []byte("test"), 0644)
			},
		},
		nil, nil,
	)

	dest := filepath.Join(t.TempDir(), "out.mpk")
	path, err := svc.Download(context.Background(), 170, "", dest)
	if err != nil {
		t.Fatal(err)
	}
	if path != dest {
		t.Errorf("got %q, want %q", path, dest)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "test" {
		t.Errorf("got %q, want test", string(data))
	}
}

func TestService_Download_SelectsSpecificVersion(t *testing.T) {
	svc := NewService(nil,
		&mockVersionRepo{
			getVersionsFn: func(_ context.Context, _ domain.ContentID) ([]*domain.Version, error) {
				return []*domain.Version{
					{VersionNumber: "1.0.0"},
					{VersionNumber: "2.0.0"},
				}, nil
			},
		},
		&mockDownloadRepo{
			downloadFn: func(_ context.Context, v *domain.Version, dest string) error {
				if v.VersionNumber != "1.0.0" {
					return errors.New("expected version 1.0.0")
				}
				return os.WriteFile(dest, []byte("v1"), 0644)
			},
		},
		nil, nil,
	)

	dest := filepath.Join(t.TempDir(), "out.mpk")
	path, err := svc.Download(context.Background(), 170, "1.0.0", dest)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "v1" {
		t.Errorf("got %q, want v1", string(data))
	}
}

func TestService_Download_VersionNotFound(t *testing.T) {
	svc := NewService(nil,
		&mockVersionRepo{
			getVersionsFn: func(_ context.Context, _ domain.ContentID) ([]*domain.Version, error) {
				return []*domain.Version{{VersionNumber: "2.0.0"}}, nil
			},
		},
		nil, nil, nil,
	)
	_, err := svc.Download(context.Background(), 170, "9.9.9", "")
	if err == nil {
		t.Fatal("expected error for nonexistent version")
	}
}

func TestService_Install_ChecksModuleType(t *testing.T) {
	svc := NewService(
		&mockContentRepo{
			getFn: func(_ context.Context, _ domain.ContentID) (*domain.Content, error) {
				return &domain.Content{Type: "Widget"}, nil
			},
		},
		nil, nil, nil, nil,
	)
	err := svc.Install(context.Background(), 170, "", "/tmp/test.mpr")
	if err == nil {
		t.Fatal("expected error for non-Module type")
	}
}

func TestService_ListInstalled_Delegates(t *testing.T) {
	svc := NewService(nil, nil, nil,
		&mockModuleLister{
			listFn: func(_ string) ([]domain.InstalledModule, error) {
				return []domain.InstalledModule{
					{Name: "DB Connector", AppStoreGuid: "2888", AppStoreVersion: "7.0.2"},
				}, nil
			},
		},
		nil,
	)
	modules, err := svc.ListInstalled(context.Background(), "/tmp/test.mpr")
	if err != nil {
		t.Fatal(err)
	}
	if len(modules) != 1 || modules[0].Name != "DB Connector" {
		t.Errorf("unexpected: %+v", modules)
	}
}

func TestService_Update_ReportsState(t *testing.T) {
	svc := NewService(
		&mockContentRepo{
			getFn: func(_ context.Context, id domain.ContentID) (*domain.Content, error) {
				return &domain.Content{
					LatestVersion: &domain.Version{VersionNumber: "7.0.3"},
				}, nil
			},
		},
		nil, nil,
		&mockModuleLister{
			listFn: func(_ string) ([]domain.InstalledModule, error) {
				return []domain.InstalledModule{
					{Name: "DB Connector", AppStoreGuid: "2888", AppStoreVersion: "7.0.2"},
				}, nil
			},
		},
		nil,
	)
	result, err := svc.Update(context.Background(), 2888, "/tmp/test.mpr")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "update-available" {
		t.Errorf("status: got %q, want update-available", result.Status)
	}
	if result.InstalledVersion != "7.0.2" || result.LatestVersion != "7.0.3" {
		t.Errorf("versions: %+v", result)
	}
}

func TestService_Update_ReportsUpToDate(t *testing.T) {
	svc := NewService(
		&mockContentRepo{
			getFn: func(_ context.Context, _ domain.ContentID) (*domain.Content, error) {
				return &domain.Content{
					LatestVersion: &domain.Version{VersionNumber: "7.0.3"},
				}, nil
			},
		},
		nil, nil,
		&mockModuleLister{
			listFn: func(_ string) ([]domain.InstalledModule, error) {
				return []domain.InstalledModule{
					{Name: "DB Connector", AppStoreGuid: "2888", AppStoreVersion: "7.0.3"},
				}, nil
			},
		},
		nil,
	)
	result, err := svc.Update(context.Background(), 2888, "/tmp/test.mpr")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "up-to-date" {
		t.Errorf("status: got %q, want up-to-date", result.Status)
	}
}

func TestService_Update_ModuleNotFound(t *testing.T) {
	svc := NewService(nil, nil, nil,
		&mockModuleLister{
			listFn: func(_ string) ([]domain.InstalledModule, error) {
				return []domain.InstalledModule{
					{Name: "DB Connector", AppStoreGuid: "2888"},
				}, nil
			},
		},
		nil,
	)
	_, err := svc.Update(context.Background(), 999, "/tmp/test.mpr")
	if err == nil {
		t.Fatal("expected error for module not found")
	}
}

func TestService_UpdateAll(t *testing.T) {
	svc := NewService(
		&mockContentRepo{
			getFn: func(_ context.Context, id domain.ContentID) (*domain.Content, error) {
				if id == 2888 {
					return &domain.Content{LatestVersion: &domain.Version{VersionNumber: "7.0.3"}}, nil
				}
				return &domain.Content{LatestVersion: &domain.Version{VersionNumber: "11.5.1"}}, nil
			},
		},
		nil, nil,
		&mockModuleLister{
			listFn: func(_ string) ([]domain.InstalledModule, error) {
				return []domain.InstalledModule{
					{Name: "DB Connector", AppStoreGuid: "2888", AppStoreVersion: "7.0.2"},
					{Name: "Commons", AppStoreGuid: "170", AppStoreVersion: "11.5.1"},
				}, nil
			},
		},
		nil,
	)
	results, err := svc.UpdateAll(context.Background(), "/tmp/test.mpr")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ModuleName != "Commons" || results[0].Status != "up-to-date" {
		t.Errorf("result 0 (Commons): %+v", results[0])
	}
	if results[1].ModuleName != "DB Connector" || results[1].Status != "update-available" {
		t.Errorf("result 1 (DB Connector): %+v", results[1])
	}
}
