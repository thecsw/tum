package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thecsw/tum/internal/recipe"
)

var appsCmd = &cobra.Command{
	Use:   "apps",
	Short: "List available (built-in) app recipes",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		names := recipe.List()
		if len(names) == 0 {
			fmt.Println("no recipes found")
		}
		for _, n := range names {
			fmt.Println(n)
		}
		return nil
	},
}

func init() { rootCmd.AddCommand(appsCmd) }
