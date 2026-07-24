package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/thecsw/tum/internal/appload"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tum-managed (suffixed) apps on the device",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		entries, err := appload.List(dev(), cfg.ApploadRoot, cfg.DefaultSuffix)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Printf("no tum-managed apps (suffix -%s) on %s\n", cfg.DefaultSuffix, cfg.Device)
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "APP\tTILE\tDIRECTORY")
		for _, e := range entries {
			fmt.Fprintf(w, "%s\t%s\t%s\n", e.App, e.Name, e.Dir)
		}
		return w.Flush()
	},
}

func init() { rootCmd.AddCommand(listCmd) }
