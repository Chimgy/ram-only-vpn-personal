//go:build windows

package tunnel

import (
	"embed"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

//go:embed embed/*
var wintunFiles embed.FS

func ensureDll() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	binDir := filepath.Dir(exePath)
	dllPath := filepath.Join(binDir, "wintun.dll")

	// check if its already there
	if _, err := os.Stat(dllPath); os.IsNotExist(err) {
		// now determine which one to grab from the embed/ dir with runtime.goarch
		srcPath := fmt.Sprintf("embed/%s/wintun.dll", runtime.GOARCH)

		dllBytes, err := wintunFiles.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("architecture %s not supported: %w", runtime.GOARCH, err)
		}

		return os.WriteFile(dllPath, dllBytes, 0644)
	}
	return nil
}

func b64ToHex(b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func Up(privateKey, tunnelIP, serverPubkey, serverEndpoint string) error {
	if err := ensureDll(); err != nil {
		return err
	}
	// Create the Wintun adapter
	// requires Administrator rights (handled by wails.json)
	interfaceName := "VPNClient"
	tunDevice, err := tun.CreateTUN(interfaceName, 1420)
	if err != nil {
		return fmt.Errorf("could not create tun: %w", err)
	}

	// Create the WireGuard device

	logger := device.NewLogger(device.LogLevelSilent, "[VPN] ")
	dev := device.NewDevice(tunDevice, conn.NewDefaultBind(), logger)
	activeDevice = dev

	privHex, err := b64ToHex(privateKey)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}
	pubHex, err := b64ToHex(serverPubkey)
	if err != nil {
		return fmt.Errorf("invalid server pubkey: %w", err)
	}

	// apply the conf
	uapiConf := fmt.Sprintf(`private_key=%s
public_key=%s
endpoint=%s
allowed_ips=0.0.0.0/0
`, privHex, pubHex, serverEndpoint)

	err = dev.IpcSet(uapiConf)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	// bring up the interface
	dev.Up()

	ip := strings.SplitN(tunnelIP, "/", 2)[0]
	// Set the IP address on the interface
	setIP := exec.Command("netsh", "interface", "ip", "set", "address",
		"name="+interfaceName, "static", ip, "255.255.255.0")
	if out, err := setIP.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set IP: %s %w", string(out), err)
	}

	return nil
}

var activeDevice *device.Device

func Down() error {
	if activeDevice != nil {
		activeDevice.Close()
		activeDevice = nil
	}
	return nil
}
