// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"fmt"
	"sort"

	"github.com/mendixlabs/mxcli/internal/marketplace/domain"
)

type Service struct {
	contentRepo  domain.ContentRepository
	versionRepo  domain.VersionRepository
	downloadRepo domain.DownloadRepository
	lister       domain.InstalledModuleLister
	installer    domain.ModuleInstaller
}

func NewService(
	contentRepo domain.ContentRepository,
	versionRepo domain.VersionRepository,
	downloadRepo domain.DownloadRepository,
	lister domain.InstalledModuleLister,
	installer domain.ModuleInstaller,
) *Service {
	return &Service{
		contentRepo:  contentRepo,
		versionRepo:  versionRepo,
		downloadRepo: downloadRepo,
		lister:       lister,
		installer:    installer,
	}
}

func (s *Service) Search(ctx context.Context, query string, limit int) ([]*domain.Content, error) {
	if s.contentRepo == nil {
		return nil, nil
	}
	return s.contentRepo.Search(ctx, query, limit)
}

func (s *Service) Get(ctx context.Context, id domain.ContentID) (*domain.Content, error) {
	if s.contentRepo == nil {
		return nil, fmt.Errorf("content repo not available")
	}
	return s.contentRepo.Get(ctx, id)
}

func (s *Service) GetVersions(ctx context.Context, id domain.ContentID) ([]*domain.Version, error) {
	if s.versionRepo == nil {
		return nil, fmt.Errorf("version repo not available")
	}
	return s.versionRepo.GetVersions(ctx, id)
}

func (s *Service) Download(ctx context.Context, id domain.ContentID, versionNumber, outputPath string) (string, error) {
	if s.versionRepo == nil || s.downloadRepo == nil {
		return "", fmt.Errorf("download not available")
	}
	versions, err := s.versionRepo.GetVersions(ctx, id)
	if err != nil {
		return "", err
	}
	version := selectVersion(versions, versionNumber)
	if version == nil {
		if versionNumber != "" {
			return "", fmt.Errorf("version %q not found", versionNumber)
		}
		return "", fmt.Errorf("no versions available")
	}
	if outputPath == "" {
		outputPath = fmt.Sprintf("%s_%s.mpk", version.Name, version.VersionNumber)
	}
	if err := s.downloadRepo.DownloadVersion(ctx, version, outputPath); err != nil {
		return "", err
	}
	return outputPath, nil
}

func (s *Service) Install(ctx context.Context, id domain.ContentID, versionNumber, projectPath string) error {
	if s.contentRepo == nil {
		return fmt.Errorf("content repo not available")
	}
	content, err := s.contentRepo.Get(ctx, id)
	if err != nil {
		return err
	}
	if content.Type != "Module" {
		return fmt.Errorf("content %d is type %q, not a Module", id, content.Type)
	}
	if s.lister != nil {
		installed, _ := s.lister.ListInstalledModules(projectPath)
		for _, m := range installed {
			if m.AppStoreGuid == fmt.Sprintf("%d", id) {
				return fmt.Errorf(
					"module %q is already installed (version %s).\nTarget version: %s.\nIn-place module updates are not applied automatically (they can discard local edits and change persistent entity IDs, which loses data). Update via Studio Pro.",
					m.Name, m.AppStoreVersion, versionNumber,
				)
			}
		}
	}
	mpkPath, err := s.Download(ctx, id, versionNumber, "")
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	if s.installer != nil {
		if err := s.installer.InstallModule(ctx, mpkPath, projectPath); err != nil {
			return fmt.Errorf("install failed: %w", err)
		}
	}
	return nil
}

func (s *Service) ListInstalled(ctx context.Context, projectPath string) ([]domain.InstalledModule, error) {
	if s.lister == nil {
		return nil, fmt.Errorf("project lister not available")
	}
	return s.lister.ListInstalledModules(projectPath)
}

type UpdateResult struct {
	ModuleName       string
	InstalledVersion string
	LatestVersion    string
	Status           string
	Error            string
}

func (s *Service) Update(ctx context.Context, id domain.ContentID, projectPath string) (*UpdateResult, error) {
	if s.lister == nil || s.contentRepo == nil {
		return nil, fmt.Errorf("update not available")
	}
	installed, err := s.lister.ListInstalledModules(projectPath)
	if err != nil {
		return nil, err
	}
	targetID := fmt.Sprintf("%d", id)
	for _, m := range installed {
		if m.AppStoreGuid == targetID {
			content, err := s.contentRepo.Get(ctx, id)
			if err != nil {
				return &UpdateResult{ModuleName: m.Name, Status: "error", Error: err.Error()}, nil
			}
			if content.LatestVersion == nil {
				return &UpdateResult{ModuleName: m.Name, Status: "error", Error: "no version info"}, nil
			}
			latest := content.LatestVersion.VersionNumber
			if latest == m.AppStoreVersion {
				return &UpdateResult{ModuleName: m.Name, InstalledVersion: m.AppStoreVersion, LatestVersion: latest, Status: "up-to-date"}, nil
			}
			return &UpdateResult{
				ModuleName: m.Name, InstalledVersion: m.AppStoreVersion,
				LatestVersion: latest, Status: "update-available",
			}, nil
		}
	}
	return nil, fmt.Errorf("module with content ID %d not found in project", id)
}

func (s *Service) UpdateAll(ctx context.Context, projectPath string) ([]UpdateResult, error) {
	if s.lister == nil {
		return nil, fmt.Errorf("project lister not available")
	}
	installed, err := s.lister.ListInstalledModules(projectPath)
	if err != nil {
		return nil, err
	}
	var results []UpdateResult
	for _, m := range installed {
		id := parseInt(m.AppStoreGuid)
		if id == 0 {
			continue
		}
		r, err := s.Update(ctx, domain.ContentID(id), projectPath)
		if err != nil {
			results = append(results, UpdateResult{ModuleName: m.Name, Status: "error", Error: err.Error()})
			continue
		}
		results = append(results, *r)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].ModuleName < results[j].ModuleName })
	return results, nil
}

func selectVersion(versions []*domain.Version, versionNumber string) *domain.Version {
	if versionNumber != "" {
		for _, v := range versions {
			if v.VersionNumber == versionNumber {
				return v
			}
		}
		return nil
	}
	var newest *domain.Version
	for _, v := range versions {
		if newest == nil || v.PublicationDate.After(newest.PublicationDate) {
			newest = v
		}
	}
	return newest
}

func parseInt(s string) int {
	var n int
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}
