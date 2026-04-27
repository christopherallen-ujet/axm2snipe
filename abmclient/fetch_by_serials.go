package abmclient

import (
	"context"
	"strings"
	"sync"
)

// fetchBySerialWorkers is the number of concurrent single-device fetch goroutines.
// Tuned conservatively to stay well below ABM's rate limits.
const fetchBySerialWorkers = 10

// FetchDevicesBySerials fetches devices individually by serial number using
// the single-device endpoint. This is required for released/unassigned devices
// that don't appear in the bulk /v1/orgDevices endpoint.
//
// Returns successfully-fetched devices. Serials that ABM returns 404 for
// (truly unknown to ABM, never enrolled, or aged out) are silently skipped.
// Other errors are logged at warn level but don't abort the batch.
//
// Uses a worker pool of fetchBySerialWorkers goroutines.
func (c *Client) FetchDevicesBySerials(ctx context.Context, serials []string) ([]Device, error) {
	if len(serials) == 0 {
		return nil, nil
	}

	type result struct {
		device *Device
		serial string
		err    error
	}

	jobs := make(chan string, len(serials))
	results := make(chan result, len(serials))

	workers := fetchBySerialWorkers
	if workers > len(serials) {
		workers = len(serials)
	}

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for serial := range jobs {
				if ctx.Err() != nil {
					results <- result{serial: serial, err: ctx.Err()}
					continue
				}
				device, err := c.GetDevice(ctx, serial)
				results <- result{device: device, serial: serial, err: err}
			}
		}()
	}

	for _, s := range serials {
		jobs <- s
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var devices []Device
	notFound := 0
	otherErrors := 0
	for r := range results {
		if r.err != nil {
			if isNotFoundErr(r.err) {
				notFound++
				log.WithField("serial", r.serial).Debug("Device not in ABM (404), skipping")
			} else {
				otherErrors++
				log.WithError(r.err).WithField("serial", r.serial).Debug("Failed to fetch device by serial")
			}
			continue
		}
		if r.device != nil {
			devices = append(devices, *r.device)
		}
	}

	log.WithFields(map[string]interface{}{
		"requested":   len(serials),
		"fetched":     len(devices),
		"not_in_abm":  notFound,
		"errors":      otherErrors,
	}).Info("Released device fetch complete")

	return devices, nil
}

// isNotFoundErr returns true if an error from the ABM API represents a 404 /
// NOT_FOUND response. The upstream abm package wraps these in standard error
// strings that contain "status=404" or "NOT_FOUND".
func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "status=404") ||
		strings.Contains(msg, "NOT_FOUND") ||
		strings.Contains(msg, "not found")
}
