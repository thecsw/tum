package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thecsw/tum/internal/device"
)

// setupCmd installs the essential xovi/appload infrastructure that sandy
// apps depend on. This is the safety net for when `vellum remove` cascades
// and purges the whole appload/xovi stack (which happened when we removed
// the stock yaft — its deps got pulled too).
//
// The packages: appload, xovi, xovi-extensions, qt-resource-rebuilder.
// These are what make the launcher sidebar appear and provide the qtfb-shim
// LD_PRELOAD framework that sandy apps run under.
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install essential xovi/appload deps on the device (safety net)",
	Long: `Install the vellum packages that sandy apps depend on:
  appload, xovi, xovi-extensions, qt-resource-rebuilder

This is the safety net for when 'vellum remove' cascades and purges the
whole appload/xovi stack. After running this, restart xochitl to get the
launcher sidebar back.`,
	Args: cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		d := dev()

		essential := []string{
			"appload",
			"xovi",
			"xovi-extensions",
			"qt-resource-rebuilder",
		}

		fmt.Println("Checking essential xovi/appload packages…")
		var missing []string
		for _, pkg := range essential {
			installed, err := vellumIsInstalled(d, pkg)
			if err != nil {
				fmt.Printf("  %s: ? (unreachable: %v) — will try to install\n", pkg, err)
				missing = append(missing, pkg)
				continue
			}
			if installed {
				fmt.Printf("  %s: ✓ installed\n", pkg)
			} else {
				fmt.Printf("  %s: ✗ MISSING\n", pkg)
				missing = append(missing, pkg)
			}
		}

		if len(missing) == 0 {
			fmt.Println("\nAll essential packages are installed. Nothing to do.")
			return nil
		}

		fmt.Printf("\nInstalling %d missing package(s): %s\n",
			len(missing), strings.Join(missing, " "))

		addCmd := fmt.Sprintf("%s add %s", cfg.Vellum, strings.Join(missing, " "))
		out, err := d.Run(addCmd)
		if err != nil {
			return fmt.Errorf("vellum add failed: %v\n%s", err, out)
		}
		fmt.Print(out)

		// Restart xochitl so the appload sidebar reappears.
		fmt.Println("\nRestarting xochitl…")
		out, err = d.Run("bash /home/root/xovi/start >/dev/null 2>&1; sleep 4; pidof xochitl")
		if err != nil || strings.TrimSpace(out) == "" {
			fmt.Println("  WARNING: xochitl not running after restart — check device")
		} else {
			fmt.Printf("  xochitl running (pid %s) — sidebar should be back\n", strings.TrimSpace(out))
		}

		return nil
	},
}

// vellumIsInstalled checks whether a package is marked [installed] by vellum.
func vellumIsInstalled(d device.Client, pkg string) (bool, error) {
	out, err := d.Run(cfg.Vellum + " list 2>/dev/null")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, pkg+"-") && strings.Contains(line, "[installed]") {
			return true, nil
		}
	}
	return false, nil
}

func init() { rootCmd.AddCommand(setupCmd) }
