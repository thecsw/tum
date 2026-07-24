package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/thecsw/tum/internal/recipe"
)

// emulateCmd builds and runs the app in host SDL emulation mode (rMlib
// EMULATE=ON): an SDL window emulates the reMarkable screen and input, so you
// can test rotation, refresh and the keyboard without deploying to the device.
//
// On macOS it builds natively with the rM2-stuff 'dev-host' preset (needs
// homebrew cmake/ninja/clang++/sdl2). On Linux it can fall back to the
// tum-emulate Docker container.
var emulateCmd = &cobra.Command{
	Use:   "emulate <app>",
	Short: "Build and run an app in an SDL window emulating the reMarkable",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := recipe.Load(args[0], flagRecipes)
		if err != nil {
			return err
		}
		root := tumRoot()
		srcDir := r.SourceDir
		if !filepath.IsAbs(srcDir) {
			srcDir = filepath.Join(root, srcDir)
		}

		if runtime.GOOS == "darwin" {
			return emulateNative(r, srcDir)
		}
		return emulateDocker(r, root, srcDir)
	},
}

// emulateNative builds with the rM2-stuff 'dev-host' preset and runs the
// resulting SDL binary directly (native macOS window).
func emulateNative(r *recipe.Recipe, srcDir string) error {
	// 1. Configure (idempotent — skip if already configured).
	buildDir := filepath.Join(srcDir, "build/dev-host")
	if _, err := os.Stat(filepath.Join(buildDir, "build.ninja")); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "tum: configuring emulation build (dev-host)...")
		c := exec.Command("cmake", "--preset", "dev-host")
		c.Dir = srcDir
		c.Stdout = os.Stderr
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("cmake configure: %w", err)
		}
	}

	// 2. Build.
	fmt.Fprintf(os.Stderr, "tum: building %s (emulate)...\n", r.Name)
	c := exec.Command("cmake", "--build", buildDir, "--target", r.Name)
	c.Dir = srcDir
	c.Stdout = os.Stderr
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("cmake build: %w", err)
	}

	// 3. Run — SDL opens a native window emulating the rM screen.
	bin := filepath.Join(buildDir, "apps", r.Name, r.Name)
	fmt.Fprintf(os.Stderr, "tum: launching %s in SDL window (Ctrl-C to quit)...\n", bin)
	run := exec.Command(bin)
	run.Stdin = os.Stdin
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	return run.Run()
}

// emulateDocker builds and runs inside the tum-emulate container (Linux).
func emulateDocker(r *recipe.Recipe, root, srcDir string) error {
	bin := filepath.Join(srcDir, "build/emulate/apps", r.Name, r.Name)
	buildSh := fmt.Sprintf(
		`set -e; cd /src/%s; `+
			`cmake -B build/emulate -DEMULATE=ON -DBUILTIN_FONT=ON -G Ninja `+
			`-DCMAKE_C_COMPILER=clang -DCMAKE_CXX_COMPILER=clang++ && `+
			`cmake --build build/emulate --target %s`,
		r.SourceDir, r.Name)
	c := exec.Command("docker", "run", "--rm",
		"-v", root+":/src:cached",
		"-e", "DISPLAY="+os.Getenv("DISPLAY"),
		"tum-emulate", "sh", "-c", buildSh)
	c.Stdout = os.Stderr
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("emulate build: %w", err)
	}
	run := exec.Command("docker", "run", "--rm", "-it",
		"-v", root+":/src:cached",
		"-e", "DISPLAY="+os.Getenv("DISPLAY"),
		"tum-emulate", bin)
	run.Stdin = os.Stdin
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	return run.Run()
}

func init() { rootCmd.AddCommand(emulateCmd) }
