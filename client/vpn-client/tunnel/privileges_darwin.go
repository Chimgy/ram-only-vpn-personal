//go:build darwin

package tunnel

import (
	"fmt"
	"os"
	"os/exec"
)

func EnsurePrivileges() error {
	if _, err := os.Stat("/etc/sudoers.d/vpnclient"); err == nil {
		return nil
	}

	wgQuick := wgQuickPath()

	// Write sudoers content to a temp file — avoids all shell/AppleScript quoting issues.
	content := fmt.Sprintf(
		"ALL ALL=(root) NOPASSWD: SETENV: %s up /tmp/vpnclient.conf\nALL ALL=(root) NOPASSWD: SETENV: %s down /tmp/vpnclient.conf\n",
		wgQuick, wgQuick,
	)
	tmp, err := os.CreateTemp("", "vpnclient-sudoers")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	tmp.Close()

	// Simple cp — no special characters in the command, no quoting issues.
	shellCmd := fmt.Sprintf("cp %s /etc/sudoers.d/vpnclient && chmod 440 /etc/sudoers.d/vpnclient", tmpPath)
	script := `do shell script "` + shellCmd + `" with administrator privileges`

	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("privilege setup: %s", string(out))
	}
	return nil
}
