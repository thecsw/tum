// Package config holds tum's host-side configuration (tum.toml).
package config

// Config is the shape of tum.toml (or ~/.tum.toml). All fields can be
// overridden by global flags (--device, --suffix).
type Config struct {
	// Device is the SSH host, e.g. "root@10.11.99.1".
	Device string `toml:"device"`
	// DefaultSuffix is the suffix applied when -s/--suffix is not given.
	DefaultSuffix string `toml:"default_suffix"`
	// ApploadRoot is the on-device xovi appload directory.
	ApploadRoot string `toml:"appload_root"`
	// Vellum is the on-device path to the vellum binary (for `tum remote`).
	Vellum string `toml:"vellum"`
}
