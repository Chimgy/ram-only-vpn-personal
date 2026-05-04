//go:build darwin

package tunnel

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const confPath = "/tmp/vpnclient.conf"

func wgQuickPath() string {
	if p, err := exec.LookPath("wg-quick"); err == nil {
		return p
	}
	return "/opt/homebrew/bin/wg-quick"
}

// wgEnv returns the current environment with Homebrew prepended to PATH.
// GUI apps on macOS inherit a minimal PATH that excludes /opt/homebrew/bin,
// so wg-quick (a bash script) would fall back to /bin/bash (3.2) — but
// wg-quick requires bash 4+.
func wgEnv() []string {
	newPath := "/opt/homebrew/bin:/usr/local/bin:" + os.Getenv("PATH")
	env := make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "PATH=") {
			env = append(env, e)
		}
	}
	return append(env, "PATH="+newPath)
}

func Up(privateKey, tunnelIP, serverPubkey, serverEndpoint string) error {
	conf := fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/24
DNS = 1.1.1.1, 8.8.8.8

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
`, privateKey, tunnelIP, serverPubkey, serverEndpoint)

	if err := os.WriteFile(confPath, []byte(conf), 0600); err != nil {
		return fmt.Errorf("writing wg config: %w", err)
	}
	cmd := exec.Command("sudo", "-E", wgQuickPath(), "up", confPath)
	cmd.Env = wgEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg-quick up: %s: %w", string(out), err)
	}
	return nil
}

func Down() error {
	cmd := exec.Command("sudo", "-E", wgQuickPath(), "down", confPath)
	cmd.Env = wgEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg-quick down: %s: %w", string(out), err)
	}
	_ = os.Remove(confPath)
	return nil
}
