// SPDX-License-Identifier: Apache-2.0

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mendixlabs/mxcli/internal/marketplace/domain"
)

const searchFetchLimit = 200

const DefaultBaseURL = "https://marketplace-api.mendix.com"

type APIClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewAPIClient(httpClient *http.Client, baseURL string) *APIClient {
	return &APIClient{httpClient: httpClient, baseURL: baseURL}
}

func (c *APIClient) Search(ctx context.Context, query string, limit int) ([]*domain.Content, error) {
	q := url.Values{}
	fetchLimit := limit
	if query != "" {
		q.Set("search", query)
		fetchLimit = searchFetchLimit
	}
	if fetchLimit > 0 {
		q.Set("limit", strconv.Itoa(fetchLimit))
	}
	path := "/v1/content"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var result struct {
		Items []*domain.Content `json:"items"`
	}
	if err := c.get(ctx, path, &result); err != nil {
		return nil, err
	}
	if query != "" {
		matched := filterItems(result.Items, query)
		if limit > 0 && len(matched) > limit {
			matched = matched[:limit]
		}
		return matched, nil
	}
	return result.Items, nil
}

func filterItems(items []*domain.Content, query string) []*domain.Content {
	q := strings.ToLower(query)
	var out []*domain.Content
	for _, item := range items {
		name := ""
		if item.LatestVersion != nil {
			name = strings.ToLower(item.LatestVersion.Name)
		}
		if strings.Contains(name, q) || strings.Contains(strings.ToLower(item.Publisher), q) {
			out = append(out, item)
		}
	}
	return out
}

func (c *APIClient) Get(ctx context.Context, id domain.ContentID) (*domain.Content, error) {
	var out domain.Content
	if err := c.get(ctx, fmt.Sprintf("/v1/content/%d", id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *APIClient) GetVersions(ctx context.Context, id domain.ContentID) ([]*domain.Version, error) {
	var result struct {
		Items []*domain.Version `json:"items"`
	}
	if err := c.get(ctx, fmt.Sprintf("/v1/content/%d/versions", id), &result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

func (c *APIClient) get(ctx context.Context, path string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("marketplace %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("marketplace %s: HTTP %d: %s", path, resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("marketplace %s: decode: %w", path, err)
	}
	return nil
}
