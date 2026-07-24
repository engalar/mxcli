// SPDX-License-Identifier: Apache-2.0

package domain

import "context"

type ContentRepository interface {
	Search(ctx context.Context, query string, limit int) ([]*Content, error)
	Get(ctx context.Context, id ContentID) (*Content, error)
}

type VersionRepository interface {
	GetVersions(ctx context.Context, id ContentID) ([]*Version, error)
}

type DownloadRepository interface {
	DownloadVersion(ctx context.Context, version *Version, destPath string) error
}

type ProjectRepository interface {
	ListInstalledModules(projectPath string) ([]InstalledModule, error)
	InstallModule(ctx context.Context, mpkPath, projectPath string) error
}
