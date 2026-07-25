package cmd

import "os/exec"

// execLookPath wraps exec.LookPath for the openFile helper.
func execLookPath(name string) (string, error) { return exec.LookPath(name) }

// runSimple runs a command, discarding output.
func runSimple(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}
