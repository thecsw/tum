package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/thecsw/tum/internal/fbview"
)

var (
	captureRotate bool
	captureOpen   bool
	captureRun    string
	captureWait   int
)

// captureCmd grabs the live qtfb framebuffer from the device, analyzes its
// orientation, and saves a PNG. This closes the visual feedback loop: you can
// verify an app's rendering without eyes on the e-ink screen.
//
// How it works: the qtfb-shim exposes its virtual framebuffer as a shared
// memory file at /dev/shm/qtfb_<key>. tum finds the newest one over SSH,
// downloads it, decodes the RGB565 pixels, detects text orientation, and
// writes a PNG.
var captureCmd = &cobra.Command{
	Use:   "capture [app]",
	Short: "Capture the device's live framebuffer, analyze orientation, save PNG",
	Long: `Capture and analyze the reMarkable's live qtfb framebuffer.

Grabs the newest /dev/shm/qtfb_* shared-memory framebuffer, decodes the RGB565
pixels, detects text orientation (portrait vs landscape-rotated), and writes a
PNG. Closes the visual feedback loop for verifying app rendering.

If an app name is given, launches it first (like appload does), waits, then
captures. Examples:

  tum capture                 # capture whatever's currently on screen
  tum capture yaft            # launch yaft-sandy, wait, capture
  tum capture yaft --run /bin/sh -c 'yes HELLO | head -40; sleep 10'
  tum capture --rotate        # save PNG rotated 90° CW for landscape viewing
  tum capture --open          # open the PNG with the system viewer`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		d := dev()

		// Optionally launch an app first.
		if len(args) == 1 {
			appName := args[0]
			// Find the app's appload dir.
			entries, err := listAppload()
			if err != nil {
				return err
			}
			dir, ok := entries[appName]
			if !ok {
				return fmt.Errorf("app %q not found in appload (try `tum list`)", appName)
			}
			runCmd := captureRun
			if runCmd == "" {
				runCmd = "/bin/sleep " + fmt.Sprint(captureWait+5)
			}
			// Read the manifest's environment (like appload does) so we launch
			// with the correct QTFB_SHIM_MODEL etc. Also generate a QTFB_KEY —
			// appload does this dynamically, and the shim needs it to create its
			// shared-memory framebuffer (without it, RM2 hits "unsupported device").
			manifestPath := dir + "/external.manifest.json"
			envStr, err := d.Run("cat " + shQuote(manifestPath))
			if err != nil {
				return fmt.Errorf("read manifest: %w", err)
			}
			envVars := parseManifestEnv(envStr)
			envVars["QTFB_KEY"] = fmt.Sprintf("%d", time.Now().UnixNano()%100000)
			envVars["HOME"] = "/home/root"

			// Build the env assignment string.
			envAssign := ""
			for k, v := range envVars {
				envAssign += k + "=" + shQuote(v) + " "
			}

			fmt.Fprintf(os.Stderr, "tum: launching %s for %ds (env: %s)…\n", appName, captureWait, envSummary(envVars))
			shell := fmt.Sprintf(
				`cd %s && env %s ./%s %s >/tmp/tum-capture.log 2>&1 &`,
				shQuote(dir), envAssign, shQuote(appName+"-"+cfg.DefaultSuffix), runCmd)
			if _, err := d.Run(shell); err != nil {
				return fmt.Errorf("launch %s: %w", appName, err)
			}
			time.Sleep(time.Duration(captureWait) * time.Second)
		}

		// Find the newest qtfb shm file.
		shm, err := d.Run("ls -t /dev/shm/qtfb_* 2>/dev/null | head -n 1")
		if err != nil {
			return err
		}
		shm = strings.TrimSpace(shm)
		if shm == "" {
			return fmt.Errorf("no qtfb framebuffer found (is a qtfb app running?)")
		}
		fmt.Fprintf(os.Stderr, "tum: capturing %s\n", shm)

		// Download it via a temp file (can't scp /dev/shm directly reliably).
		tmp, err := os.CreateTemp("", "tum-fb-*.rgb565")
		if err != nil {
			return err
		}
		tmpPath := tmp.Name()
		tmp.Close()
		defer os.Remove(tmpPath)

		// Use cat over ssh to get the raw bytes (scp can fail on shm).
		fmt.Fprintf(os.Stderr, "tum: downloading framebuffer...\n")
		shell := "cat " + shm
		out, err := d.Run(shell)
		if err != nil {
			return err
		}
		if err := os.WriteFile(tmpPath, []byte(out), 0644); err != nil {
			return err
		}

		// Analyze.
		data, err := os.ReadFile(tmpPath)
		if err != nil {
			return err
		}
		analysis := fbview.Analyze(data)
		fmt.Printf("tum: %s\n", analysis.String())

		// Write PNG.
		pngPath := "capture.png"
		if err := fbview.WritePNG(data, pngPath, captureRotate); err != nil {
			return fmt.Errorf("write png: %w", err)
		}
		fmt.Printf("tum: saved %s (%d bytes, %dx%d", pngPath, len(data), fbview.FbWidth, fbview.FbHeight)
		if captureRotate {
			fmt.Printf(", rotated 90° CW")
		}
		fmt.Println(")")

		// Optionally open.
		if captureOpen {
			openFile(pngPath)
		}

		// Clean up: kill the launched app if we started one.
		if len(args) == 1 {
			_, _ = d.Run("pkill -f " + shQuote(args[0]+"-"+cfg.DefaultSuffix))
		}
		return nil
	},
}

func init() {
	captureCmd.Flags().BoolVarP(&captureRotate, "rotate", "r", false, "rotate PNG 90° CW for landscape viewing")
	captureCmd.Flags().BoolVarP(&captureOpen, "open", "o", false, "open the PNG with the system viewer")
	captureCmd.Flags().StringVar(&captureRun, "run", "", "command to run in the app (default: sleep)")
	captureCmd.Flags().IntVarP(&captureWait, "wait", "w", 3, "seconds to wait before capturing (with app arg)")
	rootCmd.AddCommand(captureCmd)
}

// listAppload returns a map of app name → on-device dir path.
func listAppload() (map[string]string, error) {
	out, err := dev().Run("ls -d " + cfg.ApploadRoot + "/*-" + cfg.DefaultSuffix + " 2>/dev/null")
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	for _, line := range strings.Fields(out) {
		// line = .../yaft-sandy → app=yaft
		base := line
		if idx := strings.LastIndex(base, "/"); idx >= 0 {
			base = base[idx+1:]
		}
		base = strings.TrimSuffix(base, "-"+cfg.DefaultSuffix)
		m[base] = line
	}
	return m, nil
}

func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

func openFile(path string) {
	// best-effort open
	for _, cmd := range [][]string{
		{"open", path},     // macOS
		{"xdg-open", path}, // Linux
		{"wslview", path},  // WSL
	} {
		if _, err := execLookPath(cmd[0]); err == nil {
			_ = runSimple(cmd[0], cmd[1:]...)
			return
		}
	}
}
