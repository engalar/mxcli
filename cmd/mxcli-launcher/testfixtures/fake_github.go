// SPDX-License-Identifier: Apache-2.0

package testfixtures

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

// FakeGitHub is a configurable httptest.Server that mimics GitHub releases API.
// Fields must be set before calling NewFakeGitHub; do not modify after creation.
type FakeGitHub struct {
	// LatestTag is the version returned by the releases/latest endpoint.
	LatestTag string
	// Payload holds the fake daemon archive; built by BuildDaemonPayload.
	Payload *DaemonPayload
	// CorruptBinary causes the archive download to return wrong content.
	CorruptBinary bool
	// DownloadCut truncates the download response after this many bytes (0 = no cut).
	DownloadCut int
	// StatusCode, if non-zero, overrides all responses with this HTTP status.
	StatusCode int

	server     *httptest.Server
	mu         sync.Mutex
	requestLog []string
}

// NewFakeGitHub starts the fake server and registers t.Cleanup to close it.
func NewFakeGitHub(t *testing.T, cfg *FakeGitHub) *FakeGitHub {
	t.Helper()
	cfg.server = httptest.NewServer(http.HandlerFunc(cfg.handle))
	t.Cleanup(cfg.server.Close)
	return cfg
}

// Client returns an *http.Client that redirects api.github.com and github.com
// traffic to the fake server. Inject into Env.HTTPClient.
func (f *FakeGitHub) Client() *http.Client {
	fakeURL, _ := url.Parse(f.server.URL)
	return &http.Client{
		Transport: &redirectTransport{
			base:    http.DefaultTransport,
			fakeURL: fakeURL,
		},
	}
}

// RequestLog returns a snapshot of all request paths received so far.
func (f *FakeGitHub) RequestLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.requestLog))
	copy(out, f.requestLog)
	return out
}

func (f *FakeGitHub) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requestLog = append(f.requestLog, r.URL.Path)
	f.mu.Unlock()

	if f.StatusCode != 0 {
		http.Error(w, "injected error", f.StatusCode)
		return
	}

	path := r.URL.Path
	switch {
	case strings.Contains(path, "/releases/latest"):
		fmt.Fprintf(w, `{"tag_name":%q}`, f.LatestTag)

	case strings.Contains(path, "SHA256SUMS"):
		checksum := f.Payload.Checksum
		if f.CorruptBinary {
			checksum = CorruptChecksum()
		}
		fmt.Fprintf(w, "%s  %s\n", checksum, f.Payload.AssetName)

	case strings.Contains(path, ".tar.zst") || strings.Contains(path, ".zip"):
		data := f.Payload.Archive
		if f.CorruptBinary {
			data = []byte("this is not a valid archive")
		}
		if f.DownloadCut > 0 && f.DownloadCut < len(data) {
			data = data[:f.DownloadCut]
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(data)

	case strings.Contains(path, "/releases") && !strings.Contains(path, "/releases/"):
		// Serve the releases list endpoint used by fetchLatestTagWithPrefix.
		// Returns a JSON array with the single configured release.
		fmt.Fprintf(w, `[{"tag_name":%q}]`, f.LatestTag)

	default:
		http.NotFound(w, r)
	}
}

// redirectTransport rewrites requests targeting api.github.com or github.com
// to the fake server URL.
type redirectTransport struct {
	base    http.RoundTripper
	fakeURL *url.URL
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Host
	if host == "api.github.com" || host == "github.com" {
		clone := req.Clone(req.Context())
		clone.URL.Scheme = t.fakeURL.Scheme
		clone.URL.Host = t.fakeURL.Host
		clone.Host = t.fakeURL.Host
		return t.base.RoundTrip(clone)
	}
	return t.base.RoundTrip(req)
}
