package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewTestCmd creates the test command.
func NewTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Test connections to ABM/ASM and Snipe-IT",
		Long:  "Verifies that the configured API credentials work by making a test request to both Apple Business Manager / Apple School Manager and Snipe-IT.",
		RunE:  runTest,
	}
}

func runTest(cmd *cobra.Command, args []string) error {
	if err := Cfg.Validate(); err != nil {
		return err
	}

	ctx, cancel := contextWithSignal()
	defer cancel()

	abmClient, err := newABMClient(ctx)
	if err != nil {
		return err
	}

	log.Info("Testing ABM connection...")
	total, err := abmClient.ConnectionTest(ctx)
	if err != nil {
		return fmt.Errorf("ABM connection failed: %w", err)
	}
	log.Infof("ABM connection OK (%d total devices)", total)

	snipeClient, err := newSnipeClient()
	if err != nil {
		return err
	}

	log.Info("Testing Snipe-IT connection...")
	models, err := snipeClient.ListAllModels(ctx)
	if err != nil {
		return fmt.Errorf("Snipe-IT connection failed: %w", err)
	}
	log.Infof("Snipe-IT connection OK (%d models found)", len(models))

	fmt.Println("All connections successful!")
	return nil
}
