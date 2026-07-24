// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mendixlabs/mxcli/internal/marketplace/domain"
)

type Downloader struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

func NewDownloader(httpClient *http.Client, token string) *Downloader {
	return &Downloader{httpClient: httpClient, token: token}
}

func (d *Downloader) DownloadVersion(ctx context.Context, version *domain.Version, destPath string) error {
	dlURL := version.DownloadURL
	if d.baseURL != "" {
		dlURL = d.baseURL + "/download"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", dlURL, nil)
	if err != nil {
		return fmt.Errorf("download request: %w", err)
	}
	req.Header.Set("Authorization", "MxToken "+d.token)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download auth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther && resp.StatusCode != http.StatusFound {
		return fmt.Errorf("download expected 303 redirect, got %d", resp.StatusCode)
	}

	cdnURL := resp.Header.Get("Location")
	if cdnURL == "" {
		return fmt.Errorf("download: no Location header in redirect response")
	}

	cdnResp, err := d.httpClient.Get(cdnURL)
	if err != nil {
		return fmt.Errorf("cdn download: %w", err)
	}
	defer cdnResp.Body.Close()

	if cdnResp.StatusCode != http.StatusOK {
		return fmt.Errorf("cdn download: HTTP %d", cdnResp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	tmpPath := destPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	if _, err := io.Copy(f, cdnResp.Body); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
