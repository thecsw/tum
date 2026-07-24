// Package recipe loads app build+install recipes (apps/*.toml).
//
// Built-in recipes are embedded so a single `tum` binary is self-contained.
// Override them with --recipes <dir> (files named <app>.toml there win).
package recipe

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"

	"github.com/thecsw/tum/internal/manifest"
)

//go:embed apps/*.toml
var builtinFS embed.FS

// Recipe describes how to build and install one app.
type Recipe struct {
	Name        string        `toml:"name"`
	DisplayName string        `toml:"display_name"`
	SourceDir   string        `toml:"source_dir"` // host path to the source tree
	BuildCmd    string        `toml:"build_cmd"`  // shell command; {{.SourceDir}} substituted
	Binary      string        `toml:"binary"`     // built binary, relative to SourceDir
	BinaryName  string        `toml:"binary_name"` // on-device base name (gets -suffix)
	Icon        string        `toml:"icon"`        // optional icon asset, relative to SourceDir
	Manifest    manifest.Spec `toml:"manifest"`
}

// Load finds a recipe by app name: override dir first, then built-ins.
func Load(name, overrideDir string) (*Recipe, error) {
	if overrideDir != "" {
		p := filepath.Join(overrideDir, name+".toml")
		if _, err := os.Stat(p); err == nil {
			return decodeFile(p)
		}
	}
	data, err := builtinFS.ReadFile("apps/" + name + ".toml")
	if err != nil {
		return nil, fmt.Errorf("no recipe for %q (try `tum apps`): %w", name, err)
	}
	var r Recipe
	if err := toml.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("recipe %s: %w", name, err)
	}
	return &r, nil
}

func decodeFile(p string) (*Recipe, error) {
	var r Recipe
	if _, err := toml.DecodeFile(p, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// List returns the names of all available (built-in) recipes.
func List() []string {
	entries, err := builtinFS.ReadDir("apps")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		out = append(out, strings.TrimSuffix(e.Name(), ".toml"))
	}
	sort.Strings(out)
	return out
}

// RenderBuildCmd substitutes {{.SourceDir}} into BuildCmd.
func (r *Recipe) RenderBuildCmd() (string, error) {
	if r.BuildCmd == "" {
		return "", fmt.Errorf("recipe %s: empty build_cmd", r.Name)
	}
	t, err := template.New("build").Parse(r.BuildCmd)
	if err != nil {
		return "", fmt.Errorf("recipe %s: bad build_cmd template: %w", r.Name, err)
	}
	var b strings.Builder
	if err := t.Execute(&b, map[string]string{"SourceDir": r.SourceDir}); err != nil {
		return "", err
	}
	return b.String(), nil
}
