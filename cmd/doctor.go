package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check device reachability, vellum and appload dir",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		d := dev()
		ok := true

		// 1. reachable?
		if out, err := d.Run("echo tum-ok"); err != nil || strings.TrimSpace(out) != "tum-ok" {
			fmt.Printf("  device %s: UNREACHABLE (%v)\n", cfg.Device, err)
			ok = false
		} else {
			fmt.Printf("  device %s: reachable\n", cfg.Device)
		}

		// 2. vellum present?
		if out, err := d.Run("test -x " + cfg.Vellum + " && echo yes"); err != nil || strings.TrimSpace(out) != "yes" {
			fmt.Printf("  vellum %s: NOT FOUND\n", cfg.Vellum)
			ok = false
		} else {
			fmt.Printf("  vellum %s: present\n", cfg.Vellum)
		}

		// 3. appload dir?
		if out, err := d.Run("test -d " + cfg.ApploadRoot + " && echo yes"); err != nil || strings.TrimSpace(out) != "yes" {
			fmt.Printf("  appload %s: NOT FOUND\n", cfg.ApploadRoot)
			ok = false
		} else {
			fmt.Printf("  appload %s: present\n", cfg.ApploadRoot)
		}

		// 4. suffix siblings already installed?
		out, _ := d.Run("ls -d " + cfg.ApploadRoot + "/*-" + cfg.DefaultSuffix + " 2>/dev/null")
		out = strings.TrimSpace(out)
		if out == "" {
			fmt.Printf("  suffix -%s: none installed yet\n", cfg.DefaultSuffix)
		} else {
			fmt.Printf("  suffix -%s: %s\n", cfg.DefaultSuffix, strings.Join(strings.Fields(out), ", "))
		}

		if !ok {
			return fmt.Errorf("one or more checks failed")
		}
		return nil
	},
}

func init() { rootCmd.AddCommand(doctorCmd) }
