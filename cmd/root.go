package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/CampusTech/axm2snipe/abmclient"
	"github.com/CampusTech/axm2snipe/config"
	"github.com/CampusTech/axm2snipe/notify"
	"github.com/CampusTech/axm2snipe/snipe"
	axmsync "github.com/CampusTech/axm2snipe/sync"
)

var (
	// Cfg is the global application configuration.
	Cfg *config.Config
	// ConfigFile is the path to the config file.
	ConfigFile string
	// Version is the application version, set from main.go.
	Version string

	verbose bool
	debug   bool
)

var log = logrus.New()

// LoadConfig loads config from YAML file with env var overrides, then applies
// CLI flag overrides for flags that were explicitly set.
func LoadConfig(cmd *cobra.Command) error {
	var err error
	Cfg, err = config.Load(ConfigFile)
	if err != nil {
		// Only error if the user explicitly specified --config
		if cmd.Flags().Changed("config") {
			return fmt.Errorf("loading config: %w", err)
		}
		// Default config file not found — create empty config
		Cfg = &config.Config{}
	}

	// CLI flag overrides
	applyBoolFlag(cmd, "dry-run", &Cfg.Sync.DryRun)
	applyBoolFlag(cmd, "force", &Cfg.Sync.Force)
	applyBoolFlag(cmd, "update-only", &Cfg.Sync.UpdateOnly)
	applyStringFlag(cmd, "cache-dir", &Cfg.Sync.CacheDir)

	// Configure log level
	switch {
	case debug:
		log.SetLevel(logrus.DebugLevel)
		abmclient.SetLogLevel(logrus.DebugLevel)
		axmsync.SetLogLevel(logrus.DebugLevel)
		notify.SetLogLevel(logrus.DebugLevel)
		snipe.SetLogLevel(logrus.DebugLevel)
	case verbose:
		log.SetLevel(logrus.InfoLevel)
		abmclient.SetLogLevel(logrus.InfoLevel)
		axmsync.SetLogLevel(logrus.InfoLevel)
		notify.SetLogLevel(logrus.InfoLevel)
		snipe.SetLogLevel(logrus.InfoLevel)
	default:
		log.SetLevel(logrus.WarnLevel)
		abmclient.SetLogLevel(logrus.WarnLevel)
		axmsync.SetLogLevel(logrus.WarnLevel)
		notify.SetLogLevel(logrus.WarnLevel)
		snipe.SetLogLevel(logrus.WarnLevel)
	}

	return nil
}

func applyBoolFlag(cmd *cobra.Command, name string, dst *bool) {
	if cmd.Flags().Changed(name) {
		*dst, _ = cmd.Flags().GetBool(name)
	}
}

func applyStringFlag(cmd *cobra.Command, name string, dst *string) {
	if cmd.Flags().Changed(name) {
		*dst, _ = cmd.Flags().GetString(name)
	}
}

// contextWithSignal returns a context that is canceled on SIGINT/SIGTERM.
func contextWithSignal() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigCh:
			log.Infof("Received signal %v, shutting down...", sig)
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}

// newABMClient creates and returns a new ABM client from global config.
func newABMClient(ctx context.Context) (*abmclient.Client, error) {
	log.Info("Connecting to Apple Business Manager...")
	client, err := abmclient.NewClient(ctx, Cfg.ABM.ClientID, Cfg.ABM.KeyID, Cfg.ABM.PrivateKeyValue())
	if err != nil {
		return nil, fmt.Errorf("creating ABM client: %w", err)
	}
	return client, nil
}

// newSnipeClient creates and returns a new Snipe-IT client from global config.
func newSnipeClient() (*snipe.Client, error) {
	log.Info("Connecting to Snipe-IT...")
	client, err := snipe.NewClient(Cfg.SnipeIT.URL, Cfg.SnipeIT.APIKey)
	if err != nil {
		return nil, fmt.Errorf("creating Snipe-IT client: %w", err)
	}
	client.DryRun = Cfg.Sync.DryRun
	return client, nil
}

// Execute builds the root command, registers subcommands, and runs.
func Execute() {
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	rootCmd := &cobra.Command{
		Use:          "axm2snipe",
		Short:        "Sync devices from Apple Business/School Manager into Snipe-IT",
		Long:         "axm2snipe syncs devices from Apple Business Manager (ABM) / Apple School Manager (ASM) into Snipe-IT asset management.",
		Version:      Version,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return LoadConfig(cmd)
		},
	}

	rootCmd.PersistentFlags().StringVar(&ConfigFile, "config", "settings.yaml", "Path to YAML config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output (INFO level)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Debug output (DEBUG level)")

	syncCmd := NewSyncCmd()
	downloadCmd := NewDownloadCmd()
	setupCmd := NewSetupCmd()
	testCmd := NewTestCmd()
	accessTokenCmd := NewAccessTokenCmd()
	requestCmd := NewRequestCmd()

	// --dry-run: sync, setup
	for _, cmd := range []*cobra.Command{syncCmd, setupCmd} {
		cmd.Flags().Bool("dry-run", false, "Simulate without making changes")
	}

	// --cache-dir: download, sync
	for _, cmd := range []*cobra.Command{downloadCmd, syncCmd} {
		cmd.Flags().String("cache-dir", "", `Directory for cached API responses (default ".cache")`)
	}

	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(accessTokenCmd)
	rootCmd.AddCommand(requestCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
