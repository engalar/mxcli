// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

const sampleContentList = `{
  "items": [
    {
      "contentId": 170,
      "publisher": "Mendix",
      "type": "Module",
      "categories": [{"name": "Utility"}],
      "supportCategory": "Platform",
      "licenseUrl": "http://www.apache.org/licenses/LICENSE-2.0.html",
      "isPrivate": false,
      "latestVersion": {
        "name": "Community Commons",
        "versionId": "0a03e65a-d94f-47fa-ac40-4e8e054fdcd4",
        "versionNumber": "11.5.0",
        "minSupportedMendixVersion": "10.24.0",
        "publicationDate": "2026-01-13T06:57:14.512Z",
        "downloadUrl": "https://marketplace.mendix.com/v1/versions/0a03e65a-d94f-47fa-ac40-4e8e054fdcd4/download"
      }
    }
  ]
}`

const sampleMultiContentList = `{
  "items": [
    {
      "contentId": 2888,
      "publisher": "Mendix",
      "type": "Module",
      "latestVersion": {
        "name": "Database Connector",
        "versionId": "aaaa",
        "versionNumber": "3.1.0",
        "minSupportedMendixVersion": "9.0.0",
        "publicationDate": "2025-06-01T00:00:00Z"
      }
    },
    {
      "contentId": 170,
      "publisher": "Mendix",
      "type": "Module",
      "latestVersion": {
        "name": "Community Commons",
        "versionId": "bbbb",
        "versionNumber": "11.5.0",
        "minSupportedMendixVersion": "10.24.0",
        "publicationDate": "2026-01-13T00:00:00Z"
      }
    },
    {
      "contentId": 999,
      "publisher": "ACME",
      "type": "Module",
      "latestVersion": {
        "name": "Advanced Database Tools",
        "versionId": "cccc",
        "versionNumber": "1.0.0",
        "minSupportedMendixVersion": "9.0.0",
        "publicationDate": "2024-01-01T00:00:00Z"
      }
    }
  ]
}`

const sampleContent = `{
  "contentId": 170,
  "publisher": "Mendix",
  "type": "Module",
  "categories": [{"name": "Utility"}],
  "supportCategory": "Platform",
  "licenseUrl": "http://www.apache.org/licenses/LICENSE-2.0.html",
  "isPrivate": false,
  "isCompanyApproved": true,
  "latestVersion": {
    "name": "Community Commons",
    "versionId": "0a03e65a-d94f-47fa-ac40-4e8e054fdcd4",
    "versionNumber": "11.5.0",
    "minSupportedMendixVersion": "10.24.0",
    "publicationDate": "2026-01-13T06:57:14.512Z",
    "downloadUrl": "https://marketplace.mendix.com/v1/versions/0a03e65a-d94f-47fa-ac40-4e8e054fdcd4/download"
  }
}`

const sampleVersions = `{
  "items": [
    {
      "name": "Community Commons",
      "versionId": "0a03e65a-d94f-47fa-ac40-4e8e054fdcd4",
      "versionNumber": "11.5.0",
      "minSupportedMendixVersion": "10.24.0",
      "publicationDate": "2026-01-13T06:57:14.512Z",
      "releaseNotes": "<p>We upgraded guava to 33.5.0-jre</p>",
      "versionType": "Regular",
      "downloadUrl": "https://marketplace.mendix.com/v1/versions/0a03e65a-d94f-47fa-ac40-4e8e054fdcd4/download"
    }
  ]
}`

func newMockServer(t *testing.T, handler http.HandlerFunc) (*APIClient, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	return NewAPIClient(ts.Client(), ts.URL), ts
}

func TestSearch_PassesQueryAndLimit(t *testing.T) {
	var gotPath, gotQuery string
	client, _ := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleMultiContentList))
	})

	result, err := client.Search(context.Background(), "database", 3)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if gotPath != "/v1/content" {
		t.Errorf("path: got %q, want /v1/content", gotPath)
	}
	if !strings.Contains(gotQuery, "search=database") {
		t.Errorf("query missing search param: %q", gotQuery)
	}
	if !strings.Contains(gotQuery, "limit="+strconv.Itoa(searchFetchLimit)) {
		t.Errorf("expected API to receive limit=%d, got query %q", searchFetchLimit, gotQuery)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 filtered items, got %d", len(result))
	}
}

