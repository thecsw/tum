// Package manifest models xovi/appload's external.manifest.json.
//
// tum owns the manifest for every app it installs (vellum never sees it), so
// this is the one place the on-disk JSON shape is defined.
package manifest

import "encoding/json"

// Spec is the recipe-side declaration of how an app is launched.
type Spec struct {
	AspectRatio string            `toml:"aspect_ratio"`
	Qtfb        bool              `toml:"qtfb"`
	Environment map[string]string `toml:"environment"`
}

// Manifest is the on-disk external.manifest.json shape appload expects.
type Manifest struct {
	Name        string            `json:"name"`
	Application string            `json:"application"`
	AspectRatio string            `json:"aspectRatio"`
	Qtfb        bool              `json:"qtfb"`
	Environment map[string]string `json:"environment"`
}

// For builds a Manifest for a suffixed install: the display name gets
// " (<suffix>)" appended and application is the suffixed binary name.
func For(displayName, application, suffix string, spec Spec) Manifest {
	name := displayName
	if name == "" {
		name = application
	}
	name = name + " (" + suffix + ")"
	ar := spec.AspectRatio
	if ar == "" {
		ar = "original"
	}
	return Manifest{
		Name:        name,
		Application: application,
		AspectRatio: ar,
		Qtfb:        spec.Qtfb,
		Environment: spec.Environment,
	}
}

// JSON returns pretty-printed JSON for writing to disk.
func (m Manifest) JSON() []byte {
	b, _ := json.MarshalIndent(m, "", "  ")
	return append(b, '\n')
}
