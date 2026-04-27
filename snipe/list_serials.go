package snipe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// snipeAssetSerialsPage is a minimal struct for parsing only what we need
// from the /api/v1/hardware response — serials and pagination metadata.
type snipeAssetSerialsPage struct {
	Total int `json:"total"`
	Rows  []struct {
		Serial string `json:"serial"`
	} `json:"rows"`
}

// ListAllSerials returns all asset serial numbers from Snipe-IT, including
// disposed/archived assets. Used to identify devices in Snipe-IT that aren't
// returned by ABM's bulk /v1/orgDevices endpoint (typically released devices)
// so they can be re-fetched individually.
//
// Pages through /api/v1/hardware with limit=500. Empty serials are filtered
// out — they cannot be used for ABM lookups.
func (c *Client) ListAllSerials(ctx context.Context) ([]string, error) {
	var all []string
	offset := 0
	limit := 500

	for {
		url := fmt.Sprintf("%s/api/v1/hardware?limit=%d&offset=%d", c.baseURL, limit, offset)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("creating list serials request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("listing serials: %w", err)
		}

		var page snipeAssetSerialsPage
		decErr := json.NewDecoder(resp.Body).Decode(&page)
		resp.Body.Close()

		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("listing serials: HTTP %d", resp.StatusCode)
		}
		if decErr != nil {
			return nil, fmt.Errorf("decoding serials response: %w", decErr)
		}

		for _, row := range page.Rows {
			s := strings.TrimSpace(row.Serial)
			if s != "" {
				all = append(all, s)
			}
		}

		if len(page.Rows) == 0 || len(all) >= page.Total {
			break
		}
		offset += limit
	}

	return all, nil
}
