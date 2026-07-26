package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// targetCmd shows the reMarkable's hardware/software specs needed for
// cross-compilation: architecture, CPU, ABI, libc, kernel. Useful as a
// reference when writing build configs or debugging "why won't my binary run."
var targetCmd = &cobra.Command{
	Use:   "target",
	Short: "Show the reMarkable's hardware/software specs for cross-compilation",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		d := dev()

		arch, _ := d.Run("uname -m")
		kernel, _ := d.Run("uname -r")
		cpuinfo, _ := d.Run("cat /proc/cpuinfo")
		glibc, _ := d.Run("/lib/libc.so.6 2>&1 | grep 'release version'")
		compilers, _ := d.Run("which gcc g++ cc tcc clang 2>/dev/null || echo none")

		fmt.Println("=== reMarkable target specs ===")
		fmt.Printf("Architecture:  %s\n", strings.TrimSpace(arch))
		fmt.Printf("Kernel:        %s\n", strings.TrimSpace(kernel))
		fmt.Printf("glibc:         %s\n", strings.TrimSpace(glibc))
		fmt.Printf("On-device compilers: %s\n", strings.TrimSpace(strings.ReplaceAll(compilers, "\n", " ")))

		// Parse CPU details.
		for _, line := range strings.Split(cpuinfo, "\n") {
			if strings.Contains(line, "model name") || strings.Contains(line, "Features") || strings.Contains(line, "CPU part") || strings.Contains(line, "CPU architecture") {
				fmt.Printf("CPU:           %s\n", strings.TrimSpace(line))
			}
		}

		fmt.Println()
		fmt.Println("=== Cross-compile settings ===")
		fmt.Println("Go (pure Go):    GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build")
		fmt.Println("Go (cgo):        GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=1 CC=arm-linux-gnueabihf-gcc go build")
		fmt.Println("C/C++ gcc:       arm-linux-gnueabihf-gcc -march=armv7-a -mfpu=neon -mfloat-abi=hard")
		fmt.Println("C flags:         -march=armv7-a -mfpu=neon -mfloat-abi=hard -mcpu=cortex-a7")
		fmt.Println("Link:            dynamically linked, interpreter /lib/ld-linux-armhf.so.3")
		fmt.Println("Runtime deps:    libc.so.6 (glibc 2.39), libstdc++.so.6, libudev.so.1 (for evdev)")
		fmt.Println()
		fmt.Println("=== Docker build images (tum) ===")
		fmt.Println("  rm2stuff-cross-armhf  — C/C++ apps (yaft) + git-lfs")
		fmt.Println("  tum-go-armhf         — Go cgo apps (flower)")
		fmt.Println("  tum-emulate          — host SDL emulation for testing")
		return nil
	},
}

func init() { rootCmd.AddCommand(targetCmd) }
