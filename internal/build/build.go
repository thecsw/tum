// Package build runs an app's cross-compile (e.g. via Docker) on the host.
package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/thecsw/tum/internal/recipe"
)

// Run builds the app and returns the path to the resulting host binary.
// tumRoot is the tum repo root (used for Docker volume mounts and for resolving
// relative source_dir paths in recipes).
func Run(r *recipe.Recipe, tumRoot string) (string, error) {
	srcDir := r.SourceDir
	if !filepath.IsAbs(srcDir) {
		srcDir = filepath.Join(tumRoot, srcDir)
	}
	cmd, err := r.RenderBuildCmd(tumRoot)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "tum: building %s in %s...\n", r.Name, srcDir)
	c := exec.Command("sh", "-c", cmd)
	c.Dir = srcDir
	c.Stdout = os.Stderr
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("build %s: %w", r.Name, err)
	}
	bin := filepath.Join(srcDir, r.Binary)
	if _, err := os.Stat(bin); err != nil {
		return "", fmt.Errorf("build %s: output not found at %s: %w", r.Name, bin, err)
	}
	return bin, nil
}
