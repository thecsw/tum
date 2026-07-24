// Package device is a thin SSH/SCP client that drives the reMarkable.
//
// It shells out to the system `ssh` and `scp` binaries on purpose: that way it
// reuses your ~/.ssh/config, known_hosts and keys (the same that already work
// for `ssh root@10.11.99.1`), with no Go SSH/known-host/key parsing to maintain.
package device

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Client talks to one reMarkable over SSH.
type Client struct {
	Host    string // e.g. "root@10.11.99.1"
	DryRun  bool
	Verbose bool
}

func (c Client) logf(format string, a ...any) {
	if c.Verbose || c.DryRun {
		fmt.Fprintf(os.Stderr, "tum: "+format+"\n", a...)
	}
}

// sshArgs returns the common non-interactive ssh options.
func (c Client) sshArgs(extra ...string) []string {
	return append([]string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=10"}, extra...)
}

// Run runs a shell command on the device and returns stdout. The shell string
// is passed as a single argument so ssh sends it verbatim to the remote shell
// (avoids the ssh arg-joining re-split: `sh -c echo x` would run `echo`).
func (c Client) Run(shell string) (string, error) {
	c.logf("ssh %s $ %s", c.Host, shell)
	if c.DryRun {
		return "", nil
	}
	cmd := exec.Command("ssh", append(c.sshArgs(), c.Host, shell)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return string(out), fmt.Errorf("ssh %s: %w: %s",
			c.Host, err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

// WriteFile writes content to a remote path by piping it to `cat > path`.
func (c Client) WriteFile(remote, content string) error {
	shell := "cat > " + shQuote(remote)
	c.logf("write %s:%s (%d bytes)", c.Host, remote, len(content))
	if c.DryRun {
		return nil
	}
	cmd := exec.Command("ssh", append(c.sshArgs(), c.Host, shell)...)
	cmd.Stdin = bytes.NewBufferString(content)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("write %s: %w: %s", remote, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// Upload copies a local file to the device via scp.
func (c Client) Upload(local, remote string) error {
	c.logf("scp %s -> %s:%s", local, c.Host, remote)
	if c.DryRun {
		return nil
	}
	cmd := exec.Command("scp",
		append([]string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=10"},
			local, c.Host+":"+remote)...)
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("scp %s -> %s: %w: %s", local, remote, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// Mkdir creates a directory on the device.
func (c Client) Mkdir(path string) error { _, err := c.Run("mkdir -p " + shQuote(path)); return err }

// Chmod changes a file's mode on the device.
func (c Client) Chmod(path, mode string) error {
	_, err := c.Run("chmod " + mode + " " + shQuote(path))
	return err
}

// Remove recursively deletes a path on the device.
func (c Client) Remove(path string) error { _, err := c.Run("rm -rf " + shQuote(path)); return err }

// Interactive runs a command on the device with a TTY (no BatchMode), so
// password prompts and full TTY passthrough work. Used for `tum remote`.
func (c Client) Interactive(args ...string) error {
	argv := append([]string{"-o", "ConnectTimeout=10"}, c.Host)
	argv = append(argv, args...)
	cmd := exec.Command("ssh", argv...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// shQuote single-quotes a string for safe shell interpolation.
func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
