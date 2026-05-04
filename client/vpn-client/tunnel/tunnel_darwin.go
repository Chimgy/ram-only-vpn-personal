//go:build darwin

package tunnel

import (
	"fmt"
	"os"
	"os/exec"
)

const confPath = "/tmp/vpnclient.conf"

func wgQuickPath() string {
	if p, err := exec.LookPath("wg-quick"); err == nil {
		return p
	}
	return "/opt/homebrew/bin/wg-quick"
}

// wgBash returns the Homebrew bash 4+ path.
// macOS ships bash 3.2 at /bin/bash; wg-quick requires 4+.
// Checks Apple Silicon path first, then Intel.
func wgBash() string {
	for _, p := range []string{"/opt/homebrew/bin/bash", "/usr/local/bin/bash"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "/bin/bash"
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
	out, err := exec.Command("sudo", wgBash(), wgQuickPath(), "up", confPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg-quick up: %s: %w", string(out), err)
	}
	return nil
}

func Down() error {
	out, err := exec.Command("sudo", wgBash(), wgQuickPath(), "down", confPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg-quick down: %s: %w", string(out), err)
	}
	_ = os.Remove(confPath)
	return nil
}
