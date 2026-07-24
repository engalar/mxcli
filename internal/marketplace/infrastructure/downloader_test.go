// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/mendixlabs/mxcli/internal/marketplace/domain"
)

func TestDownloader_FollowsRedirectToCDN(t *testing.T) {
	var cdnCalled bool
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cdnCalled = true
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("fake-mpk-content"))
	}))
	t.Cleanup(cdn.Close)

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", cdn.URL+"/module.mpk")
		w.WriteHeader(http.StatusSeeOther)
	}))
	t.Cleanup(api.Close)

	d := NewDownloader(http.DefaultClient, "test-token")
	d.baseURL = api.URL

	dest := filepath.Join(t.TempDir(), "module.mpk")
	err := d.DownloadVersion(context.Background(), &domain.Version{
		DownloadURL: api.URL + "/download",
	}, dest)
	if err != nil {
		t.Fatal(err)
	}
	if !cdnCalled {
		t.Error("CDN was never called")
	}
	data, _ := os.ReadFile(dest)
	if string(data) != "fake-mpk-content" {
		t.Errorf("got %q, want %q", string(data), "fake-mpk-content")
	}
}

func TestDownloader_NoRedirectReturnsError(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(api.Close)

	d := NewDownloader(http.DefaultClient, "test-token")
	d.baseURL = api.URL

	err := d.DownloadVersion(context.Background(), &domain.Version{
		DownloadURL: api.URL + "/download",
	}, filepath.Join(t.TempDir(), "module.mpk"))
	if err == nil {
		t.Fatal("expected error for non-redirect response")
	}
}

func TestDownloader_CDNFailureReturnsError(t *testing.T) {
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(cdn.Close)

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", cdn.URL+"/module.mpk")
		w.WriteHeader(http.StatusSeeOther)
	}))
	t.Cleanup(api.Close)

	d := NewDownloader(http.DefaultClient, "test-token")
	d.baseURL = api.URL

	err := d.DownloadVersion(context.Background(), &domain.Version{
		DownloadURL: api.URL + "/download",
	}, filepath.Join(t.TempDir(), "module.mpk"))
	if err == nil {
		t.Fatal("expected error for CDN failure")
	}
}
