// Package snipe wraps the go-snipeit library with dry-run enforcement and
// convenience methods for the axm2snipe sync process.
package snipe

import (
	"context"
	"fmt"
	"strings"

	snipeit "github.com/CampusTech/go-snipeit"
)

// ErrDryRun is returned when a write operation is attempted in dry-run mode.
var ErrDryRun = fmt.Errorf("write blocked: dry-run mode is enabled")

// Client wraps the go-snipeit client with dry-run enforcement.
type Client struct {
	*snipeit.Client
	DryRun bool
}

// NewClient creates a new Snipe-IT client.
func NewClient(baseURL, apiKey string) (*Client, error) {
	baseURL = strings.TrimRight(baseURL, "/")

	sc, err := snipeit.NewClient(baseURL, apiKey)
	if err != nil {
		return nil, fmt.Errorf("creating snipe-it client: %w", err)
	}

	return &Client{Client: sc}, nil
}

// ListAllModels returns all models from Snipe-IT, handling pagination.
func (c *Client) ListAllModels(ctx context.Context) ([]snipeit.Model, error) {
	var all []snipeit.Model
	offset := 0
	limit := 500

	for {
		resp, _, err := c.Models.ListContext(ctx, &snipeit.ListOptions{Limit: limit, Offset: offset})
		if err != nil {
			return nil, fmt.Errorf("listing models: %w", err)
		}
		all = append(all, resp.Rows...)
		if len(all) >= resp.Total {
			break
		}
		offset += limit
	}

	return all, nil
}

// CreateModel creates a new asset model in Snipe-IT.
func (c *Client) CreateModel(ctx context.Context, model snipeit.Model) (*snipeit.Model, error) {
	if c.DryRun {
		return nil, ErrDryRun
	}
	resp, _, err := c.Models.CreateContext(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("creating model: %w", err)
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("creating model failed: %s", resp.Message)
	}
	return &resp.Payload, nil
}

// ListAllSuppliers returns all suppliers from Snipe-IT, handling pagination.
func (c *Client) ListAllSuppliers(ctx context.Context) ([]snipeit.Supplier, error) {
	var all []snipeit.Supplier
	offset := 0
	limit := 500

	for {
		resp, _, err := c.Suppliers.ListContext(ctx, &snipeit.ListOptions{Limit: limit, Offset: offset})
		if err != nil {
			return nil, fmt.Errorf("listing suppliers: %w", err)
		}
		all = append(all, resp.Rows...)
		if len(all) >= resp.Total {
			break
		}
		offset += limit
	}

	return all, nil
}

// CreateSupplier creates a new supplier in Snipe-IT.
func (c *Client) CreateSupplier(ctx context.Context, name string) (*snipeit.Supplier, error) {
	if c.DryRun {
		return nil, ErrDryRun
	}
	supplier := snipeit.Supplier{}
	supplier.Name = name
	resp, _, err := c.Suppliers.CreateContext(ctx, supplier)
	if err != nil {
		return nil, fmt.Errorf("creating supplier: %w", err)
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("creating supplier failed: %s", resp.Message)
	}
	return &resp.Payload, nil
}

// GetAssetBySerial looks up an asset by serial number.
func (c *Client) GetAssetBySerial(ctx context.Context, serial string) (*snipeit.AssetsResponse, error) {
	resp, _, err := c.Assets.GetAssetBySerialContext(ctx, serial)
	if err != nil {
		return nil, fmt.Errorf("looking up serial %s: %w", serial, err)
	}
	return resp, nil
}

// CreateAsset creates a new hardware asset.
func (c *Client) CreateAsset(ctx context.Context, asset snipeit.Asset) (*snipeit.Asset, error) {
	if c.DryRun {
		return nil, ErrDryRun
	}
	resp, _, err := c.Assets.CreateContext(ctx, asset)
	if err != nil {
		return nil, fmt.Errorf("creating asset: %w", err)
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("creating asset failed: %s", resp.Message)
	}
	return &resp.Payload, nil
}

// PatchAsset partially updates an existing hardware asset by ID.
func (c *Client) PatchAsset(ctx context.Context, id int, asset snipeit.Asset) (*snipeit.Asset, error) {
	if c.DryRun {
		return nil, ErrDryRun
	}
	resp, _, err := c.Assets.PatchContext(ctx, id, asset)
	if err != nil {
		return nil, fmt.Errorf("updating asset %d: %w", id, err)
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("updating asset %d failed: %s", id, resp.Message)
	}
	return &resp.Payload, nil
}

// --- Custom fields setup ---

// FieldDef defines a custom field to create in Snipe-IT.
type FieldDef struct {
	Name        string // display name
	Element     string // form element type: text, textarea, radio, listbox, checkbox
	Format      string // validation format: ANY, DATE, BOOLEAN, etc.
	HelpText    string // help text shown to users
	FieldValues string // newline-separated list of allowed values (for radio/listbox)
}

// SetupFields creates or updates custom fields in Snipe-IT and associates them
// with the given fieldset. Returns a map of field name -> db_column_name.
func (c *Client) SetupFields(fieldsetID int, fields []FieldDef) (map[string]string, error) {
	existing, _, err := c.Fields.List(nil)
	if err != nil {
		return nil, fmt.Errorf("listing existing fields: %w", err)
	}
	existingByName := make(map[string]snipeit.Field)
	for _, f := range existing.Rows {
		existingByName[f.Name] = f
	}

	results := make(map[string]string)

	for _, f := range fields {
		field := snipeit.Field{}
		field.Name = f.Name
		field.Element = f.Element
		field.Format = f.Format
		field.HelpText = f.HelpText
		field.FieldValues = f.FieldValues

		var fieldID int
		var dbColumn string

		if ex, ok := existingByName[f.Name]; ok {
			resp, _, err := c.Fields.Update(ex.ID, field)
			if err != nil {
				return results, fmt.Errorf("updating field %q: %w", f.Name, err)
			}
			if resp.Status != "success" {
				return results, fmt.Errorf("updating field %q: %s", f.Name, resp.Message)
			}
			fieldID = resp.Payload.ID
			dbColumn = resp.Payload.DBColumnName
			if dbColumn == "" {
				dbColumn = ex.DBColumnName
			}
		} else {
			resp, _, err := c.Fields.Create(field)
			if err != nil {
				return results, fmt.Errorf("creating field %q: %w", f.Name, err)
			}
			if resp.Status != "success" {
				return results, fmt.Errorf("creating field %q: %s", f.Name, resp.Message)
			}
			fieldID = resp.Payload.ID
			dbColumn = resp.Payload.DBColumnName
		}

		results[f.Name] = dbColumn

		if fieldsetID > 0 {
			if _, err := c.Fields.Associate(fieldID, fieldsetID); err != nil {
				return results, fmt.Errorf("associating field %q (ID %d) with fieldset %d: %w", f.Name, fieldID, fieldsetID, err)
			}
		}
	}

	// Re-fetch to fill in any missing db_column_name values
	hasMissing := false
	for _, v := range results {
		if v == "" {
			hasMissing = true
			break
		}
	}
	if hasMissing {
		refreshed, _, err := c.Fields.List(nil)
		if err == nil {
			byName := make(map[string]string)
			for _, f := range refreshed.Rows {
				byName[f.Name] = f.DBColumnName
			}
			for name, dbCol := range results {
				if dbCol == "" {
					if col, ok := byName[name]; ok && col != "" {
						results[name] = col
					}
				}
			}
		}
	}

	return results, nil
}
