//go:build darwin

package tunnel

import (
	"fmt"
	"os"
	"os/exec"
)

func EnsurePrivileges() error {
	bash := wgBash()
	wgQuick := wgQuickPath()
	expected := fmt.Sprintf(
		"ALL ALL=(root) NOPASSWD: %s %s up /tmp/vpnclient.conf\nALL ALL=(root) NOPASSWD: %s %s down /tmp/vpnclient.conf\n",
		bash, wgQuick, bash, wgQuick,
	)

	if existing, err := os.ReadFile("/etc/sudoers.d/vpnclient"); err == nil && string(existing) == expected {
		return nil
	}

	tmp, err := os.CreateTemp("", "vpnclient-sudoers")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(expected); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	tmp.Close()

	shellCmd := fmt.Sprintf("cp %s /etc/sudoers.d/vpnclient && chmod 440 /etc/sudoers.d/vpnclient", tmpPath)
	script := `do shell script "` + shellCmd + `" with administrator privileges`

	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("privilege setup: %s", string(out))
	}
	return nil
}
