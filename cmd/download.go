package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	axmsync "github.com/CampusTech/axm2snipe/sync"
)

// NewDownloadCmd creates the download command.
func NewDownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download ABM/ASM data to a local cache file",
		Long:  "Fetches all devices and AppleCare coverage from Apple Business Manager / Apple School Manager and saves them to a JSON cache file. Use 'sync --use-cache' to sync from the cache without hitting ABM API rate limits.",
		RunE:  runDownload,
	}

	cmd.Flags().StringP("output", "o", "abm.cache.json", "Output cache file path")

	return cmd
}

func runDownload(cmd *cobra.Command, args []string) error {
	if err := Cfg.ValidateABM(); err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")

	ctx, cancel := contextWithSignal()
	defer cancel()

	abmClient, err := newABMClient(ctx)
	if err != nil {
		return err
	}

	engine := axmsync.NewDownloadEngine(abmClient, Cfg)

	if err := engine.FetchAndSaveCache(ctx, output); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Printf("ABM data saved to %s\n", output)
	return nil
}