func TestSearch_ClientSideFiltering(t *testing.T) {
	client, _ := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleMultiContentList))
	})

	result, err := client.Search(context.Background(), "community", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].ContentID != 170 {
		t.Errorf("expected only Community Commons (170), got %+v", result)
	}
}

func TestSearch_ClientSideFiltering_NoMatch(t *testing.T) {
	client, _ := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleMultiContentList))
	})

	result, err := client.Search(context.Background(), "zzzznonexistent", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results for nonexistent query, got %d", len(result))
	}
}

func TestSearch_ClientSideFiltering_LimitApplied(t *testing.T) {
	client, _ := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleMultiContentList))
	})

	result, err := client.Search(context.Background(), "database", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Errorf("expected limit of 1 applied after filtering, got %d items", len(result))
	}
}

func TestSearch_PublisherFiltering(t *testing.T) {
	client, _ := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(sampleMultiContentList))
	})

	result, err := client.Search(context.Background(), "acme", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].ContentID != 999 {
		t.Errorf("expected ACME item (999), got %+v", result)
	}
}

func TestSearch_NoQueryOrLimit(t *testing.T) {
	var gotQuery string
	client, _ := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(sampleContentList))
	})

	if _, err := client.Search(context.Background(), "", 0); err != nil {
		t.Fatal(err)
	}
	if gotQuery != "" {
		t.Errorf("expected empty query when no search or limit, got %q", gotQuery)
	}
}

func TestGet_ParsesContentDetail(t *testing.T) {
	client, _ := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/content/170" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(sampleContent))
	})

	got, err := client.Get(context.Background(), 170)
	if err != nil {
		t.Fatal(err)
	}
	if got.ContentID != 170 || got.Publisher != "Mendix" {
		t.Errorf("unexpected content: %+v", got)
	}
	if got.LatestVersion == nil || got.LatestVersion.VersionNumber != "11.5.0" {
		t.Errorf("latestVersion not parsed: %+v", got.LatestVersion)
	}
	if got.LatestVersion.DownloadURL != "https://marketplace.mendix.com/v1/versions/0a03e65a-d94f-47fa-ac40-4e8e054fdcd4/download" {
		t.Errorf("downloadUrl not parsed: %s", got.LatestVersion.DownloadURL)
	}
	if !got.IsCompanyApproved {
		t.Error("isCompanyApproved should be true")
	}
	if len(got.Categories) != 1 || got.Categories[0].Name != "Utility" {
		t.Errorf("categories not parsed: %+v", got.Categories)
	}
}

func TestVersions_ParsesList(t *testing.T) {
	client, _ := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/content/170/versions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(sampleVersions))
	})

	got, err := client.GetVersions(context.Background(), 170)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 version, got %d", len(got))
	}
	v := got[0]
	if v.VersionNumber != "11.5.0" {
		t.Errorf("versionNumber: %q", v.VersionNumber)
	}
	if v.MinSupportedMendixVersion != "10.24.0" {
		t.Errorf("minSupportedMendixVersion: %q", v.MinSupportedMendixVersion)
	}
	if !strings.Contains(v.ReleaseNotes, "guava") {
		t.Errorf("releaseNotes: %q", v.ReleaseNotes)
	}
	if v.VersionType != "Regular" {
		t.Errorf("versionType: %q", v.VersionType)
	}
	if v.DownloadURL != "https://marketplace.mendix.com/v1/versions/0a03e65a-d94f-47fa-ac40-4e8e054fdcd4/download" {
		t.Errorf("downloadUrl: %q", v.DownloadURL)
	}
	if v.PublicationDate.IsZero() {
		t.Error("publicationDate did not parse")
	}
	expected := time.Date(2026, 1, 13, 6, 57, 14, 512000000, time.UTC)
	if !v.PublicationDate.Equal(expected) {
		t.Errorf("publicationDate: got %v, want %v", v.PublicationDate, expected)
	}
}

func TestGet_HTTPErrorIsReported(t *testing.T) {
	client, _ := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	})

	_, err := client.Get(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention status: %v", err)
	}
}

func TestGet_InvalidJSONReported(t *testing.T) {
	client, _ := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	})

	_, err := client.Get(context.Background(), 170)
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("error should mention decode: %v", err)
	}
}

func TestNew_UsesDefaultBaseURL(t *testing.T) {
	c := NewAPIClient(http.DefaultClient, "https://marketplace-api.mendix.com")
	if c.baseURL != "https://marketplace-api.mendix.com" {
		t.Errorf("baseURL: %q", c.baseURL)
	}
}
