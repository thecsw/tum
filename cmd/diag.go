package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// diagCmd pulls the on-device state for an app: its config.toml, manifest,
// binary info, and a capture+orientation check. This closes the feedback loop
// so you can see what the device actually has without SSHing manually.
var diagCmd = &cobra.Command{
	Use:   "diag [app]",
	Short: "Show on-device config, manifest, and binary state for an app",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		d := dev()

		// 1. Config file (yaft-specific, but useful).
		fmt.Println("=== config.toml ===")
		cfgText, err := d.Run("cat ~/.config/yaft/config.toml 2>/dev/null")
		if err != nil || strings.TrimSpace(cfgText) == "" {
			fmt.Println("(no yaft config.toml found)")
		} else {
			fmt.Print(cfgText)
		}

		// 2. App manifests.
		fmt.Println("\n=== manifests ===")
		manifests, _ := d.Run("cat " + cfg.ApploadRoot + "/*/external.manifest.json 2>/dev/null")
		if strings.TrimSpace(manifests) == "" {
			fmt.Println("(no manifests)")
		} else {
			fmt.Print(manifests)
		}

		// 3. Binary info.
		fmt.Println("\n=== binaries ===")
		bins, _ := d.Run("ls -la " + cfg.ApploadRoot + "/yaft*/yaft* " + cfg.ApploadRoot + "/flower*/flower* 2>/dev/null")
		fmt.Print(bins)

		// 4. Input devices.
		fmt.Println("\n=== input devices ===")
		inputs, _ := d.Run("for d in /dev/input/event*; do echo \"$d -> $(cat /sys/class/input/$(basename $d)/device/name 2>/dev/null)\"; done")
		fmt.Print(inputs)

		// 5. Folio status.
		fmt.Println("\n=== folio ===")
		folio, _ := d.Run("grep -l rM_Keyboard /sys/class/input/*/device/name 2>/dev/null && echo ATTACHED || echo NOT_ATTACHED")
		fmt.Print(folio)

		// 6. Running apps.
		fmt.Println("\n=== running ===")
		running, _ := d.Run("ps w | grep -E 'yaft|flower' | grep -v grep")
		if strings.TrimSpace(running) == "" {
			fmt.Println("(none)")
		} else {
			fmt.Print(running)
		}

		// 7. Launch diagnostic (if app given).
		if len(args) == 1 {
			app := args[0]
			fmt.Printf("\n=== %s launch diagnostic ===\n", app)
			dir := cfg.ApploadRoot + "/" + app + "-" + cfg.DefaultSuffix
			bin := app + "-" + cfg.DefaultSuffix
			diag, _ := d.Run(fmt.Sprintf(
				`cd %s && LD_PRELOAD=/home/root/shims/qtfb-shim.so QTFB_SHIM_MODEL=RM1 `+
					`QTFB_SHIM_INPUT_PATH_NULL=/dev/input/touchscreen0 `+
					`QTFB_SHIM_INITIAL_DISPLAY_MODE=ANIMATE HOME=/home/root ./%s /bin/true 2>&1`,
				dir, bin))
			fmt.Print(diag)
		}
		return nil
	},
}

func init() { rootCmd.AddCommand(diagCmd) }
