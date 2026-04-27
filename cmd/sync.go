package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/CampusTech/axm2snipe/abmclient"
	axmsync "github.com/CampusTech/axm2snipe/sync"
)

// NewSyncCmd creates the sync command.
func NewSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync ABM/ASM devices into Snipe-IT",
		Long:  "Fetches devices from Apple Business Manager / Apple School Manager and creates or updates corresponding assets in Snipe-IT.",
		RunE:  runSync,
	}

	cmd.Flags().Bool("force", false, "Ignore timestamps, always update")
	cmd.Flags().String("serial", "", "Sync a single device by serial number (implies --force)")
	cmd.Flags().Bool("use-cache", false, "Use cached data instead of fetching from ABM API")
	cmd.Flags().Bool("update-only", false, "Only update existing assets, never create new ones")
	cmd.Flags().Bool("clear-cache", false, "Delete the cache directory before syncing (auto-enabled when --force is set)")

	return cmd
}

func runSync(cmd *cobra.Command, args []string) error {
	// Apply sync-specific flag overrides before validation so that
	// --use-cache skips the ABM credential check in Cfg.Validate().
	applyBoolFlag(cmd, "force", &Cfg.Sync.Force)
	applyBoolFlag(cmd, "update-only", &Cfg.Sync.UpdateOnly)
	applyBoolFlag(cmd, "use-cache", &Cfg.Sync.UseCache)

	if err := Cfg.Validate(); err != nil {
		return err
	}

	if Cfg.Sync.DryRun {
		log.Info("Running in DRY RUN mode - no changes will be made")
	}

	// Clear the cache directory when --clear-cache is passed OR when force is enabled.
	// We never clear when --use-cache is set, since that would defeat the purpose.
	clearCache, _ := cmd.Flags().GetBool("clear-cache")
	if (clearCache || Cfg.Sync.Force) && !Cfg.Sync.UseCache {
		cacheDir := Cfg.Sync.CacheDir
		if cacheDir == "" {
			cacheDir = ".cache"
		}
		// Resolve to absolute path for clearer logging
		absPath, err := filepath.Abs(cacheDir)
		if err != nil {
			absPath = cacheDir
		}
		if _, statErr := os.Stat(absPath); statErr == nil {
			if err := os.RemoveAll(absPath); err != nil {
				log.WithError(err).WithField("path", absPath).Warn("Could not clear cache directory")
			} else {
				log.WithField("path", absPath).Info("Cleared cache directory before sync")
			}
		} else {
			log.WithField("path", absPath).Debug("Cache directory does not exist — nothing to clear")
		}
	}

	ctx, cancel := contextWithSignal()
	defer cancel()

	var abmClient *abmclient.Client
	if !Cfg.Sync.UseCache {
		var err error
		abmClient, err = newABMClient(ctx)
		if err != nil {
			return err
		}
	}

	snipeClient, err := newSnipeClient()
	if err != nil {
		return err
	}

	engine := axmsync.NewEngine(abmClient, snipeClient, Cfg)

	if Cfg.Sync.UseCache {
		if err := engine.LoadCache(); err != nil {
			return fmt.Errorf("loading cache: %w", err)
		}
	}

	serial, _ := cmd.Flags().GetString("serial")
	if serial != "" {
		Cfg.Sync.Force = true // always force when targeting a single device
	}

	var stats *axmsync.Stats
	if serial != "" {
		stats, err = engine.RunSingle(ctx, serial)
	} else {
		stats, err = engine.Run(ctx)
	}
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Printf("\nSync Results:\n")
	fmt.Printf("  Total devices processed: %d\n", stats.Total)
	fmt.Printf("  Assets created:          %d\n", stats.Created)
	fmt.Printf("  Assets updated:          %d\n", stats.Updated)
	fmt.Printf("  Assets skipped:          %d\n", stats.Skipped)
	fmt.Printf("  Errors:                  %d\n", stats.Errors)
	fmt.Printf("  New models created:      %d\n", stats.ModelNew)

	return nil
}
