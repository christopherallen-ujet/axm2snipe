package abmclient

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CampusTech/abm"
)

// GetPurchaseSourcesFromCache reads devices.json from cacheDir and returns
// the unique purchase sources without making any ABM API calls.
func GetPurchaseSourcesFromCache(cacheDir string) ([]PurchaseSource, error) {
	data, err := os.ReadFile(filepath.Join(cacheDir, "devices.json"))
	if err != nil {
		return nil, fmt.Errorf("reading devices cache: %w", err)
	}
	var devices []Device
	if err := json.Unmarshal(data, &devices); err != nil {
		return nil, fmt.Errorf("parsing devices cache: %w", err)
	}
	orgDevices := make([]abm.OrgDevice, len(devices))
	for i, d := range devices {
		orgDevices[i] = d.OrgDevice
	}
	return collectPurchaseSources(orgDevices), nil
}
