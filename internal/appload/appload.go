// Package appload installs/lists/removes suffixed sibling apps on the device.
//
// Each tum install lives at <apploadRoot>/<app>-<suffix>/ with its own binary,
// icon and external.manifest.json. Because the dir name ends in -<suffix>,
// tum owns exactly those entries and vellum never touches them — zero conflict.
package appload

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/thecsw/tum/internal/device"
	"github.com/thecsw/tum/internal/manifest"
	"github.com/thecsw/tum/internal/recipe"
)

// Install deploys a built binary as a suffixed sibling appload entry.
func Install(dev device.Client, r *recipe.Recipe, suffix, hostBinary, apploadRoot string) error {
	app := r.BinaryName
	if app == "" {
		app = r.Name
	}
	dir := path.Join(apploadRoot, app+"-"+suffix)
	binName := app + "-" + suffix

	if err := dev.Mkdir(dir); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	if err := dev.Upload(hostBinary, path.Join(dir, binName)); err != nil {
		return fmt.Errorf("upload binary: %w", err)
	}
	iconUploaded := false
	if r.Icon != "" {
		iconSrc := filepath.Join(r.SourceDir, r.Icon)
		if _, err := os.Stat(iconSrc); err == nil {
			if err := dev.Upload(iconSrc, path.Join(dir, "icon.png")); err != nil {
				return fmt.Errorf("upload icon: %w", err)
			}
			iconUploaded = true
		}
	}
	if !iconUploaded {
		// Best-effort: reuse the upstream sibling app's icon (e.g. vellum's
		// yaft/icon.png) so the sandy tile isn't icon-less.
		_, _ = dev.Run("cp " + shQuote(path.Join(apploadRoot, app)) + "/icon.png " +
			shQuote(path.Join(dir, "icon.png")) + " 2>/dev/null || true")
	}
	m := manifest.For(r.DisplayName, binName, suffix, r.Manifest)
	if err := dev.WriteFile(path.Join(dir, "external.manifest.json"), string(m.JSON())); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	if err := dev.Chmod(path.Join(dir, binName), "+x"); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	fmt.Printf("installed: %s — launcher tile %q at %s\n", binName, m.Name, dir)
	return nil
}

// Entry is one tum-managed app on the device.
type Entry struct {
	Dir  string
	App  string // e.g. "yaft-sandy"
	Name string // manifest display name, e.g. "YAFT (sandy)"
}

// List returns all tum-managed apps (appload dirs ending in -<suffix>).
func List(dev device.Client, apploadRoot, suffix string) ([]Entry, error) {
	// One ssh round-trip: print "<dir>\t<manifest>" separated by NUL-free markers.
	script := fmt.Sprintf(
		`for d in %s/*-%s; do [ -d "$d" ] || continue; echo "###$d"; cat "$d/external.manifest.json" 2>/dev/null || echo "{}"; echo; done`,
		shQuote(apploadRoot), shQuote(suffix))
	out, err := dev.Run(script)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, block := range strings.Split(out, "###") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		newline := strings.IndexByte(block, '\n')
		dir, body := block, ""
		if newline >= 0 {
			dir = strings.TrimSpace(block[:newline])
			body = block[newline+1:]
		}
		var m manifest.Manifest
		_ = json.Unmarshal([]byte(body), &m)
		entries = append(entries, Entry{
			Dir:  dir,
			App:  path.Base(dir),
			Name: m.Name,
		})
	}
	return entries, nil
}

// Remove deletes a tum-managed app by name. Accepts either the bare app name
// ("yaft") or the full suffixed name ("yaft-sandy").
func Remove(dev device.Client, app, suffix, apploadRoot string) error {
	name := app
	if !strings.HasSuffix(name, "-"+suffix) {
		name = name + "-" + suffix
	}
	dir := path.Join(apploadRoot, name)
	if err := dev.Remove(dir); err != nil {
		return err
	}
	fmt.Printf("removed: %s (%s)\n", name, dir)
	return nil
}

func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
