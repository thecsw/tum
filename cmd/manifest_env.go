package cmd

import (
	"encoding/json"
	"strings"
)

// parseManifestEnv reads external.manifest.json and returns the environment
// map, so tum can launch an app with the same env appload uses (instead of
// hardcoding QTFB_SHIM_MODEL=RM1 etc).
func parseManifestEnv(jsonStr string) map[string]string {
	var m struct {
		Environment map[string]string `json:"environment"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return map[string]string{}
	}
	return m.Environment
}

// envSummary returns a short summary of env vars for logging (e.g.
// "MODEL=RM2,LD_PRELOAD=…").
func envSummary(env map[string]string) string {
	parts := []string{}
	for k, v := range env {
		if k == "HOME" || k == "QTFB_KEY" {
			continue
		}
		// shorten LD_PRELOAD path
		if k == "LD_PRELOAD" {
			v = "…qtfb-shim.so"
		}
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ",")
}
