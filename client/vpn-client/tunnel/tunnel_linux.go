//go:build linux

package tunnel

import (
	"fmt"
	"os"
	"os/exec"
)

const confPath = "/tmp/vpnclient.conf"

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
	out, err := exec.Command("sudo", "wg-quick", "up", confPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg-quick up: %s: %w", string(out), err)
	}
	return nil
}

func Down() error {
	out, err := exec.Command("sudo", "wg-quick", "down", confPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg-quick down: %s: %w", string(out), err)
	}
	_ = os.Remove(confPath)
	return nil
}
