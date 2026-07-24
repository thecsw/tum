package cmd

import (
	"github.com/spf13/cobra"

	"github.com/thecsw/tum/internal/appload"
	"github.com/thecsw/tum/internal/build"
	"github.com/thecsw/tum/internal/recipe"
)

var installNoBuild bool

var installCmd = &cobra.Command{
	Use:   "install <app>",
	Short: "Build and install an app as a suffixed sibling (e.g. yaft-sandy)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := recipe.Load(args[0], flagRecipes)
		if err != nil {
			return err
		}
		suffix := cfg.DefaultSuffix
		var hostBin string
		if installNoBuild {
			hostBin = r.SourceDir + "/" + r.Binary
		} else {
			hostBin, err = build.Run(r)
			if err != nil {
				return err
			}
		}
		return appload.Install(dev(), r, suffix, hostBin, cfg.ApploadRoot)
	},
}

func init() {
	installCmd.Flags().BoolVar(&installNoBuild, "no-build", false, "skip build, deploy the existing binary")
	rootCmd.AddCommand(installCmd)
}
