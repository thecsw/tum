// Package cmd implements the tum CLI.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"

	"github.com/thecsw/tum/internal/config"
	"github.com/thecsw/tum/internal/device"
)

var (
	flagDevice  string
	flagSuffix  string
	flagDryRun  bool
	flagVerbose bool
	flagConfig  string
	flagRecipes string

	cfg config.Config
)

var rootCmd = &cobra.Command{
	Use:   "tum",
	Short: "tum — a sandy little app manager for the reMarkable",
	Long: `tum builds and installs apps onto a reMarkable over SSH as suffixed sibling
appload entries (e.g. yaft-sandy) that never conflict with vellum-managed apps.

Each tum install lives at <apploadRoot>/<app>-<suffix>/ with its own binary,
icon and manifest, so it shows up as its own launcher tile and survives
vellum upgrades untouched.

Config is read from tum.toml in the current dir (or ~/.tum.toml); all values
can be overridden with global flags.`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		switch cmd.Name() {
		case "help", "completion", "__complete", "__completeNoDesc":
			return nil
		}
		path := flagConfig
		if path == "" {
			if p, ok := findConfig(); ok {
				path = p
			}
		}
		if path != "" {
			if _, err := toml.DecodeFile(path, &cfg); err != nil {
				return fmt.Errorf("decode %s: %w", path, err)
			}
		}
		if flagDevice != "" {
			cfg.Device = flagDevice
		}
		if flagSuffix != "" {
			cfg.DefaultSuffix = flagSuffix
		}
		if cfg.DefaultSuffix == "" {
			cfg.DefaultSuffix = "sandy"
		}
		if cfg.ApploadRoot == "" {
			cfg.ApploadRoot = "/home/root/xovi/exthome/appload"
		}
		if cfg.Vellum == "" {
			cfg.Vellum = "/home/root/.vellum/bin/vellum"
		}
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagDevice, "device", "d", "", "device host (e.g. root@10.11.99.1)")
	rootCmd.PersistentFlags().StringVarP(&flagSuffix, "suffix", "s", "", "install suffix (default: sandy)")
	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "print actions, don't run them")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVar(&flagConfig, "config", "", "path to tum.toml")
	rootCmd.PersistentFlags().StringVar(&flagRecipes, "recipes", "", "dir with override <app>.toml recipes")
}

func findConfig() (string, bool) {
	dir, _ := os.Getwd()
	for i := 0; i < 6; i++ {
		p := filepath.Join(dir, "tum.toml")
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".tum.toml")
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

// dev returns a device client wired to the global config/flags.
func dev() device.Client {
	return device.Client{Host: cfg.Device, DryRun: flagDryRun, Verbose: flagVerbose}
}
