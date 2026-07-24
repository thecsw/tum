package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thecsw/tum/internal/build"
	"github.com/thecsw/tum/internal/recipe"
)

var buildCmd = &cobra.Command{
	Use:   "build <app>",
	Short: "Cross-compile an app (does not install)",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		r, err := recipe.Load(args[0], flagRecipes)
		if err != nil {
			return err
		}
		bin, err := build.Run(r)
		if err != nil {
			return err
		}
		fmt.Printf("built: %s\n", bin)
		return nil
	},
}

func init() { rootCmd.AddCommand(buildCmd) }
