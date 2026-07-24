package cmd

import (
	"github.com/spf13/cobra"

	"github.com/thecsw/tum/internal/appload"
)

var removeCmd = &cobra.Command{
	Use:   "remove <app>",
	Short: "Remove a tum-managed app from the device",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return appload.Remove(dev(), args[0], cfg.DefaultSuffix, cfg.ApploadRoot)
	},
}

func init() { rootCmd.AddCommand(removeCmd) }
