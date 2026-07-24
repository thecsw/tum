package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// remoteCmd passes its args straight to vellum on the device, interactively.
// e.g. `tum remote add yaft`, `tum remote upgrade`, `tum remote info yaft`.
var remoteCmd = &cobra.Command{
	Use:   "remote <vellum-args...>",
	Short: "Passthrough to vellum on the device (add/upgrade/info/...)",
	Long: `Run vellum on the device with full TTY passthrough, so you can install or
upgrade upstream packages exactly as with vellum directly. Examples:

  tum remote add yaft
  tum remote upgrade
  tum remote info yaft
  tum remote search yaft`,
	DisableFlagParsing: true,
	RunE: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("usage: tum remote <vellum-args...> (e.g. tum remote add yaft)")
		}
		// Prefix with the on-device vellum binary path so PATH-less non-interactive
		// shells still find it.
		argv := append([]string{cfg.Vellum}, args...)
		fmt.Fprintf(os.Stderr, "tum: ssh %s %s\n", cfg.Device, join(argv, " "))
		return dev().Interactive(argv...)
	},
}

func join(ss []string, sep string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

func init() { rootCmd.AddCommand(remoteCmd) }
